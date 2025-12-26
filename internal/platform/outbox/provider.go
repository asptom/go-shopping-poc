package outbox

import (
	"fmt"
	"log"

	"go-shopping-poc/internal/platform/config"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"
)

// OutboxProviderImpl implements the OutboxProvider interface.
// It encapsulates outbox pattern setup and provides configured outbox writer
// and publisher instances to services.
type OutboxProviderImpl struct {
	writer    *Writer
	publisher *Publisher
}

// OutboxProvider defines the interface for providing outbox pattern components.
// This interface is implemented by OutboxProviderImpl.
type OutboxProvider interface {
	// GetOutboxWriter returns a configured outbox writer for storing events
	GetOutboxWriter() *Writer

	// GetOutboxPublisher returns a configured outbox publisher for publishing events
	GetOutboxPublisher() *Publisher
}

// NewOutboxProvider creates a new outbox provider with the given database and event bus.
// It loads platform outbox configuration, creates outbox writer and publisher instances,
// and establishes the relationship between database and event bus for reliable event publishing.
//
// Parameters:
//   - db: A configured database instance for storing outbox events
//   - eventBus: A configured event bus for publishing events to external systems
//
// Returns:
//   - A configured OutboxProvider that provides outbox pattern components
//   - An error if configuration loading or component creation fails
//
// Usage:
//
//	provider, err := outbox.NewOutboxProvider(db, eventBus)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	writer := provider.GetOutboxWriter()
//	publisher := provider.GetOutboxPublisher()
func NewOutboxProvider(db database.Database, eventBus bus.Bus) (OutboxProvider, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if eventBus == nil {
		return nil, fmt.Errorf("event bus is required")
	}

	log.Printf("[INFO] OutboxProvider: Initializing outbox provider")

	// Load platform outbox configuration
	config, err := config.LoadConfig[Config]("platform-outbox")
	if err != nil {
		log.Printf("[ERROR] OutboxProvider: Failed to load outbox config: %v", err)
		return nil, fmt.Errorf("failed to load outbox config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Printf("[ERROR] OutboxProvider: Invalid outbox config: %v", err)
		return nil, fmt.Errorf("invalid outbox config: %w", err)
	}

	log.Printf("[DEBUG] OutboxProvider: Outbox config loaded successfully - batch size: %d, delete batch size: %d, process interval: %v, max retries: %d",
		config.BatchSize, config.DeleteBatchSize, config.ProcessInterval, config.MaxRetries)

	// Create outbox writer instance
	writer := NewWriter(db)
	if writer == nil {
		log.Printf("[ERROR] OutboxProvider: Failed to create outbox writer")
		return nil, fmt.Errorf("failed to create outbox writer")
	}

	log.Printf("[DEBUG] OutboxProvider: Outbox writer created successfully")

	// Create outbox publisher instance
	publisher := NewPublisher(db, eventBus, config.BatchSize, config.DeleteBatchSize, config.ProcessInterval)
	if publisher == nil {
		log.Printf("[ERROR] OutboxProvider: Failed to create outbox publisher")
		return nil, fmt.Errorf("failed to create outbox publisher")
	}

	log.Printf("[INFO] OutboxProvider: Outbox provider initialized successfully")

	return &OutboxProviderImpl{
		writer:    writer,
		publisher: publisher,
	}, nil
}

// GetOutboxWriter returns the configured outbox writer instance.
// The writer is ready for storing events in the outbox table within database transactions.
//
// Returns:
//   - A Writer instance that can be used for writing events to the outbox
//
// Usage:
//
//	writer := provider.GetOutboxWriter()
//	err := writer.WriteEvent(ctx, tx, event)
func (p *OutboxProviderImpl) GetOutboxWriter() *Writer {
	return p.writer
}

// GetOutboxPublisher returns the configured outbox publisher instance.
// The publisher is ready for publishing events from the outbox to external systems.
// Call Start() to begin the publishing process and Stop() to gracefully shut down.
//
// Returns:
//   - A Publisher instance that can be used for publishing outbox events
//
// Usage:
//
//	publisher := provider.GetOutboxPublisher()
//	publisher.Start()
//	defer publisher.Stop()
func (p *OutboxProviderImpl) GetOutboxPublisher() *Publisher {
	return p.publisher
}
