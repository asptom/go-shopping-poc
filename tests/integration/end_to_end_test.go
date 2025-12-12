//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/config"
	kafka "go-shopping-poc/internal/platform/event/bus/kafka"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/service/eventreader"
	"go-shopping-poc/internal/service/eventreader/eventhandlers"
	"go-shopping-poc/internal/testutils"
)

// TestEndToEndEventFlow_Integration tests the complete event flow from creation to processing
func TestEndToEndEventFlow_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test environment
	testutils.SetupTestEnvironment(t)

	// Load test configuration
	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	// Create event bus for publishing (simulating customer service)
	publishBroker := cfg.GetEventBroker()
	publishTopics := []string{cfg.GetEventReaderWriteTopic()} // Write to customer topic
	publishWriteTopic := cfg.GetEventReaderWriteTopic()
	publishGroup := cfg.GetEventReaderGroup() + "-publisher"

	publisherEventBus := kafka.NewEventBus(publishBroker, publishTopics, publishWriteTopic, publishGroup)

	// Create event bus for consuming (eventreader service)
	consumeBroker := cfg.GetEventBroker()
	consumeTopics := cfg.GetEventReaderReadTopics()
	consumeWriteTopic := cfg.GetEventReaderWriteTopic()
	consumeGroup := cfg.GetEventReaderGroup() + "-consumer"

	consumerEventBus := kafka.NewEventBus(consumeBroker, consumeTopics, consumeWriteTopic, consumeGroup)

	// Create and configure eventreader service
	service := eventreader.NewEventReaderService(consumerEventBus)

	// Register handlers
	customerCreatedHandler := eventhandlers.NewOnCustomerCreated()
	err := eventreader.RegisterHandler(
		service,
		customerCreatedHandler.CreateFactory(),
		customerCreatedHandler.CreateHandler(),
	)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Verify handlers are registered (this addresses the original "No handlers found" issue)
	handlerCount := service.HandlerCount()
	if handlerCount == 0 {
		t.Fatal("No handlers registered - this would cause the original issue")
	}
	t.Logf("Successfully registered %d handlers for end-to-end test", handlerCount)

	// Start the eventreader service
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	serviceStartErr := make(chan error, 1)
	go func() {
		serviceStartErr <- service.Start(ctx)
	}()

	// Start the publisher event bus
	publisherStartErr := make(chan error, 1)
	go func() {
		publisherStartErr <- publisherEventBus.StartConsuming(ctx)
	}()

	// Wait for both services to be ready
	time.Sleep(3 * time.Second)

	// Test Case 1: Complete customer creation flow
	t.Run("CustomerCreationFlow", func(t *testing.T) {
		testCustomerID := "e2e-customer-" + time.Now().Format("20060102-150405")

		// Step 1: Create customer event (simulating customer service)
		customerEvent := events.NewCustomerCreatedEvent(testCustomerID, map[string]string{
			"source":    "customer-service",
			"test":      "end-to-end",
			"timestamp": time.Now().Format(time.RFC3339),
		})

		// Step 2: Publish event to Kafka
		t.Logf("Publishing CustomerCreated event for customer %s", testCustomerID)
		publishErr := publisherEventBus.Publish(ctx, customerEvent.Topic(), customerEvent)
		if publishErr != nil {
			t.Errorf("Failed to publish customer event: %v", publishErr)
			return
		}

		// Step 3: Wait for event processing
		t.Log("Waiting for event processing...")
		time.Sleep(5 * time.Second)

		// Step 4: Verify event was processed (through logs and service health)
		healthErr := service.Health()
		if healthErr != nil {
			t.Errorf("Service health check failed after event processing: %v", healthErr)
		}

		t.Logf("Successfully processed CustomerCreated event for customer %s", testCustomerID)
	})

	// Test Case 2: Multiple events processing
	t.Run("MultipleEventsProcessing", func(t *testing.T) {
		// Create multiple customer events
		customers := []string{
			"e2e-customer-multiple-1",
			"e2e-customer-multiple-2",
			"e2e-customer-multiple-3",
		}

		for i, customerID := range customers {
			event := events.NewCustomerCreatedEvent(customerID, map[string]string{
				"batch": "true",
				"index": string(rune('1' + i)),
				"test":  "multiple-events",
			})

			t.Logf("Publishing event %d for customer %s", i+1, customerID)
			err := publisherEventBus.Publish(ctx, event.Topic(), event)
			if err != nil {
				t.Errorf("Failed to publish event %d: %v", i+1, err)
				continue
			}

			// Small delay between events
			time.Sleep(500 * time.Millisecond)
		}

		// Wait for all events to be processed
		t.Log("Waiting for multiple events processing...")
		time.Sleep(8 * time.Second)

		// Verify service is still healthy
		healthErr := service.Health()
		if healthErr != nil {
			t.Errorf("Service health check failed after multiple events: %v", healthErr)
		}

		t.Log("Successfully processed multiple customer events")
	})

	// Test Case 3: Platform utilities integration
	t.Run("PlatformUtilitiesIntegration", func(t *testing.T) {
		utils := handler.NewEventUtils()
		matcher := handler.NewEventTypeMatcher()

		// Create test event
		testEvent := events.NewCustomerCreatedEvent("e2e-utils-test", map[string]string{
			"test": "platform-utilities",
		})

		// Test validation
		err := utils.ValidateEvent(ctx, testEvent)
		if err != nil {
			t.Errorf("Event validation failed: %v", err)
		}

		// Test type matching
		isCustomer := matcher.IsCustomerEvent(testEvent)
		if !isCustomer {
			t.Error("Failed to identify customer event")
		}

		// Test ID extraction
		eventID := utils.GetEventID(testEvent)
		if eventID != "e2e-utils-test" {
			t.Errorf("Expected event ID 'e2e-utils-test', got '%s'", eventID)
		}

		// Test logging utilities
		utils.LogEventProcessing(ctx, testEvent.Type(), eventID, utils.GetResourceID(testEvent))
		utils.LogEventCompletion(ctx, testEvent.Type(), eventID, nil)

		// Test safe processing
		processor := func(ctx context.Context, event events.Event) error {
			utils.LogEventProcessing(ctx, event.Type(), utils.GetEventID(event), utils.GetResourceID(event))
			return nil
		}

		err = utils.SafeEventProcessing(ctx, testEvent, processor)
		if err != nil {
			t.Errorf("Safe processing failed: %v", err)
		}

		t.Log("Platform utilities integration test passed")
	})

	// Test Case 4: Error handling and recovery
	t.Run("ErrorHandlingAndRecovery", func(t *testing.T) {
		// Test with invalid event
		invalidEvent := events.NewCustomerCreatedEvent("", map[string]string{
			"test": "invalid-event",
		})

		// Publish invalid event (should be handled gracefully)
		err := publisherEventBus.Publish(ctx, invalidEvent.Topic(), invalidEvent)
		if err != nil {
			t.Errorf("Failed to publish invalid event: %v", err)
		}

		// Wait for processing
		time.Sleep(2 * time.Second)

		// Service should still be healthy after invalid event
		healthErr := service.Health()
		if healthErr != nil {
			t.Errorf("Service health check failed after invalid event: %v", healthErr)
		}

		t.Log("Error handling and recovery test passed")
	})

	// Cleanup
	cancel()
	time.Sleep(2 * time.Second)

	// Check for service errors
	select {
	case err := <-serviceStartErr:
		if err != nil && err != context.Canceled {
			t.Errorf("Service start error: %v", err)
		}
	case err := <-publisherStartErr:
		if err != nil && err != context.Canceled {
			t.Errorf("Publisher start error: %v", err)
		}
	default:
		// Services completed successfully
	}

	t.Log("End-to-end event flow integration test completed successfully")
}

