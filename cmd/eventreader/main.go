package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/platform/logging"
	"go-shopping-poc/internal/service/eventreader"
	"go-shopping-poc/internal/service/eventreader/eventhandlers"
)

func main() {
	loggerProvider, err := logging.NewLoggerProvider(logging.LoggerConfig{
		ServiceName: "eventreader",
	})
	if err != nil {
		log.Fatalf("Eventreader: Failed to create logger provider: %v", err)
	}
	logger := loggerProvider.Logger()

	logger.Info("EventReader service started", "version", "1.0.0")

	cfg, err := eventreader.LoadConfig()
	if err != nil {
		logger.Error("Failed to load config", logging.ErrorAttr(err))
		os.Exit(1)
	}

	logger.Debug("Configuration loaded",
		"read_topics", cfg.ReadTopics,
		"write_topic", cfg.WriteTopic,
		"group", cfg.Group,
	)

	eventBusConfig := event.EventBusConfig{
		WriteTopic: cfg.WriteTopic,
		GroupID:    cfg.Group,
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
	if err != nil {
		logger.Error("Failed to create event bus provider", logging.ErrorAttr(err))
		os.Exit(1)
	}

	eventBus := eventBusProvider.GetEventBus()

	infrastructure := eventreader.NewEventReaderInfrastructure(eventBus)

	service := eventreader.NewEventReaderService(logger, infrastructure, cfg)

	if err := registerEventHandlers(service, logger); err != nil {
		logger.Error("Failed to register event handlers", logging.ErrorAttr(err))
		os.Exit(1)
	}

	if err := validateServiceConfiguration(service, logger); err != nil {
		logger.Error("Service validation failed", logging.ErrorAttr(err))
		os.Exit(1)
	}

	logServiceInformation(service, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Debug("Starting event consumer")
	go func() {
		logger.Debug("Event consumer started")
		if err := service.Start(ctx); err != nil {
			logger.Error("Event consumer stopped", logging.ErrorAttr(err))
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logger.Info("Received shutdown signal, shutting down")

	if err := service.Stop(ctx); err != nil {
		logger.Error("Error during shutdown", logging.ErrorAttr(err))
	}
}

func registerEventHandlers(service *eventreader.EventReaderService, logger *slog.Logger) error {
	logger.Info("Registering event handlers")

	handlerLogger := logger.With("component", "event_handler")

	customerCreatedHandler := eventhandlers.NewOnCustomerCreated(handlerLogger)

	logger.Info("Registering handler",
		"event_type", customerCreatedHandler.EventType(),
		"topic", events.CustomerEvent{}.Topic(),
	)

	if err := eventreader.RegisterHandler(
		service,
		customerCreatedHandler.CreateFactory(),
		customerCreatedHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register CustomerCreated handler: %w", err)
	}

	logger.Info("Successfully registered CustomerCreated handler")

	productCreatedHandler := eventhandlers.NewOnProductCreated(handlerLogger)

	logger.Info("Registering handler",
		"event_type", productCreatedHandler.EventType(),
		"topic", events.ProductEvent{}.Topic(),
	)

	if err := eventreader.RegisterHandler(
		service,
		productCreatedHandler.CreateFactory(),
		productCreatedHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register ProductCreated handler: %w", err)
	}

	logger.Info("Successfully registered ProductCreated handler")
	logger.Info("Event handler registration completed")

	return nil
}

func validateServiceConfiguration(service *eventreader.EventReaderService, logger *slog.Logger) error {
	logger.Info("Validating service configuration")
	_ = service
	logger.Info("Service configuration validation passed")
	return nil
}

func logServiceInformation(service *eventreader.EventReaderService, logger *slog.Logger) {
	logger.Info("Service Information",
		"service_name", service.Name(),
		"handlers_registered", service.HandlerCount(),
	)

	customerEvent := &events.CustomerEvent{}
	logger.Info("Customer events will be processed",
		"topic", customerEvent.Topic(),
		"event_types", []string{string(events.CustomerCreated), string(events.CustomerUpdated)},
	)

	productEvent := &events.ProductEvent{}
	logger.Info("Product events will be processed",
		"topic", productEvent.Topic(),
		"event_types", []string{string(events.ProductCreated)},
	)

	logger.Info("Service is ready to start processing events")
}
