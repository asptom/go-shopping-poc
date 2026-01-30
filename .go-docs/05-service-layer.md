# Service Layer

This document describes the service layer patterns used for business logic orchestration, including service structure, infrastructure management, and lifecycle handling.

## Overview

The **Service Layer** implements business logic and orchestrates between repositories, event buses, and other infrastructure. It represents the "WHAT" of the system - what operations the system performs.

## Service Structure

### Service Components

A typical service consists of:

```go
internal/service/customer/
├── entity.go        # Domain models
├── service.go       # Business logic (this document)
├── repository.go    # Data access
├── handler.go       # HTTP handlers
└── config.go        # Configuration
```

### Service Struct

```go
// internal/service/customer/service.go

// CustomerService orchestrates customer business operations
type CustomerService struct {
    *service.BaseService              // Embedded base service
    repo           CustomerRepository  // Data access
    infrastructure *CustomerInfrastructure  // External dependencies
    config         *Config             // Service configuration
}
```

**Key patterns:**
1. Embed `*service.BaseService` for lifecycle management
2. Hold repository interface (not concrete type)
3. Group infrastructure dependencies
4. Keep configuration reference

**Reference:** `internal/service/customer/service.go` (lines 106-140)

## Infrastructure Pattern

### Infrastructure Struct

Group all external dependencies in a single struct:

```go
// internal/service/customer/service.go

// CustomerInfrastructure defines infrastructure components required
type CustomerInfrastructure struct {
    Database        database.Database
    EventBus        bus.Bus
    OutboxWriter    *outbox.Writer
    OutboxPublisher *outbox.Publisher
    CORSHandler     func(http.Handler) http.Handler
}

func NewCustomerInfrastructure(
    db database.Database,
    eventBus bus.Bus,
    outboxWriter *outbox.Writer,
    outboxPublisher *outbox.Publisher,
    corsHandler func(http.Handler) http.Handler,
) *CustomerInfrastructure {
    return &CustomerInfrastructure{
        Database:        db,
        EventBus:        eventBus,
        OutboxWriter:    outboxWriter,
        OutboxPublisher: outboxPublisher,
        CORSHandler:     corsHandler,
    }
}
```

**Benefits:**
- Clean constructor signature
- Easy to pass around
- Simple to mock for testing
- Clear dependency boundary

**Reference:** `internal/service/customer/service.go` (lines 21-68)

## Constructor Patterns

### Standard Constructor

```go
func NewCustomerService(
    infrastructure *CustomerInfrastructure,
    config *Config,
) *CustomerService {
    // Create repository internally
    repo := NewCustomerRepository(infrastructure.Database, infrastructure.OutboxWriter)
    
    return &CustomerService{
        BaseService:    service.NewBaseService("customer"),
        repo:           repo,
        infrastructure: infrastructure,
        config:         config,
    }
}
```

### Testing Constructor

Provide an alternative constructor for dependency injection in tests:

```go
// NewCustomerServiceWithRepo allows injecting a mock repository
func NewCustomerServiceWithRepo(
    repo CustomerRepository,
    infrastructure *CustomerInfrastructure,
    config *Config,
) *CustomerService {
    return &CustomerService{
        BaseService:    service.NewBaseService("customer"),
        repo:           repo,
        infrastructure: infrastructure,
        config:         config,
    }
}
```

**Usage in tests:**
```go
mockRepo := &mockCustomerRepository{...}
svc := NewCustomerServiceWithRepo(mockRepo, mockInfra, &Config{})
```

**Reference:** `internal/service/customer/service.go` (lines 132-140)

## Base Service

### BaseService Implementation

The platform layer provides base service functionality:

```go
// internal/platform/service/interface.go

type Service interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() error
    Name() string
}

type BaseService struct {
    name string
}

func NewBaseService(name string) *BaseService {
    return &BaseService{name: name}
}

func (s *BaseService) Name() string {
    return s.name
}
```

**Key features:**
- Standardized service interface
- Service naming
- Lifecycle hooks (Start, Stop)
- Health check support

**Reference:** `internal/platform/service/interface.go`

### EventService Extension

For event-driven services, extend the base:

```go
// internal/platform/service/interface.go

type EventService interface {
    Service
    EventBus() bus.Bus
    HandlerCount() int
}

// EventServiceBase provides base implementation
type EventServiceBase struct {
    *BaseService
    eventBus bus.Bus
    handlers []any
}

func NewEventServiceBase(name string, eventBus bus.Bus) *EventServiceBase {
    return &EventServiceBase{
        BaseService: NewBaseService(name),
        eventBus:    eventBus,
        handlers:    make([]any, 0),
    }
}

func (s *EventServiceBase) EventBus() bus.Bus {
    return s.eventBus
}

func (s *EventServiceBase) HandlerCount() int {
    return len(s.handlers)
}
```

