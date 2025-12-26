package main

import (
	"context"
	"testing"
	"time"

	"go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/service/eventreader"
	"go-shopping-poc/internal/service/eventreader/eventhandlers"
)

func TestRegisterEventHandlers(t *testing.T) {
	// Set required environment variable for Kafka config
	t.Setenv("KAFKA_BROKERS", "localhost:9092")

	// Create mock service config
	serviceCfg := &eventreader.Config{
		WriteTopic: "test-write-topic",
		ReadTopics: []string{"test-topic"},
		Group:      "test-group",
	}

	// Create event bus provider
	eventBusConfig := event.EventBusConfig{
		WriteTopic: serviceCfg.WriteTopic,
		GroupID:    serviceCfg.Group,
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
	if err != nil {
		t.Fatalf("Failed to create event bus provider: %v", err)
	}
	eventBus := eventBusProvider.GetEventBus()

	// Create infrastructure
	infrastructure := eventreader.NewEventReaderInfrastructure(eventBus)

	// Create service
	service := eventreader.NewEventReaderService(infrastructure, serviceCfg)

	// Test registering handlers
	if err := registerEventHandlers(service); err != nil {
		t.Fatalf("Failed to register event handlers: %v", err)
	}

	// Verify that handlers were registered (we can't easily test the actual registration
	// without a more complex mock, but we can at least verify the function doesn't panic)
	if service == nil {
		t.Error("Expected service to be non-nil")
	}
}

func TestMainFunctionStructure(t *testing.T) {
	// Set required environment variable for Kafka config
	t.Setenv("KAFKA_BROKERS", "localhost:9092")

	// This test verifies that the main function structure is correct
	// We can't easily test the full main function without complex mocking,
	// but we can verify the components work correctly

	// Create mock service config instead of loading from env
	cfg := &eventreader.Config{
		WriteTopic: "test-write-topic",
		ReadTopics: []string{"CustomerEvents"},
		Group:      "test-group",
	}

	// Create event bus provider
	eventBusConfig := event.EventBusConfig{
		WriteTopic: cfg.WriteTopic,
		GroupID:    cfg.Group,
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
	if err != nil {
		t.Fatalf("Failed to create event bus provider: %v", err)
	}
	eventBus := eventBusProvider.GetEventBus()

	// Test infrastructure creation
	infrastructure := eventreader.NewEventReaderInfrastructure(eventBus)
	if infrastructure == nil {
		t.Fatalf("Expected infrastructure to be non-nil")
	}

	// Test service creation
	serviceCfg := &eventreader.Config{
		WriteTopic: "test-write-topic",
		ReadTopics: []string{"test-topic"},
		Group:      "test-group",
	}
	service := eventreader.NewEventReaderService(infrastructure, serviceCfg)
	if service == nil {
		t.Fatalf("Expected service to be non-nil")
	}

	// Test handler creation
	customerCreatedHandler := eventhandlers.NewOnCustomerCreated()
	if customerCreatedHandler == nil {
		t.Error("Expected customerCreatedHandler to be non-nil")
	}

	// Test handler registration
	if err := eventreader.RegisterHandler(
		service,
		customerCreatedHandler.CreateFactory(),
		customerCreatedHandler.CreateHandler(),
	); err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Test that we can create a context (this would be used in main)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if ctx == nil {
		t.Error("Expected context to be non-nil")
	}
}
