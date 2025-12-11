package service

import (
	"context"
	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	kafka "go-shopping-poc/internal/platform/event/bus/kafka"
)

// EventServiceBase provides a base implementation for event-driven services
// It embeds BaseService and adds event-specific functionality
type EventServiceBase struct {
	*BaseService
	eventBus bus.Bus
	handlers []any // Store any type of handler registration
}

// NewEventServiceBase creates a new event service base with the given name and event bus
func NewEventServiceBase(name string, eventBus bus.Bus) *EventServiceBase {
	return &EventServiceBase{
		BaseService: NewBaseService(name),
		eventBus:    eventBus,
		handlers:    make([]any, 0),
	}
}

// Start begins consuming events from the event bus
func (s *EventServiceBase) Start(ctx context.Context) error {
	return s.eventBus.StartConsuming(ctx)
}

// RegisterHandler adds a typed event handler for any event type to the service
func RegisterHandler[T events.Event](s Service, factory events.EventFactory[T], handler bus.HandlerFunc[T]) error {
	if eventService, ok := s.(*EventServiceBase); ok {
		// Store the registration (we could store the actual types if needed for introspection)
		eventService.handlers = append(eventService.handlers, struct {
			factory events.EventFactory[T]
			handler bus.HandlerFunc[T]
		}{
			factory: factory,
			handler: handler,
		})

		// Register with the event bus - need to assert to kafka.EventBus for SubscribeTyped
		if kafkaBus, ok := eventService.eventBus.(*kafka.EventBus); ok {
			kafka.SubscribeTyped(kafkaBus, factory, handler)
			return nil
		}

		// If we can't assert to kafka.EventBus, we could try other implementations here
		// For now, return an error indicating unsupported event bus type
		return ErrUnsupportedEventBus
	}

	return &ServiceError{
		Service: s.Name(),
		Op:      "RegisterHandler",
		Err:     ErrUnsupportedEventBus,
	}
}

// EventBus returns the underlying event bus for advanced usage
func (s *EventServiceBase) EventBus() bus.Bus {
	return s.eventBus
}

// HandlerCount returns the number of registered handlers
func (s *EventServiceBase) HandlerCount() int {
	return len(s.handlers)
}
