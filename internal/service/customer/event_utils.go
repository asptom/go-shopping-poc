// Package customer provides customer domain services and utilities.
// This package contains business logic specific to customer domain operations,
// including event processing utilities and domain-specific helpers.
package customer

import (
	"go-shopping-poc/internal/contracts/events"
)

// CustomerEventUtils provides customer-specific event processing utilities
// This contains domain-specific utilities that belong in the service layer
type CustomerEventUtils struct{}

// NewCustomerEventUtils creates a new customer event utilities instance
func NewCustomerEventUtils() *CustomerEventUtils {
	return &CustomerEventUtils{}
}

// GetCustomerID extracts the customer ID from a customer event
// This provides domain-specific ID extraction for customer events
func (u *CustomerEventUtils) GetCustomerID(event events.CustomerEvent) string {
	return event.EventPayload.CustomerID
}

// GetEventResourceID extracts the resource ID from a customer event
// This provides domain-specific resource ID extraction for customer events
func (u *CustomerEventUtils) GetEventResourceID(event events.CustomerEvent) string {
	return event.EventPayload.ResourceID
}

// GetCustomerEventType returns the event type for a customer event
// This provides domain-specific event type access
func (u *CustomerEventUtils) GetCustomerEventType(event events.CustomerEvent) string {
	return string(event.EventType)
}

// IsCustomerEventType checks if the event type is a valid customer event type
// This provides domain validation for customer event types
func (u *CustomerEventUtils) IsCustomerEventType(eventType string) bool {
	validTypes := map[string]bool{
		string(events.CustomerCreated):               true,
		string(events.CustomerUpdated):               true,
		string(events.AddressAdded):                  true,
		string(events.AddressUpdated):                true,
		string(events.AddressDeleted):                true,
		string(events.CardAdded):                     true,
		string(events.CardUpdated):                   true,
		string(events.CardDeleted):                   true,
		string(events.DefaultShippingAddressChanged): true,
		string(events.DefaultBillingAddressChanged):  true,
		string(events.DefaultCreditCardChanged):      true,
	}
	return validTypes[eventType]
}

// RequiresResourceID checks if a customer event type requires a resource ID
// This provides domain knowledge about which events need resource identifiers
func (u *CustomerEventUtils) RequiresResourceID(eventType string) bool {
	resourceEvents := map[string]bool{
		string(events.AddressAdded):                  true,
		string(events.AddressUpdated):                true,
		string(events.AddressDeleted):                true,
		string(events.CardAdded):                     true,
		string(events.CardUpdated):                   true,
		string(events.CardDeleted):                   true,
		string(events.DefaultShippingAddressChanged): true,
		string(events.DefaultBillingAddressChanged):  true,
		string(events.DefaultCreditCardChanged):      true,
	}
	return resourceEvents[eventType]
}

// GetEventDescription returns a human-readable description for a customer event
// This provides domain-specific event descriptions for logging and monitoring
func (u *CustomerEventUtils) GetEventDescription(event events.CustomerEvent) string {
	switch event.EventType {
	case events.CustomerCreated:
		return "Customer account created"
	case events.CustomerUpdated:
		return "Customer account updated"
	case events.AddressAdded:
		return "Address added to customer account"
	case events.AddressUpdated:
		return "Customer address updated"
	case events.AddressDeleted:
		return "Address removed from customer account"
	case events.CardAdded:
		return "Payment card added to customer account"
	case events.CardUpdated:
		return "Customer payment card updated"
	case events.CardDeleted:
		return "Payment card removed from customer account"
	case events.DefaultShippingAddressChanged:
		return "Customer default shipping address changed"
	case events.DefaultBillingAddressChanged:
		return "Customer default billing address changed"
	case events.DefaultCreditCardChanged:
		return "Customer default payment card changed"
	default:
		return "Unknown customer event"
	}
}

// GetEntityType returns the entity type for customer events
// This provides domain knowledge about the primary entity type
func (u *CustomerEventUtils) GetEntityType() string {
	return "customer"
}

// GetResourceType determines the resource type based on the event type
// This provides domain knowledge about resource types for different events
func (u *CustomerEventUtils) GetResourceType(eventType events.EventType) string {
	switch eventType {
	case events.AddressAdded, events.AddressUpdated, events.AddressDeleted,
		events.DefaultShippingAddressChanged, events.DefaultBillingAddressChanged:
		return "address"
	case events.CardAdded, events.CardUpdated, events.CardDeleted, events.DefaultCreditCardChanged:
		return "payment_card"
	default:
		return "unknown"
	}
}
