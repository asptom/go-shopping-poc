package customer

import (
	"testing"

	"go-shopping-poc/internal/contracts/events"
)

func TestCustomerEventUtils_GetCustomerID(t *testing.T) {
	utils := NewCustomerEventUtils()

	event := events.CustomerEvent{
		EventPayload: events.CustomerEventPayload{
			CustomerID: "test-customer-123",
		},
	}

	customerID := utils.GetCustomerID(event)
	if customerID != "test-customer-123" {
		t.Errorf("Expected customer ID 'test-customer-123', got '%s'", customerID)
	}
}

func TestCustomerEventUtils_GetEventResourceID(t *testing.T) {
	utils := NewCustomerEventUtils()

	event := events.CustomerEvent{
		EventPayload: events.CustomerEventPayload{
			ResourceID: "test-resource-456",
		},
	}

	resourceID := utils.GetEventResourceID(event)
	if resourceID != "test-resource-456" {
		t.Errorf("Expected resource ID 'test-resource-456', got '%s'", resourceID)
	}
}

func TestCustomerEventUtils_GetCustomerEventType(t *testing.T) {
	utils := NewCustomerEventUtils()

	event := events.CustomerEvent{
		EventType: events.CustomerCreated,
	}

	eventType := utils.GetCustomerEventType(event)
	if eventType != string(events.CustomerCreated) {
		t.Errorf("Expected event type '%s', got '%s'", events.CustomerCreated, eventType)
	}
}

func TestCustomerEventUtils_IsCustomerEventType(t *testing.T) {
	utils := NewCustomerEventUtils()

	// Test valid event types
	validTypes := []string{
		string(events.CustomerCreated),
		string(events.CustomerUpdated),
		string(events.AddressAdded),
		string(events.AddressUpdated),
		string(events.AddressDeleted),
		string(events.CardAdded),
		string(events.CardUpdated),
		string(events.CardDeleted),
		string(events.DefaultShippingAddressChanged),
		string(events.DefaultBillingAddressChanged),
		string(events.DefaultCreditCardChanged),
	}

	for _, eventType := range validTypes {
		if !utils.IsCustomerEventType(eventType) {
			t.Errorf("Expected event type '%s' to be valid", eventType)
		}
	}

	// Test invalid event types
	invalidTypes := []string{
		"unknown_event",
		"invalid_type",
		"",
	}

	for _, eventType := range invalidTypes {
		if utils.IsCustomerEventType(eventType) {
			t.Errorf("Expected event type '%s' to be invalid", eventType)
		}
	}
}

func TestCustomerEventUtils_RequiresResourceID(t *testing.T) {
	utils := NewCustomerEventUtils()

	// Test event types that require resource ID
	resourceEvents := []string{
		string(events.AddressAdded),
		string(events.AddressUpdated),
		string(events.AddressDeleted),
		string(events.CardAdded),
		string(events.CardUpdated),
		string(events.CardDeleted),
		string(events.DefaultShippingAddressChanged),
		string(events.DefaultBillingAddressChanged),
		string(events.DefaultCreditCardChanged),
	}

	for _, eventType := range resourceEvents {
		if !utils.RequiresResourceID(eventType) {
			t.Errorf("Expected event type '%s' to require resource ID", eventType)
		}
	}

	// Test event types that don't require resource ID
	nonResourceEvents := []string{
		string(events.CustomerCreated),
		string(events.CustomerUpdated),
		"unknown_event",
	}

	for _, eventType := range nonResourceEvents {
		if utils.RequiresResourceID(eventType) {
			t.Errorf("Expected event type '%s' to not require resource ID", eventType)
		}
	}
}

func TestCustomerEventUtils_GetEventDescription(t *testing.T) {
	utils := NewCustomerEventUtils()

	testCases := []struct {
		eventType events.EventType
		expected  string
	}{
		{events.CustomerCreated, "Customer account created"},
		{events.CustomerUpdated, "Customer account updated"},
		{events.AddressAdded, "Address added to customer account"},
		{events.AddressUpdated, "Customer address updated"},
		{events.AddressDeleted, "Address removed from customer account"},
		{events.CardAdded, "Payment card added to customer account"},
		{events.CardUpdated, "Customer payment card updated"},
		{events.CardDeleted, "Payment card removed from customer account"},
		{events.DefaultShippingAddressChanged, "Customer default shipping address changed"},
		{events.DefaultBillingAddressChanged, "Customer default billing address changed"},
		{events.DefaultCreditCardChanged, "Customer default payment card changed"},
	}

	for _, tc := range testCases {
		event := events.CustomerEvent{
			EventType: tc.eventType,
		}
		description := utils.GetEventDescription(event)
		if description != tc.expected {
			t.Errorf("Expected description '%s' for event type %s, got '%s'",
				tc.expected, tc.eventType, description)
		}
	}

	// Test unknown event type
	event := events.CustomerEvent{
		EventType: "unknown_event",
	}
	description := utils.GetEventDescription(event)
	if description != "Unknown customer event" {
		t.Errorf("Expected 'Unknown customer event' for unknown type, got '%s'", description)
	}
}

func TestCustomerEventUtils_GetEntityType(t *testing.T) {
	utils := NewCustomerEventUtils()

	entityType := utils.GetEntityType()
	if entityType != "customer" {
		t.Errorf("Expected entity type 'customer', got '%s'", entityType)
	}
}

func TestCustomerEventUtils_GetResourceType(t *testing.T) {
	utils := NewCustomerEventUtils()

	testCases := []struct {
		eventType events.EventType
		expected  string
	}{
		{events.AddressAdded, "address"},
		{events.AddressUpdated, "address"},
		{events.AddressDeleted, "address"},
		{events.DefaultShippingAddressChanged, "address"},
		{events.DefaultBillingAddressChanged, "address"},
		{events.CardAdded, "payment_card"},
		{events.CardUpdated, "payment_card"},
		{events.CardDeleted, "payment_card"},
		{events.DefaultCreditCardChanged, "payment_card"},
		{events.CustomerCreated, "unknown"},
		{events.CustomerUpdated, "unknown"},
	}

	for _, tc := range testCases {
		resourceType := utils.GetResourceType(tc.eventType)
		if resourceType != tc.expected {
			t.Errorf("Expected resource type '%s' for event type %s, got '%s'",
				tc.expected, tc.eventType, resourceType)
		}
	}
}
