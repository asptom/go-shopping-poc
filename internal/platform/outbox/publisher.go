package outbox

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sync"
	"time"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"

	"github.com/jmoiron/sqlx"
)

// Publisher is responsible for publishing events from the outbox to an external system (e.g., message broker).

type Publisher struct {
	lifecycleCtx     context.Context
	lifecycleCancel  context.CancelFunc
	wg               sync.WaitGroup
	mu               sync.Mutex
	shuttingDown     bool
	operationInProg  bool
	db               interface{} // Can be *sqlx.DB or database.Database
	publisher        bus.Bus
	batchSize        int           // Number of events to process in a single batch
	deleteBatchSize  int           // Number of events to delete as a batch after processing
	processInterval  time.Duration // Time between outbox scans
	operationTimeout time.Duration // Timeout for individual operations
}

func NewPublisher(db interface{}, publisher bus.Bus, batchSize int, deleteBatchSize int, processInterval time.Duration) *Publisher {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Publisher{
		lifecycleCtx:     ctx,
		lifecycleCancel:  cancel,
		db:               db,
		publisher:        publisher,
		batchSize:        batchSize,
		deleteBatchSize:  deleteBatchSize,
		processInterval:  processInterval,
		operationTimeout: 25 * time.Second, // Allow operations to complete within shutdown window
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
			case <-p.lifecycleCtx.Done():
				return
			case <-ticker.C:
				p.processOutbox()
			}
		}
	}()
}

// Stop stops the publishing process gracefully.
func (p *Publisher) Stop() {
	p.mu.Lock()
	p.shuttingDown = true
	p.mu.Unlock()

	// Signal shutdown
	p.lifecycleCancel()

	// Wait for goroutine to finish with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Clean shutdown
		log.Printf("[DEBUG] Outbox Publisher: Clean shutdown completed")
	case <-time.After(10 * time.Second):
		// Force shutdown after timeout
		log.Printf("[WARN] Outbox Publisher: Shutdown timeout, forcing exit")
	}
}

// processOutbox reads events from the outbox and publishes them.
func (p *Publisher) processOutbox() {
	p.mu.Lock()
	if p.shuttingDown {
		p.mu.Unlock()
		return
	}
	p.operationInProg = true
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.operationInProg = false
		p.mu.Unlock()
	}()

	log.Printf("[DEBUG] Outbox Publisher: Processing outbox events...")

	// Create operation-specific context with timeout
	operationCtx, operationCancel := context.WithTimeout(context.Background(), p.operationTimeout)
	defer operationCancel()

	query := `SELECT id, event_type, topic, event_payload, created_at, times_attempted, published_at FROM outbox.outbox WHERE published_at IS NULL LIMIT $1`

	var rows *sql.Rows
	var err error

	// Handle different database types
	switch db := p.db.(type) {
	case *sqlx.DB:
		rows, err = db.QueryContext(operationCtx, query, p.batchSize)
	case database.Database:
		rows, err = db.Query(operationCtx, query, p.batchSize)
	default:
		log.Printf("[ERROR] Outbox Publisher: Unsupported database type")
		return
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("[DEBUG] Outbox Publisher: Context cancelled during query, stopping")
			return
		}
		log.Printf("[ERROR] Outbox Publisher: Failed to read outbox events: %v", err)
		return
	}
	defer func() { _ = rows.Close() }()

	outboxEvents := []OutboxEvent{}
	for rows.Next() {
		var event OutboxEvent
		err := rows.Scan(&event.ID, &event.EventType, &event.Topic, &event.EventPayload, &event.CreatedAt, &event.TimesAttempted, &event.PublishedAt)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Printf("[DEBUG] Outbox Publisher: Context cancelled during scan, stopping iteration")
				return
			}
			log.Printf("[ERROR] Outbox Publisher: Failed to scan outbox event: %v", err)
			continue
		}
		outboxEvents = append(outboxEvents, event)
	}

	if err := rows.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("[DEBUG] Outbox Publisher: Context cancelled during iteration, stopping")
			return
		}
		log.Printf("[ERROR] Outbox Publisher: Error iterating over outbox events: %v", err)
		return
	}

	if len(outboxEvents) == 0 {
		log.Printf("[DEBUG] Outbox Publisher: No new outbox events to process")
		return
	}

	for _, outboxEvent := range outboxEvents {
		log.Printf("[DEBUG] Outbox Publisher: Will publish outbox event to topic: %s", outboxEvent.Topic)

		// Check if we're shutting down before processing each event
		p.mu.Lock()
		if p.shuttingDown {
			p.mu.Unlock()
			log.Printf("[DEBUG] Outbox Publisher: Shutdown detected, stopping event processing")
			return
		}
		p.mu.Unlock()

		// Use PublishRaw to avoid double marshaling and support both legacy and typed handlers
		if err := p.publisher.PublishRaw(operationCtx, outboxEvent.Topic, outboxEvent.EventType, []byte(outboxEvent.EventPayload)); err != nil {
			// handle publish failure exactly as before
			log.Printf("[ERROR] Outbox Publisher: Failed to publish event %v: %v", outboxEvent.ID, err)
			outboxEvent.TimesAttempted++

			updateQuery := "UPDATE outbox.outbox SET times_attempted = $1 WHERE id = $2"
			switch db := p.db.(type) {
			case *sqlx.DB:
				if _, err := db.ExecContext(operationCtx, updateQuery, outboxEvent.TimesAttempted, outboxEvent.ID); err != nil {
					log.Printf("[ERROR] Outbox Publisher: Failed to update times attempted for event %v: %v", outboxEvent.ID, err)
				}
			case database.Database:
				if _, err := db.Exec(operationCtx, updateQuery, outboxEvent.TimesAttempted, outboxEvent.ID); err != nil {
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
			if _, err := db.ExecContext(operationCtx, updateQuery, outboxEvent.PublishedAt, outboxEvent.TimesAttempted, outboxEvent.ID); err != nil {
				log.Printf("[ERROR] Outbox Publisher: Failed to mark event %v as published: %v", outboxEvent.ID, err)
				continue
			}
		case database.Database:
			if _, err := db.Exec(operationCtx, updateQuery, outboxEvent.PublishedAt, outboxEvent.TimesAttempted, outboxEvent.ID); err != nil {
				log.Printf("[ERROR] Outbox Publisher: Failed to mark event %v as published: %v", outboxEvent.ID, err)
				continue
			}
		}

		log.Printf("[INFO] Outbox Publisher: Published event ID: %v to topic: %s with payload: %s", outboxEvent.ID, outboxEvent.Topic, outboxEvent.EventPayload)
	}

	log.Printf("[DEBUG] Outbox Publisher: Processed %d outbox events", len(outboxEvents))
}
