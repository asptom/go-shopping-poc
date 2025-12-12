//go:build integration

package integration

import (
	"context"
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

// TestCleanArchitecture_Integration validates the entire clean architecture system
func TestCleanArchitecture_Integration(t *testing.T) {
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
	group := cfg.GetEventReaderGroup() + "-clean-arch-test"

	eventBus := kafka.NewEventBus(broker, readTopics, writeTopic, group)

	// Create service with clean architecture
	service := eventreader.NewEventReaderService(eventBus)

	// Register handlers using the new clean architecture pattern
	customerCreatedHandler := eventhandlers.NewOnCustomerCreated()
	err := eventreader.RegisterHandler(
		service,
		customerCreatedHandler.CreateFactory(),
		customerCreatedHandler.CreateHandler(),
	)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Verify handler registration (this should fix the "No handlers found" issue)
	handlerCount := service.HandlerCount()
	if handlerCount == 0 {
		t.Fatal("Expected handlers to be registered, but found none")
	}
	t.Logf("Successfully registered %d handlers", handlerCount)

	// Start service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startErr := make(chan error, 1)
	go func() {
		startErr <- service.Start(ctx)
	}()

	// Wait for service to be ready
	time.Sleep(2 * time.Second)

	// Test 1: Publish and process CustomerCreated event
	testCustomerID := "clean-arch-customer-123"
	testEvent := events.NewCustomerCreatedEvent(testCustomerID, map[string]string{
		"test":        "clean-architecture",
		"integration": "true",
	})

	publishErr := eventBus.Publish(ctx, testEvent.Topic(), testEvent)
	if publishErr != nil {
		t.Errorf("Failed to publish test event: %v", publishErr)
	}

	// Wait for processing
	time.Sleep(3 * time.Second)

	// Test 2: Verify platform utilities work correctly
	utils := handler.NewEventUtils()
	matcher := handler.NewEventTypeMatcher()

	// Test event validation
	validationErr := utils.ValidateEvent(ctx, testEvent)
	if validationErr != nil {
		t.Errorf("Event validation failed: %v", validationErr)
	}

	// Test event type matching
	isCustomerEvent := matcher.IsCustomerEvent(testEvent)
	if !isCustomerEvent {
		t.Error("Expected event to be identified as customer event")
	}

	// Test event ID extraction
	eventID := utils.GetEventID(testEvent)
	if eventID != testCustomerID {
		t.Errorf("Expected event ID %s, got %s", testCustomerID, eventID)
	}

	// Test 3: Test service health and information
	healthErr := service.Health()
	if healthErr != nil {
		t.Errorf("Service health check failed: %v", healthErr)
	}

	serviceName := service.Name()
	if serviceName != "eventreader" {
		t.Errorf("Expected service name 'eventreader', got '%s'", serviceName)
	}

	// Cleanup
	cancel()
	time.Sleep(1 * time.Second)

	// Check for service start errors
	select {
	case err := <-startErr:
		if err != nil && err != context.Canceled {
			t.Errorf("Service start error: %v", err)
		}
	default:
		// Service started successfully
	}

	t.Log("Clean architecture integration test completed successfully")
}

// TestHandlerRegistration_Integration specifically tests handler registration
func TestHandlerRegistration_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.SetupTestEnvironment(t)

	// Load test configuration
	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	// Create event bus
	broker := cfg.GetEventBroker()
	readTopics := cfg.GetEventReaderReadTopics()
	writeTopic := cfg.GetEventReaderWriteTopic()
	group := cfg.GetEventReaderGroup() + "-registration-test"

	eventBus := kafka.NewEventBus(broker, readTopics, writeTopic, group)

	// Create service
	service := eventreader.NewEventReaderService(eventBus)

	// Test initial state - no handlers
	handlerCount := service.HandlerCount()
	if handlerCount != 0 {
		t.Errorf("Expected 0 handlers initially, got %d", handlerCount)
	}

	// Register multiple handlers
	customerCreatedHandler := eventhandlers.NewOnCustomerCreated()

	// Register first handler
	err1 := eventreader.RegisterHandler(
		service,
		customerCreatedHandler.CreateFactory(),
		customerCreatedHandler.CreateHandler(),
	)
	if err1 != nil {
		t.Errorf("Failed to register first handler: %v", err1)
	}

	// Verify first handler registration
	handlerCount = service.HandlerCount()
	if handlerCount != 1 {
		t.Errorf("Expected 1 handler after first registration, got %d", handlerCount)
	}

	// Register the same handler again (should work)
	err2 := eventreader.RegisterHandler(
		service,
		customerCreatedHandler.CreateFactory(),
		customerCreatedHandler.CreateHandler(),
	)
	if err2 != nil {
		t.Errorf("Failed to register second handler: %v", err2)
	}

	// Verify second handler registration
	handlerCount = service.HandlerCount()
	if handlerCount != 2 {
		t.Errorf("Expected 2 handlers after second registration, got %d", handlerCount)
	}

	t.Logf("Successfully registered and verified %d handlers", handlerCount)
}

