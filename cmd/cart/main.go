package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/cors"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/platform/logging"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/outbox/providers"
	"go-shopping-poc/internal/platform/sse"
	"go-shopping-poc/internal/service/cart"
	"go-shopping-poc/internal/service/cart/eventhandlers"

	"github.com/go-chi/chi/v5"
)

func main() {
	loggerProvider, err := logging.NewLoggerProvider(logging.DefaultLoggerConfig("cart"))
	if err != nil {
		log.Fatalf("Cart: Failed to create logger provider: %v", err)
	}
	logger := loggerProvider.Logger()

	logger.Info("Cart service starting", "version", "1.0.0")

	cfg, err := cart.LoadConfig()
	if err != nil {
		logger.Error("Failed to load config", logging.ErrorAttr(err))
		os.Exit(1)
	}

	logger.Debug("Configuration loaded",
		"read_topics", cfg.ReadTopics,
		"write_topic", cfg.WriteTopic,
		"group", cfg.Group,
	)

	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		logger.Error("Database URL is required")
		os.Exit(1)
	}

	logger.Debug("Creating database provider")
	dbProvider, err := database.NewDatabaseProvider(dbURL, database.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to create database provider", logging.ErrorAttr(err))
		os.Exit(1)
	}
	db := dbProvider.GetDatabase()
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Error closing database", logging.ErrorAttr(err))
		}
	}()

	logger.Debug("Creating event bus provider")
	eventBusConfig := event.EventBusConfig{
		WriteTopic: cfg.WriteTopic,
		GroupID:    cfg.Group,
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig, event.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to create event bus provider", logging.ErrorAttr(err))
		os.Exit(1)
	}
	eventBus := eventBusProvider.GetEventBus()

	logger.Debug("Creating outbox components")
	writerProvider := providers.NewWriterProvider(db, providers.WithWriterLogger(logger))
	outboxWriter := writerProvider.GetWriter()

	outboxConfig := outbox.Config{
		BatchSize:       10,
		ProcessInterval: 5 * time.Second,
	}
	logger.Info("Outbox publisher configured",
		"interval", outboxConfig.ProcessInterval.String(),
		"batch_size", outboxConfig.BatchSize,
	)
	outboxPublisher := outbox.NewPublisher(db, eventBus, outboxConfig, outbox.WithPublisherLogger(logger))
	outboxPublisher.Start()
	defer outboxPublisher.Stop()

	logger.Debug("Creating CORS provider")
	corsProvider, err := cors.NewCORSProvider(cors.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to create CORS provider", logging.ErrorAttr(err))
		os.Exit(1)
	}
	corsHandler := corsProvider.GetCORSHandler()

	logger.Debug("Creating SSE provider")
	sseProvider := sse.NewProvider(
		sse.WithLogger(logger),
		sse.WithHandlerOptions(
			sse.WithMissingIDMessage("Missing cart ID"),
			sse.WithConnectedIDField("cart_id"),
			sse.WithLogIDKey("cart_id"),
		),
	)

	logger.Debug("Creating cart infrastructure")
	infrastructure := cart.NewCartInfrastructure(
		db, eventBus, outboxWriter, outboxPublisher, corsHandler, sseProvider,
	)

	logger.Debug("Creating cart service")
	service := cart.NewCartService(logger, infrastructure, cfg)

	logger.Debug("Registering event handlers")
	if err := registerEventHandlers(service, sseProvider.GetHub(), logger); err != nil {
		logger.Error("Failed to register event handlers", logging.ErrorAttr(err))
		os.Exit(1)
	}

	logger.Debug("Starting event consumer", "topics", service.EventBus().ReadTopics())
	go func() {
		ctx := context.Background()
		if err := service.Start(ctx); err != nil {
			logger.Error("Event consumer error", logging.ErrorAttr(err))
		}
	}()

	logger.Debug("Creating cart handler")
	handler := cart.NewCartHandler(logger, service)

	logger.Debug("Setting up HTTP router")
	router := chi.NewRouter()
	router.Use(corsHandler)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	cartRouter := chi.NewRouter()
	cartRouter.Post("/carts", handler.CreateCart)
	cartRouter.Get("/carts/{id}", handler.GetCart)
	cartRouter.Delete("/carts/{id}", handler.DeleteCart)

	cartRouter.Post("/carts/{id}/items", handler.AddItem)
	cartRouter.Put("/carts/{id}/items/{line}", handler.UpdateItem)
	cartRouter.Delete("/carts/{id}/items/{line}", handler.RemoveItem)

	cartRouter.Put("/carts/{id}/contact", handler.SetContact)

	cartRouter.Post("/carts/{id}/addresses", handler.AddAddress)

	cartRouter.Put("/carts/{id}/payment", handler.SetPayment)

	cartRouter.Post("/carts/{id}/checkout", handler.Checkout)

	cartRouter.Get("/carts/{id}/stream", sseProvider.GetHandler().ServeHTTP)

	router.Mount("/api/v1", cartRouter)

	serverAddr := "0.0.0.0" + cfg.ServicePort
	server := &http.Server{
		Addr:        serverAddr,
		Handler:     router,
		ReadTimeout: 30 * time.Second,
		IdleTimeout: 120 * time.Second,
	}

	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Starting HTTP server", "address", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start HTTP server", logging.ErrorAttr(err))
		}
	}()

	<-quit
	logger.Info("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", logging.ErrorAttr(err))
	}

	close(done)
	logger.Info("Server exited")
}

func registerEventHandlers(service *cart.CartService, sseHub *sse.Hub, logger *slog.Logger) error {
	logger.Debug("Registering event handlers")

	handlerLogger := logger.With("component", "event_handler")

	orderCreatedHandler := eventhandlers.NewOnOrderCreated(sseHub, handlerLogger)
	logger.Debug("Registering handler",
		"event_type", orderCreatedHandler.EventType(),
		"topic", events.OrderEvent{}.Topic(),
	)

	if err := cart.RegisterHandler(
		service,
		orderCreatedHandler.CreateFactory(),
		orderCreatedHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register OrderCreated handler: %w", err)
	}

	logger.Debug("Successfully registered OrderCreated handler")

	productValidatedHandler := eventhandlers.NewOnProductValidated(service.GetRepository(), sseHub, handlerLogger)
	logger.Debug("Registering handler", "event_type", productValidatedHandler.EventType())

	if err := cart.RegisterHandler(
		service,
		productValidatedHandler.CreateFactory(),
		productValidatedHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register ProductValidated handler: %w", err)
	}

	logger.Debug("Successfully registered ProductValidated handler")

	// Product cache event handler — keeps the product cache current.
	// Subscribes to ProductCreated, ProductUpdated, ProductDeleted events
	// from the ProductEvents topic.
	productCache := service.GetProductCache()
	productEventHandler := eventhandlers.NewOnProductEvent(productCache, handlerLogger)
	logger.Debug("Registering handler", "event_type", productEventHandler.EventType())

	if err := cart.RegisterHandler(
		service,
		productEventHandler.CreateFactory(),
		productEventHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register ProductEvent handler: %w", err)
	}

	logger.Debug("Successfully registered ProductEvent handler")
	logger.Debug("Event handler registration completed")

	return nil
}
