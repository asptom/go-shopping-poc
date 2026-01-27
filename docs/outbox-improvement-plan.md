# Outbox Architecture Improvement Plan

## Executive Summary

This document outlines a comprehensive architectural improvement to the outbox event processing system. The goal is to eliminate context cancellation race conditions while creating a consistent, maintainable provider architecture that supports both write-only and write+publish service patterns.

## Problem Statement

Currently, multiple services (`cmd/product`, `cmd/product-admin`, `cmd/customer`) create and start their own `OutboxPublisher` instances that read from the same outbox tables, creating race conditions during service shutdown:

```
[ERROR] Outbox Publisher: Failed to scan outbox event: context canceled
[ERROR] Outbox Publisher: Error iterating over outbox events: context canceled
```

Additionally, the current combined provider architecture (`internal/platform/outbox/provider.go`) serves no real architectural value and creates inconsistent usage patterns across services.

## Architectural Decisions

### 1. Write-Only vs Write+Publish Service Pattern

**Decision**: Services will follow one of two clear patterns:

- **Write-Only Services**: Write events to outbox table only, no publishing
  - `cmd/product` (analytics events)
  - `cmd/product-loader` (ingestion events)

- **Write+Publish Services**: Write events AND process/publish from outbox table
  - `cmd/product-admin` (domain events)
  - `cmd/customer` (customer events)

### 2. Separated Provider Architecture

**Decision**: Replace the combined provider with focused, single-responsibility providers:

```
internal/platform/outbox/
├── providers/
│   ├── writer.go       # Writer-only provider
│   └── publisher.go    # Publisher-only provider
├── publisher.go        # Publisher implementation
├── writer.go         # Writer implementation
└── config.go         # Configuration
```

**Benefits**:
- Clear separation of concerns
- Consistent API across all services
- Future extensibility (configuration, monitoring, testing)
- Eliminates architectural confusion

### 3. Breaking Change Acceptance

**Decision**: This will be a breaking change requiring coordinated deployment of all affected services. Backward compatibility will NOT be maintained to ensure clean architectural transition.

## Service Impact Analysis

### Services Requiring Updates

| Service | Current Pattern | Target Pattern | Provider Usage |
|---------|----------------|----------------|---------------|
| `cmd/product` | Write+Publish (inconsistent) | Write-Only | Writer provider only |
| `cmd/product-admin` | Write+Publish | Write+Publish | Both providers |
| `cmd/customer` | Write+Publish | Write+Publish | Both providers |
| `cmd/product-loader` | Write-Only (direct) | Write-Only | Writer provider only |

### Services NOT Requiring Updates

| Service | Reason |
|---------|--------|
| `cmd/websocket` | No outbox usage |
| `cmd/eventreader` | No outbox usage |

## Implementation Plan

### Phase 1: Create New Provider Architecture

**Objective**: Implement the separated provider structure without breaking existing services.

#### 1.1 Create Provider Package Structure

**Files to Create**:
```
internal/platform/outbox/providers/
├── writer.go
└── publisher.go
```

**Implementation Details**:

**`providers/writer.go`**:
```go
package providers

import (
    "log"
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/outbox"
)

type WriterProvider interface {
    GetWriter() *outbox.Writer
}

type WriterProviderImpl struct {
    writer *outbox.Writer
}

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

**`providers/publisher.go`**:
```go
package providers

import (
    "log"
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/platform/outbox"
    "go-shopping-poc/internal/platform/config"
)

type PublisherProvider interface {
    GetPublisher() *outbox.Publisher
}

type PublisherProviderImpl struct {
    publisher *outbox.Publisher
}

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

#### 1.2 Verification Steps

```bash
# Verify new package compiles
go build ./internal/platform/outbox/providers/...

# Run any existing tests
go test -v ./internal/platform/outbox/...
```

### Phase 2: Update All Services to Use New Providers

**Objective**: Migrate all services to use the new provider architecture.

#### 2.1 Service Update Order

**Order**: Update services in dependency order to minimize disruption:

