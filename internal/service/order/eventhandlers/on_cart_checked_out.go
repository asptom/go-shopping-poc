package eventhandlers

import (
	"context"
	"fmt"
	"log"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/service/order"
)

type OnCartCheckedOut struct {
	service *order.OrderService
}

func NewOnCartCheckedOut(service *order.OrderService) *OnCartCheckedOut {
	return &OnCartCheckedOut{service: service}
}

func (h *OnCartCheckedOut) Handle(ctx context.Context, event events.Event) error {
	var cartEvent events.CartEvent
	switch e := event.(type) {
	case events.CartEvent:
		cartEvent = e
	case *events.CartEvent:
		cartEvent = *e
	default:
		log.Printf("[ERROR] Order: Expected CartEvent, got %T", event)
		return nil
	}

	if cartEvent.EventType != events.CartCheckedOut {
		log.Printf("[DEBUG] Order: Ignoring non-CartCheckedOut event: %s", cartEvent.EventType)
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
	log.Printf("[INFO] Order: Creating order from cart %s", cartID)

	snapshot := cartEvent.EventPayload.CartSnapshot
	if snapshot == nil {
		log.Printf("[ERROR] Order: Cart snapshot is missing for cart %s", cartID)
		return fmt.Errorf("cart snapshot is required to create order")
	}

	order, err := h.service.CreateOrderFromSnapshot(ctx, cartID, snapshot)
	if err != nil {
		log.Printf("[ERROR] Order: Failed to create order from cart %s: %v", cartID, err)
		return err
	}

	log.Printf("[INFO] Order: Successfully created order %s from cart %s", order.OrderNumber, cartID)
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

var _ handler.EventHandler = (*OnCartCheckedOut)(nil)
var _ handler.HandlerFactory[events.CartEvent] = (*OnCartCheckedOut)(nil)
