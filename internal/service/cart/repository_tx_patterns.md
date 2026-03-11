# Cart Repository Transaction Patterns (Canonical Examples)

This package is the canonical reference for Phase 4 transaction and outbox orchestration examples.

## Standard Transaction Skeleton

- Begin transaction: `tx, err := r.db.BeginTx(ctx, nil)`
- Guard rollback with `committed` flag
- Execute domain writes
- Write outbox event in same transaction
- Commit and set `committed = true`

See:

- `internal/service/cart/repository_checkout.go`
- `internal/service/cart/repository_crud.go`

## In-Transaction Reuse Helper Pattern

Reusable transactional helper methods should accept `database.Tx` and avoid creating nested transactions.

See:

- `internal/service/cart/repository_items.go` (`AddItemTx`)

## Error Wrapping Pattern

Repository errors should keep domain sentinel context and underlying causes with `%w` wrapping.

See:

- `internal/service/cart/repository_checkout.go`
- `internal/service/cart/repository_contact.go`
- `internal/service/cart/repository_items.go`
- `internal/service/cart/repository_payment.go`
