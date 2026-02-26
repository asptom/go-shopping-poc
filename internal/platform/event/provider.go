package event

import (
	"fmt"
	"log/slog"
	"os"

	configPkg "go-shopping-poc/internal/platform/config"
	"go-shopping-poc/internal/platform/event/bus"
	kafkabus "go-shopping-poc/internal/platform/event/bus/kafka"
	kafkaconfig "go-shopping-poc/internal/platform/event/kafka"
)

// EventBusProviderImpl implements the EventBusProvider interface.
// It encapsulates Kafka event bus setup and provides a configured event bus
// instance to services with service-specific topic and group configuration.
type EventBusProviderImpl struct {
	eventBus bus.Bus
}

// EventBusProvider defines the interface for providing event messaging infrastructure.
// This interface is implemented by EventBusProviderImpl.
type EventBusProvider interface {
	// GetEventBus returns a configured event bus instance
	GetEventBus() bus.Bus
}

// EventBusConfig defines the configuration for creating an event bus provider.
// It allows services to specify their own topic and group settings while using
// shared platform Kafka configuration.
type EventBusConfig struct {
	// WriteTopic is the Kafka topic used for publishing events (required)
	WriteTopic string

	// GroupID is the Kafka consumer group ID for this service (required)
	GroupID string
}

var (
	logger *slog.Logger
)

func init() {
	logger = slog.New(slog.NewJSONHandler(os.Stderr, nil)).
		With("platform", "event", "component", "event_bus_provider")
}

// NewEventBusProvider creates a new event bus provider with service-specific configuration.
// It loads the platform Kafka configuration, applies service-specific overrides,
// and creates a Kafka event bus instance.
//
// Parameters:
//   - config: Service-specific event bus configuration containing topic and group ID
//
// Returns:
//   - A configured EventBusProvider that provides event messaging infrastructure
//   - An error if configuration loading or event bus creation fails
//
// Usage:
//
//	config := event.EventBusConfig{
//	    WriteTopic: "customer-events",
//	    GroupID:    "customer-service",
//	}
//	provider, err := event.NewEventBusProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	bus := provider.GetEventBus()
func NewEventBusProvider(config EventBusConfig) (EventBusProvider, error) {
	if config.WriteTopic == "" {
		return nil, fmt.Errorf("write topic is required")
	}
	if config.GroupID == "" {
		return nil, fmt.Errorf("group ID is required")
	}

	logger.Info("EventBusProvider: Initializing event bus provider", "topic", config.WriteTopic, "group", config.GroupID)

	// Load platform Kafka configuration
	kafkaCfg, err := configPkg.LoadConfig[kafkaconfig.Config]("platform-kafka")
	if err != nil {
		logger.Error("EventBusProvider: Failed to load Kafka config", "error", err)
		return nil, fmt.Errorf("failed to load Kafka config: %w", err)
	}

	logger.Debug("EventBusProvider: Platform Kafka config loaded successfully")

	// Apply service-specific configuration overrides
	kafkaCfg.Topic = config.WriteTopic
	kafkaCfg.GroupID = config.GroupID

	logger.Debug("EventBusProvider: Applied service-specific config", "topic", kafkaCfg.Topic, "group", kafkaCfg.GroupID)

	// Create Kafka event bus
	eventBus := kafkabus.NewEventBus(kafkaCfg)
	if eventBus == nil {
		logger.Error("EventBusProvider: Failed to create event bus")
		return nil, fmt.Errorf("failed to create event bus")
	}

	logger.Info("EventBusProvider: Event bus provider initialized successfully")

	return &EventBusProviderImpl{
		eventBus: eventBus,
	}, nil
}

// GetEventBus returns the configured event bus instance.
// The event bus is ready for publishing and subscribing to events.
//
// Returns:
//   - A Bus interface implementation that can be used for event messaging
//
// Usage:
//
//	bus := provider.GetEventBus()
//	err := bus.Publish(ctx, "customer-events", customerCreatedEvent)
func (p *EventBusProviderImpl) GetEventBus() bus.Bus {
	return p.eventBus
}
