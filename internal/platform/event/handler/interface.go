package handler

import (
	"context"
	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
)

// EventHandler defines the interface for all event handlers
// This interface provides a contract for implementing type-safe event handlers
// that can process events from the event bus.
type EventHandler interface {
	// Handle processes the event and returns any error
	// Implementations should handle the event appropriately and return
	// any errors that occur during processing.
	Handle(ctx context.Context, event events.Event) error

	// EventType returns the event type this handler processes
	// This is used to route events to the correct handlers.
	EventType() string
}

// HandlerFactory creates event handlers with their factories
// This interface provides a generic factory pattern for creating typed handlers
// that can be registered with the event bus for any event type.
//
// Generic Parameter:
//
//	T: The specific event type (must implement events.Event interface)
//	   Examples: events.CustomerEvent, events.OrderEvent, events.ProductEvent
//
// Usage Example:
//
//	type MyEventHandler struct{}
//
//	func (h MyEventHandler) CreateFactory() events.EventFactory[MyEvent] {
//	    return &MyEventFactory{}
//	}
//
//	func (h MyEventHandler) CreateHandler() bus.HandlerFunc[MyEvent] {
//	    return func(ctx context.Context, event MyEvent) error {
//	        // Handle event business logic
//	        return nil
//	    }
//	}
type HandlerFactory[T events.Event] interface {
	// CreateFactory returns the event factory for this handler
	// The factory is used to reconstruct events from raw message data.
	CreateFactory() events.EventFactory[T]

	// CreateHandler returns the handler function
	// The handler function processes events of the specific type.
	CreateHandler() bus.HandlerFunc[T]
}
