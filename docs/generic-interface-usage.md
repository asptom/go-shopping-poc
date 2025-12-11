# Generic Event Handler Interface Usage

This document provides a comprehensive guide for using the generic event handler interface across all services in the project.

## Overview

The EventReaderService uses a generic interface that can handle any event type while maintaining full type safety. The service supports registration of event handlers for any event type that implements the `events.Event` interface.

## Usage Examples

### Basic Event Handler Registration

```go
service := eventreader.NewEventReaderService(eventBus)

// Register CustomerCreated handler
customerCreatedHandler := eventhandlers.NewOnCustomerCreated()
service.RegisterHandler(
    customerCreatedHandler.CreateFactory(),
    customerCreatedHandler.CreateHandler(),
)
```

### Multiple Event Type Registration

```go
service := eventreader.NewEventReaderService(eventBus)

// Register CustomerCreated handler
customerCreatedHandler := eventhandlers.NewOnCustomerCreated()
service.RegisterHandler(
    customerCreatedHandler.CreateFactory(),
    customerCreatedHandler.CreateHandler(),
)

// Register OrderCreated handler
orderCreatedHandler := eventhandlers.NewOnOrderCreated()
service.RegisterHandler(
    orderCreatedHandler.CreateFactory(),
    orderCreatedHandler.CreateHandler(),
)

// Register PaymentProcessed handler
paymentHandler := eventhandlers.NewOnPaymentProcessed()
service.RegisterHandler(
    paymentHandler.CreateFactory(),
    paymentHandler.CreateHandler(),
)
```

## Implementation Details

### Service Interface

```go
type Service interface {
    Start(ctx context.Context) error
    RegisterHandler(factory events.EventFactory[events.CustomerEvent], handler bus.HandlerFunc[events.CustomerEvent])
    Stop(ctx context.Context) error
}
```

### Event Handler Pattern

Each event handler follows this pattern:

```go
type OnCustomerCreated struct {
    // Handler dependencies
}

func NewOnCustomerCreated() *OnCustomerCreated {
    return &OnCustomerCreated{}
}

func (h *OnCustomerCreated) CreateFactory() events.EventFactory[events.CustomerEvent] {
    return events.CustomerEventFactory{}
}

func (h *OnCustomerCreated) CreateHandler() bus.HandlerFunc[events.CustomerEvent] {
    return func(ctx context.Context, event events.CustomerEvent) error {
        // Handle the event
        return nil
    }
}
```

### Key Benefits

1. **Type Safety**: Generic constraints ensure compile-time type checking
2. **Extensibility**: Easy to add handlers for any new event type
3. **Clean Architecture**: Consistent interface across all event types
4. **Maintainability**: Standardized pattern for event handling

## Type Safety

The generic approach maintains full type safety:

```go
// This compiles - CustomerEvent implements events.Event
service.RegisterHandler(customerFactory, customerHandler)

// This compiles - OrderEvent implements events.Event  
service.RegisterHandler(orderFactory, orderHandler)

// This would NOT compile - MyStruct doesn't implement events.Event
// service.RegisterHandler(myFactory, myHandler) // Compile-time error
```

## Service Implementation Pattern

### Complete Service Example

```go
func registerAllEventHandlers(service *eventreader.EventReaderService) {
    // Customer events
    customerCreatedHandler := eventhandlers.NewOnCustomerCreated()
    service.RegisterHandler(
        customerCreatedHandler.CreateFactory(),
        customerCreatedHandler.CreateHandler(),
    )
    
    // Order events  
    orderCreatedHandler := eventhandlers.NewOnOrderCreated()
    service.RegisterHandler(
        orderCreatedHandler.CreateFactory(),
        orderCreatedHandler.CreateHandler(),
    )
        
    // Payment events
    paymentHandler := eventhandlers.NewOnPaymentProcessed()
    service.RegisterHandler(
        paymentHandler.CreateFactory(),
        paymentHandler.CreateHandler(),
    )
}

func main() {
    eventBus := kafka.NewEventBus(config.LoadKafkaConfig())
    service := eventreader.NewEventReaderService(eventBus)
    
    // Register all handlers
    registerAllEventHandlers(service)
    
    // Start the service
    ctx := context.Background()
    if err := service.Start(ctx); err != nil {
        log.Fatal(err)
    }
}
```

## Best Practices

### 1. Handler Organization

Organize handlers by event type and domain:

```
internal/service/eventreader/
├── eventhandlers/
│   ├── customer/
│   │   ├── on_customer_created.go
│   │   └── on_customer_updated.go
│   ├── order/
│   │   ├── on_order_created.go
│   │   └── on_order_updated.go
│   └── payment/
│       ├── on_payment_processed.go
│       └── on_payment_failed.go
```

### 2. Error Handling

Always handle errors properly in handlers:

```go
func (h *OnCustomerCreated) CreateHandler() bus.HandlerFunc[events.CustomerEvent] {
    return func(ctx context.Context, event events.CustomerEvent) error {
        if err := h.processCustomerEvent(ctx, event); err != nil {
            return fmt.Errorf("failed to process customer event: %w", err)
        }
        return nil
    }
}
```

### 3. Dependency Injection

Pass dependencies through handler constructor:

```go
func NewOnCustomerCreated(db *sql.DB, logger logging.Logger) *OnCustomerCreated {
    return &OnCustomerCreated{
        db:     db,
        logger: logger,
    }
}
```

### 4. Testing

Test handlers independently:

```go
func TestOnCustomerCreated_Handle(t *testing.T) {
    handler := NewOnCustomerCreated(mockDB, mockLogger)
    
    event := events.CustomerEvent{
        ID:        uuid.New(),
        Type:      "CustomerCreated",
        Timestamp: time.Now(),
        Data:      events.CustomerCreatedData{ /* ... */ },
    }
    
    err := handler.CreateHandler()(context.Background(), event)
    assert.NoError(t, err)
}
```

## Event Factory Pattern

Each event type has a corresponding factory:

```go
type CustomerEventFactory struct{}

func (f CustomerEventFactory) CreateEvent(eventType string, data json.RawMessage) (events.CustomerEvent, error) {
    switch eventType {
    case "CustomerCreated":
        return events.NewCustomerCreatedEvent(data)
    case "CustomerUpdated":
        return events.NewCustomerUpdatedEvent(data)
    default:
        return events.CustomerEvent{}, fmt.Errorf("unknown customer event type: %s", eventType)
    }
}
```

## Future Extensibility

This pattern makes it easy to add support for new event types:

1. **Define the event structure** in `internal/contracts/events/`
2. **Create the event factory** implementing `EventFactory[T]`
3. **Implement the handler** following the standard pattern
4. **Register the handler** in your service

The service can handle any number of event types while maintaining type safety and consistent behavior.