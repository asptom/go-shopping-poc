package outbox

// Outbox writer is used to write events to the database outbox table.
//
// The outbox pattern guarantees that if a local transaction is committed, the
// corresponding event will eventually be published.

import (
	"context"

	"github.com/jmoiron/sqlx"

	"go-shopping-poc/pkg/event"
)

type Writer struct {
	db *sqlx.DB
}

// NewWriter creates a new Writer instance for writing outbox events.

func NewWriter(db *sqlx.DB) *Writer {
	return &Writer{db: db}
}

// WriteEvent writes an event to the outbox in the database within the provided transaction.

func (w *Writer) WriteEvent(ctx context.Context, tx sqlx.Tx, event event.EventInterface) error {

	// Convert the event to JSON payload
	payload, err := event.ToJSON()
	if err != nil {
		return err
	}

	// Insert the event into the outbox table
	query := `INSERT INTO outbox.outbox (event_type, topic, event_payload) VALUES ($1, $2, $3)`

	_, err = tx.ExecContext(ctx, query, event.GetType(), event.GetTopic(), payload)
	if err != nil {
		return err
	}
	return nil
}
