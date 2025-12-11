package handler

import (
	"context"

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/logging"
)

// CustomerEventHandler provides reusable customer event handling logic
type CustomerEventHandler struct{}

// NewCustomerEventHandler creates a new customer event handler
func NewCustomerEventHandler() *CustomerEventHandler {
	return &CustomerEventHandler{}
}

// HandleCustomerCreated handles customer created events
func (h *CustomerEventHandler) HandleCustomerCreated(ctx context.Context, event events.CustomerEvent) error {
	logging.Info("Customer created: %s", event.EventPayload.CustomerID)

	// Add reusable customer created logic here
	// For example: send welcome email, initialize customer data, etc.

	return nil
}

// HandleCustomerUpdated handles customer updated events
func (h *CustomerEventHandler) HandleCustomerUpdated(ctx context.Context, event events.CustomerEvent) error {
	logging.Info("Customer updated: %s", event.EventPayload.CustomerID)

	// Add reusable customer updated logic here
	// For example: update cache, notify other services, etc.

	return nil
}

// HandleAddressAdded handles address added events
func (h *CustomerEventHandler) HandleAddressAdded(ctx context.Context, event events.CustomerEvent) error {
	logging.Info("Address added for customer: %s, address: %s", event.EventPayload.CustomerID, event.EventPayload.ResourceID)

	// Add reusable address added logic here

	return nil
}

// HandleAddressUpdated handles address updated events
func (h *CustomerEventHandler) HandleAddressUpdated(ctx context.Context, event events.CustomerEvent) error {
	logging.Info("Address updated for customer: %s, address: %s", event.EventPayload.CustomerID, event.EventPayload.ResourceID)

	// Add reusable address updated logic here

	return nil
}

// HandleAddressDeleted handles address deleted events
func (h *CustomerEventHandler) HandleAddressDeleted(ctx context.Context, event events.CustomerEvent) error {
	logging.Info("Address deleted for customer: %s, address: %s", event.EventPayload.CustomerID, event.EventPayload.ResourceID)

	// Add reusable address deleted logic here

	return nil
}

// HandleCardAdded handles card added events
func (h *CustomerEventHandler) HandleCardAdded(ctx context.Context, event events.CustomerEvent) error {
	logging.Info("Card added for customer: %s, card: %s", event.EventPayload.CustomerID, event.EventPayload.ResourceID)

	// Add reusable card added logic here

	return nil
}

// HandleCardUpdated handles card updated events
func (h *CustomerEventHandler) HandleCardUpdated(ctx context.Context, event events.CustomerEvent) error {
	logging.Info("Card updated for customer: %s, card: %s", event.EventPayload.CustomerID, event.EventPayload.ResourceID)

	// Add reusable card updated logic here

	return nil
}

// HandleCardDeleted handles card deleted events
func (h *CustomerEventHandler) HandleCardDeleted(ctx context.Context, event events.CustomerEvent) error {
	logging.Info("Card deleted for customer: %s, card: %s", event.EventPayload.CustomerID, event.EventPayload.ResourceID)

	// Add reusable card deleted logic here

	return nil
}

// HandleDefaultShippingAddressChanged handles default shipping address changed events
func (h *CustomerEventHandler) HandleDefaultShippingAddressChanged(ctx context.Context, event events.CustomerEvent) error {
	logging.Info("Default shipping address changed for customer: %s, address: %s", event.EventPayload.CustomerID, event.EventPayload.ResourceID)

	// Add reusable default shipping address changed logic here

	return nil
}

// HandleDefaultBillingAddressChanged handles default billing address changed events
func (h *CustomerEventHandler) HandleDefaultBillingAddressChanged(ctx context.Context, event events.CustomerEvent) error {
	logging.Info("Default billing address changed for customer: %s, address: %s", event.EventPayload.CustomerID, event.EventPayload.ResourceID)

	// Add reusable default billing address changed logic here

	return nil
}

// HandleDefaultCreditCardChanged handles default credit card changed events
func (h *CustomerEventHandler) HandleDefaultCreditCardChanged(ctx context.Context, event events.CustomerEvent) error {
	logging.Info("Default credit card changed for customer: %s, card: %s", event.EventPayload.CustomerID, event.EventPayload.ResourceID)

	// Add reusable default credit card changed logic here

	return nil
}

// HandleAllCustomerEvents provides a generic handler that delegates to specific handlers
func (h *CustomerEventHandler) HandleAllCustomerEvents(ctx context.Context, event events.CustomerEvent) error {
	switch event.EventType {
	case events.CustomerCreated:
		return h.HandleCustomerCreated(ctx, event)
	case events.CustomerUpdated:
		return h.HandleCustomerUpdated(ctx, event)
	case events.AddressAdded:
		return h.HandleAddressAdded(ctx, event)
	case events.AddressUpdated:
		return h.HandleAddressUpdated(ctx, event)
	case events.AddressDeleted:
		return h.HandleAddressDeleted(ctx, event)
	case events.CardAdded:
		return h.HandleCardAdded(ctx, event)
	case events.CardUpdated:
		return h.HandleCardUpdated(ctx, event)
	case events.CardDeleted:
		return h.HandleCardDeleted(ctx, event)
	case events.DefaultShippingAddressChanged:
		return h.HandleDefaultShippingAddressChanged(ctx, event)
	case events.DefaultBillingAddressChanged:
		return h.HandleDefaultBillingAddressChanged(ctx, event)
	case events.DefaultCreditCardChanged:
		return h.HandleDefaultCreditCardChanged(ctx, event)
	default:
		logging.Debug("Unknown customer event type: %s", event.EventType)
		return nil
	}
}
