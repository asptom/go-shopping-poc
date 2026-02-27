package outbox

import (
	"context"
	"errors"
	"log/slog"

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/database"
)

// WriterOption is a functional option for configuring Writer.
type WriterOption func(*Writer)

// WithLogger sets the logger for the Writer.
func WithLogger(logger *slog.Logger) WriterOption {
	return func(w *Writer) {
		w.logger = logger
	}
}

// Writer is a sql-backed outbox writer.
type Writer struct {
	db     database.Database
	logger *slog.Logger
}

// NewWriter creates a new Writer instance for writing outbox events.
//
// Parameters:
//   - db: Database instance
//   - opts: Optional functional options
//
// Usage:
//
//	writer := outbox.NewWriter(db)
//	// or with custom logger
//	writer := outbox.NewWriter(db, outbox.WithLogger(logger))
func NewWriter(db database.Database, opts ...WriterOption) *Writer {
	w := &Writer{db: db}

	for _, opt := range opts {
		opt(w)
	}

	if w.logger == nil {
		w.logger = Logger()
	}

	w.logger = w.logger.With("platform", "outbox", "component", "writer")

	return w
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
