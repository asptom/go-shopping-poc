// Package customer provides customer domain services and validation.
// This package contains business logic specific to customer domain operations,
// including event validation, processing, and domain-specific utilities.
package customer

import (
	"context"
	"fmt"
	"log"

	"go-shopping-poc/internal/contracts/events"
)

// CustomerEventValidator provides validation for customer events
// This contains domain-specific validation rules for customer events
type CustomerEventValidator struct{}

// NewCustomerEventValidator creates a new customer event validator
func NewCustomerEventValidator() *CustomerEventValidator {
	return &CustomerEventValidator{}
}

// ValidateCustomerEvent validates customer-specific event fields and business rules
// This implements domain validation that belongs in the service layer, not platform
func (v *CustomerEventValidator) ValidateCustomerEvent(ctx context.Context, event events.CustomerEvent) error {
	// Validate required customer ID
	if event.EventPayload.CustomerID == "" {
		log.Printf("[ERROR] Customer event validation failed: missing CustomerID")
		return fmt.Errorf("customer ID is required")
	}

	// Validate event type is not empty
	if event.EventType == "" {
		log.Printf("[ERROR] Customer event validation failed: missing EventType")
		return fmt.Errorf("event type is required")
	}

	// Domain-specific validation: ensure customer ID format is valid
	if len(event.EventPayload.CustomerID) < 3 {
		log.Printf("[ERROR] Customer event validation failed: CustomerID too short: %s", event.EventPayload.CustomerID)
		return fmt.Errorf("customer ID must be at least 3 characters")
	}

	// Validate event type is a known customer event type
	validEventTypes := map[string]bool{
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

	if !validEventTypes[string(event.EventType)] {
		log.Printf("[ERROR] Customer event validation failed: unknown event type: %s", event.EventType)
		return fmt.Errorf("unknown customer event type: %s", event.EventType)
	}

	// Validate resource ID is present for resource-specific events
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

	if resourceEvents[string(event.EventType)] && event.EventPayload.ResourceID == "" {
		log.Printf("[ERROR] Customer event validation failed: missing ResourceID for event type: %s", event.EventType)
		return fmt.Errorf("resource ID is required for event type: %s", event.EventType)
	}

	log.Printf("[DEBUG] Customer event validation passed for customer %s, event type %s",
		event.EventPayload.CustomerID, event.EventType)
	return nil
}

// ValidateCustomerEventPayload validates the customer event payload structure
// This ensures the payload contains all required fields for processing
func (v *CustomerEventValidator) ValidateCustomerEventPayload(ctx context.Context, payload events.CustomerEventPayload) error {
	if payload.CustomerID == "" {
		return fmt.Errorf("customer ID cannot be empty")
	}

	if payload.EventType == "" {
		return fmt.Errorf("event type cannot be empty")
	}

	// Additional payload validation can be added here as business rules evolve

	return nil
}