1. **cmd/product-loader** (Simplest - write-only)
2. **cmd/product** (Write-only, removing publisher)
3. **cmd/product-admin** (Write+publish, most complex)
4. **cmd/customer** (Write+publish, similar to product-admin)

#### 2.2 Service Update Pattern

For each service:

**Step 1: Update Imports**
```go
// OLD:
"go-shopping-poc/internal/platform/outbox"

// NEW:
"go-shopping-poc/internal/platform/outbox/providers"
```

**Step 2: Replace Provider Creation**
```go
// OLD:
outboxProvider, err := outbox.NewOutboxProvider(db, eventBus)
outboxWriter := outboxProvider.GetOutboxWriter()
outboxPublisher := outboxProvider.GetOutboxPublisher()

// NEW (Write-Only):
writerProvider := providers.NewWriterProvider(db)
outboxWriter := writerProvider.GetWriter()

// NEW (Write+Publish):
writerProvider := providers.NewWriterProvider(db)
publisherProvider := providers.NewPublisherProvider(db, eventBus)
outboxWriter := writerProvider.GetWriter()
outboxPublisher := publisherProvider.GetPublisher()
```

**Step 3: Update Infrastructure Structs**
```go
// Update infrastructure definitions to use new provider types
// This is in service/*.go files, not main.go files
```

#### 2.3 Service-Specific Changes

**cmd/product-loader**:
- Change from `outbox.NewWriter(platformDB)` to `providers.NewWriterProvider(platformDB).GetWriter()`
- No other changes needed

**cmd/product**:
- Remove EventBus creation and configuration
- Remove OutboxPublisher usage
- Use writer provider only
- Remove Kafka-related environment variables

**cmd/product-admin**:
- Replace combined provider with separate providers
- No functional changes, just provider usage

**cmd/customer**:
- Replace combined provider with separate providers
- No functional changes, just provider usage

#### 2.4 Configuration Updates

**Files to Update**:
- `deploy/k8s/service/product/product-configmap.yaml`
- `deploy/k8s/service/product/product-deploy.yaml`
- `internal/service/product/config.go`

**Changes**:
- Remove Kafka configuration from product service
- Update Config struct validation
- Remove environment variables from deployment

### Phase 3: Remove Combined Provider

**Objective**: Delete the old combined provider to prevent backward compatibility.

#### 3.1 Deletion Steps

```bash
# Delete the old provider file
rm internal/platform/outbox/provider.go
```

#### 3.2 Verification

```bash
# Verify no references to old provider remain
grep -r "NewOutboxProvider" internal/ cmd/
grep -r "GetOutboxWriter" internal/ cmd/
grep -r "GetOutboxPublisher" internal/ cmd/

# Should return no results
```

### Phase 4: Testing and Validation

**Objective**: Comprehensive testing of the new architecture.

#### 4.1 Unit Testing

```bash
# Test each service individually
make product-loader-build && make product-loader-test
make product-build && make product-test
make product-admin-build && make product-admin-test
make customer-build && make customer-test
```

#### 4.2 Integration Testing

```bash
# Build all services
make services-build

# Test all services
make services-test

# Lint all services
make services-lint
```

#### 4.3 Functional Testing

**Test Scenarios**:
1. **Product Service**: Write-only functionality, no publisher errors
2. **Product-Admin Service**: Processes both admin and product events
3. **Customer Service**: Normal event processing
4. **Product-Loader Service**: Write-only ingestion events

#### 4.4 Race Condition Verification

**Test Steps**:
1. Start all services
2. Generate events in product service
3. Verify events appear in outbox table
4. Verify product-admin processes and publishes events
5. Restart services multiple times
6. Check for context cancellation errors (should be none)

### Phase 5: Coordinated Deployment

**Objective**: Deploy all services simultaneously to maintain system consistency.

#### 5.1 Pre-Deployment Checklist

- [ ] All services build successfully
- [ ] All unit tests pass
- [ ] Integration tests pass
- [ ] Configuration files updated
- [ ] Documentation updated
- [ ] Rollback procedures prepared

