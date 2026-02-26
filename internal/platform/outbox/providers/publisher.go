package providers

import (
	"go-shopping-poc/internal/platform/config"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/outbox"
)

// PublisherProvider provides outbox publisher functionality
type PublisherProvider interface {
	GetPublisher() *outbox.Publisher
}

// PublisherProviderImpl implements publisher-only provider
type PublisherProviderImpl struct {
	publisher *outbox.Publisher
}

// NewPublisherProvider creates a publisher-only provider
func NewPublisherProvider(db database.Database, eventBus bus.Bus) PublisherProvider {
	// Load platform outbox configuration
	cfg, err := config.LoadConfig[outbox.Config]("platform-outbox")
	if err != nil {
		logger.Error("PublisherProvider: Failed to load outbox config", "error", err)
		return nil
	}

	logger.Debug("PublisherProvider: Creating publisher-only provider")
	publisher := outbox.NewPublisher(db, eventBus, *cfg)
	return &PublisherProviderImpl{
		publisher: publisher,
	}
}

// GetPublisher returns the outbox publisher instance
func (p *PublisherProviderImpl) GetPublisher() *outbox.Publisher {
	return p.publisher
}
