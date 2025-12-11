package kafka

import (
	"context"
	"fmt"

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
	evt, err := h.factory.FromJSON(data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return h.handler(ctx, evt)
}