// TestPlatformUtilities_Integration tests the platform utilities with real events
func TestPlatformUtilities_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.SetupTestEnvironment(t)

	ctx := context.Background()
	utils := handler.NewEventUtils()
	matcher := handler.NewEventTypeMatcher()

	// Test 1: Customer event validation
	customerEvent := events.NewCustomerCreatedEvent("util-test-customer", map[string]string{
		"test": "platform-utilities",
	})

	err := utils.ValidateEvent(ctx, customerEvent)
	if err != nil {
		t.Errorf("Customer event validation failed: %v", err)
	}

	// Test 2: Invalid customer event validation
	invalidCustomerEvent := events.NewCustomerCreatedEvent("", map[string]string{
		"test": "invalid-event",
	})

	invalidErr := utils.ValidateEvent(ctx, invalidCustomerEvent)
	if invalidErr == nil {
		t.Error("Expected validation error for invalid customer event")
	}

	// Test 3: Event type matching
	isCustomer := matcher.IsCustomerEvent(customerEvent)
	if !isCustomer {
		t.Error("Failed to identify customer event correctly")
	}

	matchesMultiple := matcher.MatchEventType(customerEvent,
		string(events.CustomerCreated),
		string(events.CustomerUpdated))
	if !matchesMultiple {
		t.Error("Failed to match event against multiple types")
	}

	// Test 4: Event ID and resource extraction
	eventID := utils.GetEventID(customerEvent)
	if eventID != "util-test-customer" {
		t.Errorf("Expected event ID 'util-test-customer', got '%s'", eventID)
	}

	resourceID := utils.GetResourceID(customerEvent)
	// Resource ID might be empty, that's fine for this test

	// Test 5: Logging utilities
	utils.LogEventProcessing(ctx, customerEvent.Type(), eventID, resourceID)
	utils.LogEventCompletion(ctx, customerEvent.Type(), eventID, nil)

	// Test 6: Safe event processing
	processor := func(ctx context.Context, event events.Event) error {
		utils.LogEventProcessing(ctx, event.Type(), utils.GetEventID(event), utils.GetResourceID(event))
		return nil
	}

	safeErr := utils.SafeEventProcessing(ctx, customerEvent, processor)
	if safeErr != nil {
		t.Errorf("Safe event processing failed: %v", safeErr)
	}

	// Test 7: Safe event processing with panic
	panicProcessor := func(ctx context.Context, event events.Event) error {
		panic("test panic for recovery")
	}

	panicErr := utils.SafeEventProcessing(ctx, customerEvent, panicProcessor)
	if panicErr == nil {
		t.Error("Expected error from panic recovery, got nil")
	}

	// Test 8: Handle event with validation
	validationProcessor := func(ctx context.Context, event events.Event) error {
		utils.LogEventProcessing(ctx, event.Type(), utils.GetEventID(event), utils.GetResourceID(event))
		return nil
	}

	combinedErr := utils.HandleEventWithValidation(ctx, customerEvent, validationProcessor)
	if combinedErr != nil {
		t.Errorf("Handle event with validation failed: %v", combinedErr)
	}

	t.Log("Platform utilities integration test completed successfully")
}

// TestBusinessLogicExecution_Integration tests the actual business logic execution
func TestBusinessLogicExecution_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.SetupTestEnvironment(t)

	ctx := context.Background()

	// Create handler instance
	handler := eventhandlers.NewOnCustomerCreated()

	// Test 1: Create valid customer event
	customerEvent := events.NewCustomerCreatedEvent("business-logic-test-123", map[string]string{
		"source": "integration-test",
		"test":   "business-logic",
	})

	// Test 2: Execute business logic directly
	err := handler.Handle(ctx, customerEvent)
	if err != nil {
		t.Errorf("Business logic execution failed: %v", err)
	}

	// Test 3: Test with wrong event type (should be ignored)
	wrongEvent := events.NewCustomerUpdatedEvent("business-logic-test-123", map[string]string{
		"test": "wrong-event-type",
	})

	wrongErr := handler.Handle(ctx, wrongEvent)
	if wrongErr != nil {
		t.Errorf("Wrong event type handling should not return error, got: %v", wrongErr)
	}

	// Test 4: Test with non-customer event (should be handled gracefully)
	nonCustomerEvent := struct {
		events.Event
	}{}

	nonCustomerErr := handler.Handle(ctx, nonCustomerEvent)
	if nonCustomerErr != nil {
		t.Errorf("Non-customer event handling should not return error, got: %v", nonCustomerErr)
	}

	// Test 5: Verify handler factory and creation
	factory := handler.CreateFactory()
	if factory == nil {
		t.Error("Event factory should not be nil")
	}

	handlerFunc := handler.CreateHandler()
	if handlerFunc == nil {
		t.Error("Handler function should not be nil")
	}

	// Test 6: Test handler function directly
	factoryErr := handlerFunc(ctx, *customerEvent)
	if factoryErr != nil {
		t.Errorf("Handler function execution failed: %v", factoryErr)
	}

	// Test 7: Verify event type
	eventType := handler.EventType()
	if eventType != string(events.CustomerCreated) {
		t.Errorf("Expected event type %s, got %s", string(events.CustomerCreated), eventType)
	}

	t.Log("Business logic execution integration test completed successfully")
}

// TestErrorHandlingAndRecovery_Integration tests error handling scenarios
func TestErrorHandlingAndRecovery_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.SetupTestEnvironment(t)

	// Load test configuration
	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	// Create event bus
	broker := cfg.GetEventBroker()
	readTopics := cfg.GetEventReaderReadTopics()
	writeTopic := cfg.GetEventReaderWriteTopic()
	group := cfg.GetEventReaderGroup() + "-error-test"

	eventBus := kafka.NewEventBus(broker, readTopics, writeTopic, group)

	// Create service
	service := eventreader.NewEventReaderService(eventBus)

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
