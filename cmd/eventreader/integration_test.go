//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/config"
	kafka "go-shopping-poc/internal/platform/event/bus/kafka"
	"go-shopping-poc/internal/service/eventreader"
	"go-shopping-poc/internal/service/eventreader/eventhandlers"
	"go-shopping-poc/internal/testutils"
)

func TestEventReaderService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test environment
	testutils.SetupTestEnvironment(t)

	// Load test configuration
	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	// Create event bus
	broker := cfg.GetEventBroker()
	readTopics := cfg.GetEventReaderReadTopics()
	writeTopic := cfg.GetEventReaderWriteTopic()
	group := cfg.GetEventReaderGroup() + "-test"

	eventBus := kafka.NewEventBus(broker, readTopics, writeTopic, group)

	// Create service
	service := eventreader.NewEventReaderService(eventBus)

	// Register handlers
	customerCreatedHandler := eventhandlers.NewOnCustomerCreated()
	service.RegisterHandler(
		customerCreatedHandler.CreateFactory(),
		customerCreatedHandler.CreateHandler(),
	)

	// Start service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		if err := service.Start(ctx); err != nil {
			t.Errorf("Service start failed: %v", err)
		}
	}()

	// Wait for service to be ready
	time.Sleep(2 * time.Second)

	// Publish test event
	testEvent := events.NewCustomerCreatedEvent("test-customer-123", map[string]string{
		"test": "integration",
	})

	err := eventBus.Publish(ctx, testEvent.Topic(), testEvent)
	if err != nil {
		t.Errorf("Failed to publish test event: %v", err)
	}

	// Wait for processing
	time.Sleep(3 * time.Second)

	// Cleanup
	cancel()
	time.Sleep(1 * time.Second)
}