#### 5.2 Deployment Steps

**Staging Environment**:
```bash
# Deploy all services simultaneously
kubectl apply -f deploy/k8s/service/product/
kubectl apply -f deploy/k8s/service/product-admin/
kubectl apply -f deploy/k8s/service/product-loader/
kubectl apply -f deploy/k8s/service/customer/

# Verify all services start correctly
kubectl get pods -l app=product
kubectl get pods -l app=product-admin
kubectl get pods -l app=product-loader
kubectl get pods -l app=customer
```

**Production Environment**:
- Same steps as staging
- Monitor system behavior for 30 minutes
- Check logs for any errors

#### 5.3 Post-Deployment Verification

**Health Checks**:
```bash
# Verify all services respond
curl http://product-service:8081/health
curl http://product-admin-service:8082/health
curl http://customer-service:8080/health
```

**Event Processing Verification**:
- Check Kafka topics for events
- Verify outbox table processing
- Monitor for any publisher errors

## Risk Assessment and Mitigation

### High Risk Items

1. **Coordinated Deployment Failure**
   - **Mitigation**: Prepare rollback procedures for all services
   - **Monitoring**: Real-time log monitoring during deployment

2. **Configuration Errors**
   - **Mitigation**: Validate all configuration files in staging
   - **Monitoring**: Check service startup logs

3. **Event Processing Breakage**
   - **Mitigation**: Comprehensive testing in staging
   - **Monitoring**: Event pipeline monitoring

### Medium Risk Items

1. **Performance Impact**
   - **Mitigation**: Load testing in staging
   - **Monitoring**: Resource utilization monitoring

2. **Service Dependencies**
   - **Mitigation**: Dependency mapping and testing
   - **Monitoring**: Service health monitoring

## Rollback Plan

### Trigger Conditions
- Multiple services fail to start
- Event processing stops
- Critical errors in logs
- Performance degradation > 50%

### Rollback Steps

**Immediate Rollback**:
```bash
# Restore old provider file
git checkout HEAD~1 -- internal/platform/outbox/provider.go

# Revert all service changes
git checkout HEAD~1 -- cmd/product/main.go
git checkout HEAD~1 -- cmd/product-admin/main.go
git checkout HEAD~1 -- cmd/customer/main.go
git checkout HEAD~1 -- cmd/product-loader/main.go

# Revert configuration changes
git checkout HEAD~1 -- deploy/k8s/service/product/
git checkout HEAD~1 -- internal/service/product/config.go

# Redeploy all services
kubectl apply -f deploy/k8s/service/product/
kubectl apply -f deploy/k8s/service/product-admin/
kubectl apply -f deploy/k8s/service/product-loader/
kubectl apply -f deploy/k8s/service/customer/
```

**Verification**:
- All services start with old architecture
- Event processing resumes
- No race condition errors (original problem returns)

## Success Criteria

### Functional Success
- [ ] All services start without errors
- [ ] Product service writes events but doesn't publish
- [ ] Product-admin service processes all events
- [ ] Customer service continues normal operation
- [ ] Product-loader service writes events correctly

### Architectural Success
- [ ] No context cancellation errors during shutdown
- [ ] Consistent provider usage across all services
- [ ] Clean separation of write-only vs write+publish services
- [ ] No combined provider references remain

### Operational Success
- [ ] Deployment completes without service interruption
- [ ] Event processing continues throughout deployment
- [ ] Performance metrics remain stable
- [ ] Monitoring shows healthy system state

## Future Considerations

### Monitoring Enhancements
- Add metrics for outbox table growth per service
- Track event processing latency
- Monitor publisher success/failure rates

### Scalability Considerations
- Event volume capacity planning
- Potential for dedicated outbox processing service
- Partitioned outbox tables for high-volume scenarios

### Architectural Evolution
- Event priority processing
- Service-specific event batching
- Dynamic publisher scaling

---

**Prepared by**: OpenCode Assistant  
**Date**: 2026-01-27  
**Status**: Ready for Review  
**Implementation Status**: Not Started