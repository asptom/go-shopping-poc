package outbox

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"

	"github.com/jmoiron/sqlx"
)

// Publisher is responsible for publishing events from the outbox to an external system (e.g., message broker).

type Publisher struct {
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	db              interface{} // Can be *sqlx.DB or database.Database
	publisher       bus.Bus
	batchSize       int           // Number of events to process in a single batch
	deleteBatchSize int           // Number of events to delete as a batch after processing
	processInterval time.Duration // Time between outbox scans
}

func NewPublisher(db interface{}, publisher bus.Bus, batchSize int, deleteBatchSize int, processInterval time.Duration) *Publisher {
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
	log.Printf("[DEBUG] Outbox Publisher: Processing outbox events...")

	query := `SELECT id, event_type, topic, event_payload, created_at, times_attempted, published_at FROM outbox.outbox WHERE published_at IS NULL LIMIT $1`

	var rows *sql.Rows
	var err error

	// Handle different database types
	switch db := p.db.(type) {
	case *sqlx.DB:
		rows, err = db.QueryContext(p.ctx, query, p.batchSize)
	case database.Database:
		rows, err = db.Query(p.ctx, query, p.batchSize)
	default:
		log.Printf("[ERROR] Outbox Publisher: Unsupported database type")
		return
	}

	if err != nil {
		log.Printf("[ERROR] Outbox Publisher: Failed to read outbox events: %v", err)
		return
	}
	defer func() { _ = rows.Close() }()

	outboxEvents := []OutboxEvent{}
	for rows.Next() {
		var event OutboxEvent
		err := rows.Scan(&event.ID, &event.EventType, &event.Topic, &event.EventPayload, &event.CreatedAt, &event.TimesAttempted, &event.PublishedAt)
		if err != nil {
			log.Printf("[ERROR] Outbox Publisher: Failed to scan outbox event: %v", err)
			continue
		}
		outboxEvents = append(outboxEvents, event)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] Outbox Publisher: Error iterating over outbox events: %v", err)
		return
	}

	if len(outboxEvents) == 0 {
		log.Printf("[DEBUG] Outbox Publisher: No new outbox events to process")
		return
	}

	for _, outboxEvent := range outboxEvents {
		log.Printf("[DEBUG] Outbox Publisher: Will publish outbox event to topic: %s", outboxEvent.Topic)

		// Use PublishRaw to avoid double marshaling and support both legacy and typed handlers
		if err := p.publisher.PublishRaw(p.ctx, outboxEvent.Topic, outboxEvent.EventType, []byte(outboxEvent.EventPayload)); err != nil {
			// handle publish failure exactly as before
			log.Printf("[ERROR] Outbox Publisher: Failed to publish event %v: %v", outboxEvent.ID, err)
			outboxEvent.TimesAttempted++

			updateQuery := "UPDATE outbox.outbox SET times_attempted = $1 WHERE id = $2"
			switch db := p.db.(type) {
			case *sqlx.DB:
				if _, err := db.ExecContext(p.ctx, updateQuery, outboxEvent.TimesAttempted, outboxEvent.ID); err != nil {
					log.Printf("[ERROR] Outbox Publisher: Failed to update times attempted for event %v: %v", outboxEvent.ID, err)
				}
			case database.Database:
				if _, err := db.Exec(p.ctx, updateQuery, outboxEvent.TimesAttempted, outboxEvent.ID); err != nil {
					log.Printf("[ERROR] Outbox Publisher: Failed to update times attempted for event %v: %v", outboxEvent.ID, err)
				}
			}
			continue
		}
		// Mark as published
		outboxEvent.TimesAttempted++
		outboxEvent.PublishedAt = sql.NullTime{Time: time.Now(), Valid: true}

		updateQuery := "UPDATE outbox.outbox SET published_at = $1, times_attempted = $2 WHERE id = $3"
		switch db := p.db.(type) {
		case *sqlx.DB:
			if _, err := db.ExecContext(p.ctx, updateQuery, outboxEvent.PublishedAt, outboxEvent.TimesAttempted, outboxEvent.ID); err != nil {
				log.Printf("[ERROR] Outbox Publisher: Failed to mark event %v as published: %v", outboxEvent.ID, err)
				continue
			}
		case database.Database:
			if _, err := db.Exec(p.ctx, updateQuery, outboxEvent.PublishedAt, outboxEvent.TimesAttempted, outboxEvent.ID); err != nil {
				log.Printf("[ERROR] Outbox Publisher: Failed to mark event %v as published: %v", outboxEvent.ID, err)
				continue
			}
		}

		log.Printf("[INFO] Outbox Publisher: Published event ID: %v to topic: %s with payload: %s", outboxEvent.ID, outboxEvent.Topic, outboxEvent.EventPayload)
	}

	log.Printf("[DEBUG] Outbox Publisher: Processed %d outbox events", len(outboxEvents))
}
