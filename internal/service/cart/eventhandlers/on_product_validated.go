package eventhandlers

import (
	"context"
	"log"
	"strconv"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/platform/sse"
	"go-shopping-poc/internal/service/cart"
)

// OnProductValidated handles product validation events
// It updates cart items based on product validation results
type OnProductValidated struct {
	repo   cart.CartRepository
	sseHub *sse.Hub
}

// NewOnProductValidated creates a new product validation handler
func NewOnProductValidated(repo cart.CartRepository, sseHub *sse.Hub) *OnProductValidated {
	return &OnProductValidated{
		repo:   repo,
		sseHub: sseHub,
	}
}

// Handle processes product validation events
func (h *OnProductValidated) Handle(ctx context.Context, event events.Event) error {
	productEvent, ok := event.(events.ProductEvent)
	if !ok {
		log.Printf("[ERROR] Cart: Expected ProductEvent, got %T", event)
		return nil
	}

	// Handle both validated and unavailable events
	switch productEvent.EventType {
	case events.ProductValidated:
		return h.handleProductValidated(ctx, productEvent)
	case events.ProductUnavailable:
		return h.handleProductUnavailable(ctx, productEvent)
	default:
		log.Printf("[DEBUG] Cart: Ignoring product event type: %s", productEvent.EventType)
		return nil
	}
}

func (h *OnProductValidated) handleProductValidated(ctx context.Context, event events.ProductEvent) error {
	details := event.EventPayload.Details
	cartID := details["cart_id"]
	lineNumber := details["line_number"]
	productID := event.EventPayload.ProductID

	if cartID == "" || lineNumber == "" {
		log.Printf("[ERROR] Cart: Missing cart_id or line_number in ProductValidated event")
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(event.EventType), productID, cartID)

	// Get cart and item
	cartObj, err := h.repo.GetCartByID(ctx, cartID)
	if err != nil {
		log.Printf("[ERROR] Cart: Failed to get cart %s: %v", cartID, err)
		return err
	}

	// Find the item
	var targetItem *cart.CartItem
	for i := range cartObj.Items {
		if cartObj.Items[i].LineNumber == lineNumber {
			targetItem = &cartObj.Items[i]
			break
		}
	}

	if targetItem == nil {
		log.Printf("[DEBUG] Cart: Item %s not found in cart %s - may have been removed", lineNumber, cartID)
		return nil
	}

	// Parse unit price from details
	unitPrice := 0.0
	if priceStr := details["unit_price"]; priceStr != "" {
		if parsed, err := strconv.ParseFloat(priceStr, 64); err == nil {
			unitPrice = parsed
		}
	}

	// Confirm the item
	productName := event.EventPayload.Details["product_name"]
	if productName == "" {
		productName = targetItem.ProductName // Keep existing if not provided
	}

	if err := targetItem.ConfirmItem(productName, unitPrice); err != nil {
		log.Printf("[ERROR] Cart: Failed to confirm item %s: %v", lineNumber, err)
		return err
	}

	// Update item in database
	if err := h.repo.UpdateItemStatus(ctx, targetItem); err != nil {
		log.Printf("[ERROR] Cart: Failed to update item %s status: %v", lineNumber, err)
		return err
	}

	// Recalculate cart totals
	for i := range cartObj.Items {
		if cartObj.Items[i].LineNumber == targetItem.LineNumber {
			cartObj.Items[i] = *targetItem
			break
		}
	}
	cartObj.CalculateTotals()
	if err := h.repo.UpdateCart(ctx, cartObj); err != nil {
		log.Printf("[ERROR] Cart: Failed to update cart %s totals: %v", cartID, err)
		return err
	}

	// Emit CartItemConfirmed event for audit/completeness
	_ = events.NewCartItemConfirmedEvent(
		cartID, lineNumber, productID, productName, unitPrice, targetItem.Quantity,
	)
	// Note: This would require outbox writer access - may skip for now

	// Send SSE notification
	if h.sseHub != nil {
		h.sseHub.Publish(cartID, "cart.item.validated", map[string]interface{}{
			"lineNumber":  lineNumber,
			"productId":   productID,
			"productName": productName,
			"unitPrice":   unitPrice,
			"quantity":    targetItem.Quantity,
			"totalPrice":  targetItem.TotalPrice,
			"status":      "validated",
		})
	}

	log.Printf("[INFO] Cart: Item %s validated for cart %s", lineNumber, cartID)
	return nil
}

func (h *OnProductValidated) handleProductUnavailable(ctx context.Context, event events.ProductEvent) error {
	details := event.EventPayload.Details
	cartID := details["cart_id"]
	lineNumber := details["line_number"]
	productID := event.EventPayload.ProductID
	reason := details["reason"]

	if cartID == "" || lineNumber == "" {
		log.Printf("[ERROR] Cart: Missing cart_id or line_number in ProductUnavailable event")
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(event.EventType), productID, cartID)

	// Get cart and item
	cartObj, err := h.repo.GetCartByID(ctx, cartID)
	if err != nil {
		log.Printf("[ERROR] Cart: Failed to get cart %s: %v", cartID, err)
		return err
	}

	// Find the item
	var targetItem *cart.CartItem
	for i := range cartObj.Items {
		if cartObj.Items[i].LineNumber == lineNumber {
			targetItem = &cartObj.Items[i]
			break
		}
	}

	if targetItem == nil {
		log.Printf("[DEBUG] Cart: Item %s not found in cart %s - may have been removed", lineNumber, cartID)
		return nil
	}

	// Mark as backorder
	if err := targetItem.MarkAsBackorder(reason); err != nil {
		log.Printf("[ERROR] Cart: Failed to mark item %s as backorder: %v", lineNumber, err)
		return err
	}

	// Update item in database
	if err := h.repo.UpdateItemStatus(ctx, targetItem); err != nil {
		log.Printf("[ERROR] Cart: Failed to update item %s status: %v", lineNumber, err)
		return err
	}

	// Recalculate cart totals
	for i := range cartObj.Items {
		if cartObj.Items[i].LineNumber == targetItem.LineNumber {
			cartObj.Items[i] = *targetItem
			break
		}
	}
	cartObj.CalculateTotals()
	if err := h.repo.UpdateCart(ctx, cartObj); err != nil {
		log.Printf("[ERROR] Cart: Failed to update cart %s totals: %v", cartID, err)
		return err
	}

	// Send SSE notification
	if h.sseHub != nil {
		h.sseHub.Publish(cartID, "cart.item.backorder", map[string]interface{}{
			"lineNumber":      lineNumber,
			"productId":       productID,
			"status":          "backorder",
			"backorderReason": reason,
		})
	}

	log.Printf("[INFO] Cart: Item %s marked as backorder for cart %s: %s", lineNumber, cartID, reason)
	return nil
}

// CreateHandler returns a bus.HandlerFunc that wraps the Handle method
func (h *OnProductValidated) CreateHandler() bus.HandlerFunc[events.ProductEvent] {
	return func(ctx context.Context, event events.ProductEvent) error {
		return h.Handle(ctx, event)
	}
}

// CreateFactory returns an EventFactory for ProductEvent
func (h *OnProductValidated) CreateFactory() events.EventFactory[events.ProductEvent] {
	return events.ProductEventFactory{}
}

// Ensure OnProductValidated implements HandlerFactory
var _ handler.HandlerFactory[events.ProductEvent] = (*OnProductValidated)(nil)

// EventType returns the event type this handler processes
func (h *OnProductValidated) EventType() string {
	return string(events.ProductValidated) + "," + string(events.ProductUnavailable)
}
