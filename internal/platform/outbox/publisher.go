package outbox

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"

	"github.com/prometheus/client_golang/prometheus"
)

// Publisher is responsible for publishing events from the outbox to an external system (e.g., message broker).
type Publisher struct {
	db                database.Database
	publisher         bus.Bus
	batchSize         int           // Number of events to process in a single batch
	processInterval   time.Duration // Time between outbox scans
	shutdownCtx       context.Context
	shutdownCancel    context.CancelFunc
	immediateQueue    chan struct{} // Buffered channel for immediate processing requests
	maxConcurrent     int           // Maximum concurrent immediate processing goroutines
	currentProcessing int32         // Atomic counter for current processing goroutines
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
		immediateQueue:  make(chan struct{}, 100), // Buffer up to 100 immediate requests
		maxConcurrent:   10,                       // Max 10 concurrent processing goroutines
	}
}

// Start begins the publishing process.
func (p *Publisher) Start() {
	// Start background ticker for polling
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

	// Start immediate processing worker
	go p.processImmediateQueue()
}

func (p *Publisher) Stop() {
	p.shutdownCancel()
}

// processOutbox reads events from the outbox and publishes them.
func (p *Publisher) processOutbox() error {
	ctx := p.shutdownCtx

	query := `SELECT id, event_type, topic, event_payload, created_at, times_attempted, published_at FROM outbox.outbox WHERE published_at IS NULL LIMIT $1`

	rows, err := p.db.Query(ctx, query, p.batchSize)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Debug("Context cancelled during query, stopping")
			return nil
		}
		logger.Error("Failed to read outbox events", "error", err)
		return err
	}
	defer rows.Close()

	outboxEvents := []OutboxEvent{}
	for rows.Next() {
		var event OutboxEvent
		err := rows.Scan(&event.ID, &event.EventType, &event.Topic, &event.EventPayload, &event.CreatedAt, &event.TimesAttempted, &event.PublishedAt)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Debug("Context cancelled during scan, stopping iteration")
				return nil
			}
			logger.Error("Failed to scan outbox event", "error", err)
			continue
		}
		outboxEvents = append(outboxEvents, event)
	}

	if err := rows.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Debug("Context cancelled during iteration, stopping")
			return nil
		}
		logger.Error("Error iterating over outbox events", "error", err)
		return err
	}

	if len(outboxEvents) == 0 {
		//logger.Debug("No new outbox events to process")
		return nil
	}

	for _, outboxEvent := range outboxEvents {
		logger.Debug("Will publish outbox event to topic", "topic", outboxEvent.Topic)

		// Check if we're shutting down before processing each event
		select {
		case <-p.shutdownCtx.Done():
			logger.Debug("Shutdown detected, stopping event processing")
			return nil
		default:
		}

		// Use PublishRaw to avoid double marshaling and support both legacy and typed handlers
		err := p.publishEvent(ctx, outboxEvent)
		if err != nil {
			logger.Error("Failed to publish event", "event_id", outboxEvent.ID, "error", err)
			outboxEvent.TimesAttempted++

			updateQuery := "UPDATE outbox.outbox SET times_attempted = $1 WHERE id = $2"
			_, err = p.db.Exec(ctx, updateQuery, outboxEvent.TimesAttempted, outboxEvent.ID)
			if err != nil {
				logger.Error("Failed to update times attempted for event", "event_id", outboxEvent.ID, "error", err)
			}
			continue
		}
	}
	//logger.Debug("Processed %d outbox events", len(outboxEvents))
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

	logger.Info("Published event", "event_id", event.ID, "topic", event.Topic)
	return nil
}

// processImmediateQueue processes the immediate processing requests
func (p *Publisher) processImmediateQueue() {
	for {
		select {
		case <-p.shutdownCtx.Done():
			return
		case <-p.immediateQueue:
			// Check if we're at max concurrent
			current := atomic.LoadInt32(&p.currentProcessing)
			if current >= int32(p.maxConcurrent) {
				// Too many concurrent, skip this one (will be picked up by polling)
				logger.Debug("Max concurrent processing reached, skipping immediate")
				continue
			}

			// Increment counter and process
			atomic.AddInt32(&p.currentProcessing, 1)
			if err := p.processOutbox(); err != nil {
				logger.Error("Immediate processing error", "error", err)
			}
			atomic.AddInt32(&p.currentProcessing, -1)
		}
	}
}

// ProcessNow triggers immediate outbox processing
func (p *Publisher) ProcessNow() error {
	select {
	case <-p.shutdownCtx.Done():
		return nil
	case p.immediateQueue <- struct{}{}:
		// Queued for immediate processing
		return nil
	default:
		// Queue full, will be picked up by polling
		logger.Debug("Immediate queue full, event will be processed by polling")
		return nil
	}
}

// ProcessNowBlocking immediately processes pending outbox events and waits for completion
// Use sparingly - only when you need to ensure events are published before continuing
func (p *Publisher) ProcessNowBlocking(ctx context.Context) error {
	return p.processOutbox()
}
