# Outbox Pattern Implementation Improvement Plan

## Executive Summary

This document outlines a comprehensive improvement plan for the outbox pattern implementation in the go-shopping-poc project. Based on the user's feedback for a clean slate approach (without backward compatibility concerns), this plan addresses the specific architectural needs of the project's services and provides detailed implementation instructions.

## Current Service Integration Analysis

Based on the updated service usage information:
- **Customer Service**: Write & Publish operations
- **Product Admin Service**: Write & Publish operations  
- **Product Service**: Write only operations
- **Product Loader Service**: Write only operations
- **Eventreader Service**: Does not use outbox
- **WebSocket Service**: Does not use outbox

Note: Product, Product-Admin, and Product-Loader share the same product_db, so Product-Admin should handle publishing for all three services.

## Clean Architecture Approach (No Backward Compatibility)

This plan implements a clean slate design based on the following principles:
1. Remove dependencies on legacy patterns that don't align with current requirements
2. Use modern Go practices for error handling, interfaces, and performance
3. Implement a single publisher per service (Product-Admin) to handle all write operations
4. Eliminate race conditions during service shutdown

## Phased Implementation Plan

### Phase 1: Core Provider Refactoring (Week 1)

**Objective**: Replace legacy provider architecture with modern, clean implementation

**Implementation Details**:

#### 1.1 Refactor Writer Component
- Replace generic `interface{}` with specific database interfaces
- Replace error handling with structured error types
- Add event payload validation and size restriction

```
// New implementation for Writer
type Writer struct {
    db database.Database  // Specific interface instead of interface{}
    // Remove interface{} for transaction type
}

func (w *Writer) WriteEvent(ctx context.Context, tx database.Tx, evt events.Event) error {
    // Validate event before writing
    if err := w.validateEvent(evt); err != nil {
        return fmt.Errorf("invalid event: %w", err)
    }
    
    payload, err := evt.ToJSON()
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }
    
    query := `
        INSERT INTO outbox.outbox (event_type, topic, event_payload)
        VALUES ($1, $2, $3)
    `
    
    // Direct database call without switch statement
    _, err = tx.ExecContext(ctx, query, evt.Type(), evt.Topic(), payload)
    if err != nil {
        return fmt.Errorf("failed to insert event: %w", err)
    }
    
    return nil
}

func (w *Writer) validateEvent(evt events.Event) error {
    // Validate payload size (e.g., less than 10MB)
    payload, err := evt.ToJSON()
    if err != nil {
        return err
    }
    
    if len(payload) > 10*1024*1024 {  // 10MB limit
        return fmt.Errorf("event payload too large: %d bytes", len(payload))
    }
    
    // Validate event type and topic are not empty
    if evt.Type() == "" {
        return fmt.Errorf("event type cannot be empty")
    }
    
    if evt.Topic() == "" {
        return fmt.Errorf("event topic cannot be empty")
    }
    
    return nil
}
```

#### 1.2 Refactor Publisher Component  
- Remove shared publisher instances across services
- Implement single publisher per service with explicit lifecycle management
- Replace context cancellation handling with robust timeout patterns

```
// New implementation for Publisher
type Publisher struct {
    db database.Database
    publisher bus.Bus
    // Remove unused fields like:
    // lifecycleCtx, lifecycleCancel, mu, shuttingDown, operationInProg, etc.
    
    // Add specific configuration that's required
    batchSize        int
    processInterval  time.Duration
    operationTimeout time.Duration
}

func NewPublisher(db database.Database, publisher bus.Bus, cfg Config) *Publisher {
    return &Publisher{
        db:               db,
        publisher:        publisher,
        batchSize:        cfg.BatchSize,
        processInterval:  cfg.ProcessInterval,
        operationTimeout: cfg.OperationTimeout,
    }
}

func (p *Publisher) processOutbox() {
    // Remove thread safety and mutexes, rely on explicit lifecycle management
    
    // Use a separate context for this operation
    operationCtx, operationCancel := context.WithTimeout(context.Background(), p.operationTimeout)
    defer operationCancel()

    // Use prepared statement for query
    query := `SELECT id, event_type, topic, event_payload, created_at, times_attempted, published_at 
              FROM outbox.outbox 
              WHERE published_at IS NULL 
              LIMIT $1`
    
    // Execute with the operation context
    rows, err := p.db.Query(operationCtx, query, p.batchSize)
    if err != nil {
        log.Printf("[ERROR] Failed to read outbox events: %v", err)
        return
    }
    defer func() { _ = rows.Close() }() // Check error if needed
    
    // Process events sequentially within the operation context
    for rows.Next() {
        // ... process individual events
        
        // Publish with operation context
        if err := p.publisher.PublishRaw(operationCtx, topic, eventType, eventPayload); err != nil {
            // Handle publish error and update retry count
            updateQuery := "UPDATE outbox.outbox SET times_attempted = $1 WHERE id = $2"
            _, updateErr := p.db.Exec(operationCtx, updateQuery, retryCount, eventID)
            
            if updateErr != nil {
                log.Printf("[ERROR] Failed to update retry count for event %v: %v", eventID, updateErr)
            }
            
            continue
        }
        
        // Mark as published
        updateQuery := "UPDATE outbox.outbox SET published_at = $1, times_attempted = $2 WHERE id = $3"
        _, updateErr := p.db.Exec(operationCtx, updateQuery, now, retryCount, eventID)
        
        if updateErr != nil {
            log.Printf("[ERROR] Failed to mark event %v as published: %v", eventID, updateErr)
            continue
        }
    }
    
    // No need for complex shutdown handling - use explicit lifecycle management
}
```

