package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	event "go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// Publisher is responsible for publishing events from the outbox to an external system (e.g., message broker).

type Publisher struct {
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	db              *sqlx.DB
	publisher       *event.EventBus
	batchSize       int           // Number of events to process in a single batch
	deleteBatchSize int           // Number of events to delete as a batch after processing
	processInterval time.Duration // Time between outbox scans
}

func NewPublisher(db *sqlx.DB, publisher *event.EventBus, batchSize int, deleteBatchSize int, processInterval time.Duration) *Publisher {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Publisher{
		ctx:             ctx,
		cancel:          cancel,
		db:              db,
		publisher:       publisher,
		batchSize:       batchSize,
		deleteBatchSize: deleteBatchSize,
		processInterval: processInterval,
	}
	return p
}

// Start begins the publishing process.
func (p *Publisher) Start() {

	p.wg.Add(1)
	go func() {
		ticker := time.NewTicker(p.processInterval)

		defer ticker.Stop()
		defer p.wg.Done()
		defer p.wg.Wait()

		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				p.processOutbox()
			}
		}
	}()
}

// Stop stops the publishing process gracefully.

func (p *Publisher) Stop() {
	p.cancel()
	p.wg.Wait()
}

// processOutbox reads events from the outbox and publishes them.

func (p *Publisher) processOutbox() {
	logging.Info("Processing outbox events...")

	query := `SELECT * FROM outbox.outbox WHERE published_at IS NULL LIMIT $1`
	outboxEvents := []OutboxEvent{}

	if err := p.db.Select(&outboxEvents, query, p.batchSize); err != nil {
		logging.Error("Failed to read outbox events: %v", err)
		return
	}

	if len(outboxEvents) == 0 {
		logging.Info("No new outbox events to process")
		return
	}

	for _, outboxEvent := range outboxEvents {
		logging.Info("Will publish outbox event to topic: %s", outboxEvent.Topic)

		var evt event.Event[any]

		if err := json.Unmarshal(outboxEvent.EventPayload, &evt); err != nil {
			logging.Error("Failed to unmarshal event payload for event %s: %v", outboxEvent.ID, err)
			// Increment times attempted
			outboxEvent.TimesAttempted++
			if _, err := p.db.Exec("UPDATE outbox.outbox SET times_attempted = $1 WHERE id = $2", outboxEvent.TimesAttempted, outboxEvent.ID); err != nil {
				logging.Error("Failed to update times attempted for event %s: %v", outboxEvent.ID, err)
			}
			continue
		}

		if err := p.publisher.Publish(p.ctx, outboxEvent.Topic, &evt); err != nil {
			logging.Error("Failed to publish event %s: %v", outboxEvent.ID, err)
			// Increment times attempted
			outboxEvent.TimesAttempted++
			// Update the event in the database
			if _, err := p.db.Exec("UPDATE outbox.outbox SET times_attempted = $1 WHERE id = $2", outboxEvent.TimesAttempted, outboxEvent.ID); err != nil {
				logging.Error("Failed to update times attempted for event %s: %v", outboxEvent.ID, err)
			}
			continue
		}
		// Mark as published
		outboxEvent.TimesAttempted++
		outboxEvent.PublishedAt = sql.NullTime{Time: time.Now(), Valid: true}
		if _, err := p.db.Exec("UPDATE outbox.outbox SET published_at = $1, times_attempted = $2 WHERE id = $3", outboxEvent.PublishedAt, outboxEvent.TimesAttempted, outboxEvent.ID); err != nil {
			logging.Error("Failed to mark event %s as published: %v", outboxEvent.ID, err)
			continue
		}

		logging.Debug("Published event ID: %v to topic: %s with payload: %s", outboxEvent.ID, outboxEvent.Topic, outboxEvent.EventPayload)
	}

	logging.Info("Processed %d outbox events", len(outboxEvents))
}
