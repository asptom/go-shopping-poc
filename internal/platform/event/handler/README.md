# Shared Handler Interface Pattern

This document describes the shared handler interface pattern implemented in `internal/platform/event/handler/` for reusable event handling across all services.

## Overview

The shared handler pattern provides common interfaces and utilities for implementing type-safe event handlers that can be reused across different services in the system. This promotes consistency, reduces code duplication, and makes it easier to maintain event handling logic.

## Architecture

```
internal/platform/event/handler/
├── interface.go           # Common handler interfaces
├── interface_test.go     # Tests for shared interfaces
├── event_utils.go        # Generic event handling utilities
└── [future_handlers]    # Additional domain-specific handlers
```

## Core Interfaces

### EventHandler

The `EventHandler` interface defines the contract for all event handlers:

```go
type EventHandler interface {
    // Handle processes the event and returns any error
    Handle(ctx context.Context, event events.Event) error
    
    // EventType returns the event type this handler processes
    EventType() string
}
```

### HandlerFactory

The `HandlerFactory[T]` interface provides a generic factory pattern for creating typed handlers:

```go
type HandlerFactory[T events.Event] interface {
    // CreateFactory returns the event factory for this handler
    CreateFactory() events.EventFactory[T]
    
    // CreateHandler returns the handler function
    CreateHandler() bus.HandlerFunc[T]
}
```

**Generic Parameter:**
- `T`: The specific event type (e.g., `events.CustomerEvent`, `events.OrderEvent`)
- Must implement the `events.Event` interface

**Benefits:**
- **Type Safety**: Compile-time type checking for all event handling
- **Reusability**: Same interface works for any event type
- **Clean Architecture**: Platform layer remains generic and domain-agnostic
- **Extensibility**: Easy to add new event types without changing platform code



## Generic Event Utilities

The `EventUtils` type provides reusable, domain-agnostic utilities for event processing:

### Available Utilities

```go
utils := handler.NewEventUtils()

// Validation
err := utils.ValidateEvent(ctx, event)

// Logging
utils.LogEventProcessing(ctx, eventType, entityID, resourceID)
utils.LogEventCompletion(ctx, eventType, entityID, err)

// Safe processing with validation
err := utils.HandleEventWithValidation(ctx, event, processor)

// Panic recovery
err := utils.SafeEventProcessing(ctx, event, processor)

// Extract information
entityID := utils.GetEventID(event)
resourceID := utils.GetResourceID(event)
```

### Event Type Matching

The `EventTypeMatcher` provides type checking utilities:

```go
matcher := handler.NewEventTypeMatcher()

// Check event type
isMatch := matcher.MatchEventType(event, "customer.created", "customer.updated")

// Check specific event types
isCustomer := matcher.IsCustomerEvent(event)
```

## Usage Pattern

### 1. Implement the Interfaces

Create a handler that implements both `EventHandler` and `HandlerFactory`:

```go
package myhandlers

import (
    "context"
    "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/platform/event/handler"
    "go-shopping-poc/internal/platform/logging"
)

type MyCustomerHandler struct{}

func (h *MyCustomerHandler) Handle(ctx context.Context, event events.Event) error {
    customerEvent, ok := event.(*events.CustomerEvent)
    if !ok {
        logging.Error("Expected CustomerEvent, got %T", event)
        return nil
    }
    
    // Your business logic here
    logging.Info("Processing customer event: %s", customerEvent.EventType)
    return nil
}

func (h *MyCustomerHandler) EventType() string {
    return string(events.CustomerCreated)
}

func (h *MyCustomerHandler) CreateFactory() events.EventFactory[events.CustomerEvent] {
    return events.CustomerEventFactory{}
}

func (h *MyCustomerHandler) CreateHandler() bus.HandlerFunc[events.CustomerEvent] {
    return func(ctx context.Context, event events.CustomerEvent) error {
        return h.Handle(ctx, event)
    }
}

// Ensure interface compliance
var _ handler.EventHandler = (*MyCustomerHandler)(nil)
var _ handler.HandlerFactory = (*MyCustomerHandler)(nil)
```

### 2. Register with Service

Register the handler with your service:

```go
package myservice

import (
    "go-shopping-poc/internal/service/eventreader"
    "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/bus"
)

func SetupHandlers(service *eventreader.EventReaderService) {
    handler := &MyCustomerHandler{}
    
    service.RegisterHandler(
        handler.CreateFactory(),
        handler.CreateHandler(),
    )
}
```

### 3. Use Generic Event Utilities

Leverage generic event handling utilities:

```go
import "go-shopping-poc/internal/platform/event/handler"

func SetupCustomerHandlers(service *eventreader.EventReaderService) {
    utils := handler.NewEventUtils()
    
    // Create a handler that uses generic utilities
    handlerFunc := func(ctx context.Context, event events.CustomerEvent) error {
        // Use generic validation
        if err := utils.ValidateEvent(ctx, event); err != nil {
            return err
        }
        
        // Use generic logging
        utils.LogEventProcessing(ctx, event.EventType, event.EventPayload.CustomerID, event.EventPayload.ResourceID)
        
        // Your business logic here
        return nil
    }
    
    // Register for specific events
    service.RegisterHandler(
        events.CustomerEventFactory{},
        handlerFunc,
    )
}
```

## Benefits

1. **Consistency**: All handlers follow the same interface pattern
2. **Reusability**: Common logic can be shared across services
3. **Type Safety**: Compile-time type checking with generics
4. **Testability**: Easy to mock and test handlers
5. **Maintainability**: Centralized interface definitions
6. **Extensibility**: Easy to add new handler types

## Migration Guide

When migrating service-specific handlers to use the shared pattern:

1. **Move Interfaces**: Remove local interface definitions
2. **Update Imports**: Change to use `go-shopping-poc/internal/platform/event/handler`
3. **Update Type References**: Use `handler.EventHandler` and `handler.HandlerFactory`
4. **Add Interface Compliance**: Ensure your handlers implement the shared interfaces
5. **Update Tests**: Modify tests to use the shared interfaces

## Testing

The shared interfaces include comprehensive tests. Test your handlers by:

```go
func TestMyHandler(t *testing.T) {
    handler := &MyCustomerHandler{}
    
    // Test interface compliance
    var _ handler.EventHandler = handler
    var _ handler.HandlerFactory = handler
    
    // Test functionality
    event := events.NewCustomerCreatedEvent("test-id", nil)
    err := handler.Handle(context.Background(), event)
    
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }
}
```

## Future Extensions

The pattern is designed to be extensible:

- **New Event Types**: Add interfaces for different event domains
- **Middleware Support**: Add handler middleware for cross-cutting concerns
- **Async Processing**: Add async handler variants
- **Batch Processing**: Add batch handler interfaces

## Best Practices

1. **Always implement both interfaces** for consistency
2. **Use descriptive names** for your handlers
3. **Include comprehensive logging** for debugging
4. **Handle type assertions gracefully** (don't fail on wrong types)
5. **Write thorough tests** for all handler logic
6. **Document your handler's purpose** and behavior
7. **Use reusable handlers** when possible to avoid duplication