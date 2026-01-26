# Product Events Plan: Write-Only, Publish-Only Architecture

## Executive Summary

This document outlines a plan to modify the outbox event processing architecture for the product and product-admin services. The goal is to eliminate context cancellation race conditions while maintaining all functionality by having both services write to the outbox table, but only product-admin service publish events.

## Problem Statement

Currently, both `cmd/product` and `cmd/product-admin` services create and start their own `OutboxPublisher` instances that read from the same outbox table in the product database. This creates race conditions during service shutdown, resulting in frequent context cancellation errors:

```
[ERROR] Outbox Publisher: Failed to scan outbox event: context canceled
[ERROR] Outbox Publisher: Error iterating over outbox events: context canceled
```

## Proposed Solution

**Write-Only, Publish-Only Architecture:**
- **cmd/product**: Write events to outbox table only (no publishing)
- **cmd/product-admin**: Write events AND process/publish from outbox table

This centralizes event publishing logic in product-admin while allowing both services to reliably persist events.

## Current Architecture Analysis

### Event Types Published

#### cmd/product Service (Analytics Events - Non-Critical)
- `ProductViewedEvent` - when individual products are viewed
- `ProductSearchExecutedEvent` - when product searches are performed  
- `ProductCategoryViewedEvent` - when product categories are browsed

#### cmd/product-admin Service (Domain Events - Critical)
- `ProductCreatedEvent` - when new products are created
- `ProductUpdatedEvent` - when products are updated
- `ProductDeletedEvent` - when products are deleted
- `ProductImageAddedEvent` - when product images are added
- `ProductImageUpdatedEvent` - when product images are updated
- `ProductImageDeletedEvent` - when product images are deleted
- `ProductIngestionStartedEvent` - when CSV ingestion begins
- `ProductIngestionCompletedEvent` - when CSV ingestion completes
- `ProductIngestionFailedEvent` - when CSV ingestion fails

### Consumer Dependencies
- **EventReader Service**: Consumes from `ProductEvents` topic
- **Analytics Consumers**: No current consumers for product analytics events
- **Domain Consumers**: EventReader processes admin events for system state

### Technical Components

#### Outbox.Writer (✅ Well Separated)
- **Purpose**: Writes events to outbox table within database transactions
- **Dependencies**: Only database connection
- **Current Usage**: Both services use correctly

#### Outbox.Publisher (❌ Duplicated)
- **Purpose**: Reads from outbox table and publishes to Kafka
- **Dependencies**: Database + EventBus + Configuration
- **Current Usage**: Both services start their own instance

## Implementation Plan

### Phase 1: Remove Publishing from Product Service

#### 1.1 Code Changes

**File: `cmd/product/main.go`**
```go
// REMOVE these lines (54-72):
eventBusConfig := event.EventBusConfig{
    WriteTopic: "ProductEvents",
    GroupID:    "ProductCatalogGroup",
}
eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
eventBus := eventBusProvider.GetEventBus()

log.Printf("[DEBUG] Product: Creating outbox provider")
outboxProvider, err := outbox.NewOutboxProvider(platformDB, eventBus)
if err != nil {
    log.Fatalf("Product: Failed to create outbox provider: %v", err)
}
outboxPublisher := outboxProvider.GetOutboxPublisher()
outboxPublisher.Start()
defer outboxPublisher.Stop()

// REPLACE line 81 with:
catalogInfra := &product.CatalogInfrastructure{
    Database:     platformDB,
    OutboxWriter: outbox.NewWriter(platformDB), // Direct creation
}
```

**Import Cleanup:**
```go
// REMOVE these imports:
- "go-shopping-poc/internal/platform/outbox" (if not using writer)
- "go-shopping-poc/internal/platform/event" (removing EventBus)
```

#### 1.2 Configuration Changes

