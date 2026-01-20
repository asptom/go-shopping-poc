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
- Tests are currently being refactored (see below for implementation approach)

### Test Implementation Approach (Customer Service - Jan 20, 2026)

**Scope**: Targeted, business-focused tests for customer service only

**Files Created** (6 files, ~855 lines total):
1. `cmd/customer/main_test.go` (~49 lines) - Health endpoint tests
2. `cmd/customer/README.md` (~42 lines) - Bootstrap test documentation
3. `internal/service/customer/config_test.go` (~92 lines) - Configuration tests
4. `internal/service/customer/entity_test.go` (~412 lines) - Domain entity validation
5. `internal/service/customer/service_test.go` (~248 lines) - Service logic tests with mocks
6. `internal/service/customer/validation_test.go` (~103 lines) - Event validation tests

**Files Modified** (1 file, 9 lines):
- `cmd/customer/main.go` - Extracted `healthHandler` function (9 lines added)

**What Was Tested**:
- ✅ Bootstrap layer: Health endpoint (HTTP status, headers, response body)
- ✅ Configuration: LoadConfig(), Validate() (success, missing required fields)
- ✅ Entities: Customer, Address, CreditCard (validation, helper methods)
- ✅ Services: CreateCustomer, PatchCustomer, ValidatePatchData, Transform* methods
- ✅ Validation: CustomerEvent, CustomerEventPayload (success, missing fields, invalid types)

**What Was NOT Tested** (per plan):
- ❌ Repository layer (1,391 lines) - SQL queries, transactions (integration test responsibility)
- ❌ HTTP handlers (373 lines) - Thin wrappers (integration test responsibility)
- ❌ Service utilities (50 lines) - Simple logging/error functions
- ❌ Event utils (122 lines) - Platform wrapper functions

**Test Coverage Results**:
- Bootstrap layer: 3.4% coverage
- Business logic layer: 9.6% coverage
- Combined: ~9% coverage on core business logic
- All 53 tests: PASS (0 failures)
- Test execution time: <1 second total

**Implementation Requirements**:

1. **Idiomatic Go Structure**:
   - Test files alongside source: `*_test.go` (not `/tests/` subdirectory)
   - Package naming: `package customer_test` for black-box testing
   - Import standard `testing` package (no external assertion libraries)

2. **Programmatic Configuration**:
   - Use `os.Setenv()` for test environment setup
   - Use `os.Unsetenv()` in `defer` for cleanup
   - Helper functions marked with `t.Helper()` for proper error reporting
   - No `.env.test` files

3. **Minimal, Focused Tests**:
   - Test business behavior, not implementation details
   - Test domain rules and validation logic
   - No infrastructure dependencies (database, Kafka)
   - Table-driven tests with clear names

4. **Hand-Written Mocks**:
   - Mock repository implemented in test files
   - No external mock libraries
   - Implement only methods called by tests (simple no-ops)
   - Interface: `CustomerRepository` from service package

5. **Test Patterns Used**:
   - `t.Parallel()` at start of each test for concurrent execution
   - Table-driven tests with struct slices for multiple scenarios
   - Clear test names describing what's being tested

6. **Environment Variables Used**:
   - Check actual env var names in `deploy/k8s/service/<service>-configmap.yaml`
   - Example: `db_url` (not `CUSTOMER_DB_URL`)
   - Verify mapstructure tags in `Config` struct

7. **Code Refactoring**:
   - Extract inline handler functions for testability
   - Update main.go to use extracted handler
   - No behavior changes, only testability improvement

8. **Running Tests**:
   ```bash
   # Run all customer service tests
   go test -v ./cmd/customer/... ./internal/service/customer/...

   # Run specific test file
   go test -v ./internal/service/customer/... -run TestCustomerValidate

   # Run with coverage
   go test -cover ./cmd/customer/... ./internal/service/customer/...
   ```

**Duplication Checklist for Other Services**:
1. [ ] Read service's entity.go to understand validation rules
2. [ ] Read service's service.go to understand orchestration and transformation methods
3. [ ] Read service's validation.go if event validation exists
4. [ ] Check if `NewXxxServiceWithRepo()` function exists for test injection
5. [ ] Find PatchRequest, PatchAddressRequest, PatchCreditCardRequest types
6. [ ] Create test files:
   - `cmd/<service>/main_test.go` - Health endpoint tests
   - `internal/service/<service>/config_test.go` - Configuration tests
   - `internal/service/<service>/entity_test.go` - Entity validation tests
   - `internal/service/<service>/service_test.go` - Service logic tests with mocks
   - `internal/service/<service>/validation_test.go` - Event validation tests
7. [ ] Refactor main.go: Extract inline handler if needed
8. [ ] Verify tests pass: `go test -v ./cmd/<service>/... ./internal/service/<service>/...`
9. [ ] Lint: `make <service>-lint`
10. [ ] Update AGENTS.md with test summary
