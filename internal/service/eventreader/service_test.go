package eventreader

import (
	"context"
	"errors"
	"testing"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/service"
)

// MockEventBus for testing
type MockEventBus struct {
	bus.Bus
	startConsumingCalled bool
}

func (m *MockEventBus) StartConsuming(ctx context.Context) error {
	m.startConsumingCalled = true
	return nil
}

func TestEventReaderService_RegisterHandler(t *testing.T) {
	mockBus := &MockEventBus{}
	eventService := NewEventReaderService(mockBus)

	factory := events.CustomerEventFactory{}
	handler := bus.HandlerFunc[events.CustomerEvent](func(ctx context.Context, evt events.CustomerEvent) error {
		return nil
	})

	err := RegisterHandler(eventService, factory, handler)

	// This should return an error since mockBus is not a kafka.EventBus
	if err == nil {
		t.Error("Expected error for non-kafka event bus")
	}

	// Check if the error contains ErrUnsupportedEventBus (it's wrapped in ServiceError)
	if !errors.Is(err, service.ErrUnsupportedEventBus) {
		t.Errorf("Expected ErrUnsupportedEventBus, got %v", err)
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
