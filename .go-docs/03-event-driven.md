# Event-Driven Architecture

This document describes the event-driven patterns used in this project, including event contracts, the event bus, typed handlers, and the outbox pattern.

## Overview

The project uses an **event-driven architecture** where services communicate asynchronously through Kafka. Key patterns include:

1. **Event Contracts** - Type-safe event definitions
2. **Event Bus** - Abstraction over message transport
3. **Typed Handlers** - Generic event processing
4. **Outbox Pattern** - Transactional event publishing

## Event Contracts

Events are defined in `internal/contracts/events/` as pure data structures.

### Event Interface

All events implement the base `Event` interface:

```go
// internal/contracts/events/common.go
type Event interface {
    Type() string
    Topic() string
    Payload() any
    ToJSON() ([]byte, error)
    GetEntityID() string
    GetResourceID() string
}
```

### Event Factory Interface

For deserialization, events use a factory pattern:

```go
// internal/contracts/events/common.go
type EventFactory[T Event] interface {
    FromJSON([]byte) (T, error)
}
```

### Defining Domain Events

Each bounded context defines its own event types:

```go
// internal/contracts/events/customer.go
package events

// EventType is a typed string for well-known customer events
type EventType string

const (
    CustomerCreated EventType = "customer.created"
    CustomerUpdated EventType = "customer.updated"
    CustomerDeleted EventType = "customer.deleted"
)

// CustomerEvent represents a customer-related event
type CustomerEvent struct {
    ID           string               `json:"id"`
    EventType    EventType            `json:"type"`
    Timestamp    time.Time            `json:"timestamp"`
    EventPayload CustomerEventPayload `json:"payload"`
}

// Implement Event interface
func (e CustomerEvent) Type() string       { return string(e.EventType) }
func (e CustomerEvent) Topic() string      { return "customer-events" }
func (e CustomerEvent) Payload() any       { return e.EventPayload }
func (e CustomerEvent) ToJSON() ([]byte, error) { return json.Marshal(e) }
func (e CustomerEvent) GetEntityID() string { return e.ID }
func (e CustomerEvent) GetResourceID() string { return e.ID }

// CustomerEventFactory implements EventFactory
type CustomerEventFactory struct{}

func (f CustomerEventFactory) FromJSON(data []byte) (CustomerEvent, error) {
    var event CustomerEvent
    err := json.Unmarshal(data, &event)
    return event, err
}

// Convenience constructor
func NewCustomerCreatedEvent(customerID string, details map[string]string) *CustomerEvent {
    return &CustomerEvent{
        ID:        uuid.New().String(),
        EventType: CustomerCreated,
        Timestamp: time.Now(),
        EventPayload: CustomerEventPayload{
            CustomerID: customerID,
            Details:    details,
        },
    }
}
```

**Key patterns:**
1. Use typed constants for event types (not raw strings)
2. Implement all Event interface methods
3. Provide a factory for deserialization
4. Include convenience constructors
5. Use UUID for event IDs

**Reference:** `internal/contracts/events/customer.go`, `product.go`

## Event Bus

The event bus provides an abstraction over the message transport (Kafka).

### Bus Interface

```go
// internal/platform/event/bus/interface.go
type Bus interface {
    Publish(ctx context.Context, topic string, event events.Event) error
    PublishRaw(ctx context.Context, topic string, eventType string, data []byte) error
    StartConsuming(ctx context.Context) error
    WriteTopic() string
    ReadTopics() []string
}
```

### Handler Function Type

```go
// internal/platform/event/bus/interface.go
type HandlerFunc[T events.Event] func(ctx context.Context, event T) error
```

This generic type enables type-safe event handling.

### Kafka Implementation

The concrete implementation uses `segmentio/kafka-go`:

```go
// internal/platform/event/bus/kafka/eventbus.go
type EventBus struct {
    kafkaCfg      *kafka.Config
    writers       map[string]*kafka.Writer
    readers       map[string]*kafka.Reader
    typedHandlers map[string][]func(context.Context, []byte) error
}
```

**Key features:**
- Separate readers per topic
- Generic typed handlers
- Context-aware operations

**Reference:** `internal/platform/event/bus/kafka/eventbus.go`

## Typed Event Handlers

Handlers use Go generics for type safety.

### Handler Interface

