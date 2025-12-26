//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/config"
	kafka "go-shopping-poc/internal/platform/event/bus/kafka"
	kafkaconfig "go-shopping-poc/internal/platform/event/kafka"
	"go-shopping-poc/internal/service/eventreader"
	"go-shopping-poc/internal/testutils"
)

// TestCleanArchitecture_Integration validates the entire clean architecture system
func TestCleanArchitecture_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test environment
	testutils.SetupTestEnvironment(t)

	// Load test configuration
	cfg, err := eventreader.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load eventreader config: %v", err)
	}

	// Load Kafka configuration
	kafkaCfg, err := config.LoadConfig[kafkaconfig.Config]("platform-kafka")
	if err != nil {
		t.Fatalf("Failed to load kafka config: %v", err)
	}

	// Create event bus
	eventBus := kafka.NewEventBus(kafkaCfg)

	// Create infrastructure
	infrastructure := eventreader.NewEventReaderInfrastructure(eventBus)

	// Create service
	service := eventreader.NewEventReaderService(infrastructure, cfg)

	// Test 1: Service health without handlers
	healthErr := service.Health()
	if healthErr != nil {
		t.Errorf("Service health check failed without handlers: %v", healthErr)
	}

	// Test 2: Start service without handlers (should work)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	startErr := make(chan error, 1)
	go func() {
		startErr <- service.Start(ctx)
	}()

	// Wait for service to start
	time.Sleep(1 * time.Second)

	// Test 3: Publish event when no handlers are registered
	testEvent := events.NewCustomerCreatedEvent("error-test-customer", map[string]string{
		"test": "error-handling",
	})

	// This should not fail even if no handlers are registered
	publishErr := eventBus.Publish(ctx, testEvent.Topic(), testEvent)
	if publishErr != nil {
		t.Errorf("Event publishing failed: %v", publishErr)
	}

	// Wait for any processing
	time.Sleep(2 * time.Second)

	// Test 4: Stop service
	stopErr := service.Stop(ctx)
	if stopErr != nil {
		t.Errorf("Service stop failed: %v", stopErr)
	}

	// Test 5: Check service start completion
	select {
	case err := <-startErr:
		if err != nil && err != context.Canceled {
			t.Errorf("Service start error: %v", err)
		}
	case <-time.After(2 * time.Second):
		// Service should have stopped by now
	}

	t.Log("Error handling and recovery integration test completed successfully")
}
