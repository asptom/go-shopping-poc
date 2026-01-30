# Clean Architecture

This document describes the Clean Architecture patterns used in this project, including directory structure, layer separation, and dependency rules.

## Overview

The project follows **Clean Architecture** (also known as Hexagonal Architecture or Ports and Adapters) with three main layers:

1. **Contracts** - Pure data structures (events, DTOs)
2. **Platform** - Infrastructure and shared utilities (HOW things work)
3. **Service** - Business logic and domain entities (WHAT the system does)

## Directory Structure

```
/Users/tom/Projects/Go/go-shopping-poc/
├── cmd/                          # Service entry points
│   ├── customer/main.go         # Customer service entry
│   ├── product/main.go          # Product service entry
│   ├── eventreader/main.go      # Event processing service
│   └── ...
│
├── internal/
│   ├── contracts/
│   │   └── events/              # Event data structures
│   │       ├── common.go        # Event interface definitions
│   │       ├── customer.go      # Customer events
│   │       └── product.go       # Product events
│   │
│   ├── platform/                # Shared infrastructure
│   │   ├── service/             # Service base implementations
│   │   ├── event/               # Event bus abstraction
│   │   │   ├── bus/            # Transport interface
│   │   │   │   ├── interface.go
│   │   │   │   └── kafka/      # Kafka implementation
│   │   │   └── handler/        # Generic event handlers
│   │   ├── database/            # Database abstraction
│   │   ├── outbox/              # Outbox pattern
│   │   ├── config/              # Configuration loading
│   │   ├── errors/              # Error utilities
│   │   └── providers/           # Provider constructors
│   │
│   └── service/                 # Business logic
│       ├── customer/            # Customer bounded context
│       │   ├── entity.go        # Domain models
│       │   ├── service.go       # Business logic
│       │   ├── repository.go    # Data access
│       │   ├── handler.go       # HTTP handlers
│       │   └── config.go        # Service config
│       ├── product/             # Product bounded context
│       └── eventreader/         # Event processing
│
├── deploy/                       # Kubernetes manifests
└── resources/                    # Supporting files
```

## Layer Responsibilities

### 1. Contracts Layer (`internal/contracts/`)

**Purpose:** Define pure data structures for cross-service communication.

**Characteristics:**
- No business logic
- No dependencies on other layers
- JSON-serializable
- Versioned event types

**Example structure:**
```go
// internal/contracts/events/customer.go
package events

type CustomerEvent struct {
    ID           string               `json:"id"`
    EventType    EventType            `json:"type"`
    Timestamp    time.Time            `json:"timestamp"`
    EventPayload CustomerEventPayload `json:"payload"`
}

type EventType string

const (
    CustomerCreated EventType = "customer.created"
    CustomerUpdated EventType = "customer.updated"
)
```

**When to use:**
- Defining event schemas for Kafka
- Creating DTOs for API responses
- Any data structure shared between services

**Reference:** `internal/contracts/events/common.go`, `customer.go`, `product.go`

### 2. Platform Layer (`internal/platform/`)

**Purpose:** Provide reusable infrastructure (HOW things work).

**Characteristics:**
- Implements technical concerns
- Exposes interfaces for abstraction
- No business logic
- Reusable across services

**Sub-packages:**

#### `platform/service/` - Service Lifecycle
Base implementations for service management.

**Key types:**
```go
type Service interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() error
    Name() string
}

type BaseService struct {
    name string
}
```

**Reference:** `internal/platform/service/interface.go`, `base.go`

#### `platform/event/` - Event Infrastructure
Event bus abstraction and implementations.

**Key components:**
- `bus/interface.go` - Transport interface (Kafka abstraction)
- `bus/kafka/` - Kafka-specific implementation
- `handler/interface.go` - Generic event handler utilities

**Reference:** `internal/platform/event/bus/interface.go`, `handler/interface.go`

#### `platform/database/` - Database Abstraction
Database interface hiding implementation details.

**Key interface:**
```go
type Database interface {
    Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
    Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
    BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
    // ...
}
```

**Reference:** `internal/platform/database/interface.go`

#### `platform/outbox/` - Transactional Events
Outbox pattern for reliable event publishing.

**Key components:**
- `Writer` - Writes events to outbox table
- `Publisher` - Polls and publishes events to Kafka

**Reference:** `internal/platform/outbox/writer.go`, `publisher.go`

#### `platform/config/` - Configuration
Generic configuration loading using Viper.

**Key function:**
```go
func LoadConfig[T any](serviceName string) (*T, error)
```

**Reference:** `internal/platform/config/loader.go`

### 3. Service Layer (`internal/service/`)

**Purpose:** Implement business logic (WHAT the system does).

**Characteristics:**
- Domain-specific
- Uses platform interfaces
- Contains business rules
- Organized by bounded context

**Package structure per service:**
```go
internal/service/customer/
├── entity.go        # Domain models and validation
├── service.go       # Business logic orchestration
├── repository.go    # Data access interface + implementation
├── handler.go       # HTTP handlers
├── config.go        # Service-specific configuration
└── validation.go    # Input validation (if complex)
```

**Reference:** `internal/service/customer/` as canonical example

## Dependency Rules

### The Dependency Rule

Dependencies must point **inward**:

```
Service → Platform → Contracts
   ↓         ↓
(internal)  (internal)
```

