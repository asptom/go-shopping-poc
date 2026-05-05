package eventhandlers

import (
	"context"
	"log/slog"
	"strconv"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/service/cart"
)

// OnProductEvent keeps the cart's product cache current by processing
// ProductCreated, ProductUpdated, ProductDeleted, ProductValidated, and
// ProductUnavailable events from the ProductEvents topic.
//
// This handler is the sole writer to the ProductCache. All cache updates
// come from events, ensuring consistency with the product service's
// database (the source of truth).
type OnProductEvent struct {
	cache  *cart.ProductCache
	logger *slog.Logger
}

// NewOnProductEvent creates a new product event handler.
func NewOnProductEvent(cache *cart.ProductCache, logger *slog.Logger) *OnProductEvent {
	if logger == nil {
		logger = slog.Default()
	}
	return &OnProductEvent{
		cache:  cache,
		logger: logger.With("component", "cart_on_product_event"),
	}
}

// Handle processes product lifecycle events.
//
// It dispatches to handler methods based on event type:
//
//	- ProductCreated: Insert new entry into cache
//	- ProductUpdated: Upsert existing entry in cache
//	- ProductDeleted: Remove entry from cache
//	- ProductValidated: Upsert cache entry with validation data (stock, price, name)
//	- ProductUnavailable: Upsert cache entry marking product as out of stock
//
// All other event types are silently ignored.
func (h *OnProductEvent) Handle(ctx context.Context, event events.Event) error {
	productEvent, ok := event.(events.ProductEvent)
	if !ok {
		h.logger.Error("Expected ProductEvent", "actual_type", event.Type())
		return nil
	}

	switch productEvent.EventType {
	case events.ProductCreated:
		return h.handleProductCreated(ctx, productEvent)
	case events.ProductUpdated:
		return h.handleProductUpdated(ctx, productEvent)
	case events.ProductDeleted:
		return h.handleProductDeleted(ctx, productEvent)
	case events.ProductValidated:
		return h.handleProductValidated(ctx, productEvent)
	case events.ProductUnavailable:
		return h.handleProductUnavailable(ctx, productEvent)
	default:
		return nil
	}
}

func (h *OnProductEvent) handleProductCreated(_ context.Context, event events.ProductEvent) error {
	h.upsertProduct(event)
	h.logger.Debug("Product cache updated",
		"product_id", event.EventPayload.ProductID,
		"event_type", string(event.EventType),
	)
	return nil
}

func (h *OnProductEvent) handleProductUpdated(_ context.Context, event events.ProductEvent) error {
	h.upsertProduct(event)
	h.logger.Debug("Product cache updated",
		"product_id", event.EventPayload.ProductID,
		"event_type", string(event.EventType),
	)
	return nil
}

func (h *OnProductEvent) handleProductDeleted(_ context.Context, event events.ProductEvent) error {
	h.cache.Delete(event.EventPayload.ProductID)
	h.logger.Debug("Product cache entry removed",
		"product_id", event.EventPayload.ProductID,
		"event_type", string(event.EventType),
	)
	return nil
}

func (h *OnProductEvent) handleProductValidated(_ context.Context, event events.ProductEvent) error {
	details := event.EventPayload.Details
	if details == nil {
		details = make(map[string]string)
	}

	productID := event.EventPayload.ProductID
	if productID == "" {
		productID = event.EventPayload.ResourceID
	}

	inStock := true

	finalPrice := 0.0
	if priceStr := details["unit_price"]; priceStr != "" {
		if parsed, err := strconv.ParseFloat(priceStr, 64); err == nil {
			finalPrice = parsed
		}
	}

	name := details["product_name"]

	h.cache.Set(productID, cart.ProductEntry{
		ProductID:  productID,
		InStock:    inStock,
		FinalPrice: finalPrice,
		Name:       name,
	})

	h.logger.Debug("Product cache updated from validation",
		"product_id", productID,
		"event_type", string(event.EventType),
		"in_stock", inStock,
		"final_price", finalPrice,
		"name", name,
	)
	return nil
}

func (h *OnProductEvent) handleProductUnavailable(_ context.Context, event events.ProductEvent) error {
	details := event.EventPayload.Details
	if details == nil {
		details = make(map[string]string)
	}

	productID := event.EventPayload.ProductID
	if productID == "" {
		productID = event.EventPayload.ResourceID
	}

	inStock := false

	finalPrice := 0.0
	if priceStr := details["unit_price"]; priceStr != "" {
		if parsed, err := strconv.ParseFloat(priceStr, 64); err == nil {
			finalPrice = parsed
		}
	}

	name := details["product_name"]

	h.cache.Set(productID, cart.ProductEntry{
		ProductID:  productID,
		InStock:    inStock,
		FinalPrice: finalPrice,
		Name:       name,
	})

	h.logger.Debug("Product cache updated from unavailable event",
		"product_id", productID,
		"event_type", string(event.EventType),
		"in_stock", inStock,
	)
	return nil
}

// upsertProduct extracts product data from a ProductCreated or ProductUpdated
// event and stores it in the cache. It reads numeric fields from the event's
// Details map (which contains string values) and parses them into the
// appropriate types.
func (h *OnProductEvent) upsertProduct(event events.ProductEvent) {
	details := event.EventPayload.Details
	if details == nil {
		details = make(map[string]string)
	}

	productID := event.EventPayload.ProductID
	if productID == "" {
		productID = event.EventPayload.ResourceID
	}

	inStock := true // default: assume in stock
	if stockStr := details["in_stock"]; stockStr != "" {
		if parsed, err := strconv.ParseBool(stockStr); err == nil {
			inStock = parsed
		}
	}

	finalPrice := 0.0
	if priceStr := details["final_price"]; priceStr != "" {
		if parsed, err := strconv.ParseFloat(priceStr, 64); err == nil {
			finalPrice = parsed
		}
	}

	name := details["name"]

	h.cache.Set(productID, cart.ProductEntry{
		ProductID:  productID,
		InStock:    inStock,
		FinalPrice: finalPrice,
		Name:       name,
	})
}

// CreateHandler returns a bus.HandlerFunc that wraps the Handle method.
func (h *OnProductEvent) CreateHandler() bus.HandlerFunc[events.ProductEvent] {
	return func(ctx context.Context, event events.ProductEvent) error {
		return h.Handle(ctx, event)
	}
}

// CreateFactory returns an EventFactory for ProductEvent.
func (h *OnProductEvent) CreateFactory() events.EventFactory[events.ProductEvent] {
	return events.ProductEventFactory{}
}

// Ensure OnProductEvent implements handler.EventHandler.
var _ handler.EventHandler = (*OnProductEvent)(nil)

// EventType returns the event types this handler processes.
func (h *OnProductEvent) EventType() string {
	return string(events.ProductCreated) + "," +
		string(events.ProductUpdated) + "," +
		string(events.ProductDeleted) + "," +
		string(events.ProductValidated) + "," +
		string(events.ProductUnavailable)
}
