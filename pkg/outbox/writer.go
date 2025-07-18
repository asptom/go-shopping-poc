package outbox

import (
	"github.com/jmoiron/sqlx"
)

type Writer struct {
	db *sqlx.DB
}

// NewWriter creates a new Writer instance for writing outbox events.

func NewWriter(db *sqlx.DB) *Writer {
	return &Writer{db: db}
}

// WriteEvent writes an OutboxEvent to the database.
func (w *Writer) WriteEvent(event OutboxEvent) error {
	// Insert the event into the outbox table
	query := `INSERT INTO outbox.outbox (id, created_at, event_type, event_payload, times_attempted) VALUES (:id, :created_at, :event_type, :event_payload, :times_attempted)`
	_, err := w.db.NamedExec(query, event)
	if err != nil {
		return err
	}
	return nil
}