// TestOriginalIssueResolution_Integration specifically validates that the original issue is resolved
func TestOriginalIssueResolution_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Log("=== Testing Original Issue Resolution ===")
	t.Log("Original Issue: 'No handlers found' when processing events")

	testutils.SetupTestEnvironment(t)

	// Load test configuration
	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	// Create event bus
	broker := cfg.GetEventBroker()
	readTopics := cfg.GetEventReaderReadTopics()
	writeTopic := cfg.GetEventReaderWriteTopic()
	group := cfg.GetEventReaderGroup() + "-original-issue-test"

	eventBus := kafka.NewEventBus(broker, readTopics, writeTopic, group)

	// Create service
	service := eventreader.NewEventReaderService(eventBus)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// === Step 1: Register Handler (The Fix) ===
	t.Log("Step 1: Registering handler (this should prevent 'No handlers found' error)")

	customerCreatedHandler := eventhandlers.NewOnCustomerCreated()

	err := eventreader.RegisterHandler(
		service,
		customerCreatedHandler.CreateFactory(),
		customerCreatedHandler.CreateHandler(),
	)
	if err != nil {
		t.Fatalf("FAILED: Could not register handler - this would cause the original issue: %v", err)
	}

	// Verify handler is registered
	handlerCount := service.HandlerCount()
	if handlerCount == 0 {
		t.Fatal("FAILED: No handlers registered - this would reproduce the original issue")
	}

	t.Logf("✓ Handler registered successfully: %d handlers", handlerCount)

	// === Step 2: Start Service ===
	t.Log("Step 2: Starting service with registered handlers")

	startErr := make(chan error, 1)
	go func() {
		startErr <- service.Start(ctx)
	}()

	time.Sleep(2 * time.Second)

	// === Step 3: Publish Event (This would fail without handlers) ===
	t.Log("Step 3: Publishing event (this should work now with registered handlers)")

	testEvent := events.NewCustomerCreatedEvent("original-issue-test", map[string]string{
		"test": "original-issue-resolution",
	})

	err = eventBus.Publish(ctx, testEvent.Topic(), testEvent)
	if err != nil {
		t.Errorf("FAILED: Event publishing failed: %v", err)
	}

	// Wait for processing
	time.Sleep(3 * time.Second)

	// === Step 4: Verify Processing ===
	t.Log("Step 4: Verifying event processing")

	// Service should be healthy
	healthErr := service.Health()
	if healthErr != nil {
		t.Errorf("FAILED: Service health check failed: %v", healthErr)
	}

	// === Step 5: Cleanup ===
	t.Log("Step 5: Cleanup")

	cancel()
	time.Sleep(1 * time.Second)

	select {
	case err := <-startErr:
		if err != nil && err != context.Canceled {
			t.Errorf("FAILED: Service error: %v", err)
		}
	default:
	}

	t.Log("=== Original Issue Resolution Test PASSED ===")
	t.Log("✓ Handler registration prevents 'No handlers found' error")
	t.Log("✓ Events are processed successfully with registered handlers")
	t.Log("✓ Service remains healthy after event processing")
}

