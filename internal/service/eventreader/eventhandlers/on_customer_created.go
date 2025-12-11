package eventhandlers

import (
	"context"
	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/platform/logging"
)

// OnCustomerCreated handles CustomerCreated events
type OnCustomerCreated struct{}

// NewOnCustomerCreated creates a new CustomerCreated event handler
func NewOnCustomerCreated() *OnCustomerCreated {
	return &OnCustomerCreated{}
}

// Handle processes CustomerCreated events
func (h *OnCustomerCreated) Handle(ctx context.Context, event events.Event) error {
	customerEvent, ok := event.(*events.CustomerEvent)
	if !ok {
		logging.Error("Eventreader: Expected *CustomerEvent, got %T", event)
		return nil // Don't fail processing, just log and continue
	}

	if customerEvent.EventType != events.CustomerCreated {
		logging.Debug("Eventreader: Ignoring non-CustomerCreated event: %s", customerEvent.EventType)
		return nil
	}

	logging.Info("Eventreader: Processing CustomerCreated event")
	logging.Info("Eventreader: CustomerID=%s, EventType=%s, ResourceID=%s",
		customerEvent.EventPayload.CustomerID,
		customerEvent.EventPayload.EventType,
		customerEvent.EventPayload.ResourceID)

	// Business logic for handling customer creation
	return h.processCustomerCreated(ctx, *customerEvent)
}

// processCustomerCreated contains the actual business logic
func (h *OnCustomerCreated) processCustomerCreated(_ context.Context, event events.CustomerEvent) error {
	// TODO: Add actual business logic here
	// Examples:
	// - Update read models
	// - Send notifications
	// - Trigger other workflows
	// - Update analytics

	logging.Info("Eventreader: Successfully processed CustomerCreated event for customer %s",
		event.EventPayload.CustomerID)

	return nil
}

// EventType returns the event type this handler processes
func (h *OnCustomerCreated) EventType() string {
	return string(events.CustomerCreated)
}

// CreateFactory returns the event factory for this handler
func (h *OnCustomerCreated) CreateFactory() events.EventFactory[events.CustomerEvent] {
	return events.CustomerEventFactory{}
}

// CreateHandler returns the handler function
func (h *OnCustomerCreated) CreateHandler() bus.HandlerFunc[events.CustomerEvent] {
	return func(ctx context.Context, event events.CustomerEvent) error {
		return h.Handle(ctx, event)
	}
}

// Ensure OnCustomerCreated implements the shared interfaces
var _ handler.EventHandler = (*OnCustomerCreated)(nil)
var _ handler.HandlerFactory = (*OnCustomerCreated)(nil)
