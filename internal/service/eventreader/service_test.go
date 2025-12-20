package eventreader

import (
	"context"
	"testing"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	kafka "go-shopping-poc/internal/platform/event/bus/kafka"
	kafkaconfig "go-shopping-poc/internal/platform/event/kafka"
)

// mockConfig returns a test config for service tests
func mockConfig() *Config {
	return &Config{
		WriteTopic: "test-write",
		ReadTopics: []string{"test-read"},
		Group:      "test-group",
	}
}

// createTestKafkaEventBus creates a concrete kafka.EventBus for testing
// This allows RegisterHandler tests to work with the concrete type requirements
func createTestKafkaEventBus() *kafka.EventBus {
	// Create test Kafka config with localhost brokers (no real connection needed for registration)
	kafkaCfg := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-events",
		GroupID: "test-group",
	}

	// Create concrete EventBus instance
	return kafka.NewEventBus(kafkaCfg)
}

func TestEventReaderService_RegisterHandler(t *testing.T) {
	// Use concrete kafka.EventBus for testing RegisterHandler functionality
	concreteBus := createTestKafkaEventBus()
	eventService := NewEventReaderService(concreteBus, mockConfig())

	factory := events.CustomerEventFactory{}
	handler := bus.HandlerFunc[events.CustomerEvent](func(ctx context.Context, evt events.CustomerEvent) error {
		return nil
	})

	err := RegisterHandler(eventService, factory, handler)

	// RegisterHandler should succeed for concrete kafka.EventBus implementation
	if err != nil {
		t.Errorf("Expected no error for concrete kafka.EventBus implementation, got %v", err)
	}

	// Verify handler was registered
	if eventService.HandlerCount() != 1 {
		t.Errorf("Expected 1 handler registered, got %d", eventService.HandlerCount())
	}
}

func TestEventReaderService_Start(t *testing.T) {
	concreteBus := createTestKafkaEventBus()
	eventService := NewEventReaderService(concreteBus, mockConfig())

	// Use a cancelled context to prevent actual Kafka connection
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := eventService.Start(ctx)

	// We expect context cancellation error
	if err == nil || err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestEventReaderService_Stop(t *testing.T) {
	concreteBus := createTestKafkaEventBus()
	eventService := NewEventReaderService(concreteBus, mockConfig())

	ctx := context.Background()
	err := eventService.Stop(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestNewEventReaderService(t *testing.T) {
	concreteBus := createTestKafkaEventBus()
	eventService := NewEventReaderService(concreteBus, mockConfig())

	if eventService == nil {
		t.Error("Expected service to be non-nil")
	}

	// #nosec G601 - eventService is checked for nil above
	if eventService.Name() != "eventreader" {
		t.Errorf("Expected service name 'eventreader', got '%s'", eventService.Name())
	}

	if eventService.EventBus() != concreteBus {
		t.Error("Expected eventBus to be set correctly")
	}

	if eventService.HandlerCount() != 0 {
		t.Errorf("Expected 0 handlers initially, got %d", eventService.HandlerCount())
	}
}

func TestEventReaderService_Health(t *testing.T) {
	concreteBus := createTestKafkaEventBus()
	eventService := NewEventReaderService(concreteBus, mockConfig())

	err := eventService.Health()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