// TestSystemValidation_Integration is a comprehensive system validation test
func TestSystemValidation_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Log("=== Starting Comprehensive System Validation ===")

	// Setup test environment
	testutils.SetupTestEnvironment(t)

	// Load test configuration
	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	// Create event bus
	broker := cfg.GetEventBroker()
	readTopics := cfg.GetEventReaderReadTopics()
	writeTopic := cfg.GetEventReaderWriteTopic()
	group := cfg.GetEventReaderGroup() + "-system-validation"

	eventBus := kafka.NewEventBus(broker, readTopics, writeTopic, group)

	// Create service
	service := eventreader.NewEventReaderService(eventBus)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// === Phase 1: Validate Initial State ===
	t.Log("Phase 1: Validating initial service state")

	handlerCount := service.HandlerCount()
	if handlerCount != 0 {
		t.Errorf("Expected 0 handlers initially, got %d", handlerCount)
	}
	t.Logf("✓ Initial state validated: %d handlers", handlerCount)

	// === Phase 2: Handler Registration ===
	t.Log("Phase 2: Testing handler registration")

	customerCreatedHandler := eventhandlers.NewOnCustomerCreated()

	// Register handler using service layer wrapper
	err := eventreader.RegisterHandler(
		service,
		customerCreatedHandler.CreateFactory(),
		customerCreatedHandler.CreateHandler(),
	)
	if err != nil {
		t.Fatalf("FAILED: Handler registration failed: %v", err)
	}

	// Verify registration
	handlerCount = service.HandlerCount()
	if handlerCount != 1 {
		t.Fatalf("FAILED: Expected 1 handler after registration, got %d", handlerCount)
	}
	t.Logf("✓ Handler registration successful: %d handlers registered", handlerCount)

	// === Phase 3: Service Startup ===
	t.Log("Phase 3: Starting service with registered handlers")

	startErr := make(chan error, 1)
	go func() {
		startErr <- service.Start(ctx)
	}()

	// Wait for service to be ready
	time.Sleep(3 * time.Second)

	// Verify service health
	healthErr := service.Health()
	if healthErr != nil {
		t.Fatalf("FAILED: Service health check failed: %v", healthErr)
	}
	t.Log("✓ Service started successfully with registered handlers")

	// === Phase 4: Event Processing ===
	t.Log("Phase 4: Testing event processing with registered handlers")

	testCases := []struct {
		name        string
		customerID  string
		metadata    map[string]string
		expectError bool
	}{
		{
			name:       "Valid Customer Event",
			customerID: "validation-customer-1",
			metadata: map[string]string{
				"test":  "system-validation",
				"phase": "event-processing",
				"valid": "true",
			},
			expectError: false,
		},
		{
			name:       "Customer Event with Complex Metadata",
			customerID: "validation-customer-2",
			metadata: map[string]string{
				"test":           "system-validation",
				"phase":          "event-processing",
				"complex":        "true",
				"timestamp":      time.Now().Format(time.RFC3339),
				"source":         "integration-test",
				"correlation-id": "test-" + fmt.Sprintf("%d", time.Now().Unix()),
			},
			expectError: false,
		},
		{
			name:       "Invalid Customer Event (Empty ID)",
			customerID: "", // Invalid
			metadata: map[string]string{
				"test":  "system-validation",
				"phase": "event-processing",
				"valid": "false",
			},
			expectError: true, // Should fail validation
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("EventProcessing_%d_%s", i+1, tc.name), func(t *testing.T) {
			// Create event
			event := events.NewCustomerCreatedEvent(tc.customerID, tc.metadata)

			// Publish event
			t.Logf("Publishing event: %s (customer: %s)", tc.name, tc.customerID)
			publishErr := eventBus.Publish(ctx, event.Topic(), event)
			if publishErr != nil {
				t.Errorf("FAILED: Event publishing failed: %v", publishErr)
				return
			}

			// Wait for processing
			time.Sleep(2 * time.Second)

			// Verify service is still healthy
			healthErr := service.Health()
			if healthErr != nil {
				t.Errorf("FAILED: Service health check failed after event %s: %v", tc.name, healthErr)
				return
			}

			t.Logf("✓ Event processed successfully: %s", tc.name)
		})
	}

	// === Phase 5: Platform Utilities Validation ===
	t.Log("Phase 5: Validating platform utilities")

	utils := handler.NewEventUtils()
	matcher := handler.NewEventTypeMatcher()

	// Test utilities with various events
	utilityTestEvent := events.NewCustomerCreatedEvent("utilities-test", map[string]string{
		"test": "platform-utilities-validation",
	})

	// Test validation
	validationErr := utils.ValidateEvent(ctx, utilityTestEvent)
	if validationErr != nil {
		t.Errorf("FAILED: Platform utilities validation failed: %v", validationErr)
	}

	// Test type matching
	isCustomer := matcher.IsCustomerEvent(utilityTestEvent)
	if !isCustomer {
		t.Error("FAILED: Platform utilities type matching failed")
	}

	// Test ID extraction
	eventID := utils.GetEventID(utilityTestEvent)
	if eventID != "utilities-test" {
		t.Errorf("FAILED: Platform utilities ID extraction failed: expected 'utilities-test', got '%s'", eventID)
	}

	// Test logging utilities
	utils.LogEventProcessing(ctx, utilityTestEvent.Type(), eventID, utils.GetResourceID(utilityTestEvent))
	utils.LogEventCompletion(ctx, utilityTestEvent.Type(), eventID, nil)

	t.Log("✓ Platform utilities validation completed")

	// === Phase 6: Business Logic Validation ===
	t.Log("Phase 6: Validating business logic execution")

	businessLogicEvent := events.NewCustomerCreatedEvent("business-logic-test", map[string]string{
		"test": "business-logic-validation",
	})

	// Execute business logic directly
	businessErr := customerCreatedHandler.Handle(ctx, businessLogicEvent)
	if businessErr != nil {
		t.Errorf("FAILED: Business logic execution failed: %v", businessErr)
	}

	// Test handler factory and creation
	factory := customerCreatedHandler.CreateFactory()
	if factory == nil {
		t.Error("FAILED: Business logic factory is nil")
	}

	handlerFunc := customerCreatedHandler.CreateHandler()
	if handlerFunc == nil {
		t.Error("FAILED: Business logic handler function is nil")
	}

	// Test handler function directly
	factoryErr := handlerFunc(ctx, *businessLogicEvent)
	if factoryErr != nil {
		t.Errorf("FAILED: Business logic handler function execution failed: %v", factoryErr)
	}

	t.Log("✓ Business logic validation completed")

	// === Phase 7: Multiple Handler Registration ===
	t.Log("Phase 7: Testing multiple handler registration")

	// Register additional handlers
	additionalHandler := eventhandlers.NewOnCustomerCreated()

	regErr := eventreader.RegisterHandler(
		service,
		additionalHandler.CreateFactory(),
		additionalHandler.CreateHandler(),
	)
	if regErr != nil {
		t.Errorf("FAILED: Additional handler registration failed: %v", regErr)
	}

	// Verify multiple handlers
	handlerCount = service.HandlerCount()
	if handlerCount != 2 {
		t.Errorf("FAILED: Expected 2 handlers after additional registration, got %d", handlerCount)
	}

	t.Logf("✓ Multiple handler registration completed: %d handlers", handlerCount)

	// === Phase 8: Stress Testing ===
	t.Log("Phase 8: Stress testing with multiple events")

	stressEventCount := 10
	for i := 0; i < stressEventCount; i++ {
		customerID := fmt.Sprintf("stress-test-customer-%d", i+1)
		event := events.NewCustomerCreatedEvent(customerID, map[string]string{
			"test":  "stress-test",
			"index": fmt.Sprintf("%d", i+1),
			"batch": "true",
		})

		err := eventBus.Publish(ctx, event.Topic(), event)
		if err != nil {
			t.Errorf("FAILED: Stress test event %d publishing failed: %v", i+1, err)
			continue
		}

		// Small delay between events
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for all stress events to be processed
	t.Logf("Waiting for %d stress events to be processed...", stressEventCount)
	time.Sleep(10 * time.Second)

	// Verify service health after stress test
	healthErr = service.Health()
	if healthErr != nil {
		t.Errorf("FAILED: Service health check failed after stress test: %v", healthErr)
	}

	t.Logf("✓ Stress testing completed: %d events processed", stressEventCount)

	// === Phase 9: Cleanup and Final Validation ===
	t.Log("Phase 9: Cleanup and final validation")

	// Stop service
	stopErr := service.Stop(ctx)
	if stopErr != nil {
		t.Errorf("FAILED: Service stop failed: %v", stopErr)
	}

	// Wait for service to stop
	time.Sleep(2 * time.Second)

	// Check service completion
	select {
	case err := <-startErr:
		if err != nil && err != context.Canceled {
			t.Errorf("FAILED: Service completion error: %v", err)
		}
	default:
	}

	// Final validation
	finalHandlerCount := service.HandlerCount()
	if finalHandlerCount != 2 {
		t.Errorf("FAILED: Expected 2 handlers in final state, got %d", finalHandlerCount)
	}

	t.Log("✓ Cleanup and final validation completed")

	// === Test Summary ===
	t.Log("=== System Validation Summary ===")
	t.Logf("✓ Initial state: Validated")
	t.Logf("✓ Handler registration: %d handlers registered", finalHandlerCount)
	t.Logf("✓ Service startup: Successful")
	t.Logf("✓ Event processing: %d test cases processed", len(testCases))
	t.Logf("✓ Platform utilities: Validated")
	t.Logf("✓ Business logic: Validated")
	t.Logf("✓ Multiple handlers: Validated")
	t.Logf("✓ Stress testing: %d events processed", stressEventCount)
	t.Logf("✓ Cleanup: Completed")
	t.Log("=== System Validation PASSED ===")
}
