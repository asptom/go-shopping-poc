# Outbox Implementation Improvement Plan

## Executive Summary

This document details the comprehensive refactoring of the outbox infrastructure package (`internal/platform/outbox`) to follow idiomatic Go best practices and improve reliability, maintainability, and observability.

**Scope**: Total rewriting of outbox package with changes to 4 services, platform config updates, and Kubernetes deployment modifications.

**Impact**: High - 40% of services depend on outbox for event reliability

## Current State Analysis

### What Works Well ✅

1. **Outbox Pattern Correctness**: Atomic event writes with same transaction as domain operations
2. **Structured Logging**: Consistent use of prefixed log messages
3. **Context Management**: Service-level context passing (where used)
4. **Error Handling**: Most operations have try/catch with appropriate error wrapping
5. **Graceful Shutdown**: Background thread shutdown on signal
6. **SQL Injection Protection**: Parameterized queries throughout

### Critical Issues ❌

1. **Anti-Pattern `interface{}` Usage**: Writer and Publisher use `interface{}` for database instead of proper interfaces
2. **Zero Test Coverage**: No tests exist for outbox package
3. **Broken Context Propagation**: Publisher uses `context.Background()` for operations
4. **Transaction Rollback Missing**: Write failures don't rollback transactions
5. **Race Condition**: Holding lock during entire batch processing
6. **Hardcoded Query Parameters**: `deleteBatchSize` parameter defined but never used
7. **No Panic Recovery**: Background goroutine can crash service
8. **Error Types Not Wrapped**: No `%w` formatting for error wrapping
9. **Missing Dead Letter Queue**: Failed events are never retried properly
10. **No Metrics/Health Checking**: No observability for outbox health

### Service Usage

| Service | Usage | Outbox Operations |
|---------|-------|-------------------|
| **customer** | Write & Publish | CreateCustomer, UpdateCustomer, DeleteCustomer |
| **product** | Write & Publish | CreateProduct, UpdateProduct, DeleteProduct |
| **product-admin** | Write & Publish | CSV ingestion events |
| **product-loader** | Write | Bulk product operations |

## Improvement Strategy

**Principles**:
- Replace `interface{}` with proper interfaces
- Pass context through all layers
- Always rollback transactions on error
- Add comprehensive error wrapping
- No breaking changes to public APIs (internal refactor only)
- Targeted phases of 1-3 days each

## Service Impact Analysis

### Customer Service (`internal/service/customer/`)
- `service_admin.go`: Uses `WriteEvent()` in transactions
- All three domain events: CustomerCreated, CustomerUpdated, CustomerDeleted
- Transaction boundaries: `BeginTx` → `Operation` → `WriteEvent` → `Commit`

### Product Service (`internal/service/product/`)
- `service_admin.go`: Uses `WriteEvent()` for 9 event types
- Catalog service publishes view events in separate transactions
- Transaction boundaries: Multiple events per operation

### Product Admin Service (`internal/service/product/service_admin.go`)
- Uses both `WriteEvent()` for admin operations and `publishEvent()` helper
- Complex workflow: CSV parsing → MinIO upload → DB insert → Event publish
- Requires transaction coordination between database and outbox

### Product Loader Service (`internal/service/product/repository_bulk.go`)
- Heavy bulk insertion operations
- Minimal event publishing (bulk events)
- Transaction efficiency critical

## Implementation Plan - 8 Phases

### Phase 1: Core Interface Refactoring (2 days)
**Goal**: Replace `interface{}` with proper interfaces, fix context propagation

**Tasks**:
1.1 Update `internal/platform/database/interface.go`:
   - Add `BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)` to Database interface
   - Add `ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)` to Tx interface
   - Remove duplicate `ExecContext` from internal postgreSQL implementation (cleanup)

1.2 Refactor `internal/platform/outbox/writer.go`:
   ```go
   // Replace interface{ db interface{} } with:
   type Writer struct {
       DB database.Database // Proper interface, not interface{}
   }
   ```
   - Update constructor: `NewWriter(db database.Database) *Writer`
   - Replace type switch with interface method calls
   - Add transaction rollback on error (defer pattern)
   - Fix context: pass `tx.Context()` or accept context parameter
   - Wrap errors with `%w` formatting

