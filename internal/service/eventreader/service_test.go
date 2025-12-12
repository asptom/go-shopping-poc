package eventreader

import (
	"context"
	"testing"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
)

// MockEventBus for testing
type MockEventBus struct {
	startConsumingCalled bool
}

func (m *MockEventBus) Publish(ctx context.Context, topic string, event events.Event) error {
	return nil
}

func (m *MockEventBus) PublishRaw(ctx context.Context, topic string, eventType string, data []byte) error {
	return nil
}

func (m *MockEventBus) StartConsuming(ctx context.Context) error {
	m.startConsumingCalled = true
	return nil
}

func (m *MockEventBus) RegisterHandler(factory any, handler any) error {
	return nil
}

func (m *MockEventBus) WriteTopic() string {
	return "test-write"
}

func (m *MockEventBus) ReadTopics() []string {
	return []string{"test-read"}
}

func TestEventReaderService_RegisterHandler(t *testing.T) {
	mockBus := &MockEventBus{}
	eventService := NewEventReaderService(mockBus)

	factory := events.CustomerEventFactory{}
	handler := bus.HandlerFunc[events.CustomerEvent](func(ctx context.Context, evt events.CustomerEvent) error {
		return nil
	})

	err := RegisterHandler(eventService, factory, handler)

	// RegisterHandler should succeed for any Bus implementation
	if err != nil {
		t.Errorf("Expected no error for valid bus implementation, got %v", err)
	}
}

func TestEventReaderService_Start(t *testing.T) {
	mockBus := &MockEventBus{}
	eventService := NewEventReaderService(mockBus)

	ctx := context.Background()
	err := eventService.Start(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !mockBus.startConsumingCalled {
		t.Error("Expected StartConsuming to be called")
	}
}

func TestEventReaderService_Stop(t *testing.T) {
	mockBus := &MockEventBus{}
	eventService := NewEventReaderService(mockBus)

	ctx := context.Background()
	err := eventService.Stop(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestNewEventReaderService(t *testing.T) {
	mockBus := &MockEventBus{}
	eventService := NewEventReaderService(mockBus)

	if eventService == nil {
		t.Error("Expected service to be non-nil")
	}

	// #nosec G601 - eventService is checked for nil above
	if eventService.Name() != "eventreader" {
		t.Errorf("Expected service name 'eventreader', got '%s'", eventService.Name())
	}

	if eventService.EventBus() != mockBus {
		t.Error("Expected eventBus to be set correctly")
	}

	if eventService.HandlerCount() != 0 {
		t.Errorf("Expected 0 handlers initially, got %d", eventService.HandlerCount())
	}
}

func TestEventReaderService_Health(t *testing.T) {
	mockBus := &MockEventBus{}
	eventService := NewEventReaderService(mockBus)

	err := eventService.Health()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