```go
// internal/platform/event/handler/interface.go
type EventHandler interface {
    Handle(ctx context.Context, event events.Event) error
    EventType() string
}

type HandlerFactory[T events.Event] interface {
    CreateFactory() events.EventFactory[T]
    CreateHandler() bus.HandlerFunc[T]
}
```

### Concrete Handler Implementation

```go
// internal/service/eventreader/eventhandlers/on_customer_created.go
type OnCustomerCreated struct{}

func NewOnCustomerCreated() *OnCustomerCreated {
    return &OnCustomerCreated{}
}

func (h *OnCustomerCreated) Handle(ctx context.Context, event events.Event) error {
    // Type assertion to concrete type
    var customerEvent events.CustomerEvent
    switch e := event.(type) {
    case events.CustomerEvent:
        customerEvent = e
    case *events.CustomerEvent:
        customerEvent = *e
    default:
        log.Printf("[ERROR] Expected CustomerEvent, got %T", event)
        return nil
    }

    // Filter by event type
    if customerEvent.EventType != events.CustomerCreated {
        log.Printf("[DEBUG] Ignoring non-CustomerCreated event: %s", customerEvent.EventType)
        return nil
    }

    return h.processCustomerCreated(ctx, customerEvent)
}

func (h *OnCustomerCreated) processCustomerCreated(ctx context.Context, event events.CustomerEvent) error {
    // Business logic here
    return nil
}

// HandlerFactory implementation
func (h *OnCustomerCreated) CreateFactory() events.EventFactory[events.CustomerEvent] {
    return events.CustomerEventFactory{}
}

func (h *OnCustomerCreated) CreateHandler() bus.HandlerFunc[events.CustomerEvent] {
    return func(ctx context.Context, event events.CustomerEvent) error {
        return h.Handle(ctx, event)
    }
}

// Interface compliance checks
var _ handler.EventHandler = (*OnCustomerCreated)(nil)
var _ handler.HandlerFactory[events.CustomerEvent] = (*OnCustomerCreated)(nil)
```

**Key patterns:**
1. Type assertion with switch for flexibility
2. Event type filtering inside handler
3. Separate business logic method (`processCustomerCreated`)
4. Implement both `EventHandler` and `HandlerFactory`
5. Compile-time interface checks

**Reference:** `internal/service/eventreader/eventhandlers/on_customer_created.go`

### Registering Handlers

Handlers are registered in the service:

```go
// internal/service/eventreader/service.go
func registerEventHandlers(service *EventReaderService) error {
    // Register typed handlers
    err := service.RegisterHandler(
        events.CustomerEventFactory{},
        func(ctx context.Context, event events.CustomerEvent) error {
            handler := NewOnCustomerCreated()
            return handler.Handle(ctx, event)
        },
    )
    if err != nil {
        return fmt.Errorf("failed to register customer created handler: %w", err)
    }
    
    return nil
}
```

Or using a helper function:

```go
// internal/platform/service/event.go
func RegisterHandler[T events.Event](
    s Service,
    factory events.EventFactory[T],
    handler bus.HandlerFunc[T],
) error {
    // Get event bus from service
    var eventBus bus.Bus
    if es, ok := s.(EventService); ok {
        eventBus = es.EventBus()
    } else {
        return &ServiceError{...}
    }

    // Subscribe with concrete implementation
    if eb, ok := eventBus.(*kafka.EventBus); ok {
        kafka.SubscribeTyped(eb, factory, handler)
        return nil
    }
    return &ServiceError{...}
}
```

**Reference:** `internal/platform/service/event.go`

## Outbox Pattern

The outbox pattern ensures events are published reliably, even if the service crashes after database commit.

### How It Works

1. Business operation writes to database **AND** outbox table in same transaction
2. Separate publisher process polls outbox table
3. Publisher sends events to Kafka
4. Publisher marks events as published

### Outbox Writer

```go
// internal/platform/outbox/writer.go
type Writer struct {
    db database.Database
}

func NewWriter(db database.Database) *Writer {
    return &Writer{db: db}
}

func (w *Writer) WriteEvent(ctx context.Context, tx database.Tx, evt events.Event) error {
    if tx == nil {
        return errors.New("tx must be non-nil")
    }

    payload, err := evt.ToJSON()
    if err != nil {
        return err
    }

    query := `
        INSERT INTO outbox.outbox (event_type, topic, event_payload)
        VALUES ($1, $2, $3)
    `

    _, err = tx.Exec(ctx, query, evt.Type(), evt.Topic(), payload)
    if err != nil {
        return WrapWithContext(ErrWriteFailed, "failed to write event to outbox")
    }

    return nil
}
```