1.3 Refactor `internal/platform/outbox/publisher.go`:
   - Accept context as parameter to `processOutbox(ctx context.Context)`
   - Replace `context.Background()` with passed-in context
   - Propagate deadline cancellation
   - Use `ctx.Done()` for ticker cancellation instead of separate lifecycle context
   - Remove redundant `lifecycleCtx` usage

1.4 Fix `internal/platform/database/postgresql.go`:
   - Ensure `BeginTx` returns `Tx` interface (already implemented)
   - Add `Context() context.Context` method to Tx interface for context propagation

**Deliverables**:
- `internal/platform/outbox/writer.go` refactored with 0 test failures
- `internal/platform/outbox/publisher.go` using proper context
- Update all interface definitions in `internal/platform/database/interface.go`
- Service code unchanged (internal refactoring only)

**Testing**:
```bash
# Verify services compile
go build ./cmd/customer/...
go build ./cmd/product/...
go build ./cmd/product-admin/...
go build ./cmd/product-loader/...

# Run existing tests
make services-test
```

---

### Phase 2: Transaction Safety & Error Wrapping (2 days)
**Goal**: Ensure all transactions rollback on error, add comprehensive error wrapping

**Tasks**:
2.1 Implement transaction rollback in Writer:
   ```go
   func (w *Writer) WriteEvent(ctx context.Context, tx Tx, evt events.Event) error {
       txStarted := true // Track if we began rollback

       if err := ...; err != nil {
           _ = tx.Rollback()
           return fmt.Errorf("failed to write event: %w", err)
       }
       // Success case
       return nil
   }
   ```

2.2 Add outbox-specific error types:
   ```go
   // internal/platform/outbox/errors.go (new file)
   // OutboxError, WriteFailed, PublishFailed, TransactionError
   // All with %w wrapping
   ```

2.3 Update error handling in all services:
   - Customer service: Wrap events errors
   - Product service: Wrap events errors
   - Product admin: Wrap events errors
   - Product loader: Wrap events errors

2.4 Add context cancellation handling:
   ```go
   if errors.Is(err, context.Canceled) {
       log.Printf("[DEBUG] Outbox: Operation canceled")
       return nil // No-op on cancel
   }
   ```

**Deliverables**:
- `internal/platform/outbox/errors.go` with error types
- Transaction rollback in all WriteEvent calls
- Wrapped errors in all services (4 services × 3-9 events each)
- No leak tests for unclosed transactions

**Testing**:
```bash
# Test transaction safety
go test -run TestTransactionSafety ./internal/platform/outbox/...

# Test error wrapping
go test -run TestErrorWrapping ./internal/platform/outbox/...

# Run all service tests
make services-test
```

---

### Phase 3: Comprehensive Test Suite (3 days)
**Goal**: 80%+ test coverage for outbox package

**Tasks**:
3.1 Write Writer tests (`internal/platform/outbox/writer_test.go`):
   - Test success case: Event written to outbox
   - Test error case: Event not written, transaction rolled back
   - Test context cancellation: Return early on cancel, no writes
   - Test empty payload: Handle nil JSON
   - Test invalid event: Return error, no writes
   - Test sqlx.Tx vs Database.Tx interfaces

3.2 Write Publisher tests (`internal/platform/outbox/publisher_test.go`):
   - Test successful event publishing and deletion
   - Test update failed: Increment retry count, keep event
   - Test context cancellation: Stop batching
   - Test shutdown: Stop processing when flag set
   - Test batch processing: Multiple events in one batch
   - Test retry exhaustion: Move to dead letter queue concept (or re-queue)

3.3 Write Provider tests (`internal/platform/outbox/providers/*_test.go`):
   - WriterProvider: Mock database.Database
   - PublisherProvider: Mock bus.Bus

