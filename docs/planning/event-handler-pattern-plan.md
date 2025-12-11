# Event Handler Pattern Implementation Plan

## Overview

This document outlines the implementation of a clean event handler pattern for the eventreader service, establishing a reusable pattern that can be applied across all services in the go-shopping-poc project.

## Current State Analysis

### Current Structure Issues

The current `cmd/eventreader/main.go` file mixes two distinct concerns:

1. **Service Orchestration** (Infrastructure setup):
   - Configuration loading
   - Kafka event bus initialization
   - Context and signal handling
   - Service lifecycle management

2. **Business Logic** (Event handling):
   - Customer event processing logic
   - Event data extraction and logging
   - Domain-specific event handling

### Current Code Analysis

```go
// Current mixed concerns in main.go
func main() {
    // Service orchestration (lines 16-37)
    logging.SetLevel("INFO")
    envFile := config.ResolveEnvFile()
    cfg := config.Load(envFile)
    eventBus := kafka.NewEventBus(...)
    
    // Business logic mixed with orchestration (lines 38-48)
    factory := events.CustomerEventFactory{}
    handler := bus.HandlerFunc[events.CustomerEvent](func(ctx context.Context, evt events.CustomerEvent) error {
        logging.Info("Eventreader: Processing customer event: %s", evt.Type())
        logging.Info("Eventreader: Data in event: CustomerID=%s, EventType=%s, ResourceID=%s",
            evt.EventPayload.CustomerID, evt.EventPayload.EventType, evt.EventPayload.ResourceID)
        return nil
    })
    
    kafka.SubscribeTyped(eventBus, factory, handler)
    
    // Service orchestration continues (lines 50-65)
    ctx, cancel := context.WithCancel(context.Background())
    // ... lifecycle management
}
```

### Problems with Current Approach

1. **Mixed Responsibilities**: Infrastructure setup and business logic are intertwined
2. **Poor Testability**: Business logic cannot be tested in isolation
3. **Limited Extensibility**: Adding new event handlers requires modifying main.go
4. **Code Duplication**: Similar orchestration patterns will be repeated across services
5. **Maintenance Burden**: Changes to business logic risk breaking infrastructure setup

## Target Architecture

### Clean Separation of Concerns

```
cmd/eventreader/main.go                    = Service orchestration only
internal/service/eventreader/
├── service.go                           = Service interface/implementation  
└── eventhandlers/
    ├── handler.go                        = Common handler interfaces
    └── on_customer_created.go            = Business logic for CustomerCreated events
```

### Design Principles

1. **Single Responsibility**: Each component has one clear purpose
2. **Dependency Inversion**: Service depends on abstractions, not concretions
3. **Testability**: Business logic can be unit tested in isolation
4. **Extensibility**: New event handlers can be added without modifying core service
5. **Consistency**: Pattern can be replicated across all services

## Implementation Steps

### Phase 1: Service Layer Foundation

#### 1.1 Create Service Interface
**File**: `internal/service/eventreader/service.go`

```go
package eventreader

import (
    "context"
    "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/bus"
)

// Service defines the interface for event reader business operations
type Service interface {
    // Start begins consuming events and processing them
    Start(ctx context.Context) error
    
    // RegisterHandler adds a new event handler to the service
    RegisterHandler(factory events.EventFactory[events.Event], handler bus.HandlerFunc[events.Event])
    
    // Stop gracefully shuts down the service
    Stop(ctx context.Context) error
}

// EventReaderService implements the Service interface
type EventReaderService struct {
    eventBus bus.Bus
    handlers []EventHandlerRegistration
}

// EventHandlerRegistration represents a registered event handler
type EventHandlerRegistration struct {
    Factory events.EventFactory[events.Event]
    Handler bus.HandlerFunc[events.Event]
}
```

#### 1.2 Create Service Implementation
**File**: `internal/service/eventreader/service.go` (continued)

