package eventhandlers

import (
	"context"
	"fmt"
	"log/slog"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/platform/sse"
)

// OnOrderCreated handles order.created events and publishes SSE notifications
type OnOrderCreated struct {
	sseHub *sse.Hub
	logger *slog.Logger
}

// NewOnOrderCreated creates a new order created handler
func NewOnOrderCreated(sseHub *sse.Hub, logger *slog.Logger) *OnOrderCreated {
	if logger == nil {
		logger = slog.Default()
	}
	return &OnOrderCreated{
		sseHub: sseHub,
		logger: logger.With("component", "cart_on_order_created"),
	}
}

func (h *OnOrderCreated) Handle(ctx context.Context, event events.Event) error {
	orderEvent, ok := event.(events.OrderEvent)
	if !ok {
		h.logger.Error("Expected OrderEvent", "actual_type", fmt.Sprintf("%T", event))
		return nil
	}
	log := h.logger.With(
		"operation", "handle_order_created",
		"event_id", orderEvent.ID,
		"event_type", orderEvent.EventType,
	)

	log.Debug("Order event received",
		"topic", orderEvent.Topic(),
	)

	if orderEvent.EventType != events.OrderCreated {
		log.Debug("Ignore event type")
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(orderEvent.EventType),
		orderEvent.Data.OrderID,
		orderEvent.Data.CartID)

	log.Debug("Order created event",
		"order_id", orderEvent.Data.OrderID,
		"order_number", orderEvent.Data.OrderNumber,
		"cart_id", orderEvent.Data.CartID,
		"total", orderEvent.Data.Total,
	)

	// Push SSE event to subscribers
	if h.sseHub != nil {
		sseData := map[string]interface{}{
			"order_id":     orderEvent.Data.OrderID,
			"order_number": orderEvent.Data.OrderNumber,
			"cart_id":      orderEvent.Data.CartID,
			"total":        orderEvent.Data.Total,
		}
		log.Debug("Publish SSE order event", "cart_id", orderEvent.Data.CartID)

		h.sseHub.Publish(
			orderEvent.Data.CartID,
			"order.created",
			sseData,
		)
		log.Debug("Publish SSE order event complete", "cart_id", orderEvent.Data.CartID)
	} else {
		log.Warn("SSE hub unavailable", "cart_id", orderEvent.Data.CartID)
	}

	log.Info("Order event sent to frontend", "order_id", orderEvent.Data.OrderID, "cart_id", orderEvent.Data.CartID)
	return h.updateCartStatus(ctx, orderEvent.Data.CartID)
}

func (h *OnOrderCreated) updateCartStatus(ctx context.Context, cartID string) error {
	log := h.logger.With("operation", "update_cart_status", "cart_id", cartID)
	log.Debug("Update cart status requested")
	_ = ctx
	log.Debug("Update cart status complete", "status", "completed")
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