#### 1.3 Update Configuration
- Simplified config with specific requirements
- Add default values for all configuration parameters

```
type Config struct {
    BatchSize       int           `mapstructure:"OUTBOX_BATCH_SIZE" default:"10"`
    ProcessInterval time.Duration `mapstructure:"OUTBOX_PROCESS_INTERVAL" default:"5s"`
    OperationTimeout time.Duration `mapstructure:"OUTBOX_OPERATION_TIMEOUT" default:"25s"`
    MaxRetries      int           `mapstructure:"OUTBOX_MAX_RETRIES" default:"3"`
}
```

### Phase 2: Performance and Resource Optimization (Week 2)

**Objective**: Implement performance enhancements and robust resource management

**Implementation Details**:

#### 2.1 Prepared Statement Implementation
- Add prepared statement usage for all database operations in Publisher

```
// Global prepared statements for better performance
var (
    selectUnpublishedStmt *sqlx.Stmt
    updatePublishedStmt   *sqlx.Stmt
    updateRetryStmt       *sqlx.Stmt
)

// Initialize in service startup
func (p *Publisher) initPreparedStatements() error {
    var err error
    
    selectUnpublishedStmt, err = p.db.PreparexContext(
        p.operationTimeout, 
        `SELECT id, event_type, topic, event_payload, created_at, times_attempted, published_at 
         FROM outbox.outbox 
         WHERE published_at IS NULL 
         LIMIT $1`)
    if err != nil {
        return fmt.Errorf("failed to prepare select statement: %w", err)
    }
    
    updatePublishedStmt, err = p.db.PreparexContext(
        p.operationTimeout, 
        `UPDATE outbox.outbox SET published_at = $1, times_attempted = $2 WHERE id = $3`)
    if err != nil {
        return fmt.Errorf("failed to prepare update statement: %w", err)
    }
    
    updateRetryStmt, err = p.db.PreparexContext(
        p.operationTimeout, 
        `UPDATE outbox.outbox SET times_attempted = $1 WHERE id = $2`)
    if err != nil {
        return fmt.Errorf("failed to prepare retry statement: %w", err)
    }
    
    return nil
}
```

#### 2.2 Enhanced Error Handling
- Add specific error types for different failure modes
- Implement robust error wrapping for debugging

```
// Custom error types
type OutboxError struct {
    Code    string
    Message string
    Err     error
}

func (e *OutboxError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *OutboxError) Unwrap() error {
    return e.Err
}

// Specific error constructors
func NewEventPublishError(message string, err error) *OutboxError {
    return &OutboxError{
        Code:    "PUBLISH_ERROR",
        Message: message,
        Err:     err,
    }
}

func NewEventWriteError(message string, err error) *OutboxError {
    return &OutboxError{
        Code:    "WRITE_ERROR", 
        Message: message,
        Err:     err,
    }
}
```

### Phase 3: Lifecycle and Monitoring Enhancements (Week 3)

**Objective**: Implement proper lifecycle management and monitoring capabilities

**Implementation Details**:

#### 3.1 Simplified Lifecycle Management
- Implement explicit service lifecycle with proper context management
- Add graceful shutdown handling 

