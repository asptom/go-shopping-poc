// Package eventreader implements the event processing service for the shopping platform.
// This service consumes events from Kafka and processes them using registered handlers.
//
// Architecture:
//   - Uses platform service infrastructure for lifecycle management
//   - Registers typed event handlers with compile-time type safety
//   - Processes events asynchronously with proper error handling
//
// Handler registration:
//
//	err := eventreader.RegisterHandler(service, factory, handlerFunc)
//
// Example usage:
//
//	// Create service
//	eventBus := kafka.NewEventBus(config)
//	service := eventreader.NewEventReaderService(eventBus)
//
//	// Register customer event handler
//	err := eventreader.RegisterHandler(
//	    service,
//	    customerHandler.CreateFactory(),
//	    customerHandler.CreateHandler(),
//	)
//
//	// Start processing
//	err = service.Start(ctx)
package eventreader

import (
	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/service"
)

// EventReaderInfrastructure defines the infrastructure components required by the eventreader service.
//
// This struct encapsulates all external dependencies that the eventreader service needs
// to function, primarily the event bus for consuming and processing events from Kafka.
// It follows clean architecture principles by clearly defining the infrastructure
// boundaries that the service depends on.
type EventReaderInfrastructure struct {
	// EventBus handles consuming and processing events from the message broker
	EventBus bus.Bus
}

// NewEventReaderInfrastructure creates a new EventReaderInfrastructure instance with the provided components.
//
// Parameters:
//   - eventBus: Event bus for consuming and processing events from Kafka
//
// Returns a configured EventReaderInfrastructure ready for use by the eventreader service.
func NewEventReaderInfrastructure(eventBus bus.Bus) *EventReaderInfrastructure {
	return &EventReaderInfrastructure{
		EventBus: eventBus,
	}
}

// Service defines the interface for event reader business operations
// This extends the platform service interface with domain-specific methods
type Service interface {
	service.Service
}

// RegisterHandler adds a new event handler for any event type to the service
// This is a convenience wrapper around the platform service RegisterHandler
func RegisterHandler[T events.Event](s Service, factory events.EventFactory[T], handler bus.HandlerFunc[T]) error {
	return service.RegisterHandler(s, factory, handler)
}

// EventReaderService implements the Service interface using platform infrastructure
type EventReaderService struct {
	*service.EventServiceBase
	infrastructure *EventReaderInfrastructure // Infrastructure components
	config         *Config                    // Store config for potential future use
}

// NewEventReaderService creates a new event reader service instance
func NewEventReaderService(infrastructure *EventReaderInfrastructure, config *Config) *EventReaderService {
	return &EventReaderService{
		EventServiceBase: service.NewEventServiceBase("eventreader", infrastructure.EventBus),
		infrastructure:   infrastructure,
		config:           config,
	}
}
