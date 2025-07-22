package outbox

import (
	"context"
	event "go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type EventPublisher interface {
	Publish(ctx context.Context, topic string, event event.Event) error
}

type PublishEvent struct {
	EventType    string
	EventPayload any
}

func NewPublishEvent(eventType string, eventPayload any) PublishEvent {
	return PublishEvent{
		EventType:    eventType,
		EventPayload: eventPayload,
	}
}

func (e PublishEvent) Name() string { return e.EventType }
func (e PublishEvent) Payload() any {
	return e.EventPayload
}

type Reader struct {
	db              *sqlx.DB
	publisher       *event.KafkaEventBus
	ctx             context.Context
	cancel          context.CancelFunc
	batchSize       int // Number of events to process in a single batch
	deleteBatchSize int // Number of events to delete as a batch after processing
	wg              sync.WaitGroup
}

func NewReader(db *sqlx.DB, publisher *event.KafkaEventBus, batchSize int, deleteBatchSize int) *Reader {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Reader{
		db:              db,
		publisher:       publisher,
		ctx:             ctx,
		cancel:          cancel,
		batchSize:       batchSize,
		deleteBatchSize: deleteBatchSize,
	}
	return r
}

// Start begins reading events from the outbox and publishing them.
func (r *Reader) Start() {

	r.wg.Add(1)
	go func() {

		ticker := time.NewTicker(5 * time.Second) // Adjust the interval as needed
		logging.Debug("Outbox Reader started, processing events every 5 seconds")
		defer ticker.Stop()
		defer r.wg.Done()
		defer r.wg.Wait()

		for {
			select {
			case <-r.ctx.Done():
				return
			case <-ticker.C:
				r.processOutbox()
			}
		}
	}()
}

func (r *Reader) Stop() {
	logging.Info("Stopping Outbox Reader...")
	r.cancel()
	r.wg.Wait()
	logging.Info("Outbox Reader stopped")
}

// processOutbox reads events from the outbox and publishes them.
func (r *Reader) processOutbox() {
	logging.Info("Processing outbox events...")

	query := `SELECT id, created_at, event_type, event_payload, times_attempted FROM outbox.outbox LIMIT $1`
	outboxEvents := []OutboxEvent{}

	if err := r.db.Select(&outboxEvents, query, r.batchSize); err != nil {
		logging.Error("Failed to read outbox events: %v", err)
		return
	}

	if len(outboxEvents) == 0 {
		logging.Info("No new outbox events to process")
		return
	}
	topic := "CustomerEvent" // Assuming a single topic for simplicity - FIX THIS
	for _, outboxEvent := range outboxEvents {
		logging.Debug("Publishing event ID: %s, Type: %s, Created At: %s, Times Attempted: %d, with payload: %s",
			outboxEvent.ID, outboxEvent.EventType, outboxEvent.CreatedAt.Format(time.RFC3339), outboxEvent.TimesAttempted, string(outboxEvent.EventPayload))
		pe := NewPublishEvent(outboxEvent.EventType, outboxEvent.EventPayload)
		if err := r.publisher.Publish(r.ctx, topic, pe); err != nil {
			logging.Error("Failed to publish event %s: %v", pe, err)
			continue
		}
		logging.Debug("Published event ID: %s to topic: %s with payload: %s", outboxEvent.ID, topic, pe.Payload())

		logging.Info("Processed %d outbox events", len(outboxEvents))

		r.deleteProcessedEvents(outboxEvents)
		r.logProcessedEvents(outboxEvents)
	}
}

// deleteProcessedEvents deletes processed events from the outbox.
func (r *Reader) deleteProcessedEvents(events []OutboxEvent) {
	if len(events) == 0 {
		return
	}

	ids := make([]uuid.UUID, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}

	query := `DELETE FROM outbox.outbox WHERE id = ANY($1)`
	if _, err := r.db.Exec(query, ids); err != nil {
		logging.Error("Failed to delete processed outbox events: %v", err)
		return
	}

	logging.Info("Deleted %d processed outbox events", len(events))
}

func (r *Reader) logProcessedEvents(events []OutboxEvent) {
	query := `INSERT INTO outbox.processed_events (event_id, event_type, time_processed) VALUES (:id, :event_type, CURRENT_TIMESTAMP)`
	for _, event := range events {
		_, err := r.db.NamedExec(query, event)
		if err != nil {
			logging.Error("Failed to log processed event %s: %v", event.ID, err)
		}
		logging.Debug("Event ID: %s, Type: %s, Created At: %s, Times Attempted: %d",
			event.ID, event.EventType, event.CreatedAt.Format(time.RFC3339), event.TimesAttempted)
	}
}
