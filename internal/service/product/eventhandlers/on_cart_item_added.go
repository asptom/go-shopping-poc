package eventhandlers

import (
	"context"
	"fmt"
	"log"
	"strconv"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/service/product"
)

// OnCartItemAdded handles cart item addition events
// It validates the product and emits product validation events
type OnCartItemAdded struct {
	service *product.CatalogService
}

// NewOnCartItemAdded creates a new cart item added handler
func NewOnCartItemAdded(service *product.CatalogService) *OnCartItemAdded {
	return &OnCartItemAdded{
		service: service,
	}
}

// Handle processes cart item addition events
func (h *OnCartItemAdded) Handle(ctx context.Context, event events.Event) error {
	cartItemEvent, ok := event.(events.CartItemEvent)
	if !ok {
		log.Printf("[ERROR] Product: Expected CartItemEvent, got %T", event)
		return nil
	}

	// Only handle CartItemAdded events
	if cartItemEvent.EventType != events.CartItemAdded {
		log.Printf("[DEBUG] Product: Ignoring event type: %s", cartItemEvent.EventType)
		return nil
	}

	payload := cartItemEvent.Data

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(cartItemEvent.EventType),
		payload.ProductID,
		payload.CartID)

	log.Printf("[DEBUG] Product: Processing cart item added - CartID: %s, ProductID: %s, Quantity: %d",
		payload.CartID, payload.ProductID, payload.Quantity)

	// Parse product ID
	productID, err := strconv.ParseInt(payload.ProductID, 10, 64)
	if err != nil {
		log.Printf("[ERROR] Product: Invalid product ID format: %s", payload.ProductID)
		return h.publishValidationResult(ctx, payload, false, 0, "invalid_product_id")
	}

	// Get product details
	product, err := h.service.GetProductByID(ctx, productID)
	if err != nil {
		log.Printf("[DEBUG] Product: Product %s not found: %v", payload.ProductID, err)
		return h.publishValidationResult(ctx, payload, false, 0, "product_not_found")
	}

	if !product.InStock {
		log.Printf("[DEBUG] Product: Product %s is out of stock", payload.ProductID)
		return h.publishValidationResult(ctx, payload, false, 0, "out_of_stock")
	}

	log.Printf("[DEBUG] Product: Product %s validated successfully", payload.ProductID)
	return h.publishValidationResult(ctx, payload, true, product.FinalPrice, "")
}

func (h *OnCartItemAdded) publishValidationResult(ctx context.Context, payload events.CartItemPayload, isAvailable bool, unitPrice float64, reason string) error {
	infra := h.service.GetInfrastructure()

	tx, err := infra.Database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var event *events.ProductEvent
	if isAvailable {
		// Get product name from service (we need to fetch it again or pass it)
		productID, _ := strconv.ParseInt(payload.ProductID, 10, 64)
		product, _ := h.service.GetProductByID(ctx, productID)
		productName := ""
		if product != nil {
			productName = product.Name
		}
		event = events.NewProductValidatedEvent(payload.ProductID, productName, unitPrice, payload.CartID, payload.LineNumber)
	} else {
		event = events.NewProductUnavailableEvent(payload.ProductID, reason, payload.CartID, payload.LineNumber)
	}

	if err := infra.OutboxWriter.WriteEvent(ctx, tx, event); err != nil {
		return fmt.Errorf("failed to write validation event to outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit validation event: %w", err)
	}
	committed = true

	// Trigger immediate outbox processing for low latency
	if infra.OutboxPublisher != nil {
		go func() {
			if err := infra.OutboxPublisher.ProcessNow(); err != nil {
				log.Printf("[WARN] Product: Failed to trigger immediate outbox processing: %v", err)
			}
		}()
	}

	log.Printf("[DEBUG] Product: Published validation result for product %s (available: %v)",
		payload.ProductID, isAvailable)
	return nil
}

// CreateHandler returns a bus.HandlerFunc that wraps the Handle method
func (h *OnCartItemAdded) CreateHandler() bus.HandlerFunc[events.CartItemEvent] {
	return func(ctx context.Context, event events.CartItemEvent) error {
		return h.Handle(ctx, event)
	}
}

// CreateFactory returns an EventFactory for CartItemEvent
func (h *OnCartItemAdded) CreateFactory() events.EventFactory[events.CartItemEvent] {
	return events.CartItemEventFactory{}
}

// Ensure OnCartItemAdded implements HandlerFactory
var _ handler.HandlerFactory[events.CartItemEvent] = (*OnCartItemAdded)(nil)

// EventType returns the event type this handler processes
func (h *OnCartItemAdded) EventType() string {
	return string(events.CartItemAdded)
}