```go
// NewEventReaderService creates a new event reader service instance
func NewEventReaderService(eventBus bus.Bus) *EventReaderService {
    return &EventReaderService{
        eventBus: eventBus,
        handlers: make([]EventHandlerRegistration, 0),
    }
}

// RegisterHandler adds a new event handler to the service
func (s *EventReaderService) RegisterHandler(
    factory events.EventFactory[events.Event], 
    handler bus.HandlerFunc[events.Event],
) {
    registration := EventHandlerRegistration{
        Factory: factory,
        Handler: handler,
    }
    s.handlers = append(s.handlers, registration)
    
    // Register with the event bus
    kafka.SubscribeTyped(s.eventBus, factory, handler)
}

// Start begins consuming events and processing them
func (s *EventReaderService) Start(ctx context.Context) error {
    return s.eventBus.StartConsuming(ctx)
}

// Stop gracefully shuts down the service
func (s *EventReaderService) Stop(ctx context.Context) error {
    // Implementation for graceful shutdown
    return nil
}
```

### Phase 2: Event Handler Pattern

#### 2.1 Create Common Handler Interface
**File**: `internal/service/eventreader/eventhandlers/handler.go`

```go
package eventhandlers

import (
    "context"
    "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/bus"
)

// EventHandler defines the interface for all event handlers
type EventHandler interface {
    // Handle processes the event and returns any error
    Handle(ctx context.Context, event events.Event) error
    
    // EventType returns the event type this handler processes
    EventType() string
}

// HandlerFactory creates event handlers with their factories
type HandlerFactory interface {
    // CreateFactory returns the event factory for this handler
    CreateFactory() events.EventFactory[events.Event]
    
    // CreateHandler returns the handler function
    CreateHandler() bus.HandlerFunc[events.Event]
}
```

#### 2.2 Create Customer Created Handler
**File**: `internal/service/eventreader/eventhandlers/on_customer_created.go`

```go
package eventhandlers

import (
    "context"
    events "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/platform/logging"
)

// OnCustomerCreated handles CustomerCreated events
type OnCustomerCreated struct{}

// NewOnCustomerCreated creates a new CustomerCreated event handler
func NewOnCustomerCreated() *OnCustomerCreated {
    return &OnCustomerCreated{}
}

// Handle processes CustomerCreated events
func (h *OnCustomerCreated) Handle(ctx context.Context, event events.Event) error {
    customerEvent, ok := event.(events.CustomerEvent)
    if !ok {
        logging.Error("Eventreader: Expected CustomerEvent, got %T", event)
        return nil // Don't fail processing, just log and continue
    }
    
    if customerEvent.EventType != events.CustomerCreated {
        logging.Debug("Eventreader: Ignoring non-CustomerCreated event: %s", customerEvent.EventType)
        return nil
    }
    
    logging.Info("Eventreader: Processing CustomerCreated event")
    logging.Info("Eventreader: CustomerID=%s, EventType=%s, ResourceID=%s",
        customerEvent.EventPayload.CustomerID, 
        customerEvent.EventPayload.EventType, 
        customerEvent.EventPayload.ResourceID)
    
    // Business logic for handling customer creation
    return h.processCustomerCreated(ctx, customerEvent)
}

// processCustomerCreated contains the actual business logic
func (h *OnCustomerCreated) processCustomerCreated(ctx context.Context, event events.CustomerEvent) error {
    // TODO: Add actual business logic here
    // Examples:
    // - Update read models
    // - Send notifications
    // - Trigger other workflows
    // - Update analytics
    
    logging.Info("Eventreader: Successfully processed CustomerCreated event for customer %s", 
        event.EventPayload.CustomerID)
    
    return nil
}

// EventType returns the event type this handler processes
func (h *OnCustomerCreated) EventType() string {
    return string(events.CustomerCreated)
}

// CreateFactory returns the event factory for this handler
func (h *OnCustomerCreated) CreateFactory() events.EventFactory[events.Event] {
    return events.CustomerEventFactory{}
}

// CreateHandler returns the handler function
func (h *OnCustomerCreated) CreateHandler() bus.HandlerFunc[events.Event] {
    return bus.HandlerFunc[events.Event](h.Handle)
}
```

### Phase 3: Service Orchestration Refactor