```
type Publisher struct {
    db      database.Database
    publisher bus.Bus
    // Remove all mutexes, channels, and complex shutdown handling
    
    // Add lifecycle state tracking
    isStarted bool
    stopChan  chan struct{}
}

func (p *Publisher) Start() error {
    if p.isStarted {
        return fmt.Errorf("publisher already started")
    }
    
    p.isStarted = true
    p.stopChan = make(chan struct{})
    
    go p.run()
    return nil
}

func (p *Publisher) Stop() error {
    if !p.isStarted {
        return fmt.Errorf("publisher not started")
    }
    
    // Signal stop
    close(p.stopChan)
    
    // Wait a bit for cleanup
    select {
    case <-time.After(5 * time.Second):
        log.Printf("[WARN] Publisher shutdown timeout")
    case <-time.After(1 * time.Second): // Graceful timeout
    }
    
    p.isStarted = false
    return nil
}

func (p *Publisher) run() {
    ticker := time.NewTicker(p.processInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-p.stopChan:
            log.Printf("[DEBUG] Publisher stopping")
            return
        case <-ticker.C:
            p.processOutbox()
        }
    }
}
```

#### 3.2 Monitoring Integration

```
// Add Prometheus metrics
var (
    eventsProcessedTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "outbox_events_processed_total",
            Help: "Total number of events processed",
        },
        []string{"topic", "status"},
    )
    
    eventProcessingTime = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "outbox_processing_duration_seconds",
            Help: "Event processing duration",
        },
        []string{"topic"},
    )
)

func init() {
    prometheus.MustRegister(eventsProcessedTotal)
    prometheus.MustRegister(eventProcessingTime)
}
```

### Phase 4: Service Coordination and Deployment (Week 4)

**Objective**: Ensure proper service coordination and deployment configuration

**Implementation Details**:

#### 4.1 Product-Admin Service Coordination
- Configure Product-Admin to handle outbox publishing for all three services
- Update startup process to use shared database

```
// In product-admin/main.go
func main() {
    // ... existing setup code ...
    
    // Shared database connection for all services
    sharedDB := platformDB  
    
    // Create outbox publisher once 
    writerProvider := providers.NewWriterProvider(sharedDB)
    publisherProvider := providers.NewPublisherProvider(sharedDB, eventBus)
    outboxPublisher := publisherProvider.GetPublisher()
    outboxPublisher.Start()
    defer outboxPublisher.Stop()
    
    // ... rest of service setup using shared instances
}
```

#### 4.2 Update Service Dependencies
Update all service main files to use the new approach:

```
// In product/main.go
func main() {
    // ... existing code ...
    
    // Product service uses writer only - no publisher needed
    productWriter := writerProvider.GetWriter()
    infrastructure := product.NewProductInfrastructure(db, productWriter)
    
    // ... rest of setup ...
}

// In product-loader/main.go  
func main() {
    // ... existing code ...
    
    // Product loader uses writer only - no publisher needed
    loaderWriter := writerProvider.GetWriter()
    infrastructure := product.NewProductLoaderInfrastructure(db, loaderWriter)
    
    // ... rest of setup ...
}
```

## Service Integration Changes

### Required Service Modifications

1. **Product-Admin Service**:
   - Remove shared publisher instances (create only one for all services)
   - Update service configuration to reflect new publisher setup

2. **Product Service**:
   - Remove publisher initialization
   - Keep writer only for outbox event publication

3. **Product-Loader Service**:
   - Remove publisher initialization  
   - Keep writer only for outbox event publication

4. **All Services**:
   - Update import paths for new error types
   - Update provider initialization to use new APIs

## Deployment and Platform Changes

### Configuration Updates
Update environment variables to include new options:
```
OUTBOX_BATCH_SIZE=10
OUTBOX_PROCESS_INTERVAL=5s
OUTBOX_OPERATION_TIMEOUT=25s
OUTBOX_MAX_RETRIES=3
```

### Monitoring Integration
Add new metrics:
- `outbox_events_processed_total` with labels for topic and status
- `outbox_processing_duration_seconds` histogram for event processing times
- `outbox_errors_total` counter for outbox-related errors

## Risk Assessment

### Low Risk Items
- Interface refinements (only service code changes required)
- Code style improvements  

### Medium Risk Items  
- Lifecycle management changes (require service startup/shutdown testing)
- Performance optimizations (test for regressions)

### High Risk Items
- Complete rewrite of publisher architecture (extensive test coverage needed)
- New error handling approach (requires extensive validation)

## Success Metrics

1. **Service Stability**: 0% race conditions during shutdown
2. **Performance**: 40% reduction in average event processing time
3. **Error Handling**: 100% of errors include contextual information
4. **Reliability**: 99.9% event delivery success rate
5. **Maintainability**: 95% fewer complex state tracking mechanisms