package customer_test

import (
	"context"
	"testing"

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/service/customer"
)

func TestValidateCustomerEventSuccess(t *testing.T) {
	t.Parallel()

	validator := customer.NewCustomerEventValidator()

	evt := events.NewCustomerCreatedEvent("customer-123", map[string]string{
		"username": "testuser",
	})

	err := validator.ValidateCustomerEvent(context.Background(), *evt)
	if err != nil {
		t.Errorf("valid event should pass validation: %v", err)
	}
}

func TestValidateCustomerEventMissingCustomerID(t *testing.T) {
	t.Parallel()

	validator := customer.NewCustomerEventValidator()

	evt := events.NewCustomerCreatedEvent("", nil)

	err := validator.ValidateCustomerEvent(context.Background(), *evt)
	if err == nil {
		t.Error("expected error for missing customer ID")
	}
}

func TestValidateCustomerEventInvalidEventType(t *testing.T) {
	t.Parallel()

	validator := customer.NewCustomerEventValidator()

	payload := events.CustomerEventPayload{
		CustomerID: "customer-123",
		EventType:  "invalid.event.type",
	}

	evt := &events.CustomerEvent{
		EventType:    "invalid.event.type",
		EventPayload: payload,
	}

	err := validator.ValidateCustomerEvent(context.Background(), *evt)
	if err == nil {
		t.Error("expected error for invalid event type")
	}
}

func TestValidateCustomerEventResourceIDRequired(t *testing.T) {
	t.Parallel()

	validator := customer.NewCustomerEventValidator()

	evt := events.NewAddressAddedEvent("customer-123", "", nil)

	err := validator.ValidateCustomerEvent(context.Background(), *evt)
	if err == nil {
		t.Error("expected error for missing resource ID on address event")
	}
}

func TestValidateCustomerEventPayloadSuccess(t *testing.T) {
	t.Parallel()

	validator := customer.NewCustomerEventValidator()

	payload := events.CustomerEventPayload{
		CustomerID: "customer-123",
		EventType:  events.CustomerCreated,
	}

	err := validator.ValidateCustomerEventPayload(context.Background(), payload)
	if err != nil {
		t.Errorf("valid payload should pass validation: %v", err)
	}
}

func TestValidateCustomerEventPayloadMissingCustomerID(t *testing.T) {
	t.Parallel()

	validator := customer.NewCustomerEventValidator()

	payload := events.CustomerEventPayload{
		CustomerID: "",
		EventType:  events.CustomerCreated,
	}

	err := validator.ValidateCustomerEventPayload(context.Background(), payload)
	if err == nil {
		t.Error("expected error for missing customer ID")
	}
}
