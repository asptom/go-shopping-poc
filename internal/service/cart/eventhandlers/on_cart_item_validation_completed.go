package eventhandlers

import (
	"context"
	"log"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/platform/sse"
	"go-shopping-poc/internal/service/cart"
)

// OnCartItemValidationCompleted handles cart item validation results
// It updates the item status and notifies the frontend via SSE
type OnCartItemValidationCompleted struct {
	repo   cart.CartRepository
	sseHub *sse.Hub
}

// NewOnCartItemValidationCompleted creates a new validation completed handler
func NewOnCartItemValidationCompleted(repo cart.CartRepository, sseHub *sse.Hub) *OnCartItemValidationCompleted {
	return &OnCartItemValidationCompleted{
		repo:   repo,
		sseHub: sseHub,
	}
}

// Handle processes the validation result event
func (h *OnCartItemValidationCompleted) Handle(ctx context.Context, event events.Event) error {
	log.Printf("[DEBUG] Cart: Received event - Type: %T, processing...", event)

	validationEvent, ok := event.(events.CartValidationEvent)
	if !ok {
		log.Printf("[ERROR] Cart: Expected CartValidationEvent, got %T", event)
		return nil
	}

	log.Printf("[DEBUG] Cart: Event received - Type: %s, ID: %s, Topic: %s", validationEvent.EventType, validationEvent.ID, validationEvent.Topic())

	if validationEvent.EventType != events.CartItemValidationCompleted {
		log.Printf("[DEBUG] Cart: Ignoring event type: %s", validationEvent.EventType)
		return nil
	}

	payload, ok := validationEvent.EventPayload.(events.CartValidationResultPayload)
	if !ok {
		log.Printf("[ERROR] Cart: Invalid payload type for validation result, got %T", validationEvent.EventPayload)
		return nil
	}

	log.Printf("[DEBUG] Cart: Validation result received - CorrelationID: %s, IsValid: %v, InStock: %v, ProductName: %s", payload.CorrelationID, payload.IsValid, payload.InStock, payload.ProductName)

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(validationEvent.EventType),
		payload.CorrelationID,
		"")

	// Find cart item by validation ID (correlation ID)
	item, err := h.repo.GetItemByValidationID(ctx, payload.CorrelationID)
	if err != nil {
		// Item not found - may have been removed by user before validation completed
		log.Printf("[DEBUG] Cart: Item not found for validation ID %s - may have been removed by user", payload.CorrelationID)
		return nil
	}

	// Get cart for totals recalculation
	cartObj, err := h.repo.GetCartByID(ctx, item.CartID.String())
	if err != nil {
		log.Printf("[ERROR] Cart: Failed to get cart %s for validation update: %v", item.CartID, err)
		return err
	}

	// Update item based on validation result
	if payload.IsValid && payload.InStock {
		// Confirm item with product details
		if err := item.ConfirmItem(payload.ProductName, payload.UnitPrice); err != nil {
			log.Printf("[ERROR] Cart: Failed to confirm item %s: %v", item.LineNumber, err)
			return err
		}
		log.Printf("[INFO] Cart: Item %s confirmed for cart %s", item.LineNumber, item.CartID)
	} else {
		// Mark as backorder
		reason := payload.Reason
		if reason == "" {
			reason = "validation_failed"
		}
		if err := item.MarkAsBackorder(reason); err != nil {
			log.Printf("[ERROR] Cart: Failed to mark item %s as backorder: %v", item.LineNumber, err)
			return err
		}
		log.Printf("[INFO] Cart: Item %s marked as backorder for cart %s: %s", item.LineNumber, item.CartID, reason)
	}

	// Update item in database
	if err := h.repo.UpdateItemStatus(ctx, item); err != nil {
		log.Printf("[ERROR] Cart: Failed to update item %s status in database: %v", item.LineNumber, err)
		return err
	}

	// Recalculate cart totals
	for i := range cartObj.Items {
		if cartObj.Items[i].LineNumber == item.LineNumber {
			cartObj.Items[i] = *item
			break
		}
	}
	cartObj.CalculateTotals()

	if err := h.repo.UpdateCart(ctx, cartObj); err != nil {
		log.Printf("[ERROR] Cart: Failed to update cart %s totals after validation: %v", cartObj.CartID, err)
		return err
	}

	// Push SSE notification to frontend
	if h.sseHub != nil {
		eventType := "cart.item.validated"
		if item.IsBackorder() {
			eventType = "cart.item.backorder"
		}

		sseData := map[string]interface{}{
			"lineNumber":      item.LineNumber,
			"productId":       item.ProductID,
			"status":          item.Status,
			"productName":     item.ProductName,
			"unitPrice":       item.UnitPrice,
			"quantity":        item.Quantity,
			"totalPrice":      item.TotalPrice,
			"backorderReason": item.BackorderReason,
		}
		log.Printf("[DEBUG] SSE: Publishing event '%s' for cart %s - lineNumber: %s, productId: %s, status: %s", eventType, item.CartID, item.LineNumber, item.ProductID, item.Status)

		h.sseHub.Publish(
			item.CartID.String(),
			eventType,
			sseData,
		)
		log.Printf("[DEBUG] SSE: Successfully published event '%s' for cart %s", eventType, item.CartID)
	} else {
		log.Printf("[WARN] SSE: sseHub is nil, cannot publish event for cart %s", item.CartID)
	}

	return nil
}

// EventType returns the event type this handler processes
func (h *OnCartItemValidationCompleted) EventType() string {
	return string(events.CartItemValidationCompleted)
}

// CreateHandler returns a bus.HandlerFunc that wraps the Handle method
func (h *OnCartItemValidationCompleted) CreateHandler() bus.HandlerFunc[events.CartValidationEvent] {
	return func(ctx context.Context, event events.CartValidationEvent) error {
		return h.Handle(ctx, event)
	}
}

// CreateFactory returns an EventFactory for CartValidationEvent
func (h *OnCartItemValidationCompleted) CreateFactory() events.EventFactory[events.CartValidationEvent] {
	return events.CartValidationEventFactory{}
}

// Ensure OnCartItemValidationCompleted implements the required interfaces
var _ handler.EventHandler = (*OnCartItemValidationCompleted)(nil)
var _ handler.HandlerFactory[events.CartValidationEvent] = (*OnCartItemValidationCompleted)(nil)