**Reference:** `internal/platform/service/base.go`

## Business Logic Methods

### Method Structure

Service methods follow a consistent pattern:

```go
func (s *CustomerService) CreateCustomer(
    ctx context.Context,
    customer *Customer,
) error {
    // 1. Validate input
    if err := customer.Validate(); err != nil {
        return fmt.Errorf("invalid customer: %w", err)
    }
    
    // 2. Check business rules
    existing, err := s.repo.GetCustomerByEmail(ctx, customer.Email)
    if err != nil && !errors.Is(err, ErrCustomerNotFound) {
        return fmt.Errorf("failed to check existing customer: %w", err)
    }
    if existing != nil {
        return ErrDuplicateEmail
    }
    
    // 3. Perform operation
    if err := s.repo.InsertCustomer(ctx, customer); err != nil {
        return fmt.Errorf("failed to create customer: %w", err)
    }
    
    // 4. Side effects (events, notifications)
    // Events are handled by repository via outbox pattern
    
    return nil
}
```

**Key patterns:**
1. Accept `context.Context` as first parameter
2. Validate input first
3. Check business rules
4. Delegate to repository for data access
5. Handle errors with context
6. Return domain-specific errors

**Reference:** `internal/service/customer/service.go`

### Complex Operations

For operations spanning multiple entities:

```go
func (s *CustomerService) RegisterCustomerWithAddresses(
    ctx context.Context,
    customer *Customer,
    addresses []Address,
) (*Customer, error) {
    // Validate all inputs first
    if err := customer.Validate(); err != nil {
        return nil, fmt.Errorf("invalid customer: %w", err)
    }
    
    for _, addr := range addresses {
        if err := addr.Validate(); err != nil {
            return nil, fmt.Errorf("invalid address: %w", err)
        }
    }
    
    // Check business rules
    if err := s.validateUniqueEmail(ctx, customer.Email); err != nil {
        return nil, err
    }
    
    // Repository handles transaction and event publishing
    if err := s.repo.InsertCustomerWithAddresses(ctx, customer, addresses); err != nil {
        return nil, fmt.Errorf("failed to register customer: %w", err)
    }
    
    return customer, nil
}
```

**Key patterns:**
1. Validate all inputs upfront
2. Repository handles transactions for atomicity
3. Single error return with wrapped context
4. Return created/updated entities

## Service Lifecycle

### Starting a Service

Services implement the `Start` method:

```go
// internal/service/eventreader/service.go

func (s *EventReaderService) Start(ctx context.Context) error {
    // Validate service is ready
    if s.infrastructure.EventBus == nil {
        return &service.ServiceError{
            Service: s.Name(),
            Op:      "Start",
            Err:     errors.New("event bus not configured"),
        }
    }
    
    // Start consuming events
    if err := s.infrastructure.EventBus.StartConsuming(ctx); err != nil {
        return fmt.Errorf("failed to start event consumption: %w", err)
    }
    
    return nil
}
```

**Reference:** `internal/service/eventreader/service.go`

### Stopping a Service

Graceful shutdown:

```go
func (s *EventReaderService) Stop(ctx context.Context) error {
    // Stop event bus
    if s.infrastructure.EventBus != nil {
        // Implementation-specific stop logic
    }
    
    return nil
}
```

### Health Checks

```go
func (s *CustomerService) Health() error {
    // Check database connectivity
    if err := s.infrastructure.Database.Ping(context.Background()); err != nil {
        return fmt.Errorf("database unhealthy: %w", err)
    }
    
    return nil
}
```

## Event Handling in Services

### Registering Event Handlers

Event-driven services register handlers during initialization:

```go
// cmd/eventreader/main.go

func registerEventHandlers(service *EventReaderService) error {
    // Register customer created handler
    err := service.RegisterHandler(
        events.CustomerEventFactory{},
        func(ctx context.Context, event events.CustomerEvent) error {
            handler := NewOnCustomerCreated()
            return handler.Handle(ctx, event)
        },
    )
    if err != nil {
        return fmt.Errorf("failed to register customer handler: %w", err)
    }
    
    return nil
}
```

**Reference:** `cmd/eventreader/main.go`

### Emitting Events

Services emit events through the event bus:

