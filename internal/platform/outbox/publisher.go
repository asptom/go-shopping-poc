package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	logger            *slog.Logger
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

// PublisherOption is a functional option for configuring Publisher.
type PublisherOption func(*Publisher)

// WithPublisherLogger sets the logger for the Publisher.
func WithPublisherLogger(logger *slog.Logger) PublisherOption {
	return func(p *Publisher) {
		p.logger = logger
	}
}

func NewPublisher(db database.Database, publisher bus.Bus, cfg Config, opts ...PublisherOption) *Publisher {
	p := &Publisher{
		db:              db,
		publisher:       publisher,
		batchSize:       cfg.BatchSize,
		processInterval: cfg.ProcessInterval,
		immediateQueue:  make(chan struct{}, 100), // Buffer up to 100 immediate requests
		maxConcurrent:   10,                       // Max 10 concurrent processing goroutines
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.logger == nil {
		p.logger = Logger()
	}

	p.logger = p.logger.With("platform", "outbox", "component", "outbox_publisher")

	ctx, cancel := context.WithCancel(context.Background())
	p.shutdownCtx = ctx
	p.shutdownCancel = cancel

	return p
}

// Start begins the publishing process.
func (p *Publisher) Start() {
	p.logger.Info("Outbox publisher started", "operation", "start_publisher", "process_interval", p.processInterval.String(), "batch_size", p.batchSize)

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
	p.logger.Info("Outbox publisher stopping", "operation", "stop_publisher")
	p.shutdownCancel()
}

// processOutbox reads events from the outbox and publishes them.
func (p *Publisher) processOutbox() error {
	ctx := p.shutdownCtx
	startedAt := time.Now()
	log := p.logger.With("operation", "process_outbox")
	processedCount := 0

	query := `SELECT id, event_type, topic, event_payload, created_at, times_attempted, published_at FROM outbox.outbox WHERE published_at IS NULL LIMIT $1`

	rows, err := p.db.Query(ctx, query, p.batchSize)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Debug("Outbox query cancelled")
			return nil
		}
		log.Error("Outbox read failed", "error", err)
		return err
	}
	defer rows.Close()

	outboxEvents := []OutboxEvent{}
	for rows.Next() {
		var event OutboxEvent
		err := rows.Scan(&event.ID, &event.EventType, &event.Topic, &event.EventPayload, &event.CreatedAt, &event.TimesAttempted, &event.PublishedAt)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Debug("Outbox scan cancelled")
				return nil
			}
			log.Error("Outbox scan failed", "error", err)
			continue
		}
		outboxEvents = append(outboxEvents, event)
	}

	if err := rows.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Debug("Outbox iteration cancelled")
			return nil
		}
		log.Error("Outbox iteration failed", "error", err)
		return err
	}

	if len(outboxEvents) == 0 {
		//logger.Debug("No new outbox events to process")
		return nil
	}

	for _, outboxEvent := range outboxEvents {
		log.Debug("Publish outbox event", "event_id", outboxEvent.ID, "event_type", outboxEvent.EventType, "topic", outboxEvent.Topic)

		// Check if we're shutting down before processing each event
		select {
		case <-p.shutdownCtx.Done():
			log.Debug("Outbox processing stopped due to shutdown")
			return nil
		default:
		}

		// Use PublishRaw to avoid double marshaling and support both legacy and typed handlers
		err := p.publishEvent(ctx, outboxEvent)
		if err != nil {
			log.Error("Publish outbox event failed", "event_id", outboxEvent.ID, "event_type", outboxEvent.EventType, "error", err)
			outboxEvent.TimesAttempted++

			updateQuery := "UPDATE outbox.outbox SET times_attempted = $1 WHERE id = $2"
			_, err = p.db.Exec(ctx, updateQuery, outboxEvent.TimesAttempted, outboxEvent.ID)
			if err != nil {
				log.Error("Update outbox retry count failed", "event_id", outboxEvent.ID, "error", err)
			}
			continue
		}
		processedCount++
	}
	log.Debug("Outbox batch processed", "status", "completed", "processed_count", processedCount, "duration_ms", time.Since(startedAt).Milliseconds())
	//logger.Debug("Processed %d outbox events", len(outboxEvents))
	return nil
}

// publishEvent publishes a single event and marks it as published.
func (p *Publisher) publishEvent(ctx context.Context, event OutboxEvent) error {
	log := p.logger.With(
		"operation", "publish_event",
		"event_id", event.ID,
		"event_type", event.EventType,
		"topic", event.Topic,
	)
	if err := p.publisher.PublishRaw(ctx, event.Topic, event.EventType, []byte(event.EventPayload)); err != nil {
		log.Warn("Publish raw event failed", "error", err)
		return WrapWithContext(ErrPublishFailed, fmt.Sprintf("failed to publish event %d", event.ID))
	}

	updateQuery := "UPDATE outbox.outbox SET published_at = NOW(), times_attempted = $1 WHERE id = $2"
	_, err := p.db.Exec(ctx, updateQuery, event.TimesAttempted+1, event.ID)
	if err != nil {
		log.Error("Mark outbox event as published failed", "error", err)
		return WrapWithContext(errors.New("failed to update event status"), "failed to mark event as published")
	}

	log.Debug("Outbox event published")
	return nil
}

// processImmediateQueue processes the immediate processing requests
func (p *Publisher) processImmediateQueue() {
	log := p.logger.With("operation", "process_immediate_queue")
	for {
		select {
		case <-p.shutdownCtx.Done():
			log.Debug("Immediate queue worker stopped")
			return
		case <-p.immediateQueue:
			// Check if we're at max concurrent
			current := atomic.LoadInt32(&p.currentProcessing)
			if current >= int32(p.maxConcurrent) {
				// Too many concurrent, skip this one (will be picked up by polling)
				log.Warn("Immediate processing skipped", "status", "max_concurrent", "max_concurrent", p.maxConcurrent)
				continue
			}

			// Increment counter and process
			atomic.AddInt32(&p.currentProcessing, 1)
			if err := p.processOutbox(); err != nil {
				log.Error("Immediate processing failed", "error", err)
			}
			atomic.AddInt32(&p.currentProcessing, -1)
		}
	}
}

// ProcessNow triggers immediate outbox processing
func (p *Publisher) ProcessNow() error {
	log := p.logger.With("operation", "trigger_process_now")
	select {
	case <-p.shutdownCtx.Done():
		return nil
	case p.immediateQueue <- struct{}{}:
		log.Debug("Immediate processing queued")
		// Queued for immediate processing
		return nil
	default:
		// Queue full, will be picked up by polling
		log.Warn("Immediate queue full", "status", "queued_for_polling")
		return nil
	}
}

// ProcessNowBlocking immediately processes pending outbox events and waits for completion
// Use sparingly - only when you need to ensure events are published before continuing
func (p *Publisher) ProcessNowBlocking(ctx context.Context) error {
	return p.processOutbox()
}
