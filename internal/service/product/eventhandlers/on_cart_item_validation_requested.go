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

// OnCartItemValidationRequested handles cart item validation requests
// It validates the product exists and is in stock, then publishes the result
type OnCartItemValidationRequested struct {
	service *product.CatalogService
}

// NewOnCartItemValidationRequested creates a new validation handler
func NewOnCartItemValidationRequested(service *product.CatalogService) *OnCartItemValidationRequested {
	return &OnCartItemValidationRequested{
		service: service,
	}
}

// Handle processes the validation request event
func (h *OnCartItemValidationRequested) Handle(ctx context.Context, event events.Event) error {
	log.Printf("[DEBUG] SSE: Received event - Type: %T, processing...", event)

	validationEvent, ok := event.(events.CartValidationEvent)
	if !ok {
		log.Printf("[ERROR] Product: Expected CartValidationEvent, got %T", event)
		return nil
	}

	log.Printf("[DEBUG] SSE: Event received - Type: %s, ID: %s, Topic: %s", validationEvent.EventType, validationEvent.ID, validationEvent.Topic())

	if validationEvent.EventType != events.CartItemValidationRequested {
		log.Printf("[DEBUG] Product: Ignoring event type: %s", validationEvent.EventType)
		return nil
	}

	payload, ok := validationEvent.EventPayload.(events.CartValidationPayload)
	if !ok {
		log.Printf("[ERROR] Product: Invalid payload type for validation request, got %T", validationEvent.EventPayload)
		return nil
	}

	log.Printf("[DEBUG] Product: Validation request received - CorrelationID: %s, CartID: %s, ProductID: %s, Quantity: %d", payload.CorrelationID, payload.CartID, payload.ProductID, payload.Quantity)

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(validationEvent.EventType),
		payload.ProductID,
		payload.CartID)

	log.Printf("[DEBUG] Product: Validating product %s for cart %s", payload.ProductID, payload.CartID)

	// Parse product ID
	productID, err := strconv.ParseInt(payload.ProductID, 10, 64)
	if err != nil {
		log.Printf("[ERROR] Product: Invalid product ID format: %s", payload.ProductID)
		return h.publishValidationResult(ctx, payload.CorrelationID, false, false, "", 0, "invalid_product_id")
	}

	// Validate product exists and is in stock
	product, err := h.service.GetProductByID(ctx, productID)

	result := events.CartValidationResultPayload{
		CorrelationID: payload.CorrelationID,
	}

	if err != nil {
		result.IsValid = false
		result.Reason = "product_not_found"
		log.Printf("[DEBUG] Product: Product %s not found: %v", payload.ProductID, err)
	} else if !product.InStock {
		result.IsValid = false
		result.Reason = "out_of_stock"
		result.InStock = false
		result.ProductName = product.Name
		log.Printf("[DEBUG] Product: Product %s is out of stock", payload.ProductID)
	} else {
		result.IsValid = true
		result.InStock = true
		result.ProductName = product.Name
		result.UnitPrice = product.FinalPrice
		log.Printf("[DEBUG] Product: Product %s validated successfully", payload.ProductID)
	}

	return h.publishValidationResult(ctx, result.CorrelationID, result.IsValid, result.InStock,
		result.ProductName, result.UnitPrice, result.Reason)
}

// publishValidationResult writes the validation result to the outbox
func (h *OnCartItemValidationRequested) publishValidationResult(ctx context.Context, correlationID string,
	isValid, inStock bool, productName string, unitPrice float64, reason string) error {

	log.Printf("[DEBUG] Product: Publishing validation result - CorrelationID: %s, IsValid: %v, InStock: %v, ProductName: %s, UnitPrice: %.2f, Reason: %s", correlationID, isValid, inStock, productName, unitPrice, reason)

	infra := h.service.GetInfrastructure()

	// Write result to outbox within a transaction
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

	resultEvent := events.NewCartItemValidationCompletedEvent(correlationID, isValid, inStock, productName, unitPrice, reason)
	log.Printf("[DEBUG] Product: Validation result event created - EventID: %s, Type: %s", resultEvent.ID, resultEvent.EventType)

	if err := infra.OutboxWriter.WriteEvent(ctx, tx, resultEvent); err != nil {
		return fmt.Errorf("failed to write validation result to outbox: %w", err)
	}
	log.Printf("[DEBUG] Product: Validation result event written to outbox for CorrelationID: %s", correlationID)

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit validation result: %w", err)
	}
	committed = true
	log.Printf("[DEBUG] Product: Validation result committed for CorrelationID: %s", correlationID)

	return nil
}

// EventType returns the event type this handler processes
func (h *OnCartItemValidationRequested) EventType() string {
	return string(events.CartItemValidationRequested)
}

// CreateHandler returns a bus.HandlerFunc that wraps the Handle method
func (h *OnCartItemValidationRequested) CreateHandler() bus.HandlerFunc[events.CartValidationEvent] {
	return func(ctx context.Context, event events.CartValidationEvent) error {
		return h.Handle(ctx, event)
	}
}

// CreateFactory returns an EventFactory for CartValidationEvent
func (h *OnCartItemValidationRequested) CreateFactory() events.EventFactory[events.CartValidationEvent] {
	return events.CartValidationEventFactory{}
}

// Ensure OnCartItemValidationRequested implements the required interfaces
var _ handler.EventHandler = (*OnCartItemValidationRequested)(nil)
var _ handler.HandlerFactory[events.CartValidationEvent] = (*OnCartItemValidationRequested)(nil)
