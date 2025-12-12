package handler

import (
	"context"
	"go-shopping-poc/internal/contracts/events"
)

// ExampleHandlerFactory demonstrates how to use the generic HandlerFactory[T]
// This example shows how to create handlers for different event types using the same interface.

// ExampleOrderHandler demonstrates a handler for OrderEvent (hypothetical)
type ExampleOrderHandler struct{}

// Assume we have an OrderEvent type that implements events.Event
// type OrderEvent struct { ... }

func (h *ExampleOrderHandler) Handle(ctx context.Context, event events.Event) error {
	// Handle order event logic
	return nil
}

func (h *ExampleOrderHandler) EventType() string {
	return "order.created"
}

// CreateFactory returns the factory for OrderEvent
// func (h *ExampleOrderHandler) CreateFactory() events.EventFactory[OrderEvent] {
//     return &OrderEventFactory{}
// }

// CreateHandler returns the typed handler for OrderEvent
// func (h *ExampleOrderHandler) CreateHandler() func(context.Context, OrderEvent) error {
//     return func(ctx context.Context, event OrderEvent) error {
//         return h.Handle(ctx, event)
//     }
// }

// Example usage of the generic HandlerFactory interface:

// func RegisterGenericHandler[T events.Event](
//     eventBus any, // bus.Bus interface
//     factory HandlerFactory[T],
// ) error {
//     handlerFunc := factory.CreateHandler()
//     // return eventBus.SubscribeTyped(handlerFunc)
//     return nil
// }

// This allows the same registration function to work for any event type:
//
//     customerHandler := &OnCustomerCreated{}
//     RegisterGenericHandler(eventBus, customerHandler)  // Works with CustomerEvent
//
//     orderHandler := &ExampleOrderHandler{}
//     RegisterGenericHandler(eventBus, orderHandler)     // Works with OrderEvent

// The generic interface provides:
// 1. Type safety at compile time
// 2. Reusable patterns across different event types
// 3. Clean separation between platform and domain layers
// 4. No coupling to specific event types in the platform layer
