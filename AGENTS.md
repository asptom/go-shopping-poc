# AGENTS.md - Guidelines for Coding Agents

This document provides build commands and code style guidelines for agentic coding assistants working in this repository.

## Build / Lint / Test Commands

### Build Commands
```bash
# Build a specific service
make <service>-build
# Examples: make customer-build, make eventreader-build, make product-build

# Build all services
make services-build

# Direct Go build
go build -o bin/<service> ./cmd/<service>

# Build product loader
make product-loader-build
```

### Lint Commands
```bash
# Lint a specific service
make <service>-lint
# Examples: make customer-lint, make eventreader-lint

# Lint all services
make services-lint

# Direct linting
golangci-lint run ./cmd/<service>/...
```

### Test Commands
```bash
# Test a specific service
make <service>-test

# Test all services
make services-test

# Direct testing
go test ./cmd/<service>/...

# Run a single test
go test -run TestFunctionName ./path/to/package

# Run tests with verbose output
go test -v ./path/to/package
```

### Deployment Commands
```bash
# Deploy platform services (Postgres, Kafka, Minio, Keycloak)
make platform

# Deploy application services
make services

# Deploy everything
make install

# Uninstall all services
make uninstall
```

## Architecture Overview

This project follows **Clean Architecture** with strict layer separation:

```
internal/
├── contracts/events/          # Pure DTOs - Event data structures only
├── platform/                  # Shared infrastructure (HOW)
│   ├── service/              # Service lifecycle management
│   ├── event/bus/            # Message transport abstraction
│   ├── event/handler/        # Generic event utilities
│   ├── database/             # Database abstraction
│   ├── outbox/               # Outbox pattern implementation
│   └── cors/                 # CORS middleware
└── service/<domain>/          # Business logic + domain entities (WHAT)
    ├── entity.go             # Domain models and validation
    ├── service.go            # Business logic
    ├── repository.go         # Data access
    └── handler.go            # HTTP handlers
```

### Key Principles
- **Contracts-first**: Event contracts defined in `internal/contracts/events/` before any implementation
- **Domain-driven**: Each service has its own domain with entities, business rules, and validation
- **Interface-based**: All infrastructure dependencies abstracted behind interfaces
- **Event-driven**: All domain changes publish events via outbox pattern

## Code Style Guidelines

### Package Organization
- Each service package (`internal/service/<domain>/`) contains: `entity.go`, `service.go`, `repository.go`, `handler.go`, `config.go`, `validation.go`
- Platform packages provide shared infrastructure abstractions
- Event contracts are pure data structures with no business logic

### Imports
Order imports as follows (with blank line separators):
1. Standard library (alphabetical)
2. Third-party packages (alphabetical)
3. Internal packages (alphabetical)

```go
import (
    "context"
    "errors"
    "time"

    "github.com/google/uuid"
    "github.com/go-chi/chi/v5"

    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/service/customer"
)
```

### Naming Conventions
- **Packages**: lowercase, no underscores (e.g., `customer`, `handler`, `bus`)
- **Types/Interfaces**: PascalCase (e.g., `Customer`, `CustomerRepository`, `EventBus`)
- **Exported Methods**: PascalCase (e.g., `GetCustomerByID`, `CreateCustomer`)
- **Private Methods**: camelCase (e.g., `getAddressesByCustomerID`, `validateCustomer`)
- **Variables**: camelCase (e.g., `customerID`, `addressList`)
- **Constants**:
  - Type-specific: PascalCase (e.g., `CustomerCreated`)
  - String constants: UPPER_SCREAMING_SNAKE (e.g., `EVENT_TYPE_CUSTOMER_CREATED`)
- **Errors**: `ErrSomething` pattern at package level (e.g., `ErrCustomerNotFound`, `ErrInvalidUUID`)

### Struct Tags
```go
type Customer struct {
    CustomerID string `json:"customer_id" db:"customer_id"`
    Email      string `json:"email,omitempty" db:"email"`
}
```
- JSON tags: snake_case, use `omitempty` for optional fields
- DB tags: snake_case, match database column names exactly

