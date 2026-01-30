# Quick Reference Index

This project follows Clean Architecture with event-driven microservices. Use this index to quickly find the right pattern.

## Directory Structure

```
cmd/                    # Service entry points (one per service)
├── customer/main.go
├── product/main.go
├── eventreader/main.go
└── ...

internal/
├── contracts/events/   # Event DTOs - pure data structures
├── platform/           # Shared infrastructure (HOW)
│   ├── service/        # Base service lifecycle
│   ├── event/          # Event bus & handlers
│   ├── database/       # Database abstraction
│   ├── outbox/         # Transactional event publishing
│   ├── config/         # Configuration loading
│   └── errors/         # Error utilities
└── service/            # Business logic (WHAT)
    ├── customer/       # Customer bounded context
    ├── product/        # Product bounded context
    └── eventreader/    # Event processing
```

## Pattern Decision Tree

**Starting a new service?**
→ See: 02-clean-architecture.md, 05-service-layer.md, 10-project-examples.md

**Implementing event-driven features?**
→ See: 03-event-driven.md, 10-project-examples.md

**Adding database operations?**
→ See: 04-repository-pattern.md, 10-project-examples.md

**Handling errors?**
→ See: 06-error-handling.md

**Loading configuration?**
→ See: 07-configuration.md

**Writing tests?**
→ See: 08-testing.md, 10-project-examples.md

**Using goroutines/channels?**
→ See: 09-concurrency.md

## Key Conventions (At a Glance)

| Aspect | Convention | Reference |
|--------|------------|-----------|
| Interface naming | No "I" prefix: `Service`, `Database` | 01-general-style.md |
| Constructor pattern | `New{Type}()`: `NewCustomerService()` | 01-general-style.md |
| Error variables | `Err{Description}`: `ErrNotFound` | 01-general-style.md |
| Boolean naming | Prefix with `is`, `has`, `can`: `isValid` | 01-general-style.md |
| Interface compliance | `var _ Interface = (*Type)(nil)` | 01-general-style.md |
| Error handling | Check first, wrap with `%w` | 06-error-handling.md |
| File organization | One concern per file | 02-clean-architecture.md |

## Quick File References

- **Event Contracts**: `internal/contracts/events/common.go`
- **Event Bus**: `internal/platform/event/bus/interface.go`
- **Repository Pattern**: `internal/service/customer/repository.go`
- **Service Pattern**: `internal/service/customer/service.go`
- **Handler Pattern**: `internal/service/customer/handler.go`
- **Configuration**: `internal/platform/config/loader.go`
- **Main Entry**: `cmd/customer/main.go`
- **Tests**: `internal/service/customer/service_test.go`

## When to Use Which Pattern

**Use Clean Architecture layers when:**
- Building a new bounded context (customer, product, order)
- Need separation between business logic and infrastructure
- Want testable, maintainable code

**Use Event-Driven patterns when:**
- Services need to communicate asynchronously
- Implementing transactional outbox
- Building event processors

**Use Repository pattern when:**
- Abstracting database operations
- Need transaction support
- Want to swap database implementations

**Use Service layer when:**
- Orchestrating business operations
- Coordinating between repositories and events
- Implementing use cases