**File: `deploy/k8s/service/product/product-configmap.yaml`**
```yaml
# REMOVE these lines:
# PRODUCT_WRITE_TOPIC: "ProductEvents"
# PRODUCT_GROUP: "ProductGroup"

# KEEP these lines:
PRODUCT_SERVICE_PORT: ":8081"
MINIO_BUCKET: "productimages"
```

**File: `deploy/k8s/service/product/product-deploy.yaml`**
```yaml
# REMOVE from environment variables section (lines 94-121):
# Kafka Configuration section
# Outbox Pattern Configuration section

# KEEP these environment variables:
# - Database Connection Pool Settings
# - CORS Configuration  
# - Service config (PRODUCT_SERVICE_PORT, MINIO_BUCKET, DB_URL)
```

**File: `internal/service/product/config.go`**
```go
// REMOVE from Config struct:
// // Kafka configuration
// WriteTopic string `mapstructure:"product_write_topic" validate:"required"`
// Group      string `mapstructure:"product_group"`

// REMOVE from Validate() function:
// if c.WriteTopic == "" {
//     return errors.New("write topic is required")
// }
```

### Phase 2: Enhance Product-Admin Service

#### 2.1 No Changes Required
- Product-admin continues working as-is
- Already processes events from outbox table
- Will now process both admin + product events

### Phase 3: Optional Enhancements

#### 3.1 Refactor Provider Architecture (Optional)

**Current Problem:** Combined provider in `provider.go` serves no real architectural value - it's just a factory that creates both writer and publisher independently.

**Proposed Solution:** Create explicit, focused providers in dedicated `providers/` subdirectory.

**New File Structure:**
```
internal/platform/outbox/
├── providers/
│   ├── writer.go       # Writer-only provider
│   └── publisher.go    # Publisher-only provider
├── publisher.go        # Publisher implementation
├── writer.go         # Writer implementation
└── config.go         # Configuration
```

**Writer Provider (`providers/writer.go`):**
```go
package providers

import (
    "log"
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/outbox"
)

// WriterProvider provides outbox writer functionality
type WriterProvider interface {
    GetWriter() *outbox.Writer
}

// WriterProviderImpl implements writer-only provider
type WriterProviderImpl struct {
    writer *outbox.Writer
}

// NewWriterProvider creates a writer-only provider
func NewWriterProvider(db database.Database) WriterProvider {
    log.Printf("[DEBUG] WriterProvider: Creating writer-only provider")
    return &WriterProviderImpl{
        writer: outbox.NewWriter(db),
    }
}

func (p *WriterProviderImpl) GetWriter() *outbox.Writer {
    return p.writer
}
```

**Publisher Provider (`providers/publisher.go`):**
```go
package providers

import (
    "log"
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/platform/outbox"
    "go-shopping-poc/internal/platform/config"
)

// PublisherProvider provides outbox publisher functionality
type PublisherProvider interface {
    GetPublisher() *outbox.Publisher
}

// PublisherProviderImpl implements publisher-only provider
type PublisherProviderImpl struct {
    publisher *outbox.Publisher
}

// NewPublisherProvider creates a publisher-only provider
func NewPublisherProvider(db database.Database, eventBus bus.Bus) PublisherProvider {
    // Load platform outbox configuration
    config, err := config.LoadConfig[Config]("platform-outbox")
    if err != nil {
        log.Printf("[ERROR] PublisherProvider: Failed to load outbox config: %v", err)
        return nil, err
    }
    
    log.Printf("[DEBUG] PublisherProvider: Creating publisher-only provider")
    publisher := outbox.NewPublisher(db, eventBus, config.BatchSize, config.DeleteBatchSize, config.ProcessInterval)
    return &PublisherProviderImpl{
        publisher: publisher,
    }
}

func (p *PublisherProviderImpl) GetPublisher() *outbox.Publisher {
    return p.publisher
}
```

