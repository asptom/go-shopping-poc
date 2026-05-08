# Go Shopping POC

Go 1.24.2, Clean Architecture microservices. Chi, pgx/sqlx, Kafka, MinIO, Viper, JWT/Keycloak.

## Commands

- Build all: `go build ./...`
- Test all: `go test ./...`
- Vet: `go vet ./...`
- CI (gofmt, goimports, vet, tests, arch guards, logging drift): `scripts/ci/standardization_checks.sh`

## Architecture

Three-layer Clean Architecture. Dependencies point INWARD:
`service/` → `platform/` → `contracts/events/`

- `internal/contracts/events/` — pure event DTOs with Event interface
- `internal/platform/` — shared infra (database, event bus, outbox, httpx, httperr, auth, config, logging, sse, websocket, storage)
- `internal/service/<domain>/` — business logic per bounded context: `entity.go`, `service.go`, `repository.go`, `handler.go`, `config.go`
- Entry point: `cmd/<name>/main.go` — load config → create infra → wire service → start HTTP

## Conventions

- **Interfaces**: no "I" prefix (`Service`, `Database`, `CustomerRepository`)
- **Constructors**: `New{Type}()` (e.g., `NewCustomerService`)
- **Error vars**: `Err{Description}` (e.g., `ErrNotFound`), wrap with `%w`
- **Repository**: split by operation — `_crud.go`, `_query.go`, `_address.go`, etc.
- **HTTP**: `httpx.DecodeJSON` → logic → `httpx.WriteJSON`, errors via `httperr.*`
- **Logging**: structured slog with `component`, `operation`, IDs, `request_id`, `duration_ms`, `error`
- **Outbox**: Writer (enqueue in DB tx) + Publisher (poll table → Kafka topic)
- **Events**: typed generics — `bus.SubscribeTyped[T](factory, HandlerFunc[T])`

## Delegation Protocol (MANDATORY)

You must delegate via the Task tool. Do not research or implement directly.

- **@explore** — ALL code reading, searching, file investigation. Trigger: any task requiring >1 file read.
- **@general** — ALL multi-step implementation. Trigger: any task with >2 steps or >1 file write.
- **Primary agent role**: strategy, decisions, quality review only. Never read or write files directly beyond verifying a single line.
- Not delegating wastes context and produces lower quality work. Delegate early, delegate often.

## Reference Docs (read the one matching your task)

- LLM behavior rules: `.go-docs/00-LLMrules.md`
- Event-driven patterns: `.go-docs/03-event-driven.md`
- Repository/transaction: `.go-docs/04-repository-pattern.md`
- Service layer: `.go-docs/05-service-layer.md`
- Error handling: `.go-docs/06-error-handling.md`
- Testing: `.go-docs/08-testing.md`
- Code examples: `.go-docs/10-project-examples.md`
- Standardization templates: `docs/standardization/templates/`
