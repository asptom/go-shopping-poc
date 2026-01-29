package outbox

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"

	"github.com/prometheus/client_golang/prometheus"
)

// Publisher is responsible for publishing events from the outbox to an external system (e.g., message broker).
type Publisher struct {
	db               database.Database
	publisher        bus.Bus
	batchSize        int           // Number of events to process in a single batch
	processInterval  time.Duration // Time between outbox scans
	operationTimeout time.Duration // Timeout for individual operations

	// Lifecycle management
	isStarted bool
	stopChan  chan struct{}
	mu        sync.Mutex
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
	return &Publisher{
		db:               db,
		publisher:        publisher,
		batchSize:        cfg.BatchSize,
		processInterval:  cfg.ProcessInterval,
		operationTimeout: cfg.OperationTimeout,
	}
}

// Start begins the publishing process.
func (p *Publisher) Start() error {
	p.mu.Lock()
	if p.isStarted {
		p.mu.Unlock()
		return fmt.Errorf("publisher already started")
	}
	p.isStarted = true
	p.stopChan = make(chan struct{})
	p.mu.Unlock()

	go p.run()
	return nil
}

// Stop stops the publishing process gracefully.
func (p *Publisher) Stop() error {
	p.mu.Lock()
	if !p.isStarted {
		p.mu.Unlock()
		return fmt.Errorf("publisher not started")
	}
	p.isStarted = false
	p.mu.Unlock()

	// Signal stop
	close(p.stopChan)

	// Wait a bit for cleanup
	select {
	case <-time.After(5 * time.Second):
		log.Printf("[WARN] Publisher shutdown timeout")
	case <-time.After(1 * time.Second): // Graceful timeout
	}

	return nil
}

// run handles the main publisher loop
func (p *Publisher) run() {
	ticker := time.NewTicker(p.processInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			log.Printf("[DEBUG] Publisher stopping")
			return
		case <-ticker.C:
			p.processOutbox()
		}
	}
}

func (p *Publisher) processOutbox() {
	operationCtx, operationCancel := context.WithTimeout(context.Background(), p.operationTimeout)
	defer operationCancel()

	query := `SELECT id, event_type, topic, event_payload, created_at, times_attempted, published_at 
               FROM outbox.outbox 
               WHERE published_at IS NULL 
               LIMIT $1`

	log.Printf("[DEBUG] Batch size for query: %v", p.batchSize)
	log.Printf("[DEBUG] Query: %s", query)

	rows, err := p.db.Query(operationCtx, query, p.batchSize)
	if err != nil {
		log.Printf("[ERROR] Failed to read outbox events: %v", err)
		return
	}
	defer func() { _ = rows.Close() }()

	log.Printf("[DEBUG] Query executed successfully, checking rows")

	rowCount := 0
	for rows.Next() {
		rowCount++
		log.Printf("[DEBUG] Processing row %d", rowCount)

		var eventID int64
		var eventType string
		var topic string
		var eventPayload string
		var createdAt time.Time
		var timesAttempted int
		var publishedAt sql.NullTime

		if err := rows.Scan(&eventID, &eventType, &topic, &eventPayload, &createdAt, &timesAttempted, &publishedAt); err != nil {
			log.Printf("[ERROR] Failed to scan outbox event %d: %v", rowCount, err)
			continue
		}

		log.Printf("[DEBUG] Row %d - eventID: %d, eventType: %s, topic: %s", rowCount, eventID, eventType, topic)
		log.Printf("[DEBUG] Will publish outbox event to topic: %s", topic)

		start := time.Now()

		if err := p.publisher.PublishRaw(operationCtx, topic, eventType, []byte(eventPayload)); err != nil {
			log.Printf("[ERROR] Failed to publish event %d: %v", eventID, err)
			retryCount := timesAttempted + 1
			updateQuery := "UPDATE outbox.outbox SET times_attempted = $1 WHERE id = $2"
			_, updateErr := p.db.Exec(operationCtx, updateQuery, retryCount, eventID)
			if updateErr != nil {
				log.Printf("[ERROR] Failed to update retry count for event %v: %v", eventID, updateErr)
			}
			eventsProcessedTotal.WithLabelValues(topic, "failed").Inc()
			continue
		}

		// Mark as published
		now := time.Now()
		retryCount := timesAttempted + 1
		updateQuery := "UPDATE outbox.outbox SET published_at = $1, times_attempted = $2 WHERE id = $3"
		_, updateErr := p.db.Exec(operationCtx, updateQuery, now, retryCount, eventID)
		if updateErr != nil {
			log.Printf("[ERROR] Failed to mark event %v as published: %v", eventID, updateErr)
			continue
		}

		eventsProcessedTotal.WithLabelValues(topic, "success").Inc()
		eventProcessingTime.WithLabelValues(topic).Observe(time.Since(start).Seconds())
		log.Printf("[INFO] Published event ID: %v to topic: %s", eventID, topic)
	}

	// Check for iteration errors
	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] Error during row iteration: %v", err)
	}

	log.Printf("[DEBUG] Processed outbox events, total rows: %d", rowCount)
}

