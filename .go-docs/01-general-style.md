# General Go Style and Idioms

This document covers Go language idioms, formatting, naming, and general conventions used throughout this project.

## Formatting

### gofmt
Always run `gofmt` on your code. The project follows standard Go formatting without custom rules.

**Key formatting rules:**
- Use **tabs** for indentation (gofmt default)
- No line length limit, but wrap long lines for readability
- Opening braces on same line: `func foo() {`
- No parentheses around control structures: `if err != nil {`

**Before committing:**
```bash
gofmt -w .
goimports -w .
```

### Import Organization
Imports must be grouped and ordered:

```go
import (
    // Standard library packages
    "context"
    "database/sql"
    "errors"
    "fmt"
    "time"
    
    // Project internal packages
    "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/service/customer"
    
    // External dependencies
    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
)
```

**Rules:**
1. Standard library first (no blank line after)
2. Blank line
3. Project internal packages
4. Blank line
5. External dependencies

Use `goimports` to enforce this automatically.

## Naming Conventions

### General Principles
- Use **CamelCase** for exported identifiers
- Use **camelCase** for unexported identifiers
- Keep names short but descriptive
- Avoid stutter: `customer.CustomerService` not `customer.CustomerServiceManager`

### Interfaces
**DO NOT** prefix with "I". Go interfaces are implicit and naming should reflect capability.

```go
// Good
 type Database interface { ... }
 type Service interface { ... }
 type EventHandler interface { ... }

// Bad
 type IDatabase interface { ... }
 type IService interface { ... }
```

### Structs
Use descriptive names. Exported structs start with uppercase.

```go
// Exported (public API)
type CustomerService struct { ... }
type EventBus struct { ... }

// Unexported (internal implementation)
type customerRepository struct { ... }
type eventHandler struct { ... }
```

### Constructor Functions
Use `New{Type}` pattern consistently.

```go
func NewCustomerService(infra *CustomerInfrastructure, cfg *Config) *CustomerService
func NewEventBus(config *Config) (*EventBus, error)
func NewWriter(db database.Database) *Writer
```

For testing with dependency injection, provide alternative constructors:

```go
// Standard constructor
func NewCustomerService(infra *CustomerInfrastructure, cfg *Config) *CustomerService

// Testing constructor with injected repository
func NewCustomerServiceWithRepo(repo CustomerRepository, infra *CustomerInfrastructure, cfg *Config) *CustomerService
```

### Error Variables
Use `Err` prefix for package-level error variables.

```go
var (
    ErrCustomerNotFound   = errors.New("customer not found")
    ErrDatabaseOperation  = errors.New("database operation failed")
    ErrTransactionFailed  = errors.New("transaction failed")
    ErrInvalidEvent       = errors.New("invalid event")
)
```

### Boolean Variables
Prefix with `is`, `has`, `can`, `allow`, or similar.

```go
var (
    isValid      bool
    hasConflict  bool
    canManage    bool
    allowAccess  bool
)
```

### Function and Method Names
- Use verbs for functions: `CreateCustomer`, `ValidateInput`, `ProcessOrder`
- Use nouns for pure functions that return values: `CustomerID`, `EventPayload`
- Keep receiver names short: 1-2 letters

```go
// Method receiver naming
func (c *Customer) Validate() error
func (r *customerRepository) GetByID(id string) (*Customer, error)
func (s *CustomerService) Create(ctx context.Context, customer *Customer) error
```

### Package Names
- Use short, lowercase, singular names
- No underscores, no mixed caps
- Name should match directory name

```go
// Good
package customer
package database
package events

// Bad
package customer_management
package CustomerPackage
package db_utils
```

## Documentation Comments

### Package Comments
Every package should have a package-level comment explaining its purpose.

```go
// Package customer provides business logic for customer management,
// including registration, authentication, and profile management.
package customer
```

### Function and Method Comments
Follow Go convention: complete sentence starting with the function name.

```go
// CreateCustomer creates a new customer record and emits a customer.created event.
// It validates the input, checks for duplicate emails, and returns an error
// if the customer cannot be created.
func (s *CustomerService) CreateCustomer(ctx context.Context, customer *Customer) error
```