**Key requirements:**
- Must use transaction passed from caller
- Serialize event to JSON
- Insert into outbox table within same transaction

**Reference:** `internal/platform/outbox/writer.go`

### Repository Integration

Repositories use the outbox writer within transactions:

```go
// internal/service/customer/repository.go
func (r *customerRepository) insertCustomerWithRelations(ctx context.Context, customer *Customer) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    committed := false
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()

    // Insert customer record
    if err := r.insertCustomerRecordInTransaction(ctx, tx, customer); err != nil {
        return err
    }

    // Write event to outbox (same transaction!)
    evt := events.NewCustomerCreatedEvent(customer.CustomerID, nil)
    if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
        return fmt.Errorf("failed to publish customer created event: %w", err)
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    committed = true
    
    return nil
}
```

**Critical pattern:**
- Begin transaction
- Defer rollback (set committed flag on success)
- All database operations use same transaction
- Write event to outbox before commit
- Commit transaction

**Reference:** `internal/service/customer/repository.go`

### Outbox Publisher

The publisher runs as a background process:

```go
// internal/platform/outbox/publisher.go
type Publisher struct {
    db     database.Database
    bus    bus.Bus
    ticker *time.Ticker
    quit   chan struct{}
}

func (p *Publisher) Start() {
    go p.run()
}

func (p *Publisher) run() {
    for {
        select {
        case <-p.ticker.C:
            if err := p.publishPending(); err != nil {
                log.Printf("[ERROR] Failed to publish pending events: %v", err)
            }
        case <-p.quit:
            return
        }
    }
}

func (p *Publisher) publishPending() error {
    // Query unpublished events
    // Publish each to Kafka
    // Mark as published in database
    return nil
}
```

**Reference:** `internal/platform/outbox/publisher.go`

## Event Publishing Directly

For non-transactional scenarios, publish directly to the event bus:

```go
func (s *CustomerService) emitEvent(ctx context.Context, event events.Event) error {
    if err := s.infrastructure.EventBus.Publish(ctx, event.Topic(), event); err != nil {
        return fmt.Errorf("failed to publish event: %w", err)
    }
    return nil
}
```

**When to use:**
- Event is not tied to a database transaction
- Event is a side effect (e.g., notification)
- Idempotency is handled elsewhere

## Error Handling in Event Processing

Event handlers should be resilient:

```go
func (h *OnCustomerCreated) Handle(ctx context.Context, event events.Event) error {
    // Log and continue on type mismatch (don't retry)
    if !isExpectedType(event) {
        log.Printf("[WARN] Unexpected event type: %T", event)
        return nil  // Acknowledge but don't process
    }

    // Retryable errors return error (will be retried)
    if err := h.process(ctx, event); err != nil {
        return fmt.Errorf("failed to process customer created: %w", err)
    }

    return nil  // Success
}
```

**Error handling strategy:**
- Return `nil` for non-retryable errors (bad event format)
- Return error for retryable errors (database unavailable)
- Log all errors with appropriate level

## Testing Event Handlers

Test handlers with mock event bus:

```go
func TestOnCustomerCreated_Handle(t *testing.T) {
    handler := NewOnCustomerCreated()
    
    event := events.NewCustomerCreatedEvent("cust-123", map[string]string{
        "name": "John Doe",
    })
    
    err := handler.Handle(context.Background(), event)
    if err != nil {
        t.Errorf("Handle() error = %v", err)
    }
}
```

**Reference:** See test patterns in 08-testing.md

## When to Use Events vs Direct Calls

### Use Events When:
- Services need to be decoupled
- Operation can be asynchronous
- Multiple services need to react
- Event sourcing or audit trail needed

### Use Direct Calls When:
- Synchronous response required
- Strong consistency needed
- Simple request-response pattern
- Services are tightly coupled by design

## Migration Guide

### Adding a New Event Type

1. Define event type constant in `contracts/events/{domain}.go`
2. Create event struct with JSON tags
3. Implement `Event` interface methods
4. Create `EventFactory` implementation
5. Add convenience constructors
6. Create handler in `service/eventreader/eventhandlers/`
7. Register handler in event reader service

### Adding Event Publishing to a Service

1. Add `EventBus` to service infrastructure
2. Create method to emit events
3. Use outbox writer for transactional operations
4. Publish directly for non-transactional events
