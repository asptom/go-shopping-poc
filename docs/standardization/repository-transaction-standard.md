# Repository and Transaction Standard

This standard captures the Phase 4 conventions for repository error semantics, transaction orchestration, outbox usage, and read-modify-write concurrency safety.

## 1) Repository Conventions

- Sentinel errors are package-owned in each domain repository package (`internal/service/{cart,customer,order,product}/repository.go`).
- Repository implementations wrap lower-level failures with `%w` to preserve unwrapping for both domain sentinels and root causes.
- Repository file layout remains split by concern (CRUD/query/related entities/utilities), preserving existing package boundaries.

## 2) Transaction Pattern (Required)

Use this sequence for multi-step writes:

1. `BeginTx(ctx, nil)`
2. `committed := false`
3. `defer rollback guard` (`if !committed { _ = tx.Rollback() }`)
4. Execute all domain writes and in-transaction outbox writes
5. `tx.Commit()`
6. Set `committed = true`

Reference implementations:

- `internal/service/cart/repository_checkout.go`
- `internal/service/customer/repository_crud.go`
- `internal/service/order/repository_crud.go`
- `internal/service/product/repository_crud.go`

## 3) Outbox Usage Pattern (Required)

- Domain events are written through `outboxWriter.WriteEvent(ctx, tx, evt)` inside the same transaction as state mutation.
- Event write failure aborts the transaction path and surfaces wrapped domain error.
- Outbox publishing trigger remains post-commit at service orchestration level (for example asynchronous trigger in `internal/service/order/service.go`), not inside repository transactional mutations.

Reference implementations:

- `internal/service/cart/repository_checkout.go`
- `internal/service/customer/repository_util.go`
- `internal/service/order/repository_crud.go`
- `internal/service/product/repository_crud.go`

## 4) Read-Modify-Write Race Risk Audit and Mitigation

### High-risk patterns observed

- Status transition check before transactional update:
  - `internal/service/order/repository_crud.go` (`GetOrderByID` + `SetStatus` before write transaction)
- Patch flow with pre-read outside transaction:
  - `internal/service/customer/repository_crud.go` (`PatchCustomer` reads existing before `executePatchCustomer` tx)
- Item quantity update with read then write without lock:
  - `internal/service/cart/repository_items.go` (`UpdateItemQuantity`)

### Mitigation pattern (for follow-up migrations)

- Prefer a single transaction with one of:
  - `SELECT ... FOR UPDATE` for mutable rows before domain decision checks, or
  - optimistic compare-and-swap (`UPDATE ... WHERE current_status = $expected`) and row-count validation.
- Keep side effects (outbox writes) in the same transaction.
- Keep retry policy explicit in service layer for contention paths.

No behavior-changing locking strategy is introduced in this phase; this document defines the required pattern for incremental adoption.