#### 3.1 Refactor Main.go
**File**: `cmd/eventreader/main.go`

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"

    "go-shopping-poc/internal/platform/config"
    kafka "go-shopping-poc/internal/platform/event/bus/kafka"
    "go-shopping-poc/internal/platform/logging"
    "go-shopping-poc/internal/service/eventreader"
    "go-shopping-poc/internal/service/eventreader/eventhandlers"
)

func main() {
    logging.SetLevel("INFO")
    logging.Info("Eventreader: EventReader service started")

    // Load configuration
    envFile := config.ResolveEnvFile()
    cfg := config.Load(envFile)

    logging.Debug("Eventreader: Configuration loaded from %s", envFile)
    logging.Debug("Eventreader: Config: %v", cfg)

    // Setup infrastructure
    broker := cfg.GetEventBroker()
    readTopics := cfg.GetEventReaderReadTopics()
    writeTopic := cfg.GetEventReaderWriteTopic()
    group := cfg.GetEventReaderGroup()

    logging.Debug("Eventreader: Event Broker: %s, Read Topics: %v, Write Topic: %v, Group: %s", 
        broker, readTopics, writeTopic, group)

    eventBus := kafka.NewEventBus(broker, readTopics, writeTopic, group)

    // Create service
    service := eventreader.NewEventReaderService(eventBus)

    // Register event handlers
    registerEventHandlers(service)

    // Start service
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    logging.Debug("Eventreader: Starting event consumer...")
    go func() {
        logging.Debug("Eventreader: Event consumer started")
        if err := service.Start(ctx); err != nil {
            logging.Error("Eventreader: Event consumer stopped:", err)
        }
    }()

    // Wait for shutdown signal
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    <-sig
    logging.Debug("Eventreader: Received shutdown signal, shutting down...")
    
    // Graceful shutdown
    if err := service.Stop(ctx); err != nil {
        logging.Error("Eventreader: Error during shutdown:", err)
    }
}

// registerEventHandlers registers all event handlers with the service
func registerEventHandlers(service *eventreader.EventReaderService) {
    // Register CustomerCreated handler
    customerCreatedHandler := eventhandlers.NewOnCustomerCreated()
    service.RegisterHandler(
        customerCreatedHandler.CreateFactory(),
        customerCreatedHandler.CreateHandler(),
    )
    
    // Future handlers can be registered here
    // customerUpdatedHandler := eventhandlers.NewOnCustomerUpdated()
    // service.RegisterHandler(
    //     customerUpdatedHandler.CreateFactory(),
    //     customerUpdatedHandler.CreateHandler(),
    // )
}
```

### Phase 4: Testing Strategy

#### 4.1 Service Layer Tests
**File**: `internal/service/eventreader/service_test.go`

```go
package eventreader

import (
    "context"
    "testing"
    "time"

    events "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/testutils"
)

// MockEventBus for testing
type MockEventBus struct {
    bus.Bus
    startConsumingCalled bool
    handlers            []interface{}
}

func (m *MockEventBus) StartConsuming(ctx context.Context) error {
    m.startConsumingCalled = true
    return nil
}

func TestEventReaderService_RegisterHandler(t *testing.T) {
    mockBus := &MockEventBus{}
    service := NewEventReaderService(mockBus)
    
    factory := events.CustomerEventFactory{}
    handler := bus.HandlerFunc[events.Event](func(ctx context.Context, evt events.Event) error {
        return nil
    })
    
    service.RegisterHandler(factory, handler)
    
    if len(service.handlers) != 1 {
        t.Errorf("Expected 1 handler, got %d", len(service.handlers))
    }
}

func TestEventReaderService_Start(t *testing.T) {
    mockBus := &MockEventBus{}
    service := NewEventReaderService(mockBus)
    
    ctx := context.Background()
    err := service.Start(ctx)
    
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }
    
    if !mockBus.startConsumingCalled {
        t.Error("Expected StartConsuming to be called")
    }
}
```

#### 4.2 Event Handler Tests
**File**: `internal/service/eventreader/eventhandlers/on_customer_created_test.go`

```go
package eventhandlers

import (
    "context"
    "testing"

    events "go-shopping-poc/internal/contracts/events"
)

