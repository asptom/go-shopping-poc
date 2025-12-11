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
// This interface provides a factory pattern for creating typed handlers
// that can be registered with the event bus.
type HandlerFactory interface {
	// CreateFactory returns the event factory for this handler
	// The factory is used to reconstruct events from raw message data.
	CreateFactory() events.EventFactory[events.CustomerEvent]

	// CreateHandler returns the handler function
	// The handler function processes events of the specific type.
	CreateHandler() bus.HandlerFunc[events.CustomerEvent]
}
