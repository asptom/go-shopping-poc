# Platform Service Infrastructure

This package provides first-class service infrastructure for the Go Shopping POC platform. It implements clean architecture principles with proper abstraction hierarchy between platform-level infrastructure and domain-specific services.

## Architecture

```
Platform Level: internal/platform/service/     # Service infrastructure (HOW)
Domain Level: internal/service/eventreader/     # Specific service (WHAT)
```

## Components

### Service Interface

The `Service` interface defines the common lifecycle for all services:

```go
type Service interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() error
    Name() string
}
```

### BaseService

`BaseService` provides a default implementation that can be embedded:

```go
type BaseService struct {
    name string
}

func NewBaseService(name string) *BaseService
```

### EventServiceBase

`EventServiceBase` extends `BaseService` with event-specific functionality:

```go
type EventServiceBase struct {
    *BaseService
    eventBus bus.Bus
    handlers []any
}

func NewEventServiceBase(name string, eventBus bus.Bus) *EventServiceBase
```

## Usage Patterns

### Basic Service

```go
type MyService struct {
    *service.BaseService
    // additional fields
}

func NewMyService() *MyService {
    return &MyService{
        BaseService: service.NewBaseService("my-service"),
    }
}

func (s *MyService) Start(ctx context.Context) error {
    // custom start logic
    return nil
}
```

### Event-Driven Service

```go
type MyEventService struct {
    *service.EventServiceBase
    // additional fields
}

func NewMyEventService(eventBus bus.Bus) *MyEventService {
    return &MyEventService{
        EventServiceBase: service.NewEventServiceBase("my-event-service", eventBus),
    }
}

// Register event handlers
func (s *MyEventService) SetupHandlers() error {
    factory := events.CustomerEventFactory{}
    handler := bus.HandlerFunc[events.CustomerEvent](func(ctx context.Context, evt events.CustomerEvent) error {
        // handle event
        return nil
    })
    
    return service.RegisterHandler(s, factory, handler)
}
```

## Key Features

### Service-Type Agnostic

- **Event-driven services**: Use `EventServiceBase`
- **HTTP services**: Embed `BaseService` and add HTTP server logic
- **gRPC services**: Embed `BaseService` and add gRPC server logic
- **Custom services**: Implement `Service` interface directly

### Clean Separation of Concerns

- `platform/service/` = Service infrastructure patterns
- `platform/event/` = Event-specific infrastructure  
- `service/*/` = Domain-specific business logic

### Extensibility

- Easy to add new service types
- Common functionality shared across services
- Base implementations that can be extended
- Consistent patterns for all services

## Error Handling

The package provides structured error handling:

```go
type ServiceError struct {
    Service string
    Op      string
    Err     error
}
```

Common errors:
- `ErrUnsupportedEventBus`: Returned when event bus type is not supported

## Utilities

### Service Information

```go
info, err := service.GetEventServiceInfo(eventService)
// Returns: name, handler count, topics, health status
```

### Handler Management

```go
registrations, err := service.ListHandlers(eventService)
// Returns: list of registered handler information
```

### Validation

```go
err := service.ValidateEventBus(eventService)
// Validates event bus configuration
```

## Testing

The package includes comprehensive tests covering:

- Base service functionality
- Event service base functionality  
- Error handling scenarios
- Context cancellation
- Mock implementations for testing

Run tests with:
```bash
go test ./internal/platform/service/...
```

## Migration from EventReader

The EventReader service has been updated to use this shared infrastructure:

**Before:**
```go
type EventReaderService struct {
    eventBus bus.Bus
    handlers []any
}
```

**After:**
```go
type EventReaderService struct {
    *service.EventServiceBase
}
```

This eliminates code duplication and provides consistent service patterns across the platform.

## Best Practices

1. **Always embed BaseService** for consistent behavior
2. **Use EventServiceBase** for event-driven services
3. **Implement proper error handling** with ServiceError
4. **Add health checks** for service monitoring
5. **Handle context cancellation** gracefully
6. **Write comprehensive tests** using the provided patterns

## Future Enhancements

- HTTP service base implementation
- gRPC service base implementation
- Service discovery integration
- Metrics and monitoring integration
- Service dependency management