func TestOnCustomerCreated_Handle(t *testing.T) {
    handler := NewOnCustomerCreated()
    
    // Create a CustomerCreated event
    event := events.NewCustomerCreatedEvent("customer-123", map[string]string{
        "source": "test",
    })
    
    ctx := context.Background()
    err := handler.Handle(ctx, event)
    
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }
}

func TestOnCustomerCreated_HandleWrongEventType(t *testing.T) {
    handler := NewOnCustomerCreated()
    
    // Create a different event type
    event := events.NewCustomerUpdatedEvent("customer-123", map[string]string{
        "source": "test",
    })
    
    ctx := context.Background()
    err := handler.Handle(ctx, event)
    
    if err != nil {
        t.Errorf("Expected no error for wrong event type, got %v", err)
    }
}

func TestOnCustomerCreated_EventType(t *testing.T) {
    handler := NewOnCustomerCreated()
    
    expectedType := string(events.CustomerCreated)
    actualType := handler.EventType()
    
    if actualType != expectedType {
        t.Errorf("Expected event type %s, got %s", expectedType, actualType)
    }
}

func TestOnCustomerCreated_FactoryAndHandler(t *testing.T) {
    handler := NewOnCustomerCreated()
    
    factory := handler.CreateFactory()
    if factory == nil {
        t.Error("Expected factory to be non-nil")
    }
    
    handlerFunc := handler.CreateHandler()
    if handlerFunc == nil {
        t.Error("Expected handler to be non-nil")
    }
}
```

#### 4.3 Integration Tests
**File**: `cmd/eventreader/integration_test.go`

```go
//go:build integration

package main

import (
    "context"
    "testing"
    "time"

    events "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/config"
    kafka "go-shopping-poc/internal/platform/event/bus/kafka"
    "go-shopping-poc/internal/service/eventreader"
    "go-shopping-poc/internal/service/eventreader/eventhandlers"
    "go-shopping-poc/internal/testutils"
)

func TestEventReaderService_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Setup test environment
    testutils.SetupTestEnvironment(t)
    
    // Load test configuration
    envFile := config.ResolveEnvFile()
    cfg := config.Load(envFile)
    
    // Create event bus
    broker := cfg.GetEventBroker()
    readTopics := cfg.GetEventReaderReadTopics()
    writeTopic := cfg.GetEventReaderWriteTopic()
    group := cfg.GetEventReaderGroup() + "-test"
    
    eventBus := kafka.NewEventBus(broker, readTopics, writeTopic, group)
    
    // Create service
    service := eventreader.NewEventReaderService(eventBus)
    
    // Register handlers
    customerCreatedHandler := eventhandlers.NewOnCustomerCreated()
    service.RegisterHandler(
        customerCreatedHandler.CreateFactory(),
        customerCreatedHandler.CreateHandler(),
    )
    
    // Start service
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    go func() {
        if err := service.Start(ctx); err != nil {
            t.Errorf("Service start failed: %v", err)
        }
    }()
    
    // Wait for service to be ready
    time.Sleep(2 * time.Second)
    
    // Publish test event
    testEvent := events.NewCustomerCreatedEvent("test-customer-123", map[string]string{
        "test": "integration",
    })
    
    err := eventBus.Publish(ctx, testEvent.Topic(), testEvent)
    if err != nil {
        t.Errorf("Failed to publish test event: %v", err)
    }
    
    // Wait for processing
    time.Sleep(3 * time.Second)
    
    // Cleanup
    cancel()
    time.Sleep(1 * time.Second)
}
```

## Migration Strategy

### Step-by-Step Migration

1. **Create New Directory Structure**
   ```bash
   mkdir -p internal/service/eventreader/eventhandlers
   ```

2. **Implement Service Layer** (Phase 1)
   - Create service interface and implementation
   - Add basic tests
   - Verify compilation

3. **Implement Event Handler Pattern** (Phase 2)
   - Create common handler interfaces
   - Implement CustomerCreated handler
   - Add comprehensive tests
   - Verify functionality

4. **Refactor Main.go** (Phase 3)
   - Extract business logic from main.go
   - Implement clean orchestration
   - Test service startup and shutdown

5. **Add Comprehensive Testing** (Phase 4)
   - Unit tests for service layer
   - Unit tests for event handlers
   - Integration tests
   - Verify test coverage

6. **Verification and Cleanup**
   - Run all existing tests to ensure no regressions
   - Run new tests to verify new functionality
   - Update documentation
   - Clean up any unused code

### Risk Mitigation

1. **Incremental Changes**: Implement changes in small, testable increments
2. **Backward Compatibility**: Ensure existing functionality continues to work
3. **Comprehensive Testing**: Test each component in isolation and integration
4. **Rollback Plan**: Keep original code until migration is complete and verified

## Pattern Documentation

### Reusable Pattern for Other Services

This implementation establishes a pattern that can be replicated across all services:

#### 1. Service Structure
```
internal/service/{servicename}/
├── service.go              # Service interface and implementation
└── eventhandlers/          # Event handling logic
    ├── handler.go          # Common handler interfaces
    └── on_{event}_go       # Specific event handlers
