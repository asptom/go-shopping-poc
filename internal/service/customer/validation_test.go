package customer

import (
	"context"
	"strings"
	"testing"

	"go-shopping-poc/internal/contracts/events"
)

func TestCustomerEventValidator_ValidateCustomerEvent(t *testing.T) {
	validator := NewCustomerEventValidator()

	// Test valid customer event
	validEvent := events.CustomerEvent{
		EventType: events.CustomerCreated,
		EventPayload: events.CustomerEventPayload{
			CustomerID: "test-customer-123",
			EventType:  events.CustomerCreated,
		},
	}

	err := validator.ValidateCustomerEvent(context.Background(), validEvent)
	if err != nil {
		t.Errorf("Expected valid event to pass validation, got error: %v", err)
	}

	// Test event with empty customer ID
	invalidEvent := events.CustomerEvent{
		EventType: events.CustomerCreated,
		EventPayload: events.CustomerEventPayload{
			CustomerID: "",
			EventType:  events.CustomerCreated,
		},
	}

	err = validator.ValidateCustomerEvent(context.Background(), invalidEvent)
	if err == nil {
		t.Error("Expected error for empty customer ID")
	}
	if !strings.Contains(err.Error(), "customer ID is required") {
		t.Errorf("Expected customer ID validation error, got: %v", err)
	}

	// Test event with short customer ID
	invalidEvent = events.CustomerEvent{
		EventType: events.CustomerCreated,
		EventPayload: events.CustomerEventPayload{
			CustomerID: "ab", // Too short
			EventType:  events.CustomerCreated,
		},
	}

	err = validator.ValidateCustomerEvent(context.Background(), invalidEvent)
	if err == nil {
		t.Error("Expected error for short customer ID")
	}
	if !strings.Contains(err.Error(), "customer ID must be at least 3 characters") {
		t.Errorf("Expected customer ID length validation error, got: %v", err)
	}

	// Test event with empty event type
	invalidEvent = events.CustomerEvent{
		EventType: "",
		EventPayload: events.CustomerEventPayload{
			CustomerID: "test-customer-123",
			EventType:  "",
		},
	}

	err = validator.ValidateCustomerEvent(context.Background(), invalidEvent)
	if err == nil {
		t.Error("Expected error for empty event type")
	}
	if !strings.Contains(err.Error(), "event type is required") {
		t.Errorf("Expected event type validation error, got: %v", err)
	}

	// Test event with unknown event type
	invalidEvent = events.CustomerEvent{
		EventType: "unknown_event_type",
		EventPayload: events.CustomerEventPayload{
			CustomerID: "test-customer-123",
			EventType:  "unknown_event_type",
		},
	}

	err = validator.ValidateCustomerEvent(context.Background(), invalidEvent)
	if err == nil {
		t.Error("Expected error for unknown event type")
	}
	if !strings.Contains(err.Error(), "unknown customer event type") {
		t.Errorf("Expected unknown event type validation error, got: %v", err)
	}
}

func TestCustomerEventValidator_ValidateCustomerEvent_ResourceID(t *testing.T) {
	validator := NewCustomerEventValidator()

	// Test resource event without resource ID
	resourceEvent := events.CustomerEvent{
		EventType: events.AddressAdded,
		EventPayload: events.CustomerEventPayload{
			CustomerID: "test-customer-123",
			ResourceID: "", // Missing resource ID
			EventType:  events.AddressAdded,
		},
	}

	err := validator.ValidateCustomerEvent(context.Background(), resourceEvent)
	if err == nil {
		t.Error("Expected error for resource event without resource ID")
	}
	if !strings.Contains(err.Error(), "resource ID is required for event type") {
		t.Errorf("Expected resource ID validation error, got: %v", err)
	}

	// Test resource event with resource ID (should pass)
	resourceEvent.EventPayload.ResourceID = "test-resource-456"

	err = validator.ValidateCustomerEvent(context.Background(), resourceEvent)
	if err != nil {
		t.Errorf("Expected resource event with resource ID to pass validation, got error: %v", err)
	}

	// Test non-resource event without resource ID (should pass)
	nonResourceEvent := events.CustomerEvent{
		EventType: events.CustomerCreated,
		EventPayload: events.CustomerEventPayload{
			CustomerID: "test-customer-123",
			ResourceID: "", // No resource ID needed
			EventType:  events.CustomerCreated,
		},
	}

	err = validator.ValidateCustomerEvent(context.Background(), nonResourceEvent)
	if err != nil {
		t.Errorf("Expected non-resource event without resource ID to pass validation, got error: %v", err)
	}
}

func TestCustomerEventValidator_ValidateCustomerEventPayload(t *testing.T) {
	validator := NewCustomerEventValidator()

	// Test valid payload
	validPayload := events.CustomerEventPayload{
		CustomerID: "test-customer-123",
		EventType:  events.CustomerCreated,
	}

	err := validator.ValidateCustomerEventPayload(context.Background(), validPayload)
	if err != nil {
		t.Errorf("Expected valid payload to pass validation, got error: %v", err)
	}

	// Test payload with empty customer ID
	invalidPayload := events.CustomerEventPayload{
		CustomerID: "",
		EventType:  events.CustomerCreated,
	}

	err = validator.ValidateCustomerEventPayload(context.Background(), invalidPayload)
	if err == nil {
		t.Error("Expected error for empty customer ID in payload")
	}
	if !strings.Contains(err.Error(), "customer ID cannot be empty") {
		t.Errorf("Expected customer ID validation error, got: %v", err)
	}

	// Test payload with empty event type
	invalidPayload = events.CustomerEventPayload{
		CustomerID: "test-customer-123",
		EventType:  "",
	}

	err = validator.ValidateCustomerEventPayload(context.Background(), invalidPayload)
	if err == nil {
		t.Error("Expected error for empty event type in payload")
	}
	if !strings.Contains(err.Error(), "event type cannot be empty") {
		t.Errorf("Expected event type validation error, got: %v", err)
	}
}
