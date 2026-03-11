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
		logger:  logger.With("component", "order_on_cart_checked_out"),
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
	log := h.logger.With(
		"operation", "handle_cart_checked_out",
		"event_id", cartEvent.ID,
		"event_type", cartEvent.EventType,
		"cart_id", cartEvent.EventPayload.CartID,
	)

	if cartEvent.EventType != events.CartCheckedOut {
		log.Debug("Ignore event type")
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(cartEvent.EventType),
		cartEvent.EventPayload.CartID,
		"")

	err := h.createOrderFromCart(ctx, cartEvent)

	utils.LogEventCompletion(ctx, string(cartEvent.EventType),
		cartEvent.EventPayload.CartID, err)
	if err != nil {
		log.Error("Cart checked out processing failed", "error", err.Error())
		return err
	}
	log.Info("Cart checked out processed")
	return err
}

func (h *OnCartCheckedOut) createOrderFromCart(ctx context.Context, cartEvent events.CartEvent) error {
	cartID := cartEvent.EventPayload.CartID
	log := h.logger.With("operation", "create_order_from_cart", "cart_id", cartID)
	log.Debug("Create order from cart")

	snapshot := cartEvent.EventPayload.CartSnapshot
	if snapshot == nil {
		log.Error("Cart snapshot missing")
		return fmt.Errorf("cart snapshot is required to create order")
	}

	order, err := h.service.CreateOrderFromSnapshot(ctx, cartID, snapshot)
	if err != nil {
		log.Error("Create order from cart failed", "error", err.Error())
		return err
	}

	log.Info("Order created from cart", "order_number", order.OrderNumber)
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