3.4 Write integration tests (`cmd/*/test_outbox_integration_test.go`):
   - End-to-end test: Write event → DB commit → Publisher picks up → Deletes from DB

**Deliverables**:
- `internal/platform/outbox/writer_test.go` (~250 lines)
- `internal/platform/outbox/publisher_test.go` (~350 lines)
- `internal/platform/outbox/providers/*_test.go` (~100 lines)
- `cmd/customer/cmd_customer_outbox_test.go` or similar

**Coverage Target**:
```
internal/platform/outbox/writer.go    85%+
internal/platform/outbox/publisher.go 80%+
internal/platform/outbox/outbox.go    100%
internal/platform/outbox/config.go    90%+
All providers                          85%+
Total Outbox Package                   80%+
```

**Testing**:
```bash
go test -v -cover ./internal/platform/outbox/...
go test -race ./internal/platform/outbox/...
go test -coverprofile=coverage.out ./internal/platform/outbox/...
go tool cover -html=coverage.out
```

---

### Phase 4: Publisher Optimization (2 days)
**Goal**: Fix race conditions, implement proper batching, add panic recovery

**Tasks**:
4.1 Refactor locking strategy:
   ```go
   func (p *Publisher) processOutbox(ctx context.Context) error {
       batch := p.fetchBatch(ctx) // Lock only for batch fetch
       p.publishBatch(ctx, batch) // Lock per event only
       p.deleteBatch(ctx, batch)
   }
   ```

4.2 Implement batching properly:
   ```go
   // fetchBatch pulls events, respects batchSize
   // publishBatch processes events with error handling
   // deleteBatch removes processed events in single query
   ```

4.3 Add panic recovery wrapper:
   ```go
   func (p *Publisher) runLoop() {
       defer func() {
           if r := recover(); r != nil {
               log.Printf("[CRITICAL] Outbox Publisher PANIC: %v", r)
               // Restart after delay
           }
       }()
       // ... main loop
   }
   ```

4.4 Add metrics/health endpoint:
   ```go
   type PublisherStats struct {
       EventsProcessed    int64
       EventsFailed       int64
       EventsPending      int64
       LastError          string
       HealthStatus       string
   }
   ```

4.5 Add health check method:
   ```go
   func (p *Publisher) Health() bool {
       return atomic.LoadInt64(&p.eventsProcessed) > 0
   }
   ```

**Deliverables**:
- Publisher with per-batch locking
- Batch deletion query using `deleteBatchSize`
- Panic recovery in background loop
- `Health()` method for service health checks
- Prometheus metrics or simple counters

**Testing**:
```bash
# Test concurrent batching
go test -run TestPublisherConcurrency ./internal/platform/outbox/...

# Test panic recovery
go test -run TestPanicRecovery ./internal/platform/outbox/...

# Test health check
go test -run TestPublisherHealth ./internal/platform/outbox/...
```

---

### Phase 5: Service Code Updates (2 days)
**Goal**: Update services to use refactored interfaces, fix context usage

**Tasks**:
5.1 Review all service transaction patterns:
   - Customer: `BeginTx` → `Operation` → `WriteEvent` → `Commit`
   - Product: Same pattern, multiple events
   - Product-admin: Complex multi-step transactions
   - Product-loader: Bulk operations

5.2 Ensure context propagation in all services:
   ```go
   // Before:
   outboxWriter.WriteEvent(ctx, tx, evt)

   // After: Already correct
   outboxWriter.WriteEvent(ctx, tx, evt)
   ```

5.3 Fix Product-admin image processing:
   - Ensure transaction atomicity between DB insert and event publish
   - Add error handling for image upload failures

5.4 Update error handling:
   - Add proper `%w` wrapping
   - Log event publish failures without crashing operation

5.5 Refactor main.go's event handler integration:
   - Ensure Publisher is started with proper context
   - Start publisher as goroutine on service start
   - Graceful shutdown with Publisher.Stop()

**Deliverables**:
- All 4 services compile and run with new outbox interfaces
- Context passed through all WriteEvent calls
- Error messages include causal chain with `%w`
- Service health checks include outbox status