### Interface Comments
Document what the interface represents and when to use it.

```go
// Database defines the interface for database operations.
// Implementations should handle connection pooling and support transactions.
type Database interface { ... }
```

## Interface Compliance Verification

Always verify interface compliance with compile-time checks.

```go
var _ EventHandler = (*OnCustomerCreated)(nil)
var _ HandlerFactory[events.CustomerEvent] = (*OnCustomerCreated)(nil)
var _ Service = (*CustomerService)(nil)
```

This ensures the type implements the interface at compile time, not runtime.

## Variable Declaration

### Group Related Variables

```go
var (
    // sentinel errors
    ErrNotFound      = errors.New("not found")
    ErrInvalidInput  = errors.New("invalid input")
    
    // configuration defaults
    defaultTimeout   = 30 * time.Second
    defaultBatchSize = 100
)
```

### Prefer Short Variable Names in Small Scopes

```go
// Good - short names in tight scopes
for i, item := range items {
    process(item)
}

// Good - longer names for package-level variables
var customerRepository CustomerRepository
```

## String Handling

### Use StringBuilder for Concatenation

```go
var builder strings.Builder
builder.Grow(estimatedSize)
for _, part := range parts {
    builder.WriteString(part)
}
result := builder.String()
```

### Use fmt.Sprintf for Formatting

```go
// Good
msg := fmt.Sprintf("Processing customer %s (ID: %s)", customer.Name, customer.ID)

// Avoid - less readable
msg := "Processing customer " + customer.Name + " (ID: " + customer.ID + ")"
```

## Constants

### Use iota for Sequential Values

```go
type EventType string

const (
    CustomerCreated  EventType = "customer.created"
    CustomerUpdated  EventType = "customer.updated"
    CustomerDeleted  EventType = "customer.deleted"
)
```

### Group Related Constants

```go
const (
    defaultPageSize = 20
    maxPageSize     = 100
    
    defaultTimeout  = 30 * time.Second
    maxRetries      = 3
)
```

## Type Declarations

### Define Domain Types

Use type aliases for domain concepts to improve type safety.

```go
type CustomerID string
type EventType string
type OrderStatus string
```

### Use Struct Tags Properly

```go
type Config struct {
    DatabaseURL  string `mapstructure:"db_url" validate:"required"`
    ServicePort  string `mapstructure:"service_port" validate:"required"`
}
```

## Control Flow

### Indent Error Flow

Keep the success path at minimum indentation.

```go
// Good - error handling first
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}
// success path continues here

// Bad - success path nested
if err == nil {
    // success logic nested one level
} else {
    return err
}
```

### Early Returns

Use early returns to reduce nesting.

```go
// Good
func (s *Service) Process(ctx context.Context, id string) error {
    if id == "" {
        return ErrInvalidInput
    }
    
    item, err := s.repo.Get(ctx, id)
    if err != nil {
        return fmt.Errorf("failed to get item: %w", err)
    }
    
    return s.processItem(item)
}
```

## Context Usage

### Always Accept Context as First Parameter

```go
func (r *Repository) Get(ctx context.Context, id string) (*Entity, error)
func (s *Service) Process(ctx context.Context, input *Input) error
func Publish(ctx context.Context, topic string, event Event) error
```

### Pass Context Through Call Chain

Never store context in a struct. Pass it through the call chain.

```go
// Good
func (s *Service) DoWork(ctx context.Context) error {
    return s.repo.Query(ctx, "SELECT ...")
}

// Bad - storing context
 type Service struct {
     ctx context.Context  // Don't do this
 }
```

## Avoiding Common Pitfalls

### Don't Use `init()` for Production Code

Avoid `init()` functions. Use explicit initialization in constructors.

### Don't Panic

Return errors instead of panicking. Only panic in truly exceptional situations.

### Don't Use Reflection Unnecessarily

Prefer type safety over reflection. Use generics where appropriate (Go 1.18+).

### Avoid Global State

Pass dependencies through constructors rather than using global variables.
