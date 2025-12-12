package handler

import (
	"context"
	"testing"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
)

// MockEventHandler implements EventHandler for testing
type MockEventHandler struct {
	eventType string
}

func NewMockEventHandler(eventType string) *MockEventHandler {
	return &MockEventHandler{eventType: eventType}
}

func (h *MockEventHandler) Handle(ctx context.Context, event events.Event) error {
	// Mock implementation
	return nil
}

func (h *MockEventHandler) EventType() string {
	return h.eventType
}

func (h *MockEventHandler) CreateFactory() events.EventFactory[events.CustomerEvent] {
	return events.CustomerEventFactory{}
}

func (h *MockEventHandler) CreateHandler() bus.HandlerFunc[events.CustomerEvent] {
	return func(ctx context.Context, event events.CustomerEvent) error {
		return h.Handle(ctx, event)
	}
}

func TestEventHandler_Interface(t *testing.T) {
	handler := NewMockEventHandler("test-event")

	// Test that it implements EventHandler
	var _ EventHandler = handler

	if handler.EventType() != "test-event" {
		t.Errorf("Expected event type 'test-event', got '%s'", handler.EventType())
	}
}

func TestHandlerFactory_Interface(t *testing.T) {
	handler := NewMockEventHandler("test-event")

	// Test that it implements HandlerFactory[events.CustomerEvent]
	var _ HandlerFactory[events.CustomerEvent] = handler

	factory := handler.CreateFactory()
	if factory == nil {
		t.Error("Expected factory to be non-nil")
	}

	handlerFunc := handler.CreateHandler()
	if handlerFunc == nil {
		t.Error("Expected handler function to be non-nil")
	}
}

func TestSharedHandlerInterfaces(t *testing.T) {
	// Test that the shared interfaces can be used by different implementations
	handler := NewMockEventHandler(string(events.CustomerCreated))

	// Verify interface compliance
	var eventHandler EventHandler = handler
	var factoryHandler HandlerFactory[events.CustomerEvent] = handler

	// Test methods
	ctx := context.Background()
	event := events.NewCustomerCreatedEvent("customer-123", map[string]string{"source": "test"})

	err := eventHandler.Handle(ctx, event)
	if err != nil {
		t.Errorf("Expected no error from Handle, got %v", err)
	}

	expectedType := string(events.CustomerCreated)
	if eventHandler.EventType() != expectedType {
		t.Errorf("Expected event type %s, got %s", expectedType, eventHandler.EventType())
	}

	factory := factoryHandler.CreateFactory()
	if factory == nil {
		t.Error("Expected factory to be non-nil")
	}

	handlerFunc := factoryHandler.CreateHandler()
	if handlerFunc == nil {
		t.Error("Expected handler function to be non-nil")
	}
}