**Testing**:
```bash
# Individual service tests
go test ./cmd/customer/... ./internal/service/customer/...
go test ./cmd/product/... ./internal/service/product/...

# Integration test
go test -run TestFullEventFlow ./cmd/customer/cmd_customer_test.go

# Verify outbox events are published
# Run service, make request, check outbox table for committed events
```

---

### Phase 6: Configuration & Schema Updates (1 day)
**Goal**: Update config structs, database schema, and environment variables

**Tasks**:
6.1 Update `internal/platform/outbox/config.go`:
   ```go
   type Config struct {
       BatchSize           int           `mapstructure:"outbox_batch_size" default:"10"`
       DeleteBatchSize     int           `mapstructure:"outbox_delete_batch_size" default:"50"`
       ProcessInterval     time.Duration `mapstructure:"outbox_process_interval" default:"5s"`
       MaxRetries          int           `mapstructure:"outbox_max_retries" default:"3"`
       DeadLetterQueueSize int           `mapstructure:"outbox_dead_letter_size" default:"1000"`
       OperationTimeout    time.Duration `mapstructure:"outbox_operation_timeout" default:"30s"`
   }
   ```

6.2 Add database migration for outbox improvements:
   ```sql
   -- Add idempotency token
   ALTER TABLE outbox.outbox ADD COLUMN IF NOT EXISTS idempotency_token TEXT UNIQUE;

   -- Add status field
   ALTER TABLE outbox.outbox ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'pending';

   -- Add retry limit
   ALTER TABLE outbox.outbox ADD COLUMN IF NOT EXISTS max_retries INTEGER DEFAULT 3;
   ```

6.3 Update environment variable documentation:
   - `OUTBOX_BATCH_SIZE`: Batch size for publisher
   - `OUTBOX_DELETE_BATCH_SIZE`: Delete batch size
   - `OUTBOX_PROCESS_INTERVAL`: Process interval in seconds
   - `OUTBOX_MAX_RETRIES`: Maximum retry attempts
   - `OUTBOX_DEAD_LETTER_SIZE`: Dead letter queue size

6.4 Update ConfigMap examples in Kubernetes files:
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: platform-config
   data:
     outbox_batch_size: "10"
     outbox_delete_batch_size: "50"
     outbox_process_interval: "5"
   ```

**Deliverables**:
- Updated `internal/platform/outbox/config.go`
- Database migration script: `migrations/001_outbox_enhancements.sql`
- Environment variable documentation in each service's README
- Updated ConfigMap templates in K8s manifests

---

### Phase 7: Kubernetes Deployment Updates (2 days)
**Goal**: Update all service deployments to use new outbox publisher

**Tasks**:
7.1 Create outbox publisher deployment:
   - Separate pod for outbox publisher (not per-service)
   - Use platform ConfigMap for outbox config
   - Resource requests: CPU=100m, Memory=64Mi

7.2 Update each service deployment:
   - Add Publisher startup in `main.go`:
     ```go
     publisher := providers.NewPublisherProvider(platformDB.GetDatabase(), eventBus)
     publisher.GetPublisher().Start()
     defer publisher.GetPublisher().Stop()
     ```
   - Ensure proper cleanup in signal handler
   - Add health check for outbox publisher status

7.3 Update platform ConfigMap (`deploy/k8s/platform/platform-configmap.yaml`):
   ```yaml
   data:
     outbox_batch_size: "10"
     outbox_delete_batch_size: "50"
     outbox_process_interval: "5"
   ```

7.4 Add outbox namespace to platform:
   - Ensure outbox DB table created in shared schema
   - Update RBAC if needed (publisher service account)

7.5 Rollout strategy:
   - Deploy publisher first
   - Roll out services one by one
   - Monitor after each service deployment
   - Kubernetes liveness probes for outbox

**Deliverables**:
- `deploy/k8s/platform/outbox-deployment.yaml` (new file)
- Updated `deploy/k8s/service/customer-deployment.yaml`
- Updated `deploy/k8s/service/product-deployment.yaml`
- Updated `deploy/k8s/service/product-admin-deployment.yaml`
- Updated `deploy/k8s/service/product-loader-deployment.yaml`
- Updated `deploy/k8s/platform/platform-configmap.yaml`

**Testing**:
```bash
# Deploy platform + outbox
make platform

