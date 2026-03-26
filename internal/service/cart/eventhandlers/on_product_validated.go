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
		logger: logger.With("component", "cart_on_product_validated"),
	}
}

// Handle processes product validation events
func (h *OnProductValidated) Handle(ctx context.Context, event events.Event) error {
	log := h.logger.With("operation", "handle_product_event")

	log.Debug("Product event received",
		"event_type", event.Type(),
		"event_id", event.GetEntityID(),
		"topic", event.Topic(),
	)

	productEvent, ok := event.(events.ProductEvent)
	if !ok {
		log.Error("Expected ProductEvent", "actual_type", fmt.Sprintf("%T", event))
		return nil
	}

	// Handle both validated and unavailable events
	switch productEvent.EventType {
	case events.ProductValidated:
		log.Debug("Process product validated event", "product_id", productEvent.EventPayload.ProductID)
		return h.handleProductValidated(ctx, productEvent)
	case events.ProductUnavailable:
		log.Debug("Process product unavailable event", "product_id", productEvent.EventPayload.ProductID)
		return h.handleProductUnavailable(ctx, productEvent)
	default:
		log.Debug("Ignore product event type", "event_type", productEvent.EventType)
		return nil
	}
}

func (h *OnProductValidated) handleProductValidated(ctx context.Context, event events.ProductEvent) error {
	details := event.EventPayload.Details
	cartID := details["cart_id"]
	lineNumber := details["line_number"]
	productID := event.EventPayload.ProductID
	log := h.logger.With(
		"operation", "handle_product_validated",
		"event_type", event.EventType,
		"event_id", event.ID,
		"product_id", productID,
	)

	if cartID == "" || lineNumber == "" {
		log.Error("Missing product validation routing fields")
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(event.EventType), productID, cartID)

	// Get cart and item
	cartObj, err := h.repo.GetCartByID(ctx, cartID)
	if err != nil {
		log.Error("Get cart failed", "cart_id", cartID, "error", err.Error())
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
		log.Debug("Item not found in cart", "line_number", lineNumber, "cart_id", cartID)
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
		log.Error("Confirm cart item failed", "line_number", lineNumber, "error", err.Error())
		return err
	}

	// Update item in database
	if err := h.repo.UpdateItemStatus(ctx, targetItem); err != nil {
		log.Error("Update item status failed", "line_number", lineNumber, "error", err.Error())
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
		log.Error("Update cart totals failed", "cart_id", cartID, "error", err.Error())
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
			"line_number":  lineNumber,
			"product_id":   productID,
			"product_name": productName,
			"unit_price":   unitPrice,
			"quantity":     targetItem.Quantity,
			"total_price":  targetItem.TotalPrice,
			"status":       "validated",
		})
	}

	log.Info("Cart item validated", "line_number", lineNumber, "cart_id", cartID)
	return nil
}

func (h *OnProductValidated) handleProductUnavailable(ctx context.Context, event events.ProductEvent) error {
	details := event.EventPayload.Details
	cartID := details["cart_id"]
	lineNumber := details["line_number"]
	productID := event.EventPayload.ProductID
	reason := details["reason"]
	log := h.logger.With(
		"operation", "handle_product_unavailable",
		"event_type", event.EventType,
		"event_id", event.ID,
		"product_id", productID,
	)

	if cartID == "" || lineNumber == "" {
		log.Error("Missing product unavailable routing fields")
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(event.EventType), productID, cartID)

	// Get cart and item
	cartObj, err := h.repo.GetCartByID(ctx, cartID)
	if err != nil {
		log.Error("Get cart failed", "cart_id", cartID, "error", err.Error())
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
		log.Debug("Item not found in cart", "line_number", lineNumber, "cart_id", cartID)
		return nil
	}

	// Mark as backorder
	if err := targetItem.MarkAsBackorder(reason); err != nil {
		log.Error("Mark item as backorder failed", "line_number", lineNumber, "error", err.Error())
		return err
	}

	// Update item in database
	if err := h.repo.UpdateItemStatus(ctx, targetItem); err != nil {
		log.Error("Update item status failed", "line_number", lineNumber, "error", err.Error())
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
		log.Error("Update cart totals failed", "cart_id", cartID, "error", err.Error())
		return err
	}

	// Send SSE notification
	if h.sseHub != nil {
		h.sseHub.Publish(cartID, "cart.item.backorder", map[string]interface{}{
			"line_number":      lineNumber,
			"product_id":       productID,
			"status":           "backorder",
			"backorder_reason": reason,
		})
	}

	log.Info("Cart item backorder set", "line_number", lineNumber, "cart_id", cartID, "reason", reason)
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
