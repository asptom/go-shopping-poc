# Project Examples

This document provides real code examples from the go-shopping-poc project, demonstrating how patterns from other guides are applied in practice.

## Table of Contents

1. [Repository Implementation](#repository-implementation)
2. [Service Implementation](#service-implementation)
3. [Event Handler Implementation](#event-handler-implementation)
4. [Event Contract Definition](#event-contract-definition)
5. [Configuration Implementation](#configuration-implementation)
6. [Main Entry Point](#main-entry-point)
7. [HTTP Handler](#http-handler)
8. [Testing with Mocks](#testing-with-mocks)

---

## Repository Implementation

### Full Repository with Transactions

See: `internal/service/customer/repository.go`

```go
package customer

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    
    "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/outbox"
)

// Domain-specific sentinel errors
var (
    ErrCustomerNotFound   = errors.New("customer not found")
    ErrAddressNotFound    = errors.New("address not found")
    ErrDatabaseOperation  = errors.New("database operation failed")
    ErrTransactionFailed  = errors.New("transaction failed")
)

// CustomerRepository interface
type CustomerRepository interface {
    InsertCustomer(ctx context.Context, customer *Customer) error
    GetCustomerByID(ctx context.Context, customerID string) (*Customer, error)
    GetCustomerByEmail(ctx context.Context, email string) (*Customer, error)
    UpdateCustomer(ctx context.Context, customer *Customer) error
}

// Unexported implementation
type customerRepository struct {
    db           database.Database
    outboxWriter *outbox.Writer
}

// Constructor
func NewCustomerRepository(db database.Database, outbox *outbox.Writer) CustomerRepository {
    return &customerRepository{
        db:           db,
        outboxWriter: outbox,
    }
}

// Interface compliance check
var _ CustomerRepository = (*customerRepository)(nil)

// Transactional insert with outbox
func (r *customerRepository) InsertCustomer(ctx context.Context, customer *Customer) error {
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
    query := `
        INSERT INTO customers (customer_id, username, email)
        VALUES (:customer_id, :username, :email)
    `
    _, err = tx.NamedExecContext(ctx, query, customer)
    if err != nil {
        return fmt.Errorf("failed to insert customer: %w", err)
    }
    
    // Write event to outbox (same transaction!)
    evt := events.NewCustomerCreatedEvent(customer.CustomerID, nil)
    if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
        return fmt.Errorf("failed to write event to outbox: %w", err)
    }
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    committed = true
    
    return nil
}

// Query with error conversion
func (r *customerRepository) GetCustomerByID(ctx context.Context, customerID string) (*Customer, error) {
    query := `SELECT customer_id, username, email FROM customers WHERE customer_id = $1`
    
    var customer Customer
    err := r.db.GetContext(ctx, &customer, query, customerID)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrCustomerNotFound
        }
        return nil, fmt.Errorf("failed to get customer: %w", err)
    }
    
    return &customer, nil
}
```

---

## Service Implementation

### Complete Service with Infrastructure

See: `internal/service/customer/service.go`

```go
package customer

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    
    "go-shopping-poc/internal/platform/bus"
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/outbox"
    "go-shopping-poc/internal/platform/service"
)

// Infrastructure bundle
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

// Service implementation
type CustomerService struct {
    *service.BaseService
    repo           CustomerRepository
    infrastructure *CustomerInfrastructure
    config         *Config
}

// Standard constructor
func NewCustomerService(
    infrastructure *CustomerInfrastructure,
    config *Config,
) *CustomerService {
    repo := NewCustomerRepository(infrastructure.Database, infrastructure.OutboxWriter)
    
    return &CustomerService{
        BaseService:    service.NewBaseService("customer"),
        repo:           repo,
        infrastructure: infrastructure,
        config:         config,
    }
}

// Testing constructor with injected repository
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

// Business logic method
func (s *CustomerService) CreateCustomer(ctx context.Context, customer *Customer) error {
    // Validate input
    if err := customer.Validate(); err != nil {
        return fmt.Errorf("invalid customer data: %w", err)
    }
    
    // Check business rules
    existing, err := s.repo.GetCustomerByEmail(ctx, customer.Email)
    if err != nil && !errors.Is(err, ErrCustomerNotFound) {
        return fmt.Errorf("failed to check existing customer: %w", err)
    }
    if existing != nil {
        return ErrDuplicateEmail
    }
    
    // Execute operation
    if err := s.repo.InsertCustomer(ctx, customer); err != nil {
        return fmt.Errorf("failed to create customer: %w", err)
    }
    
    return nil
}
```

---

## Event Handler Implementation

### Typed Event Handler

See: `internal/service/eventreader/eventhandlers/on_customer_created.go`

```go
package eventhandlers

import (
    "context"
    "log"
    
    "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/platform/event/handler"
)

type OnCustomerCreated struct{}

func NewOnCustomerCreated() *OnCustomerCreated {
    return &OnCustomerCreated{}
}

func (h *OnCustomerCreated) Handle(ctx context.Context, event events.Event) error {
    // Type assertion
    var customerEvent events.CustomerEvent
    switch e := event.(type) {
    case events.CustomerEvent:
        customerEvent = e
    case *events.CustomerEvent:
        customerEvent = *e
    default:
        log.Printf("[ERROR] Expected CustomerEvent, got %T", event)
        return nil  // Don't retry
    }
    
    // Event type filtering
    if customerEvent.EventType != events.CustomerCreated {
        log.Printf("[DEBUG] Ignoring event type: %s", customerEvent.EventType)
        return nil
    }
    
    // Process event
    return h.processCustomerCreated(ctx, customerEvent)
}

func (h *OnCustomerCreated) processCustomerCreated(
    ctx context.Context,
    event events.CustomerEvent,
) error {
    // Business logic here
    log.Printf("[INFO] Processing customer created: %s", event.EventPayload.CustomerID)
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

// Interface compliance
var _ handler.EventHandler = (*OnCustomerCreated)(nil)
var _ handler.HandlerFactory[events.CustomerEvent] = (*OnCustomerCreated)(nil)
```

---

## Event Contract Definition

### Domain Event Definition

See: `internal/contracts/events/customer.go`

```go
package events

import (
    "encoding/json"
    "time"
    
    "github.com/google/uuid"
)

// Event type constants
type EventType string

const (
    CustomerCreated EventType = "customer.created"
    CustomerUpdated EventType = "customer.updated"
    CustomerDeleted EventType = "customer.deleted"
)

// Domain event struct
type CustomerEvent struct {
    ID           string               `json:"id"`
    EventType    EventType            `json:"type"`
    Timestamp    time.Time            `json:"timestamp"`
    EventPayload CustomerEventPayload `json:"payload"`
}

// Event payload
type CustomerEventPayload struct {
    CustomerID string            `json:"customer_id"`
    Details    map[string]string `json:"details,omitempty"`
}

// Event interface implementation
func (e CustomerEvent) Type() string        { return string(e.EventType) }
func (e CustomerEvent) Topic() string       { return "customer-events" }
func (e CustomerEvent) Payload() any        { return e.EventPayload }
func (e CustomerEvent) ToJSON() ([]byte, error) { return json.Marshal(e) }
func (e CustomerEvent) GetEntityID() string { return e.EventPayload.CustomerID }
func (e CustomerEvent) GetResourceID() string { return e.ID }

// Event factory
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

---

## Configuration Implementation

### Service Configuration

See: `internal/service/customer/config.go`

```go
package customer

import (
    "errors"
    
    "go-shopping-poc/internal/platform/config"
)

// Config struct with mapstructure tags
type Config struct {
    DatabaseURL  string `mapstructure:"db_url" validate:"required"`
    ServicePort  string `mapstructure:"customer_service_port" validate:"required"`
    WriteTopic   string `mapstructure:"customer_write_topic" validate:"required"`
    Group        string `mapstructure:"customer_group"`
}

// LoadConfig function
func LoadConfig() (*Config, error) {
    return config.LoadConfig[Config]("customer")
}

// Validate method
func (c *Config) Validate() error {
    if c.DatabaseURL == "" {
        return errors.New("database URL is required")
    }
    if c.ServicePort == "" {
        return errors.New("service port is required")
    }
    if c.WriteTopic == "" {
        return errors.New("write topic is required")
    }
    return nil
}
```

---

## Main Entry Point

### Service Main Function

See: `cmd/customer/main.go`

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/go-chi/chi/v5"
    
    "go-shopping-poc/internal/platform/cors"
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/event"
    "go-shopping-poc/internal/platform/providers"
    "go-shopping-poc/internal/service/customer"
)

func main() {
    defer func() {
        if r := recover(); r != nil {
            log.Fatalf("[FATAL] Panic recovered: %v", r)
        }
    }()
    
    // Load configuration
    cfg, err := customer.LoadConfig()
    if err != nil {
        log.Fatalf("[FATAL] Failed to load config: %v", err)
    }
    
    // Create infrastructure providers
    dbProvider, err := database.NewDatabaseProvider(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("[FATAL] Failed to create database provider: %v", err)
    }
    db := dbProvider.GetDatabase()
    defer db.Close()
    
    eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
    if err != nil {
        log.Fatalf("[FATAL] Failed to create event bus provider: %v", err)
    }
    eventBus := eventBusProvider.GetEventBus()
    
    // Create outbox
    writerProvider := providers.NewWriterProvider(db)
    publisherProvider := providers.NewPublisherProvider(db, eventBus)
    outboxPublisher := publisherProvider.GetPublisher()
    outboxPublisher.Start()
    defer outboxPublisher.Stop()
    
    corsProvider, err := cors.NewCORSProvider()
    if err != nil {
        log.Fatalf("[FATAL] Failed to create CORS provider: %v", err)
    }
    corsHandler := corsProvider.GetCORSHandler()
    
    // Create infrastructure and service
    infrastructure := customer.NewCustomerInfrastructure(
        db, eventBus, writerProvider.GetWriter(),
        outboxPublisher, corsHandler,
    )
    svc := customer.NewCustomerService(infrastructure, cfg)
    handler := customer.NewCustomerHandler(svc)
    
    // Setup HTTP server
    router := chi.NewRouter()
    router.Use(corsHandler)
    
    router.Post("/customers", handler.CreateCustomer)
    router.Get("/customers/{id}", handler.GetCustomer)
    
    server := &http.Server{
        Addr:         ":" + cfg.ServicePort,
        Handler:      router,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
    }
    
    // Start server in goroutine
    go func() {
        log.Printf("[INFO] Starting customer service on port %s", cfg.ServicePort)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("[FATAL] Server error: %v", err)
        }
    }()
    
    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    
    log.Println("[INFO] Shutting down...")
    
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Printf("[ERROR] Shutdown error: %v", err)
    }
}
```

---

## HTTP Handler

### REST Handler Implementation

See: `internal/service/customer/handler.go`

```go
package customer

import (
    "encoding/json"
    "errors"
    "net/http"
    
    "github.com/go-chi/chi/v5"
    
    "go-shopping-poc/internal/platform/errors"
)

type CustomerHandler struct {
    service *CustomerService
}

func NewCustomerHandler(service *CustomerService) *CustomerHandler {
    return &CustomerHandler{service: service}
}

func (h *CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
    var req CreateCustomerRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid request body")
        return
    }
    
    customer := &Customer{
        Username: req.Username,
        Email:    req.Email,
    }
    
    if err := h.service.CreateCustomer(r.Context(), customer); err != nil {
        if errors.Is(err, ErrDuplicateEmail) {
            errors.SendError(w, http.StatusConflict, errors.ErrorTypeValidation, "Email already exists")
            return
        }
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to create customer")
        return
    }
    
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(customer)
}

func (h *CustomerHandler) GetCustomer(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    
    customer, err := h.service.GetCustomer(r.Context(), id)
    if err != nil {
        if errors.Is(err, ErrCustomerNotFound) {
            errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Customer not found")
            return
        }
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve customer")
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(customer)
}
```

---

## Testing with Mocks

### Complete Test Example

See: `internal/service/customer/service_test.go`

```go
package customer_test

import (
    "context"
    "testing"
    
    "go-shopping-poc/internal/service/customer"
)

// Mock repository implementation
type mockCustomerRepository struct {
    insertCustomerFunc     func(ctx context.Context, c *customer.Customer) error
    getCustomerByIDFunc    func(ctx context.Context, id string) (*customer.Customer, error)
    getCustomerByEmailFunc func(ctx context.Context, email string) (*customer.Customer, error)
}

func (m *mockCustomerRepository) InsertCustomer(ctx context.Context, c *customer.Customer) error {
    return m.insertCustomerFunc(ctx, c)
}

func (m *mockCustomerRepository) GetCustomerByID(ctx context.Context, id string) (*customer.Customer, error) {
    return m.getCustomerByIDFunc(ctx, id)
}

func (m *mockCustomerRepository) GetCustomerByEmail(ctx context.Context, email string) (*customer.Customer, error) {
    return m.getCustomerByEmailFunc(ctx, email)
}

// Implement remaining interface methods...

func TestCustomerService_CreateCustomer_Success(t *testing.T) {
    t.Parallel()
    
    // Setup mock
    mockRepo := &mockCustomerRepository{
        insertCustomerFunc: func(ctx context.Context, c *customer.Customer) error {
            return nil
        },
        getCustomerByEmailFunc: func(ctx context.Context, email string) (*customer.Customer, error) {
            return nil, customer.ErrCustomerNotFound  // No duplicate
        },
    }
    
    // Create service with mock
    mockInfra := &customer.CustomerInfrastructure{}
    svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})
    
    // Test
    testCustomer := &customer.Customer{
        Username: "testuser",
        Email:    "test@example.com",
    }
    
    err := svc.CreateCustomer(context.Background(), testCustomer)
    if err != nil {
        t.Errorf("CreateCustomer() failed: %v", err)
    }
}

// Table-driven test example
func TestCustomerService_ValidatePatchData(t *testing.T) {
    t.Parallel()
    
    tests := []struct {
        name      string
        patchData *customer.PatchCustomerRequest
        wantError bool
    }{
        {
            name:      "nil patch data",
            patchData: nil,
            wantError: true,
        },
        {
            name: "valid patch",
            patchData: &customer.PatchCustomerRequest{
                UserName: strPtr("newusername"),
            },
            wantError: false,
        },
    }
    
    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            
            mockRepo := &mockCustomerRepository{}
            mockInfra := &customer.CustomerInfrastructure{}
            svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})
            
            err := svc.ValidatePatchData(tt.patchData)
            if (err != nil) != tt.wantError {
                t.Errorf("ValidatePatchData() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}

func strPtr(s string) *string {
    return &s
}
```

---

## Quick Reference: Pattern Checklist

When implementing new code, verify against this checklist:

### Repository
- [ ] Define domain-specific sentinel errors
- [ ] Create interface with context-first methods
- [ ] Implement unexported struct
- [ ] Add constructor returning interface
- [ ] Verify interface compliance: `var _ Interface = (*Type)(nil)`
- [ ] Use transactions for multi-step operations
- [ ] Write outbox events within transactions
- [ ] Convert sql.ErrNoRows to domain errors

### Service
- [ ] Define Infrastructure struct
- [ ] Embed *service.BaseService
- [ ] Provide standard and testing constructors
- [ ] Validate inputs before business logic
- [ ] Return domain errors, wrap infrastructure errors
- [ ] Implement Start, Stop, Health if needed

### Event Handler
- [ ] Implement EventHandler interface
- [ ] Implement HandlerFactory[T] interface
- [ ] Type assertion with switch for flexibility
- [ ] Filter by event type
- [ ] Separate business logic method
- [ ] Return nil for non-retryable errors

### Configuration
- [ ] Define Config struct with mapstructure tags
- [ ] Create LoadConfig() function
- [ ] Implement Validate() method
- [ ] Fail fast on invalid configuration

### Testing
- [ ] Use _test package for black box testing
- [ ] Create mock implementations
- [ ] Use table-driven tests
- [ ] Run tests in parallel with t.Parallel()
- [ ] Capture range variable: tt := tt
