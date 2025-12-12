package service

import (
	"context"
	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"reflect"
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
	if eventService, ok := s.(EventService); ok {
		// Store the registration in the service for introspection
		// This works for both direct EventServiceBase and embedded cases
		storeHandlerInService(s, factory, handler)

		// Use interface method for registration (works across all environments)
		return eventService.EventBus().RegisterHandler(factory, handler)
	}

	return &ServiceError{
		Service: s.Name(),
		Op:      "RegisterHandler",
		Err:     ErrUnsupportedEventBus,
	}
}

// storeHandlerInService stores a handler registration in the service for introspection
// This handles both direct EventServiceBase and embedded EventServiceBase cases
func storeHandlerInService[T events.Event](s Service, factory events.EventFactory[T], handler bus.HandlerFunc[T]) {
	// Try direct type assertion first (for services that are directly EventServiceBase)
	if esb, ok := s.(*EventServiceBase); ok {
		esb.handlers = append(esb.handlers, struct {
			factory events.EventFactory[T]
			handler bus.HandlerFunc[T]
		}{
			factory: factory,
			handler: handler,
		})
		return
	}

	// Try to access embedded EventServiceBase using reflection
	// This handles cases like EventReaderService which embeds *EventServiceBase
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
		if v.Kind() == reflect.Struct {
			// Look for embedded EventServiceBase field
			for i := 0; i < v.NumField(); i++ {
				field := v.Field(i)
				if field.Kind() == reflect.Ptr && !field.IsNil() {
					fieldType := field.Type()
					if fieldType.Kind() == reflect.Ptr && fieldType.Elem().Name() == "EventServiceBase" {
						// Found embedded EventServiceBase
						if esb, ok := field.Interface().(*EventServiceBase); ok {
							esb.handlers = append(esb.handlers, struct {
								factory events.EventFactory[T]
								handler bus.HandlerFunc[T]
							}{
								factory: factory,
								handler: handler,
							})
							return
						}
					}
				}
			}
		}
	}

	// If we can't store the handler, that's okay - the functionality still works
	// The handler is registered with the eventbus and will process events correctly
}

// EventBus returns the underlying event bus for advanced usage
func (s *EventServiceBase) EventBus() bus.Bus {
	return s.eventBus
}

// HandlerCount returns the number of registered handlers
func (s *EventServiceBase) HandlerCount() int {
	return len(s.handlers)
}