**Usage in Product Service (Write-Only):**
```go
import "go-shopping-poc/internal/platform/outbox/providers"

// In cmd/product/main.go:
writerProvider := providers.NewWriterProvider(platformDB)
catalogInfra := &product.CatalogInfrastructure{
    Database:     platformDB,
    OutboxWriter: writerProvider.GetWriter(),
}
```

**Usage in Product-Admin Service (Write + Publish):**
```go
import "go-shopping-poc/internal/platform/outbox/providers"

// In cmd/product-admin/main.go:
writerProvider := providers.NewWriterProvider(platformDB)
publisherProvider := providers.NewPublisherProvider(platformDB, eventBus)
adminInfra := &product.AdminInfrastructure{
    Database:       platformDB,
    OutboxWriter:   writerProvider.GetWriter(),
    OutboxPublisher: publisherProvider.GetPublisher(),
}
```

#### 3.2 Remove Combined Provider and Update All Services
- **Delete `internal/platform/outbox/provider.go` completely** (no backward compatibility)
- **Update import paths in ALL services using outbox providers**
- **Mandatory adoption** - all services must use new `providers/` package

**Services Requiring Updates:**
1. **cmd/customer/main.go** - Currently uses combined provider
2. **cmd/eventreader/main.go** - If it uses outbox provider (check)
3. **cmd/websocket/main.go** - If it uses outbox provider (check)
4. **Any other services** - Audit codebase for additional usages

**Migration Approach:**
- **Single coordinated deployment**: Update all services simultaneously
- **Breaking change acceptance**: No backward compatibility maintained
- **Testing requirement**: All services must pass tests with new provider structure

**Import Path Changes:**
```go
// OLD (remove):
"github.com/.../internal/platform/outbox"

// NEW (use):
"github.com/.../internal/platform/outbox/providers"
```

**Service Update Checklist:**
- [ ] Update import statements
- [ ] Replace `outbox.NewOutboxProvider()` with appropriate provider calls
- [ ] Update `GetOutboxWriter()` → `GetWriter()` 
- [ ] Update `GetOutboxPublisher()` → `GetPublisher()`
- [ ] Test service startup and functionality
- [ ] Update deployment configurations if needed

#### 3.3 Event Processing Priorities (Optional)
- Consider if admin events should be processed before analytics events
- Modify publisher to prioritize by event type if needed

#### 3.2 Event Processing Priorities (Optional)
- Consider if admin events should be processed before analytics events
- Modify publisher to prioritize by event type if needed

## Benefits Analysis

### 1. Eliminates Race Condition
- **Before**: Two OutboxPublisher instances compete for same outbox table
- **After**: Single OutboxPublisher instance processes all events
- **Result**: No more context cancellation errors during shutdown

### 2. Resource Efficiency
- **Product Service**: Fewer goroutines, reduced memory footprint
- **Kafka Connections**: Half the number of producer connections
- **Database Load**: Reduced query contention on outbox table

### 3. Architecture Clarity
- **Clear Responsibilities**: Read-only vs Write/Admin operations
- **Simplified Dependencies**: Product service has fewer external deps
- **Centralized Publishing**: Event processing logic in one place

### 4. Maintainability
- **Single Point of Monitoring**: Event processing metrics in product-admin
- **Easier Debugging**: One place to investigate publishing issues
- **Consistent Configuration**: Publishing settings centralized

## Impact Analysis

### Services Requiring Updates
Based on current outbox provider usage, the following services MUST be updated:

1. **cmd/customer/main.go** - Uses combined provider for event publishing
2. **cmd/product-admin/main.go** - Uses combined provider for write + publish  
3. **cmd/eventreader/main.go** - Check if uses outbox provider
4. **cmd/websocket/main.go** - Check if uses outbox provider
5. **Any other services** - Complete codebase audit required

### Breaking Change Impact
- **HIGH IMPACT**: All services using outbox providers will break
- **Coordinated Deployment Required**: Cannot deploy incrementally
- **Import Path Changes**: All services must update imports
- **Method Name Changes**: `GetOutboxWriter()` → `GetWriter()`, `GetOutboxPublisher()` → `GetPublisher()`

