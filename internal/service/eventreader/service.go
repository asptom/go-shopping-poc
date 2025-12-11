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
}

// NewEventReaderService creates a new event reader service instance
func NewEventReaderService(eventBus bus.Bus) *EventReaderService {
	return &EventReaderService{
		EventServiceBase: service.NewEventServiceBase("eventreader", eventBus),
	}
}
