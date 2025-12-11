package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go-shopping-poc/internal/platform/config"
	kafka "go-shopping-poc/internal/platform/event/bus/kafka"
	"go-shopping-poc/internal/platform/logging"
	"go-shopping-poc/internal/service/eventreader"
	"go-shopping-poc/internal/service/eventreader/eventhandlers"
)

func main() {

	logging.SetLevel("INFO")
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

	// Register event handlers
	registerEventHandlers(service)

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

// registerEventHandlers registers all event handlers with the service
func registerEventHandlers(service *eventreader.EventReaderService) {
	// Register CustomerCreated handler using the clean generic method
	customerCreatedHandler := eventhandlers.NewOnCustomerCreated()
	eventreader.RegisterHandler(
		service,
		customerCreatedHandler.CreateFactory(),
		customerCreatedHandler.CreateHandler(),
	)

	// Future handlers can be registered here using the same generic method:
	// customerUpdatedHandler := eventhandlers.NewOnCustomerUpdated()
	// eventreader.RegisterHandler(
	//     service,
	//     customerUpdatedHandler.CreateFactory(),
	//     customerUpdatedHandler.CreateHandler(),
	// )

	// orderCreatedHandler := eventhandlers.NewOnOrderCreated()
	// eventreader.RegisterHandler(
	//     service,
	//     orderCreatedHandler.CreateFactory(),
	//     orderCreatedHandler.CreateHandler(),
	// )
}
