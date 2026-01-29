package eventhandlers

import (
	"context"
	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"log"
)

// OnProductCreated handles ProductCreated events
type OnProductCreated struct{}

// NewOnProductCreated creates a new ProductCreated event handler
func NewOnProductCreated() *OnProductCreated {
	return &OnProductCreated{}
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
		log.Printf("[ERROR] Eventreader: Expected ProductEvent, got %T", event)
		return nil // Don't fail processing, just log and continue
	}

	if productEvent.EventType != events.ProductCreated {
		log.Printf("[DEBUG] Eventreader: Ignoring non-ProductCreated event: %s", productEvent.EventType)
		return nil
	}

	// Use platform utilities for consistent logging
	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(productEvent.EventType),
		productEvent.EventPayload.ProductID,
		productEvent.EventPayload.ResourceID)

	// Business logic for handling customer creation
	return h.processProductCreated(ctx, productEvent)
}

// processProductCreated contains the actual business logic
func (h *OnProductCreated) processProductCreated(ctx context.Context, event events.ProductEvent) error {
	productID := event.EventPayload.ProductID
	utils := handler.NewEventUtils()

	// Business logic for reacting to Product creation
	if err := h.celebrateNewProduct(ctx, productID); err != nil {
		utils.LogEventCompletion(ctx, string(event.EventType), productID, err)
		// Continue processing even if email fails
	}

	// Log successful completion
	utils.LogEventCompletion(ctx, string(event.EventType), productID, nil)
	return nil
}

// sendWelcomeEmail sends a welcome email to the new customer
func (h *OnProductCreated) celebrateNewProduct(ctx context.Context, productID string) error {
	// Check for context cancellation before proceeding
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	log.Printf("[INFO] Eventreader: HOORAY!  A new product is here: %s", productID)

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

// Ensure OnProductCreated implements the shared interfaces
var _ handler.EventHandler = (*OnProductCreated)(nil)
var _ handler.HandlerFactory[events.ProductEvent] = (*OnProductCreated)(nil)