```

#### 2. Service Interface Pattern
```go
type Service interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    RegisterHandler(factory events.EventFactory[events.Event], handler bus.HandlerFunc[events.Event])
}
```

#### 3. Event Handler Pattern
```go
type EventHandler interface {
    Handle(ctx context.Context, event events.Event) error
    EventType() string
}

type HandlerFactory interface {
    CreateFactory() events.EventFactory[events.Event]
    CreateHandler() bus.HandlerFunc[events.Event]
}
```

#### 4. Main.go Pattern
```go
func main() {
    // 1. Load configuration
    // 2. Setup infrastructure
    // 3. Create service
    // 4. Register handlers (via registerEventHandlers function)
    // 5. Start service
    // 6. Handle lifecycle
}

func registerEventHandlers(service *ServiceName) {
    // Register all handlers here
}
```

### Benefits of This Pattern

1. **Consistency**: Same structure across all services
2. **Maintainability**: Clear separation of concerns
3. **Testability**: Each component can be tested in isolation
4. **Extensibility**: Easy to add new event handlers
5. **Documentation**: Self-documenting structure
6. **Onboarding**: New developers can quickly understand the pattern

## Success Criteria

### Functional Requirements
- ✅ Business logic separated from service orchestration
- ✅ All existing functionality preserved
- ✅ EventReader service processes CustomerCreated events correctly
- ✅ Service can be started and stopped gracefully
- ✅ New event handlers can be added without modifying main.go

### Quality Requirements
- ✅ Unit test coverage > 80% for new code
- ✅ Integration tests verify end-to-end functionality
- ✅ Code follows existing project conventions
- ✅ No performance regressions
- ✅ Clear documentation and comments

### Architectural Requirements
- ✅ Clean separation of concerns
- ✅ Dependency inversion principle followed
- ✅ Single responsibility principle applied
- ✅ Open/closed principle (extensible without modification)
- ✅ Interface segregation principle

## Future Enhancements

### Short-term (Next Sprint)
1. **Additional Event Handlers**: Implement handlers for CustomerUpdated, AddressAdded, etc.
2. **Error Handling**: Implement robust error handling and retry logic
3. **Metrics**: Add metrics for event processing
4. **Health Checks**: Implement health check endpoints

### Medium-term (Next Quarter)
1. **Event Sourcing**: Implement event sourcing for read models
2. **CQRS**: Separate command and query responsibilities
3. **Distributed Tracing**: Add tracing for event flows
4. **Circuit Breakers**: Implement resilience patterns

### Long-term (Next 6 Months)
1. **Event Schema Registry**: Implement schema validation
2. **Event Replay**: Add capability to replay events
3. **Multi-tenant Support**: Support multiple tenants
4. **Performance Optimization**: Optimize for high-throughput scenarios

## Conclusion

This implementation plan establishes a clean, testable, and extensible event handler pattern for the eventreader service. The pattern can be replicated across all services, providing consistency and maintainability while preserving all existing functionality.

The phased approach ensures minimal risk while delivering immediate benefits in code organization and testability. The comprehensive testing strategy ensures reliability and confidence in the new architecture.

By following this plan, we'll create a solid foundation for event-driven architecture that can scale with the growing needs of the go-shopping-poc project.