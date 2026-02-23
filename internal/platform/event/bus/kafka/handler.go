package kafka

import (
	"context"
	"fmt"
	"log"

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
)

// TypedHandler provides type-safe event handling using generics
type TypedHandler[T events.Event] struct {
	factory events.EventFactory[T]
	handler bus.HandlerFunc[T]
}

// NewTypedHandler creates a new typed handler with the given factory and handler function
func NewTypedHandler[T events.Event](factory events.EventFactory[T], handler bus.HandlerFunc[T]) *TypedHandler[T] {
	return &TypedHandler[T]{
		factory: factory,
		handler: handler,
	}
}

// Handle processes raw JSON data by unmarshaling it using the factory and then calling the handler
func (h *TypedHandler[T]) Handle(ctx context.Context, data []byte) error {
	log.Printf("[DEBUG] Eventbus: TypedHandler.Handle called with %d bytes of JSON data", len(data))
	log.Printf("[DEBUG] Eventbus: Raw JSON data: %s", string(data))

	evt, err := h.factory.FromJSON(data)
	if err != nil {
		log.Printf("[ERROR] Eventbus: Failed to unmarshal event: %v", err)
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("[DEBUG] Eventbus: Event unmarshaled successfully - type: %T", evt)

	result := h.handler(ctx, evt)
	if result != nil {
		log.Printf("[ERROR] Eventbus: Handler returned error: %v", result)
	} else {
		log.Printf("[DEBUG] Eventbus: Handler completed successfully")
	}
	return result
}
