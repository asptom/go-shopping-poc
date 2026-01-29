package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/service/eventreader"
	"go-shopping-poc/internal/service/eventreader/eventhandlers"
)

func main() {
	log.SetFlags(log.LstdFlags)

	log.Printf("[INFO] Eventreader: EventReader service started")

	// Load service-specific configuration
	cfg, err := eventreader.LoadConfig()
	if err != nil {
		log.Printf("[ERROR] Eventreader: Failed to load config: %v", err)
		os.Exit(1)
	}

	log.Printf("[DEBUG] Eventreader: Configuration loaded successfully")
	log.Printf("[DEBUG] Eventreader: Read Topics: %v, Write Topic: %v, Group: %s",
		cfg.ReadTopics, cfg.WriteTopic, cfg.Group)

	// Create event bus provider
	eventBusConfig := event.EventBusConfig{
		WriteTopic: cfg.WriteTopic,
		GroupID:    cfg.Group,
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
	if err != nil {
		log.Printf("[ERROR] Eventreader: Failed to create event bus provider: %v", err)
		os.Exit(1)
	}

	// Get event bus from provider
	eventBus := eventBusProvider.GetEventBus()

	// Create infrastructure from provider
	infrastructure := eventreader.NewEventReaderInfrastructure(eventBus)

	// Create service
	service := eventreader.NewEventReaderService(infrastructure, cfg)

	// Register event handlers with validation
	if err := registerEventHandlers(service); err != nil {
		log.Printf("[ERROR] Eventreader: Failed to register event handlers: %v", err)
		os.Exit(1)
	}

	// Validate service configuration
	if err := validateServiceConfiguration(service); err != nil {
		log.Printf("[ERROR] Eventreader: Service validation failed: %v", err)
		os.Exit(1)
	}

	// Log service information
	logServiceInformation(service)

	// Start service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("[DEBUG] Eventreader: Starting event consumer...")
	go func() {
		log.Printf("[DEBUG] Eventreader: Event consumer started")
		if err := service.Start(ctx); err != nil {
			log.Printf("[ERROR] Eventreader: Event consumer stopped: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Printf("[DEBUG] Eventreader: Received shutdown signal, shutting down...")

	// Graceful shutdown
	if err := service.Stop(ctx); err != nil {
		log.Printf("[ERROR] Eventreader: Error during shutdown: %v", err)
	}
}

// registerEventHandlers registers all event handlers with the service and validates registration
func registerEventHandlers(service *eventreader.EventReaderService) error {
	log.Printf("[INFO] Eventreader: Registering event handlers...")

	// Register CustomerCreated handler using the clean generic method
	customerCreatedHandler := eventhandlers.NewOnCustomerCreated()

	// Log handler registration details
	log.Printf("[INFO] Eventreader: Registering handler for event type: %s", customerCreatedHandler.EventType())
	log.Printf("[INFO] Eventreader: Handler will process events from topic: %s", events.CustomerEvent{}.Topic())

	if err := eventreader.RegisterHandler(
		service,
		customerCreatedHandler.CreateFactory(),
		customerCreatedHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register CustomerCreated handler: %w", err)
	}

	log.Printf("[INFO] Eventreader: Successfully registered CustomerCreated handler")

	// Register CustomerCreated handler using the clean generic method
	productCreatedHandler := eventhandlers.NewOnProductCreated()

	// Log handler registration details
	log.Printf("[INFO] Eventreader: Registering handler for event type: %s", productCreatedHandler.EventType())
	log.Printf("[INFO] Eventreader: Handler will process events from topic: %s", events.ProductEvent{}.Topic())

	if err := eventreader.RegisterHandler(
		service,
		productCreatedHandler.CreateFactory(),
		productCreatedHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register ProductAdded handler: %w", err)
	}

	log.Printf("[INFO] Eventreader: Successfully registered ProductCreated reated handler")

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

	log.Printf("[INFO] Eventreader: Event handler registration completed")
	return nil
}

// validateServiceConfiguration validates the service configuration before starting
func validateServiceConfiguration(service *eventreader.EventReaderService) error {
	log.Printf("[INFO] Eventreader: Validating service configuration...")

	// Service configuration validation passed - handlers are registered during startup
	// In the future, this could validate service health, configuration, etc.
	_ = service // Will be used for future validation logic

	log.Printf("[INFO] Eventreader: Service configuration validation passed")
	return nil
}

// logServiceInformation logs detailed information about the service configuration
func logServiceInformation(service *eventreader.EventReaderService) {
	log.Printf("[INFO] Eventreader: Service Information:")
	log.Printf("[INFO] Eventreader:   Service Name: %s", service.Name())
	log.Printf("[INFO] Eventreader:   Handlers Registered: %d", service.HandlerCount())

	// Log specific topic mapping for customer events
	customerEvent := &events.CustomerEvent{}
	log.Printf("[INFO] Eventreader:   Customer events will be processed from topic: %s", customerEvent.Topic())
	log.Printf("[INFO] Eventreader:   Customer event types include: %s, %s",
		events.CustomerCreated, events.CustomerUpdated)

	productEvent := &events.ProductEvent{}
	log.Printf("[INFO] Eventreader:   Product events will be processed from topic: %s", productEvent.Topic())
	log.Printf("[INFO] Eventreader:   Product event types include: %s",
		events.ProductCreated)

	log.Printf("[INFO] Eventreader: Service is ready to start processing events")
}
