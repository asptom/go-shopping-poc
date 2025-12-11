package eventhandlers

import (
	"context"
	"testing"

	events "go-shopping-poc/internal/contracts/events"
	handlerpkg "go-shopping-poc/internal/platform/event/handler"
)

func TestOnCustomerCreated_Handle(t *testing.T) {
	handler := NewOnCustomerCreated()

	// Create a CustomerCreated event
	event := events.NewCustomerCreatedEvent("customer-123", map[string]string{
		"source": "test",
	})

	ctx := context.Background()
	err := handler.Handle(ctx, event)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestOnCustomerCreated_HandleWrongEventType(t *testing.T) {
	handler := NewOnCustomerCreated()

	// Create a different event type
	event := events.NewCustomerUpdatedEvent("customer-123", map[string]string{
		"source": "test",
	})

	ctx := context.Background()
	err := handler.Handle(ctx, event)

	if err != nil {
		t.Errorf("Expected no error for wrong event type, got %v", err)
	}
}

func TestOnCustomerCreated_HandleWrongEventInterface(t *testing.T) {
	handler := NewOnCustomerCreated()

	// Create a mock event that doesn't implement CustomerEvent
	mockEvent := &struct {
		events.Event
	}{
		Event: events.CustomerEvent{}, // This will fail the type assertion
	}

	ctx := context.Background()
	err := handler.Handle(ctx, mockEvent)

	if err != nil {
		t.Errorf("Expected no error for wrong event interface, got %v", err)
	}
}

func TestOnCustomerCreated_EventType(t *testing.T) {
	handler := NewOnCustomerCreated()

	expectedType := string(events.CustomerCreated)
	actualType := handler.EventType()

	if actualType != expectedType {
		t.Errorf("Expected event type %s, got %s", expectedType, actualType)
	}
}

func TestOnCustomerCreated_FactoryAndHandler(t *testing.T) {
	handler := NewOnCustomerCreated()

	factory := handler.CreateFactory()
	if factory == nil {
		t.Error("Expected factory to be non-nil")
	}

	handlerFunc := handler.CreateHandler()
	if handlerFunc == nil {
		t.Error("Expected handler to be non-nil")
	}
}

func TestOnCustomerCreated_processCustomerCreated(t *testing.T) {
	handler := NewOnCustomerCreated()

	event := events.NewCustomerCreatedEvent("customer-123", map[string]string{
		"source": "test",
	})

	ctx := context.Background()
	err := handler.processCustomerCreated(ctx, *event)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestNewOnCustomerCreated(t *testing.T) {
	handler := NewOnCustomerCreated()

	if handler == nil {
		t.Error("Expected handler to be non-nil")
	}

	// Test that it implements the expected interfaces
	var _ handlerpkg.EventHandler = handler
	var _ handlerpkg.HandlerFactory = handler
}
