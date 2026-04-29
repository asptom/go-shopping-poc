# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go Shopping POC is a microservices-based e-commerce proof-of-concept built with Go, demonstrating Clean Architecture, event-driven design (Kafka), and Kubernetes deployment. It uses Go 1.24.2 with Chi for HTTP, pgx/sqlx for PostgreSQL, segmentio/kafka-go for messaging, MinIO for object storage, and Viper for config.

## Commands

### Building and Running

```bash
go build ./...                          # Build all packages
go run cmd/customer/main.go             # Run a single service locally
```

### Testing

```bash
go test ./...                           # Run all tests
go test -v ./cmd/customer/... ./internal/service/customer/...  # Service-specific tests
gmake customer-test                     # Run customer service tests (alias)
```

### Linting and Standards

```bash
scripts/ci/standardization_checks.sh    # Run full CI checks: formatting, vet, tests, architecture guards, logging drift
```

### Deployment

```bash
gmake platform                           # Deploy platform services (Postgres, Kafka, MinIO, Keycloak, Grafana)
gmake services                           # Deploy application services + DB migrations
gmake install                            # Run platform + services (full deploy)
gmake uninstall                          # Uninstall all services and namespaces
```

## Architecture

### Three-Layer Structure

Dependencies point **inward**: Service -> Platform -> Contracts

```
internal/
  contracts/events/    # Pure DTOs and Event interfaces (no business logic)
  platform/            # Shared infrastructure (HOW things work)
    service/           # Service lifecycle management (BaseService, EventServiceBase)
    event/             # Event bus abstraction (bus/interface.go) + Kafka impl (bus/kafka/)
    outbox/            # Transactional outbox pattern (Writer + Publisher)
    database/          # PostgreSQL abstraction (pgx)
    httpx/             # HTTP transport helpers (decode/encode/params)
    httperr/           # Structured transport error responses
    config/            # Viper-based generic config loading
    sse/               # Server-sent events primitives
    websocket/         # WebSocket primitives
    storage/minio/     # MinIO object storage
    auth/              # JWT validation
    logging/           # Structured logging helpers
  service/             # Business logic by bounded context (WHAT the system does)
    customer/          # Customer CRUD + addresses + credit cards
    product/           # Product catalog
    order/             # Order processing
    cart/              # Shopping cart
    eventreader/       # Event processing service
```

### Service Entry Points

Each service lives in `cmd/<name>/main.go` and follows the same bootstrap pattern:
1. Load config -> 2. Create infrastructure (DB, event bus, outbox, CORS) -> 3. Wire service/handler -> 4. Start HTTP server with graceful shutdown

### Event System

Contracts-first event-driven communication via Kafka:
- Define events in `internal/contracts/events/` implementing the `Event` interface
- Use outbox pattern for transactional event publishing (write to outbox table in same DB tx as business data)
- `platform/outbox/Publisher` polls the outbox table and pushes to Kafka
- Services subscribe via `EventBus.SubscribeTyped()` with generic `HandlerFunc[T]`

### Each Service Package Contains

```
internal/service/<domain>/
  entity.go        # Domain models + validation
  service.go       # Business logic
  repository.go    # Data access (implements interfaces from contracts)
  handler.go       # HTTP handlers (Chi routers)
  config.go        # Service-specific config loading
```

### Key Platform Patterns

- **Provider pattern**: Infrastructure is constructed via provider functions in `main.go` (lazy init, DI)
- **EventServiceBase**: Embeds BaseService, adds event bus + handler registration for event-driven services
- **Generics**: Event handling uses Go generics for type safety (`HandlerFunc[T events.Event]`)

### Deployment

Services deploy to Kubernetes via Helm/manifests under `deploy/k8s/`. Platform services (Postgres, Kafka, MinIO, Keycloak) are deployed separately from application services. Config is split between platform ConfigMap (shared) and service-specific ConfigMaps.

## Reference Docs

Detailed architecture, patterns, and conventions are documented in `.go-docs/`:
- `02-clean-architecture.md` - Layer separation, dependency rules, file organization
- `03-event-driven.md` - Event contracts, event bus, typed handlers, outbox pattern
- `04-repository-pattern.md` - Repository interface patterns
- `05-service-layer.md` - Business logic patterns
- `06-error-handling.md` - Error handling conventions
- `07-configuration.md` - Configuration architecture
- `08-testing.md` - Testing patterns (mocks, table-driven tests)
- `00-LLMrules.md` - LLM-specific rules for this project
- `10-project-examples.md` - Code examples across patterns
