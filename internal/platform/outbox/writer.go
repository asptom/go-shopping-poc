package outbox

import (
	"context"
	"errors"
	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/database"
)

// Writer is a sql-backed outbox writer.
type Writer struct {
	db database.Database //
}

// NewWriter creates a new Writer instance for writing outbox events.
func NewWriter(db database.Database) *Writer {
	return &Writer{db: db}
}

// WriteEvent writes an Event to the outbox table using the provided transaction.
// tx can be *sqlx.Tx or database.Tx; this function does not Commit/Rollback.
func (w *Writer) WriteEvent(ctx context.Context, tx database.Tx, evt events.Event) error {
	if tx == nil {
		return errors.New("tx must be non-nil")
	}

	payload, err := evt.ToJSON()
	if err != nil {
		return err
	}

	query := `
        INSERT INTO outbox.outbox (event_type, topic, event_payload)
        VALUES ($1, $2, $3)
    `

	_, err = tx.Exec(ctx, query, evt.Type(), evt.Topic(), payload)
	if err != nil {
		return WrapWithContext(ErrWriteFailed, "failed to write event to outbox")
	}

	return nil
}
