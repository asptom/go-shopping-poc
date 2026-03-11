# Repository Transaction Template

Use this skeleton for multi-step write flows in repositories.

## Goals

- Keep transaction orchestration consistent.
- Preserve `%w` error wrapping for `errors.Is` compatibility.
- Keep outbox writes in the same transaction as state mutation.

## Skeleton

```go
package mydomain

import (
    "context"
    "fmt"
)

func (r *repository) ExecuteWrite(ctx context.Context, input Input) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }

    committed := false
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()

    if _, err := tx.ExecContext(ctx, `UPDATE ...`, input.ID); err != nil {
        return fmt.Errorf("update entity: %w", err)
    }

    evt := BuildDomainEvent(input)
    if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
        return fmt.Errorf("write outbox event: %w", err)
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }
    committed = true

    return nil
}
```

## Checklist

- Transaction pattern: `BeginTx -> rollback guard -> writes -> outbox write -> Commit -> committed=true`.
- Uses `%w` wrapping for all chained repository errors.
- No post-commit side effects inside repository transaction method.
- Service layer owns retry/backoff policy for contention and async trigger behavior.
