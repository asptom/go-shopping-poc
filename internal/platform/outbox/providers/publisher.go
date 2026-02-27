package providers

import (
	"log/slog"

	"go-shopping-poc/internal/platform/config"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/outbox"
)

// PublisherProvider provides outbox publisher functionality
type PublisherProvider interface {
	GetPublisher() *outbox.Publisher
}

// PublisherOption is a functional option for configuring PublisherProviderImpl.
type PublisherOption func(*PublisherProviderImpl)

// WithPublisherLogger sets the logger for the PublisherProviderImpl.
func WithPublisherLogger(logger *slog.Logger) PublisherOption {
	return func(p *PublisherProviderImpl) {
		p.logger = logger
	}
}

// PublisherProviderImpl implements publisher-only provider
type PublisherProviderImpl struct {
	publisher *outbox.Publisher
	logger    *slog.Logger
}

// NewPublisherProvider creates a publisher-only provider
//
// Parameters:
//   - db: Database instance for the outbox
//   - eventBus: Event bus for publishing events
//   - opts: Optional functional options for configuring the provider
//
// Usage:
//
//	provider := providers.NewPublisherProvider(db, eventBus)
//	// or with custom logger
//	provider := providers.NewPublisherProvider(db, eventBus, providers.WithPublisherLogger(logger))
func NewPublisherProvider(db database.Database, eventBus bus.Bus, opts ...PublisherOption) PublisherProvider {
	p := &PublisherProviderImpl{}

	for _, opt := range opts {
		opt(p)
	}

	if p.logger == nil {
		p.logger = Logger()
	}

	outboxLogger := p.logger.With("platform", "outbox", "component", "publisher")

	// Load platform outbox configuration
	cfg, err := config.LoadConfig[outbox.Config]("platform-outbox")
	if err != nil {
		p.logger.Error("PublisherProvider: Failed to load outbox config", "error", err)
		return nil
	}

	p.logger.Debug("PublisherProvider: Creating publisher-only provider")
	publisher := outbox.NewPublisher(db, eventBus, *cfg, outbox.WithPublisherLogger(outboxLogger))
	return &PublisherProviderImpl{
		publisher: publisher,
		logger:    p.logger,
	}
}

// GetPublisher returns the outbox publisher instance
func (p *PublisherProviderImpl) GetPublisher() *outbox.Publisher {
	return p.publisher
}
