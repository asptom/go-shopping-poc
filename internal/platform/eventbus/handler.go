package eventbus

import (
	"context"
	"fmt"
	event "go-shopping-poc/internal/platform/event"
)

// HandlerFunc defines a function type for handling typed events
type HandlerFunc[T event.Event] func(ctx context.Context, event T) error

// TypedHandler provides type-safe event handling using generics
type TypedHandler[T event.Event] struct {
	factory event.EventFactory[T]
	handler HandlerFunc[T]
}

// NewTypedHandler creates a new typed handler with the given factory and handler function
func NewTypedHandler[T event.Event](factory event.EventFactory[T], handler HandlerFunc[T]) *TypedHandler[T] {
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
