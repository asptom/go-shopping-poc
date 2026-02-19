package eventhandlers

import (
	"context"
	"log"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/platform/sse"
)

// OnOrderCreated handles order.created events and publishes SSE notifications
type OnOrderCreated struct {
	sseHub *sse.Hub
}

// NewOnOrderCreated creates a new order created handler
func NewOnOrderCreated(sseHub *sse.Hub) *OnOrderCreated {
	return &OnOrderCreated{
		sseHub: sseHub,
	}
}

func (h *OnOrderCreated) Handle(ctx context.Context, event events.Event) error {
	log.Printf("[DEBUG] SSE: Received event - Type: %T, processing...", event)

	orderEvent, ok := event.(events.OrderEvent)
	if !ok {
		log.Printf("[ERROR] Cart: Expected OrderEvent, got %T", event)
		return nil
	}

	log.Printf("[DEBUG] SSE: Event received - Type: %s, ID: %s, Topic: %s", orderEvent.EventType, orderEvent.ID, orderEvent.Topic())

	if orderEvent.EventType != events.OrderCreated {
		log.Printf("[DEBUG] Cart: Ignoring event type: %s", orderEvent.EventType)
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(orderEvent.EventType),
		orderEvent.Data.OrderID,
		orderEvent.Data.CartID)

	log.Printf("[DEBUG] SSE: Order created event - OrderID: %s, OrderNumber: %s, CartID: %s, Total: %.2f", orderEvent.Data.OrderID, orderEvent.Data.OrderNumber, orderEvent.Data.CartID, orderEvent.Data.Total)

	// Push SSE event to subscribers
	if h.sseHub != nil {
		sseData := map[string]interface{}{
			"orderId":     orderEvent.Data.OrderID,
			"orderNumber": orderEvent.Data.OrderNumber,
			"cartId":      orderEvent.Data.CartID,
			"total":       orderEvent.Data.Total,
		}
		log.Printf("[DEBUG] SSE: Publishing 'order.created' event for cart %s", orderEvent.Data.CartID)

		h.sseHub.Publish(
			orderEvent.Data.CartID,
			"order.created",
			sseData,
		)
		log.Printf("[DEBUG] SSE: Successfully published 'order.created' event for cart %s", orderEvent.Data.CartID)
	} else {
		log.Printf("[WARN] SSE: sseHub is nil, cannot publish event for cart %s", orderEvent.Data.CartID)
	}

	return h.updateCartStatus(ctx, orderEvent.Data.CartID)
}

func (h *OnOrderCreated) updateCartStatus(ctx context.Context, cartID string) error {
	//Put the actual business logic for updating the cart status to completed here
	log.Printf("[INFO] Cart: Processing OrderCreated event for cart %s", cartID)
	_ = ctx // Placeholder to avoid unused variable error, replace with actual context usage in real implementation
	// Simulate processing time
	//time.Sleep(2 * time.Second)
	log.Printf("[INFO] Cart: Updating cart %s status to completed", cartID)
	return nil
}

// EventType returns the event type this handler processes
func (h *OnOrderCreated) EventType() string {
	return string(events.OrderCreated)
}

// CreateHandler returns a bus.HandlerFunc that wraps the Handle method
func (h *OnOrderCreated) CreateHandler() bus.HandlerFunc[events.OrderEvent] {
	return func(ctx context.Context, event events.OrderEvent) error {
		return h.Handle(ctx, event)
	}
}

// CreateFactory returns an EventFactory for OrderEvent
func (h *OnOrderCreated) CreateFactory() events.EventFactory[events.OrderEvent] {
	return events.OrderEventFactory{}
}

// Ensure OnOrderCreated implements the shared interfaces
var _ handler.EventHandler = (*OnOrderCreated)(nil)
var _ handler.HandlerFactory[events.OrderEvent] = (*OnOrderCreated)(nil)
