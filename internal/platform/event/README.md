# Event Bus Provider

The Event Bus Provider implements the provider pattern for Kafka event messaging infrastructure. It encapsulates event bus setup and provides a configured event bus instance to services with service-specific topic and group configuration.

## Overview

The EventBusProvider interface allows services to access event messaging infrastructure through dependency injection, maintaining clean architecture principles. The provider loads platform Kafka configuration and applies service-specific overrides for topic and group settings.

## Interface

```go
type EventBusProvider interface {
    GetEventBus() bus.Bus
}
```

## Usage

### Basic Usage

```go
// Create provider with service-specific configuration
config := event.EventBusConfig{
    WriteTopic: "customer-events",
    GroupID:    "customer-service",
}

provider, err := event.NewEventBusProvider(config)
if err != nil {
    log.Fatal(err)
}

// Get configured event bus
bus := provider.GetEventBus()

// Use the event bus for publishing
err = bus.Publish(ctx, "customer-events", customerCreatedEvent)
```

### Service Integration

Services should depend on the `EventBusProvider` interface rather than concrete implementations:

```go
type MyService struct {
    eventBusProvider EventBusProvider
    // other dependencies...
}

func NewMyService(eventBusProvider EventBusProvider) *MyService {
    return &MyService{
        eventBusProvider: eventBusProvider,
    }
}

func (s *MyService) HandleEvent(ctx context.Context) error {
    bus := s.eventBusProvider.GetEventBus()
    return bus.Publish(ctx, "my-topic", myEvent)
}
```

## Configuration

The provider requires service-specific configuration:

```go
type EventBusConfig struct {
    WriteTopic string  // Kafka topic for publishing events (required)
    GroupID    string  // Kafka consumer group ID (required)
}
```

### Platform Configuration

The provider loads shared Kafka configuration from environment variables
provided by `platform-configmap-for-services` ConfigMap in Kubernetes deployment:

```env
KAFKA_BROKERS=["localhost:9092"]
kafka.topic=events          # Default topic (overridden by service config)
kafka.group_id=default-group # Default group (overridden by service config)
```

## Error Handling

The provider handles various error conditions:

- **Configuration Loading**: Fails if platform Kafka config cannot be loaded
- **Validation**: Validates required service configuration parameters
- **Event Bus Creation**: Ensures event bus is properly initialized

All errors are wrapped with context for better debugging.

## Architecture

The provider follows clean architecture principles:

- **Platform Layer**: Infrastructure concerns (Kafka setup, configuration)
- **Service Layer**: Business logic with dependency injection
- **Contracts Layer**: Event interfaces and data structures

## Testing

The provider includes comprehensive tests covering:

- Configuration validation
- Error handling scenarios
- Interface compliance
- Event bus initialization

Tests gracefully skip when Kafka configuration is not available in test environments.

## Migration from Direct Usage

### Before (Direct Usage)

```go
// Load config directly
kafkaCfg, err := config.LoadConfig[kafka.Config]("platform-kafka")
if err != nil {
    return err
}

// Override with service config
kafkaCfg.Topic = cfg.WriteTopic
kafkaCfg.GroupID = cfg.Group

// Create bus directly
bus := kafka.NewEventBus(kafkaCfg)
```

### After (Provider Pattern)

```go
// Create provider with service config
config := event.EventBusConfig{
    WriteTopic: cfg.WriteTopic,
    GroupID:    cfg.Group,
}

provider, err := event.NewEventBusProvider(config)
if err != nil {
    return err
}

// Get bus through provider
bus := provider.GetEventBus()
```

## Benefits

- **Clean Architecture**: Proper separation of concerns
- **Dependency Injection**: Services depend on interfaces, not implementations
- **Configuration Management**: Centralized config loading with service overrides
- **Error Handling**: Comprehensive error handling and logging
- **Testability**: Easy to mock and test with interface-based design
- **Maintainability**: Single source of truth for event bus setup</content>
<parameter name="filePath">internal/platform/event/README.md