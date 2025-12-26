package outbox

import (
	"context"
	"errors"
	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/database"

	"github.com/jmoiron/sqlx"
)

// Writer is a sql-backed outbox writer.
type Writer struct {
	db interface{} // Can be *sqlx.DB or database.Database
}

// NewWriter creates a new Writer instance for writing outbox events.
func NewWriter(db interface{}) *Writer {
	return &Writer{db: db}
}

// WriteEvent writes an Event to the outbox table using the provided transaction.
// tx can be *sqlx.Tx or database.Tx; this function does not Commit/Rollback.
func (w *Writer) WriteEvent(ctx context.Context, tx interface{}, evt events.Event) error {
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

	// Handle different transaction types
	switch t := tx.(type) {
	case *sqlx.Tx:
		_, err = t.ExecContext(ctx, query, evt.Type(), evt.Topic(), payload)
	case database.Tx:
		_, err = t.Exec(ctx, query, evt.Type(), evt.Topic(), payload)
	default:
		return errors.New("unsupported transaction type")
	}

	return err
}