## Risk Assessment

### MEDIUM RISK (Increased due to breaking changes)
- **Breaking Change**: All services must be updated simultaneously
- **Coordinated Deployment**: Higher deployment complexity
- **Service Dependencies**: Multiple services affected simultaneously

### Risk Mitigations
1. **Complete Audit**: Thoroughly identify ALL affected services before implementation
2. **Comprehensive Testing**: Test ALL services with new provider structure
3. **Coordinated Deployment**: Deploy all services simultaneously to staging
4. **Rollback Preparation**: Prepare coordinated rollback procedures
5. **Monitoring**: Monitor entire system during deployment

## Verification Steps

### Pre-Deployment Testing
```bash
# Build verification
make product-build
make product-lint
make product-test

# Unit tests
go test -v ./cmd/product/... ./internal/service/product/...
```

### Post-Deployment Verification
1. **Product Service**:
   - Starts successfully without Kafka configuration
   - APIs respond normally
   - Writes analytics events to outbox table
   - No OutboxPublisher errors in logs

2. **Product-Admin Service**:
   - Continues normal operation
   - Processes both admin + product events
   - Publishes to Kafka successfully

3. **Integration Testing**:
   - EventReader receives all expected events
   - No event processing delays
   - System state consistency maintained

4. **Load Testing**:
   - High traffic scenarios
   - Service shutdown/restart scenarios
   - Event ordering consistency

## Implementation Timeline

### Week 1: Planning & Review
- [ ] Review and approve this plan
- [ ] Discuss any concerns or modifications
- [ ] Audit codebase for ALL services using combined provider
- [ ] Prepare test environments

### Week 2: Implementation (Breaking Changes)
- [ ] Create new providers/ package structure
- [ ] Implement writer and publisher providers
- [ ] **DELETE combined provider** (no backward compatibility)
- [ ] **Update ALL services** (customer, eventreader, websocket, others)
- [ ] Update configuration files for all affected services
- [ ] Run unit and integration tests for ALL services
- [ ] Document rollback procedures for coordinated rollback

### Week 3: Coordinated Deployment
- [ ] **Deploy ALL services simultaneously** to staging environment
- [ ] Verify ALL services functionality works with new providers
- [ ] **Deploy ALL services simultaneously** to production
- [ ] Monitor system behavior across entire system

### Week 4: Optimization
- [ ] Analyze performance metrics
- [ ] Verify race condition elimination in product service
- [ ] Confirm event processing works across all services
- [ ] Update documentation

## Rollback Plan

**BREAKING CHANGE ROLLBACK:**
Since backward compatibility is NOT maintained, rollback requires:

1. **Restore Combined Provider**:
   ```bash
   # Recreate internal/platform/outbox/provider.go from git history
   # Restore old import paths in ALL services
   # Revert import statements to use outbox package directly
   # Redeploy ALL affected services simultaneously
   ```

2. **Service Verification**:
   ```bash
   # Test ALL services start with combined provider
   # Verify event processing across entire system
   # Monitor for race condition errors (original problem)
   ```

**Rollback Complexity**: HIGH (multiple services, coordinated deployment)

## Future Considerations

### Scaling Considerations
- If event volume increases significantly, consider:
  - Dedicated outbox processing service
  - Event batching optimizations
  - Partitioned outbox tables

### Analytics Evolution
- If analytics consumers are added, consider:
  - Separate analytics topic
  - Different processing priorities
  - Retention policies for analytics events

### Monitoring Enhancements
- Add metrics for:
  - Event processing latency per service
  - Outbox table growth rates
  - Publisher success/failure rates

---

**Prepared by**: OpenCode Assistant  
**Date**: 2026-01-26  
**Review Status**: Pending  
**Implementation Status**: Not Started