**Valid dependencies:**
- Service can import Platform
- Service can import Contracts
- Platform can import Contracts
- Platform can import other Platform packages

**Invalid dependencies:**
- Platform cannot import Service
- Contracts cannot import Platform or Service
- Services should not directly import other services (use events)

### Interface Segregation

Keep interfaces small and focused:

```go
// Good - focused interface
type CustomerRepository interface {
    GetByID(ctx context.Context, id string) (*Customer, error)
    Create(ctx context.Context, customer *Customer) error
    Update(ctx context.Context, customer *Customer) error
}

// Good - separate concerns
type Database interface {
    Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
    Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
    BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
}
```

## File Organization

### One Concern Per File

Organize files by responsibility, not by type:

```go
// entity.go - Domain models and validation
package customer

type Customer struct { ... }
type Address struct { ... }
func (c *Customer) Validate() error { ... }

// repository.go - Data access
package customer

type CustomerRepository interface { ... }
type customerRepository struct { ... }
func NewCustomerRepository(...) *customerRepository { ... }

// service.go - Business logic
package customer

type CustomerService struct { ... }
func NewCustomerService(...) *CustomerService { ... }
func (s *CustomerService) Create(...) error { ... }

// handler.go - HTTP handlers
package customer

type CustomerHandler struct { ... }
func NewCustomerHandler(...) *CustomerHandler { ... }
func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) { ... }
```

### Interface and Implementation

Keep interface and primary implementation in the same file or same package:

```go
// repository.go
package customer

// Interface defined here
type CustomerRepository interface {
    GetByID(ctx context.Context, id string) (*Customer, error)
}

// Implementation in same file (unexported)
type customerRepository struct {
    db database.Database
}

// Constructor exported
func NewCustomerRepository(db database.Database) CustomerRepository {
    return &customerRepository{db: db}
}
```

## Bounded Contexts

Each service package represents a bounded context:

```
service/
├── customer/        # Customer management context
├── product/         # Product catalog context
├── order/           # Order processing context
└── eventreader/     # Event processing context (cross-cutting)
```

**Principles:**
- Each context owns its data and business rules
- Contexts communicate via events, not direct calls
- No shared domain models between contexts
- Each context has its own repository, service, and entity definitions

## Provider Pattern

Use provider functions to construct infrastructure in `main.go`:

```go
// platform/providers/providers.go
type Provider func() (interface{}, error)

// database provider
func NewDatabaseProvider(cfg *database.Config) (*DatabaseProvider, error) { ... }

// event bus provider  
func NewEventBusProvider(cfg *event.Config) (*EventBusProvider, error) { ... }
```

**Benefits:**
- Lazy initialization
- Dependency injection
- Testable composition

**Reference:** `internal/platform/providers/providers.go`

## Entry Points (cmd/)

Each service has its own entry point:

```go
// cmd/customer/main.go
func main() {
    // 1. Load configuration
    cfg, err := customer.LoadConfig()
    
    // 2. Create infrastructure providers
    dbProvider, _ := database.NewDatabaseProvider(cfg.DatabaseURL)
    eventBusProvider, _ := event.NewEventBusProvider(cfg.EventConfig)
    
    // 3. Create infrastructure
    infrastructure := customer.NewCustomerInfrastructure(
        dbProvider.GetDatabase(),
        eventBusProvider.GetEventBus(),
        // ...
    )
    
    // 4. Create service
    service := customer.NewCustomerService(infrastructure, cfg)
    
    // 5. Setup HTTP handlers
    handler := customer.NewCustomerHandler(service)
    
    // 6. Start server
    // ...
}
```

**Reference:** `cmd/customer/main.go` as canonical example

## Testing Structure

Test files follow the same organization:

```go
// service_test.go - tests for service.go
package customer_test  // Note: separate package

import "go-shopping-poc/internal/service/customer"

func TestCustomerService_Create(t *testing.T) { ... }

// repository_test.go - tests for repository.go
func TestCustomerRepository_GetByID(t *testing.T) { ... }
```

**Key patterns:**
- Use `xxx_test` package (black box testing)
- Mock interfaces for unit tests
- Table-driven tests
- Parallel execution with `t.Parallel()`

**Reference:** `internal/service/customer/service_test.go`

## When to Create New Platform Package

Create a new package in `platform/` when:
- Logic is shared across multiple services
- It's a technical concern (not business logic)
- It abstracts external dependencies
- It has a clear, single responsibility

Examples:
- ✅ New database driver support → `platform/database/mysql/`
- ✅ New message broker → `platform/event/bus/rabbitmq/`
- ✅ Shared authentication → `platform/auth/`
- ❌ Customer-specific logic → Keep in `service/customer/`
- ❌ Product business rules → Keep in `service/product/`

## Migration Guide

### Adding a New Bounded Context

1. Create `internal/service/{domain}/`
2. Define entities in `entity.go`
3. Define repository interface in `repository.go`
4. Implement service in `service.go`
5. Create HTTP handlers in `handler.go`
6. Add configuration in `config.go`
7. Create entry point in `cmd/{domain}/main.go`
8. Add tests in `*_test.go`

### Adding Shared Infrastructure

1. Create `internal/platform/{concern}/`
2. Define interfaces in `interface.go`
3. Implement in appropriate sub-packages
4. Add provider in `platform/providers/`
5. Update `main.go` files to use new infrastructure