# Start outbox publisher
kubectl apply -f deploy/k8s/platform/outbox-deployment.yaml

# Deploy services one by one
kubectl apply -f deploy/k8s/service/customer-deployment.yaml
kubectl apply -f deploy/k8s/service/product-deployment.yaml
kubectl apply -f deploy/k8s/service/product-admin-deployment.yaml
kubectl apply -f deploy/k8s/service/product-loader-deployment.yaml

# Verify outbox running
kubectl logs -f deployment/outbox-publisher -n platform
```

---

### Phase 8: Documentation & Observability (1 day)
**Goal**: Complete documentation, health checks, and basic monitoring

**Tasks**:
8.1 Create outbox architecture README:
   - Document outbox pattern usage
   - Explain transaction atomicity guarantee
   - Describe publisher operation
   - Troubleshooting guide for common issues

8.2 Add service-specific docs:
   - `cmd/customer/README.md`: Event flow diagram
   - `cmd/product/README.md`: Event flow diagram
   - `cmd/product-admin/README.md`: CSV ingestion events
   - `cmd/product-loader/README.md`: Bulk events

8.3 Add health check endpoints:
   ```go
   // In services:
   router.Get("/health/deep", healthHandler.DeepHealth)
   // Checks: Database OK, Kafka OK, Outbox OK
   ```

8.4 Add log aggregation format:
   - Standardize event publish logs with correlation IDs
   - Example: `[INFO] Outbox: Event published correlation-id=123`

8.5 Success criteria checklist:
   - [ ] All services compile
   - [ ] All outbox package tests pass (80%+ coverage)
   - [ ] Service tests pass
   - [ ] Outbox publisher runs without panics
   - [ ] Event delivery is reliable (no missed events)
   - [ ] Health checks pass for all components
   - [ ] Documentation complete

**Deliverables**:
- `docs/platform/outbox-architecture.md` (new file)
- Event flow diagrams for each service
- Health check endpoint implementations
- Service README updates
- Verification checklist

**Testing**:
```bash
# Smoke test: Create event, verify in outbox, verify published
curl -X POST http://localhost:<port>/api/v1/products \\
  -H "Content-Type: application/json" \\
  -d '{"name":"Test","price":100}'

# Check outbox table
psql $DB_URL -c "SELECT * FROM outbox.outbox WHERE published_at IS NULL"

# Check event broker
kafka-console-consumer --bootstrap-server localhost:9092 --topic ProductEvents

# Health check
curl http://localhost:<port>/health
curl http://localhost:<port>/health/deep

# Verify no missing events
# Make requests, check event count matches request count
```

---

## Total Timeline Estimate

- Phase 1: 2 days
- Phase 2: 2 days
- Phase 3: 3 days
- Phase 4: 2 days
- Phase 5: 2 days
- Phase 6: 1 day
- Phase 7: 2 days
- Phase 8: 1 day

**Total**: 15 days (3 weeks)

**Risk Factors**:
- Phase 3 (testing) may take longer if issues discovered
- Phase 5 (service updates) could expose integration bugs
- Phase 7 (deployment) requires careful K8s testing

## Rollback Plan

Each phase should be deployable independently:
- Tag the version after each phase
- Keep original outbox code in `outbox_legacy.go` until complete
- Enable/disable publisher via environment variable

## Success Metrics

- **Coverage**: 80%+ for outbox package
- **Latency**: Event write latency < 100ms (baseline)
- **Reliability**: < 0.1% events lost (within acceptable SLA)
- **Uptime**: Publisher downtime < 0.5% (target 99.9%)
- **Error Rate**: < 1% event publish failures per day