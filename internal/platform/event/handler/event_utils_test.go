package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-shopping-poc/internal/contracts/events"
)

// mockEvent implements events.Event for testing
type mockEvent struct {
	eventType  string
	topic      string
	payload    any
	entityID   string
	resourceID string
}

func (m *mockEvent) Type() string {
	return m.eventType
}

func (m *mockEvent) Topic() string {
	return m.topic
}

func (m *mockEvent) Payload() any {
	return m.payload
}

func (m *mockEvent) ToJSON() ([]byte, error) {
	return []byte("{}"), nil
}

func (m *mockEvent) GetEntityID() string {
	return m.entityID
}

func (m *mockEvent) GetResourceID() string {
	return m.resourceID
}

func TestEventUtils_ValidateEvent(t *testing.T) {
	utils := NewEventUtils()
	ctx := context.Background()

	t.Run("Valid Event", func(t *testing.T) {
		event := events.NewCustomerCreatedEvent("customer-123", nil)
		err := utils.ValidateEvent(ctx, event)
		if err != nil {
			t.Errorf("Expected no error for valid event, got %v", err)
		}
	})

	t.Run("Nil Event", func(t *testing.T) {
		err := utils.ValidateEvent(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil event")
		}
		if err.Error() != "event cannot be nil" {
			t.Errorf("Expected 'event cannot be nil', got %v", err)
		}
	})

	t.Run("Event Missing Type", func(t *testing.T) {
		// Create a mock event with empty type
		event := &mockEvent{
			eventType: "",
			topic:     "test-topic",
		}
		err := utils.ValidateEvent(ctx, event)
		if err == nil {
			t.Error("Expected error for missing event type")
		}
		if err.Error() != "event type is required" {
			t.Errorf("Expected 'event type is required', got %v", err)
		}
	})

	t.Run("Event Missing Topic", func(t *testing.T) {
		// Create a mock event with empty topic
		event := &mockEvent{
			eventType: "test-event",
			topic:     "",
		}
		err := utils.ValidateEvent(ctx, event)
		if err == nil {
			t.Error("Expected error for missing topic")
		}
		if err.Error() != "event topic is required" {
			t.Errorf("Expected 'event topic is required', got %v", err)
		}
	})

	t.Run("Nil Event", func(t *testing.T) {
		err := utils.ValidateEvent(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil event")
		}
		if err.Error() != "event cannot be nil" {
			t.Errorf("Expected 'event cannot be nil', got %v", err)
		}
	})

	t.Run("Customer Event Missing EventType", func(t *testing.T) {
		event := &events.CustomerEvent{
			ID:        "test-id",
			EventType: "", // Missing EventType
			Timestamp: time.Now(),
			EventPayload: events.CustomerEventPayload{
				CustomerID: "customer-123",
				EventType:  "", // Missing EventType in payload
			},
		}
		err := utils.ValidateEvent(ctx, event)
		if err == nil {
			t.Error("Expected error for missing event type")
		}
		if err.Error() != "event type is required" {
			t.Errorf("Expected 'event type is required', got %v", err)
		}
	})
}

func TestEventUtils_LogEventProcessing(t *testing.T) {
	utils := NewEventUtils()
	ctx := context.Background()

	// These tests just verify the methods don't panic
	// In real scenarios, they would output logs
	t.Run("With Resource ID", func(t *testing.T) {
		utils.LogEventProcessing(ctx, "customer.created", "customer-123", "address-456")
	})

	t.Run("Without Resource ID", func(t *testing.T) {
		utils.LogEventProcessing(ctx, "customer.created", "customer-123", "")
	})
}

func TestEventUtils_LogEventCompletion(t *testing.T) {
	utils := NewEventUtils()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		utils.LogEventCompletion(ctx, "customer.created", "customer-123", nil)
	})

	t.Run("Error", func(t *testing.T) {
		err := errors.New("test error")
		utils.LogEventCompletion(ctx, "customer.created", "customer-123", err)
	})
}

func TestEventUtils_HandleEventWithValidation(t *testing.T) {
	utils := NewEventUtils()
	ctx := context.Background()

	t.Run("Valid Event and Successful Processing", func(t *testing.T) {
		event := events.NewCustomerCreatedEvent("customer-123", nil)
		processor := func(ctx context.Context, event events.Event) error {
			return nil
		}

		err := utils.HandleEventWithValidation(ctx, event, processor)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Invalid Event", func(t *testing.T) {
		// Create a mock event with missing topic (generic validation)
		event := &mockEvent{
			eventType: "test-event",
			topic:     "", // Invalid - missing topic
		}
		processor := func(ctx context.Context, event events.Event) error {
			return nil
		}

		err := utils.HandleEventWithValidation(ctx, event, processor)
		if err == nil {
			t.Error("Expected validation error")
		}
	})

	t.Run("Processing Error", func(t *testing.T) {
		event := events.NewCustomerCreatedEvent("customer-123", nil)
		processor := func(ctx context.Context, event events.Event) error {
			return errors.New("processing error")
		}

		err := utils.HandleEventWithValidation(ctx, event, processor)
		if err == nil {
			t.Error("Expected processing error")
		}
		expectedError := "event processing failed: processing error"
		if err.Error() != expectedError {
			t.Errorf("Expected '%s', got %v", expectedError, err)
		}
	})
}

