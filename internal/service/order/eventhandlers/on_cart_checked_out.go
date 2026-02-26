package eventhandlers

import (
	"context"
	"fmt"
	"log/slog"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/service/order"
)

type OnCartCheckedOut struct {
	service *order.OrderService
	logger  *slog.Logger
}

func NewOnCartCheckedOut(service *order.OrderService, logger *slog.Logger) *OnCartCheckedOut {
	if logger == nil {
		logger = slog.Default()
	}
	return &OnCartCheckedOut{
		service: service,
		logger:  logger.With("handler", "on_cart_checked_out"),
	}
}

func (h *OnCartCheckedOut) Handle(ctx context.Context, event events.Event) error {
	var cartEvent events.CartEvent
	switch e := event.(type) {
	case events.CartEvent:
		cartEvent = e
	case *events.CartEvent:
		cartEvent = *e
	default:
		h.logger.Error("Expected CartEvent", "actual_type", fmt.Sprintf("%T", event))
		return nil
	}

	if cartEvent.EventType != events.CartCheckedOut {
		h.logger.Debug("Ignoring non-CartCheckedOut event", "event_type", cartEvent.EventType)
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(cartEvent.EventType),
		cartEvent.EventPayload.CartID,
		"")

	err := h.createOrderFromCart(ctx, cartEvent)

	utils.LogEventCompletion(ctx, string(cartEvent.EventType),
		cartEvent.EventPayload.CartID, err)

	return err
}

func (h *OnCartCheckedOut) createOrderFromCart(ctx context.Context, cartEvent events.CartEvent) error {
	cartID := cartEvent.EventPayload.CartID
	h.logger.Info("Creating order from cart", "cart_id", cartID)

	snapshot := cartEvent.EventPayload.CartSnapshot
	if snapshot == nil {
		h.logger.Error("Cart snapshot is missing", "cart_id", cartID)
		return fmt.Errorf("cart snapshot is required to create order")
	}

	order, err := h.service.CreateOrderFromSnapshot(ctx, cartID, snapshot)
	if err != nil {
		h.logger.Error("Failed to create order from cart", "cart_id", cartID, "error", err.Error())
		return err
	}

	h.logger.Info("Successfully created order", "order_number", order.OrderNumber, "cart_id", cartID)
	return nil
}

func (h *OnCartCheckedOut) EventType() string {
	return string(events.CartCheckedOut)
}

func (h *OnCartCheckedOut) CreateFactory() events.EventFactory[events.CartEvent] {
	return events.CartEventFactory{}
}

func (h *OnCartCheckedOut) CreateHandler() bus.HandlerFunc[events.CartEvent] {
	return func(ctx context.Context, event events.CartEvent) error {
		return h.Handle(ctx, event)
	}
}
