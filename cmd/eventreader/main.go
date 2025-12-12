package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/config"
	kafka "go-shopping-poc/internal/platform/event/bus/kafka"
	"go-shopping-poc/internal/platform/logging"
	"go-shopping-poc/internal/service/eventreader"
	"go-shopping-poc/internal/service/eventreader/eventhandlers"
)

func main() {

	logging.SetLevel("DEBUG")
	logging.Info("Eventreader: EventReader service started")

	// Load configuration

	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	logging.Debug("Eventreader: Configuration loaded from %s", envFile)
	logging.Debug("Eventreader: Config: %v", cfg)

	broker := cfg.GetEventBroker()
	readTopics := cfg.GetEventReaderReadTopics()
	writeTopic := cfg.GetEventReaderWriteTopic()
	group := cfg.GetEventReaderGroup()

	logging.Debug("Eventreader: Event Broker: %s, Read Topics: %v, Write Topic: %v, Group: %s", broker, readTopics, writeTopic, group)

	eventBus := kafka.NewEventBus(broker, readTopics, writeTopic, group)

	// Create service
	service := eventreader.NewEventReaderService(eventBus)

	// Register event handlers with validation
	if err := registerEventHandlers(service); err != nil {
		logging.Error("Eventreader: Failed to register event handlers: %v", err)
		os.Exit(1)
	}

	// Validate service configuration
	if err := validateServiceConfiguration(service); err != nil {
		logging.Error("Eventreader: Service validation failed: %v", err)
		os.Exit(1)
	}

	// Log service information
	logServiceInformation(service)

	// Start service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logging.Debug("Eventreader: Starting event consumer...")
	go func() {
		logging.Debug("Eventreader: Event consumer started")
		if err := service.Start(ctx); err != nil {
			logging.Error("Eventreader: Event consumer stopped:", err)
		}
	}()

	// Wait for shutdown signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logging.Debug("Eventreader: Received shutdown signal, shutting down...")

	// Graceful shutdown
	if err := service.Stop(ctx); err != nil {
		logging.Error("Eventreader: Error during shutdown:", err)
	}
}

// registerEventHandlers registers all event handlers with the service and validates registration
func registerEventHandlers(service *eventreader.EventReaderService) error {
	logging.Info("Eventreader: Registering event handlers...")

	// Register CustomerCreated handler using the clean generic method
	customerCreatedHandler := eventhandlers.NewOnCustomerCreated()

	// Log handler registration details
	logging.Info("Eventreader: Registering handler for event type: %s", customerCreatedHandler.EventType())
	logging.Info("Eventreader: Handler will process events from topic: %s", events.CustomerEvent{}.Topic())

	if err := eventreader.RegisterHandler(
		service,
		customerCreatedHandler.CreateFactory(),
		customerCreatedHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register CustomerCreated handler: %w", err)
	}

	logging.Info("Eventreader: Successfully registered CustomerCreated handler")

	// Future handlers can be registered here using the same generic method:
	// customerUpdatedHandler := eventhandlers.NewOnCustomerUpdated()
	// if err := eventreader.RegisterHandler(
	//     service,
	//     customerUpdatedHandler.CreateFactory(),
	//     customerUpdatedHandler.CreateHandler(),
	// ); err != nil {
	//     return fmt.Errorf("failed to register CustomerUpdated handler: %w", err)
	// }

	// orderCreatedHandler := eventhandlers.NewOnOrderCreated()
	// if err := eventreader.RegisterHandler(
	//     service,
	//     orderCreatedHandler.CreateFactory(),
	//     orderCreatedHandler.CreateHandler(),
	// ); err != nil {
	//     return fmt.Errorf("failed to register OrderCreated handler: %w", err)
	// }

	logging.Info("Eventreader: Event handler registration completed")
	return nil
}

// validateServiceConfiguration validates the service configuration before starting
func validateServiceConfiguration(service *eventreader.EventReaderService) error {
	logging.Info("Eventreader: Validating service configuration...")

	// Service configuration validation passed - handlers are registered during startup
	// In the future, this could validate service health, configuration, etc.
	_ = service // Will be used for future validation logic

	logging.Info("Eventreader: Service configuration validation passed")
	return nil
}

// logServiceInformation logs detailed information about the service configuration
func logServiceInformation(service *eventreader.EventReaderService) {
	logging.Info("Eventreader: Service Information:")
	logging.Info("Eventreader:   Service Name: %s", service.Name())
	logging.Info("Eventreader:   Handlers Registered: %d", service.HandlerCount())

	// Log specific topic mapping for customer events
	customerEvent := &events.CustomerEvent{}
	logging.Info("Eventreader: Customer events will be processed from topic: %s", customerEvent.Topic())
	logging.Info("Eventreader: Customer event types include: %s, %s",
		events.CustomerCreated, events.CustomerUpdated)

	logging.Info("Eventreader: Service is ready to start processing events")
}
