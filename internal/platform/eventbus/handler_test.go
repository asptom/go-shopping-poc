package eventbus

import (
	"context"
	"testing"

	events "go-shopping-poc/internal/event/customer"
	"go-shopping-poc/internal/platform/event"
)

func TestTypedHandler(t *testing.T) {
	// Create factory and handler
	factory := events.CustomerEventFactory{}

	receivedEvent := false
	handler := HandlerFunc[events.CustomerEvent](func(ctx context.Context, evt events.CustomerEvent) error {
		receivedEvent = true

		// Verify event structure
		if evt.Type() != "customer.created" {
			t.Errorf("Expected event type 'customer.created', got '%s'", evt.Type())
		}

		if evt.Topic() != "customer.changes" {
			t.Errorf("Expected topic 'customer.changes', got '%s'", evt.Topic())
		}

		if evt.EventPayload.CustomerID != "test-customer-123" {
			t.Errorf("Expected customer ID 'test-customer-123', got '%s'", evt.EventPayload.CustomerID)
		}

		return nil
	})

	typedHandler := NewTypedHandler(factory, handler)

	// Create test event JSON
	testEvent := events.NewCustomerCreatedEvent("test-customer-123", map[string]string{"test": "value"})
	eventJSON, err := testEvent.ToJSON()
	if err != nil {
		t.Fatalf("Failed to marshal test event: %v", err)
	}

	// Test the handler
	err = typedHandler.Handle(context.Background(), eventJSON)
	if err != nil {
		t.Errorf("Handler returned error: %v", err)
	}

	if !receivedEvent {
		t.Error("Handler was not called")
	}
}

func TestEventBusSubscribeTyped(t *testing.T) {
	// Create event bus
	bus := NewEventBus("localhost:9092", []string{"test-topic"}, "write-topic", "test-group")

	// Create factory and handler
	factory := events.CustomerEventFactory{}
	receivedEvent := false

	handler := HandlerFunc[events.CustomerEvent](func(ctx context.Context, evt events.CustomerEvent) error {
		receivedEvent = true
		return nil
	})

	// Subscribe using typed method
	SubscribeTyped(bus, factory, handler)

	// Verify handler was registered
	bus.mu.RLock()
	handlers := bus.typedHandlers["customer.changes"]
	bus.mu.RUnlock()

	if len(handlers) == 0 {
		t.Error("No handlers were registered for topic 'customer.changes'")
	}

	// Test the registered handler
	testEvent := events.NewCustomerCreatedEvent("test-customer-123", map[string]string{"test": "value"})
	eventJSON, err := testEvent.ToJSON()
	if err != nil {
		t.Fatalf("Failed to marshal test event: %v", err)
	}

	err = handlers[0](context.Background(), eventJSON)
	if err != nil {
		t.Errorf("Registered handler returned error: %v", err)
	}

	if !receivedEvent {
		t.Error("Registered handler was not called")
	}
}

func TestCustomerEventFactory(t *testing.T) {
	factory := events.CustomerEventFactory{}

	// Create test event JSON
	testEvent := events.NewCustomerCreatedEvent("test-customer-123", map[string]string{"test": "value"})
	eventJSON, err := testEvent.ToJSON()
	if err != nil {
		t.Fatalf("Failed to marshal test event: %v", err)
	}

	// Test factory reconstruction
	reconstructedEvent, err := factory.FromJSON(eventJSON)
	if err != nil {
		t.Fatalf("Factory failed to reconstruct event: %v", err)
	}

	// Verify reconstructed event
	if reconstructedEvent.Type() != "customer.created" {
		t.Errorf("Expected event type 'customer.created', got '%s'", reconstructedEvent.Type())
	}

	if reconstructedEvent.EventPayload.CustomerID != "test-customer-123" {
		t.Errorf("Expected customer ID 'test-customer-123', got '%s'", reconstructedEvent.EventPayload.CustomerID)
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that old Subscribe method still works
	bus := NewEventBus("localhost:9092", []string{"test-topic"}, "write-topic", "test-group")

	// Create a mock legacy handler
	handler := &mockLegacyHandler{}

	// Subscribe using legacy method
	bus.Subscribe("customer.created", handler)

	// Verify handler was registered
	bus.mu.RLock()
	handlers := bus.handlers["customer.created"]
	bus.mu.RUnlock()

	if len(handlers) == 0 {
		t.Error("No legacy handlers were registered for event type 'customer.created'")
	}
}

// Mock legacy handler for backward compatibility testing
type mockLegacyHandler struct{}

func (h *mockLegacyHandler) Handle(ctx context.Context, event event.Event) error {
	return nil
}
