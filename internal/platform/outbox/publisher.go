package outbox

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"

	"github.com/prometheus/client_golang/prometheus"
)

// Publisher is responsible for publishing events from the outbox to an external system (e.g., message broker).
type Publisher struct {
	db              database.Database
	publisher       bus.Bus
	batchSize       int           // Number of events to process in a single batch
	processInterval time.Duration // Time between outbox scans
	shutdownCtx     context.Context
	shutdownCancel  context.CancelFunc
}

var (
	eventsProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "outbox_events_processed_total",
			Help: "Total number of events processed",
		},
		[]string{"topic", "status"},
	)
	eventProcessingTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "outbox_processing_duration_seconds",
			Help: "Event processing duration",
		},
		[]string{"topic"},
	)
)

func init() {
	prometheus.MustRegister(eventsProcessedTotal)
	prometheus.MustRegister(eventProcessingTime)
}

func NewPublisher(db database.Database, publisher bus.Bus, cfg Config) *Publisher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Publisher{
		db:              db,
		publisher:       publisher,
		batchSize:       cfg.BatchSize,
		processInterval: cfg.ProcessInterval,
		shutdownCtx:     ctx,
		shutdownCancel:  cancel,
	}
}

// Start begins the publishing process.
func (p *Publisher) Start() {
	go func() {
		ticker := time.NewTicker(p.processInterval)
		defer ticker.Stop()

		for {
			select {
			case <-p.shutdownCtx.Done():
				return
			case <-ticker.C:
				p.processOutbox()
			}
		}
	}()
}

func (p *Publisher) Stop() {
	p.shutdownCancel()
}

// processOutbox reads events from the outbox and publishes them.
func (p *Publisher) processOutbox() error {
	ctx := p.shutdownCtx

	log.Printf("[DEBUG] Outbox Publisher: ---------------------------")
	log.Printf("[DEBUG] Outbox Publisher: Processing outbox events...")

	query := `SELECT id, event_type, topic, event_payload, created_at, times_attempted, published_at FROM outbox.outbox WHERE published_at IS NULL LIMIT $1`

	rows, err := p.db.Query(ctx, query, p.batchSize)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("[DEBUG] Outbox Publisher: Context cancelled during query, stopping")
			return nil
		}
		log.Printf("[ERROR] Outbox Publisher: Failed to read outbox events: %v", err)
		return err
	}
	defer rows.Close()

	outboxEvents := []OutboxEvent{}
	for rows.Next() {
		var event OutboxEvent
		err := rows.Scan(&event.ID, &event.EventType, &event.Topic, &event.EventPayload, &event.CreatedAt, &event.TimesAttempted, &event.PublishedAt)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Printf("[DEBUG] Outbox Publisher: Context cancelled during scan, stopping iteration")
				return nil
			}
			log.Printf("[ERROR] Outbox Publisher: Failed to scan outbox event: %v", err)
			continue
		}
		outboxEvents = append(outboxEvents, event)
	}

	if err := rows.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("[DEBUG] Outbox Publisher: Context cancelled during iteration, stopping")
			return nil
		}
		log.Printf("[ERROR] Outbox Publisher: Error iterating over outbox events: %v", err)
		return err
	}

	if len(outboxEvents) == 0 {
		log.Printf("[DEBUG] Outbox Publisher: No new outbox events to process")
		return nil
	}

	for _, outboxEvent := range outboxEvents {
		log.Printf("[DEBUG] Outbox Publisher: Will publish outbox event to topic: %s", outboxEvent.Topic)

		// Check if we're shutting down before processing each event
		select {
		case <-p.shutdownCtx.Done():
			log.Printf("[DEBUG] Outbox Publisher: Shutdown detected, stopping event processing")
			return nil
		default:
		}

		// Use PublishRaw to avoid double marshaling and support both legacy and typed handlers
		err := p.publishEvent(ctx, outboxEvent)
		if err != nil {
			log.Printf("[ERROR] Outbox Publisher: Failed to publish event %v: %v", outboxEvent.ID, err)
			outboxEvent.TimesAttempted++

			updateQuery := "UPDATE outbox.outbox SET times_attempted = $1 WHERE id = $2"
			_, err = p.db.Exec(ctx, updateQuery, outboxEvent.TimesAttempted, outboxEvent.ID)
			if err != nil {
				log.Printf("[ERROR] Outbox Publisher: Failed to update times attempted for event %v: %v", outboxEvent.ID, err)
			}
			continue
		}
	}
	log.Printf("[DEBUG] Outbox Publisher: Processed %d outbox events", len(outboxEvents))
	return nil
}

// publishEvent publishes a single event and marks it as published.
func (p *Publisher) publishEvent(ctx context.Context, event OutboxEvent) error {
	if err := p.publisher.PublishRaw(ctx, event.Topic, event.EventType, []byte(event.EventPayload)); err != nil {
		return WrapWithContext(ErrPublishFailed, fmt.Sprintf("failed to publish event %d", event.ID))
	}

	updateQuery := "UPDATE outbox.outbox SET published_at = NOW(), times_attempted = $1 WHERE id = $2"
	_, err := p.db.Exec(ctx, updateQuery, event.TimesAttempted+1, event.ID)
	if err != nil {
		return WrapWithContext(errors.New("failed to update event status"), "failed to mark event as published")
	}

	log.Printf("[INFO] Outbox Publisher: Published event ID: %v to topic %s", event.ID, event.Topic)
	return nil
}