### Error Handling
```go
// Define custom errors at package level
var (
    ErrCustomerNotFound  = errors.New("customer not found")
    ErrInvalidUUID       = errors.New("invalid UUID format")
    ErrDatabaseOperation = errors.New("database operation failed")
)

// Wrap errors with context
return fmt.Errorf("failed to fetch customer: %w", err)

// Return nil for "not found" (not an error)
if err == sql.ErrNoRows {
    return nil, nil
}
```

### Validation
- Domain entities have `Validate()` methods that return errors
- Handlers validate input before passing to service layer
- Service layer validates business logic before calling repository
- Use struct field validation in service layer for patch operations

### Transaction Management
```go
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

// ... perform operations ...

if err := tx.Commit(); err != nil {
    return fmt.Errorf("failed to commit transaction: %w", err)
}
committed = true
```

### Event Publishing
All domain changes publish events via outbox pattern within the same transaction:

```go
evt := events.NewCustomerCreatedEvent(customerID, details)
if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
    return fmt.Errorf("failed to publish event: %w", err)
}
```

### Logging
Use structured logging with prefixes and levels:
```go
log.Printf("[INFO] Customer: Starting service...")
log.Printf("[DEBUG] Repository: Fetching customer by ID...")
log.Printf("[ERROR] Failed to create customer: %v", err)
```

### Interface Design
- All infrastructure components use interfaces
- Repository interfaces defined in service package
- Platform provides interface implementations
- Constructor functions accept interfaces, not concrete types

### Service Pattern
```go
type CustomerService struct {
    *service.BaseService
    repo           CustomerRepository
    infrastructure *CustomerInfrastructure
    config         *Config
}

func NewCustomerService(infrastructure *CustomerInfrastructure, config *Config) *CustomerService {
    repo := NewCustomerRepository(infrastructure.Database, infrastructure.OutboxWriter)
    return &CustomerService{
        BaseService:    service.NewBaseService("customer"),
        repo:           repo,
        infrastructure: infrastructure,
        config:         config,
    }
}
```

### HTTP Handlers
- Use `github.com/go-chi/chi/v5` for routing
- Extract path parameters with `chi.URLParam(r, "param")`
- Validate input before calling service
- Use `errors.SendError()` for error responses
- Return appropriate HTTP status codes (201 for create, 204 for delete/no content, 200 for success)

### Configuration
- Each service has its own `config.go` with `LoadConfig()` function
- Configuration loaded from environment variables
- Use Viper for configuration management
- Platform config shared via ConfigMaps in Kubernetes

### Context Usage
- All service and repository methods accept `context.Context` as first parameter
- Pass context through all layers (handler → service → repository)
- Use context for request tracing and cancellation

### Pointer Usage
- Use pointers for optional struct fields (e.g., `DefaultShippingAddressID *uuid.UUID`)
- Use pointers for patch operations to distinguish "not set" from "set to empty"
- Use value types for required fields

### Database Queries
- Use `sqlx` for database operations with `NamedExecContext`, `GetContext`, `SelectContext`
- SQL keywords uppercase, identifiers lowercase (e.g., `SELECT * FROM customers.customer WHERE customer_id = $1`)
- Use parameterized queries to prevent SQL injection
- Schema names in queries (e.g., `customers.Customer`)

## Event System Usage

### Creating Events
```go
// Convenience constructor
event := events.NewCustomerCreatedEvent(customerID, details)

// Generic constructor
event := events.NewCustomerEvent(customerID, events.CustomerCreated, resourceID, details)
```

### Publishing Events
```go
publisher := outbox.NewPublisher(eventBus)
err := publisher.PublishEvent(ctx, event)
```

### Handling Events
```go
utils := handler.NewEventUtils()
err := utils.HandleEventWithValidation(ctx, event, func(ctx context.Context, event events.Event) error {
    // Business logic here
    return nil
})
```

## Go Version
- Target Go version: 1.24.2
- See `go.mod` for current version

## Testing
- Tests are currently being refactored (see README.md)
- Use table-driven tests where appropriate
- Mock repository interfaces for service layer testing
- Test files: `<name>_test.go` in the same package