```go
func (s *CustomerService) emitCustomerCreated(
    ctx context.Context,
    customer *Customer,
) error {
    event := events.NewCustomerCreatedEvent(customer.CustomerID, map[string]string{
        "email": customer.Email,
    })
    
    if err := s.infrastructure.EventBus.Publish(ctx, event.Topic(), event); err != nil {
        return fmt.Errorf("failed to publish customer created event: %w", err)
    }
    
    return nil
}
```

**Note:** For database operations, use the outbox pattern instead (see 03-event-driven.md).

## Error Handling

### ServiceError Type

Use structured errors for service operations:

```go
// internal/platform/service/interface.go

type ServiceError struct {
    Service string
    Op      string
    Err     error
}

func (e *ServiceError) Error() string {
    return fmt.Sprintf("service %s: %s: %v", e.Service, e.Op, e.Err)
}

func (e *ServiceError) Unwrap() error {
    return e.Err
}
```

**Benefits:**
- Context about which service failed
- Operation that failed
- Chainable with `errors.Is`

### Error Patterns in Services

```go
func (s *CustomerService) GetCustomer(ctx context.Context, id string) (*Customer, error) {
    // Validate input
    if id == "" {
        return nil, ErrInvalidInput
    }
    
    // Delegate to repository
    customer, err := s.repo.GetCustomerByID(ctx, id)
    if err != nil {
        if errors.Is(err, ErrCustomerNotFound) {
            return nil, err  // Pass through domain errors
        }
        return nil, fmt.Errorf("failed to get customer: %w", err)
    }
    
    return customer, nil
}
```

**Key patterns:**
1. Validate inputs before repository calls
2. Pass through domain-specific errors
3. Wrap infrastructure errors with context
4. Never expose internal errors directly

## Service Composition

### HTTP Handler Integration

Services are used by HTTP handlers:

```go
// internal/service/customer/handler.go

type CustomerHandler struct {
    service *CustomerService
}

func NewCustomerHandler(service *CustomerService) *CustomerHandler {
    return &CustomerHandler{service: service}
}

func (h *CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
    // Parse request
    var req CreateCustomerRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Call service
    customer := &Customer{...}
    if err := h.service.CreateCustomer(r.Context(), customer); err != nil {
        // Handle specific errors
        if errors.Is(err, ErrDuplicateEmail) {
            http.Error(w, "Email already exists", http.StatusConflict)
            return
        }
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }
    
    // Return response
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(customer)
}
```

**Reference:** `internal/service/customer/handler.go`

## Testing Services

### Unit Testing

Test services with mock repositories:

```go
func TestCustomerService_CreateCustomer(t *testing.T) {
    // Setup mock
    mockRepo := &mockCustomerRepository{
        insertFunc: func(ctx context.Context, c *Customer) error {
            return nil
        },
    }
    
    // Create service with mock
    svc := NewCustomerServiceWithRepo(mockRepo, mockInfra, &Config{})
    
    // Test
    customer := &Customer{
        Email: "test@example.com",
    }
    
    err := svc.CreateCustomer(context.Background(), customer)
    if err != nil {
        t.Errorf("CreateCustomer() error = %v", err)
    }
}
```

**Reference:** 08-testing.md for detailed patterns

## Best Practices

### DO:
- ✅ Embed `BaseService` for standard lifecycle
- ✅ Group infrastructure in `Infrastructure` struct
- ✅ Accept context as first parameter
- ✅ Validate inputs before business logic
- ✅ Use domain errors, wrap infrastructure errors
- ✅ Provide testing constructor with dependency injection
- ✅ Keep methods focused (single responsibility)
- ✅ Return created/updated entities
- ✅ Use repository for all data access

### DON'T:
- ❌ Mix SQL queries with business logic
- ❌ Ignore context cancellation
- ❌ Return raw infrastructure errors
- ❌ Create circular dependencies between services
- ❌ Store context in service struct
- ❌ Use global state

## Migration Guide

### Creating a New Service

1. Define `Infrastructure` struct with all dependencies
2. Create `New{Service}Infrastructure()` constructor
3. Define service struct embedding `*service.BaseService`
4. Add standard and testing constructors
5. Implement business logic methods
6. Implement `Start()`, `Stop()`, `Health()` if needed
7. Create HTTP handler that uses the service
8. Write unit tests with mock repositories

### Adding a New Operation

1. Add method to service struct
2. Validate inputs
3. Check business rules
4. Call repository method
5. Handle side effects
6. Return appropriate error or result
7. Add unit tests
8. Add HTTP handler endpoint if needed
