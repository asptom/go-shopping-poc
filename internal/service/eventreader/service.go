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
	config *Config // Store config for potential future use
}

// NewEventReaderService creates a new event reader service instance
func NewEventReaderService(eventBus bus.Bus, config *Config) *EventReaderService {
	return &EventReaderService{
		EventServiceBase: service.NewEventServiceBase("eventreader", eventBus),
		config:           config,
	}
}
