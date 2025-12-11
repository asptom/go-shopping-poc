package outbox

import (
	"context"
	"errors"
	event "go-shopping-poc/internal/platform/event"

	"github.com/jmoiron/sqlx"
)

// Writer is a sql-backed outbox writer.
type Writer struct {
	db *sqlx.DB
}

// NewWriter creates a new Writer instance for writing outbox events.
func NewWriter(db *sqlx.DB) *Writer {
	return &Writer{db: db}
}

// WriteEvent writes an Event to the outbox table using the provided sqlx transaction.
// tx must be a non-nil *sqlx.Tx started by the caller; this function does not Commit/Rollback.
func (w *Writer) WriteEvent(ctx context.Context, tx *sqlx.Tx, evt event.Event) error {
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

	_, err = tx.ExecContext(ctx, query, evt.Type(), evt.Topic(), payload)
	return err
}
