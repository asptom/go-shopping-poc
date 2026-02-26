package eventhandlers

import (
	"context"
	"fmt"
	"log/slog"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
)

// OnProductCreated handles ProductCreated events
type OnProductCreated struct {
	logger *slog.Logger
}

// NewOnProductCreated creates a new ProductCreated event handler
func NewOnProductCreated(logger *slog.Logger) *OnProductCreated {
	if logger == nil {
		logger = slog.Default()
	}
	return &OnProductCreated{
		logger: logger.With("handler", "on_product_created"),
	}
}

// Handle processes ProductCreated events
func (h *OnProductCreated) Handle(ctx context.Context, event events.Event) error {
	var productEvent events.ProductEvent
	switch e := event.(type) {
	case events.ProductEvent:
		productEvent = e
	case *events.ProductEvent:
		productEvent = *e
	default:
		h.logger.Error("Expected ProductEvent", "actual_type", fmt.Sprintf("%T", event))
		return nil
	}

	if productEvent.EventType != events.ProductCreated {
		h.logger.Debug("Ignoring non-ProductCreated event", "event_type", productEvent.EventType)
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(productEvent.EventType),
		productEvent.EventPayload.ProductID,
		productEvent.EventPayload.ResourceID)

	return h.processProductCreated(ctx, productEvent)
}

// processProductCreated contains the actual business logic
func (h *OnProductCreated) processProductCreated(ctx context.Context, event events.ProductEvent) error {
	productID := event.EventPayload.ProductID
	utils := handler.NewEventUtils()

	if err := h.celebrateNewProduct(ctx, productID); err != nil {
		utils.LogEventCompletion(ctx, string(event.EventType), productID, err)
	}

	utils.LogEventCompletion(ctx, string(event.EventType), productID, nil)
	return nil
}

// celebrateNewProduct celebrates a new product
func (h *OnProductCreated) celebrateNewProduct(ctx context.Context, productID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	h.logger.Info("A new product is here", "product_id", productID)
	return nil
}

// EventType returns the event type this handler processes
func (h *OnProductCreated) EventType() string {
	return string(events.ProductCreated)
}

// CreateFactory returns the event factory for this handler
func (h *OnProductCreated) CreateFactory() events.EventFactory[events.ProductEvent] {
	return events.ProductEventFactory{}
}

// CreateHandler returns the handler function
func (h *OnProductCreated) CreateHandler() bus.HandlerFunc[events.ProductEvent] {
	return func(ctx context.Context, event events.ProductEvent) error {
		return h.Handle(ctx, event)
	}
}

var _ handler.EventHandler = (*OnProductCreated)(nil)
var _ handler.HandlerFactory[events.ProductEvent] = (*OnProductCreated)(nil)