func TestEventUtils_SafeEventProcessing(t *testing.T) {
	utils := NewEventUtils()
	ctx := context.Background()

	t.Run("Successful Processing", func(t *testing.T) {
		event := events.NewCustomerCreatedEvent("customer-123", nil)
		processor := func(ctx context.Context, event events.Event) error {
			return nil
		}

		err := utils.SafeEventProcessing(ctx, event, processor)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Processing Error", func(t *testing.T) {
		event := events.NewCustomerCreatedEvent("customer-123", nil)
		processor := func(ctx context.Context, event events.Event) error {
			return errors.New("processing error")
		}

		err := utils.SafeEventProcessing(ctx, event, processor)
		if err == nil {
			t.Error("Expected processing error")
		}
	})

	t.Run("Panic Recovery", func(t *testing.T) {
		event := events.NewCustomerCreatedEvent("customer-123", nil)
		processor := func(ctx context.Context, event events.Event) error {
			panic("test panic")
		}

		err := utils.SafeEventProcessing(ctx, event, processor)
		if err == nil {
			t.Error("Expected panic recovery error")
		}
		if err.Error() != "panic during event processing: test panic" {
			t.Errorf("Expected panic error, got %v", err)
		}
	})
}

func TestEventUtils_GetEventType(t *testing.T) {
	utils := NewEventUtils()

	t.Run("Customer Event", func(t *testing.T) {
		event := events.NewCustomerCreatedEvent("customer-123", nil)
		eventType := utils.GetEventType(event)
		if eventType != "customer.created" {
			t.Errorf("Expected 'customer.created', got %s", eventType)
		}
	})

	t.Run("Mock Event", func(t *testing.T) {
		event := &mockEvent{
			eventType: "test-event",
			topic:     "test-topic",
		}
		eventType := utils.GetEventType(event)
		if eventType != "test-event" {
			t.Errorf("Expected 'test-event', got %s", eventType)
		}
	})

	t.Run("Nil Event", func(t *testing.T) {
		eventType := utils.GetEventType(nil)
		if eventType != "unknown" {
			t.Errorf("Expected 'unknown', got %s", eventType)
		}
	})
}

func TestEventUtils_GetEventTopic(t *testing.T) {
	utils := NewEventUtils()

	t.Run("Customer Event", func(t *testing.T) {
		event := events.NewCustomerCreatedEvent("customer-123", nil)
		topic := utils.GetEventTopic(event)
		if topic != "CustomerEvents" {
			t.Errorf("Expected 'CustomerEvents', got %s", topic)
		}
	})

	t.Run("Mock Event", func(t *testing.T) {
		event := &mockEvent{
			eventType: "test-event",
			topic:     "test-topic",
		}
		topic := utils.GetEventTopic(event)
		if topic != "test-topic" {
			t.Errorf("Expected 'test-topic', got %s", topic)
		}
	})

	t.Run("Nil Event", func(t *testing.T) {
		topic := utils.GetEventTopic(nil)
		if topic != "unknown" {
			t.Errorf("Expected 'unknown', got %s", topic)
		}
	})
}

func TestEventTypeMatcher_MatchEventType(t *testing.T) {
	matcher := NewEventTypeMatcher()

	t.Run("Matching Event Type", func(t *testing.T) {
		event := events.NewCustomerCreatedEvent("customer-123", nil)
		isMatch := matcher.MatchEventType(event, "customer.created", "customer.updated")
		if !isMatch {
			t.Error("Expected match for customer.created")
		}
	})

	t.Run("Non-Matching Event Type", func(t *testing.T) {
		event := events.NewCustomerCreatedEvent("customer-123", nil)
		isMatch := matcher.MatchEventType(event, "order.created", "order.updated")
		if isMatch {
			t.Error("Expected no match for order events")
		}
	})

	t.Run("Nil Event", func(t *testing.T) {
		isMatch := matcher.MatchEventType(nil, "customer.created")
		if isMatch {
			t.Error("Expected no match for nil event")
		}
	})
}

func TestEventTypeMatcher_IsEventType(t *testing.T) {
	matcher := NewEventTypeMatcher()

	t.Run("Matching Event Type", func(t *testing.T) {
		event := events.NewCustomerCreatedEvent("customer-123", nil)
		isMatch := matcher.IsEventType(event, "customer.created")
		if !isMatch {
			t.Error("Expected true for matching event type")
		}
	})

	t.Run("Non-Matching Event Type", func(t *testing.T) {
		event := events.NewCustomerCreatedEvent("customer-123", nil)
		isMatch := matcher.IsEventType(event, "customer.updated")
		if isMatch {
			t.Error("Expected false for non-matching event type")
		}
	})

	t.Run("Nil Event", func(t *testing.T) {
		isMatch := matcher.IsEventType(nil, "any-type")
		if isMatch {
			t.Error("Expected false for nil event")
		}
	})
}

func TestNewEventUtils(t *testing.T) {
	utils := NewEventUtils()
	if utils == nil {
		t.Error("Expected non-nil EventUtils")
	}
}

func TestNewEventTypeMatcher(t *testing.T) {
	matcher := NewEventTypeMatcher()
	if matcher == nil {
		t.Error("Expected non-nil EventTypeMatcher")
	}
}