// processOutbox reads events from the outbox and publishes them.
func (p *Publisher) processOutbox_old() {
	// Use a separate context for this operation
	operationCtx, operationCancel := context.WithTimeout(context.Background(), p.operationTimeout)
	defer operationCancel()

	query := `SELECT id, event_type, topic, event_payload, created_at, times_attempted, published_at 
               FROM outbox.outbox 
               WHERE published_at IS NULL 
               LIMIT $1`

	log.Printf("[DEBUG] Batch size for query: %v", p.batchSize)

	// Execute with the operation context
	rows, err := p.db.Query(operationCtx, query, p.batchSize)
	if err != nil {
		log.Printf("[ERROR] Failed to read outbox events: %v", err)
		return
	}
	defer func() { _ = rows.Close() }() // Check error if needed

	log.Printf("[DEBUG] Query executed successfully")

	// Process events sequentially within the operation context
	rowCount := 0
	for rows.Next() {
		rowCount++
		log.Printf("[DEBUG] Processing row %d", rowCount)

		var eventID int64
		var eventType string
		var topic string
		var eventPayload string
		var createdAt time.Time
		var timesAttempted int
		var publishedAt sql.NullTime

		if err := rows.Scan(&eventID, &eventType, &topic, &eventPayload, &createdAt, &timesAttempted, &publishedAt); err != nil {
			log.Printf("[ERROR] Failed to scan outbox event: %v", err)
			continue
		}

		log.Printf("[DEBUG] Row %d - eventID: %d, eventType: %s, topic: %s", rowCount, eventID, eventType, topic)

		log.Printf("[DEBUG] Will publish outbox event to topic: %s", topic)

		// Record start time for metrics
		start := time.Now()

		// Publish with operation context
		if err := p.publisher.PublishRaw(operationCtx, topic, eventType, []byte(eventPayload)); err != nil {
			// Handle publish error and update retry count
			retryCount := timesAttempted + 1
			updateQuery := "UPDATE outbox.outbox SET times_attempted = $1 WHERE id = $2"
			_, updateErr := p.db.Exec(operationCtx, updateQuery, retryCount, eventID)

			if updateErr != nil {
				log.Printf("[ERROR] Failed to update retry count for event %v: %v", eventID, updateErr)
			}

			// Record failed event metric
			eventsProcessedTotal.WithLabelValues(topic, "failed").Inc()

			continue
		}

		// Mark as published
		now := time.Now()
		retryCount := timesAttempted + 1
		updateQuery := "UPDATE outbox.outbox SET published_at = $1, times_attempted = $2 WHERE id = $3"
		_, updateErr := p.db.Exec(operationCtx, updateQuery, now, retryCount, eventID)

		if updateErr != nil {
			log.Printf("[ERROR] Failed to mark event %v as published: %v", eventID, updateErr)
			continue
		}

		// Record successful event metric
		eventsProcessedTotal.WithLabelValues(topic, "success").Inc()
		eventProcessingTime.WithLabelValues(topic).Observe(time.Since(start).Seconds())

		log.Printf("[INFO] Published event ID: %v to topic: %s", eventID, topic)
	}

	log.Printf("[DEBUG] Processed outbox events")
}

// IsStarted returns true if the publisher is running
func (p *Publisher) IsStarted() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.isStarted
}
