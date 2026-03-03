package eventhandlers

import (
	"context"
	"fmt"
	"log/slog"
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
	logger *slog.Logger
}

// NewOnProductValidated creates a new product validation handler
func NewOnProductValidated(repo cart.CartRepository, sseHub *sse.Hub, logger *slog.Logger) *OnProductValidated {
	if logger == nil {
		logger = slog.Default()
	}
	return &OnProductValidated{
		repo:   repo,
		sseHub: sseHub,
		logger: logger.With("handler", "on_product_validated"),
	}
}

// Handle processes product validation events
func (h *OnProductValidated) Handle(ctx context.Context, event events.Event) error {

	h.logger.Debug("Event received",
		"event_type", event.Type(),
		"event_id", event.GetEntityID(),
		"topic", event.Topic(),
	)

	productEvent, ok := event.(events.ProductEvent)
	if !ok {
		h.logger.Error("Expected ProductEvent", "actual_type", fmt.Sprintf("%T", event))
		return nil
	}

	// Handle both validated and unavailable events
	switch productEvent.EventType {
	case events.ProductValidated:
		h.logger.Debug("Processing ProductValidated event", "product_id", productEvent.EventPayload.ProductID)
		return h.handleProductValidated(ctx, productEvent)
	case events.ProductUnavailable:
		h.logger.Debug("Processing ProductUnavailable event", "product_id", productEvent.EventPayload.ProductID)
		return h.handleProductUnavailable(ctx, productEvent)
	default:
		h.logger.Debug("Ignoring product event type", "event_type", productEvent.EventType)
		return nil
	}
}

func (h *OnProductValidated) handleProductValidated(ctx context.Context, event events.ProductEvent) error {
	details := event.EventPayload.Details
	cartID := details["cart_id"]
	lineNumber := details["line_number"]
	productID := event.EventPayload.ProductID

	if cartID == "" || lineNumber == "" {
		h.logger.Error("Missing cart_id or line_number in ProductValidated event")
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(event.EventType), productID, cartID)

	// Get cart and item
	cartObj, err := h.repo.GetCartByID(ctx, cartID)
	if err != nil {
		h.logger.Error("Failed to get cart", "cart_id", cartID, "error", err.Error())
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
		h.logger.Debug("Item not found in cart - may have been removed", "line_number", lineNumber, "cart_id", cartID)
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
		h.logger.Error("Failed to confirm item", "line_number", lineNumber, "error", err.Error())
		return err
	}

	// Update item in database
	if err := h.repo.UpdateItemStatus(ctx, targetItem); err != nil {
		h.logger.Error("Failed to update item status", "line_number", lineNumber, "error", err.Error())
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
		h.logger.Error("Failed to update cart totals", "cart_id", cartID, "error", err.Error())
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

	h.logger.Info("Cart item validated", "line_number", lineNumber, "cart_id", cartID)
	return nil
}

func (h *OnProductValidated) handleProductUnavailable(ctx context.Context, event events.ProductEvent) error {
	details := event.EventPayload.Details
	cartID := details["cart_id"]
	lineNumber := details["line_number"]
	productID := event.EventPayload.ProductID
	reason := details["reason"]

	if cartID == "" || lineNumber == "" {
		h.logger.Error("Missing cart_id or line_number in ProductUnavailable event")
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(event.EventType), productID, cartID)

	// Get cart and item
	cartObj, err := h.repo.GetCartByID(ctx, cartID)
	if err != nil {
		h.logger.Error("Failed to get cart", "cart_id", cartID, "error", err.Error())
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
		h.logger.Debug("Item not found in cart - may have been removed", "line_number", lineNumber, "cart_id", cartID)
		return nil
	}

	// Mark as backorder
	if err := targetItem.MarkAsBackorder(reason); err != nil {
		h.logger.Error("Failed to mark item as backorder", "line_number", lineNumber, "error", err.Error())
		return err
	}

	// Update item in database
	if err := h.repo.UpdateItemStatus(ctx, targetItem); err != nil {
		h.logger.Error("Failed to update item status", "line_number", lineNumber, "error", err.Error())
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
		h.logger.Error("Failed to update cart totals", "cart_id", cartID, "error", err.Error())
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

	h.logger.Info("Item marked as backorder", "line_number", lineNumber, "cart_id", cartID, "reason", reason)
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
