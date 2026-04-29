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

	"go-shopping-poc/internal/platform/auth"
	"go-shopping-poc/internal/platform/cors"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/platform/logging"
	"go-shopping-poc/internal/platform/outbox/providers"
	"go-shopping-poc/internal/service/order"
	"go-shopping-poc/internal/service/order/eventhandlers"

	"github.com/go-chi/chi/v5"
)

func main() {
	loggerProvider, err := logging.NewLoggerProvider(logging.DefaultLoggerConfig("order"))
	if err != nil {
		log.Fatalf("Order: Failed to create logger provider: %v", err)
	}
	logger := loggerProvider.Logger()

	logger.Info("Order service started", "version", "1.0.0")

	cfg, err := order.LoadConfig()
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

	logger.Debug("Creating outbox providers")
	writerProvider := providers.NewWriterProvider(db, providers.WithWriterLogger(logger))
	publisherProvider := providers.NewPublisherProvider(db, eventBus, providers.WithPublisherLogger(logger))
	if publisherProvider == nil {
		logger.Error("Failed to create publisher provider")
		os.Exit(1)
	}
	outboxPublisher := publisherProvider.GetPublisher()
	outboxPublisher.Start()
	defer outboxPublisher.Stop()
	outboxWriter := writerProvider.GetWriter()

	logger.Debug("Creating CORS provider")
	corsProvider, err := cors.NewCORSProvider(cors.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to create CORS provider", logging.ErrorAttr(err))
		os.Exit(1)
	}
	corsHandler := corsProvider.GetCORSHandler()

	logger.Debug("Creating order infrastructure")
	infrastructure := order.NewOrderInfrastructure(db, eventBus, outboxWriter, outboxPublisher, corsHandler)

	logger.Debug("Creating order service")
	service := order.NewOrderService(logger, infrastructure, cfg)

	logger.Debug("Registering event handlers")
	if err := registerEventHandlers(service, logger); err != nil {
		logger.Error("Failed to register event handlers", logging.ErrorAttr(err))
		os.Exit(1)
	}

	// Bootstrap identity cache from historical CustomerEvents
	logger.Info("Bootstrapping identity cache from CustomerEvents")
	if err := service.BootstrapIdentityCache(context.Background()); err != nil {
		logger.Warn("Identity cache bootstrap had issues — on-demand fallback active", "error", err)
	}
	logger.Info("Identity cache ready", "entries", service.IdentityCacheCount())

	logger.Debug("Starting event consumer", "topics", service.EventBus().ReadTopics())
	go func() {
		if err := service.Start(context.Background()); err != nil {
			logger.Error("Event consumer error", logging.ErrorAttr(err))
			}
	}()

	logger.Debug("Creating order handler")
	handler := order.NewOrderHandler(service)

	// Set up auth middleware
	var authMiddleware func(http.Handler) http.Handler
	if cfg.KeycloakIssuer != "" && cfg.KeycloakJWKSURL != "" {
		validator := auth.NewKeycloakValidator(cfg.KeycloakIssuer, cfg.KeycloakJWKSURL)
		authMiddleware = auth.RequireAuth(validator, "")
		logger.Info("Auth middleware enabled")
	} else {
		logger.Warn("Keycloak config not set — auth middleware disabled (all endpoints open)")
	}

	logger.Debug("Setting up HTTP router")
	router := chi.NewRouter()
	router.Use(corsHandler)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	orderRouter := chi.NewRouter()
	if authMiddleware != nil {
		orderRouter.Use(authMiddleware)
	}
	orderRouter.Get("/orders/{id}", handler.GetOrder)
	orderRouter.Get("/orders/customer/{customerId}", handler.GetOrdersByCustomer)
	orderRouter.Delete("/orders/{id}", handler.CancelOrder)
	orderRouter.Patch("/orders/{id}/status", handler.UpdateOrderStatus)

	router.Mount("/api/v1", orderRouter)

	serverAddr := "0.0.0.0" + cfg.ServicePort
	server := &http.Server{
		Addr:         serverAddr,
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

func registerEventHandlers(service *order.OrderService, logger *slog.Logger) error {
	logger.Debug("Registering event handlers")

	handlerLogger := logger.With("component", "event_handler")

	cartCheckedOutHandler := eventhandlers.NewOnCartCheckedOut(service, handlerLogger)
	logger.Debug("Registering handler", "event_type", cartCheckedOutHandler.EventType())

	if err := order.RegisterHandler(
		service,
		cartCheckedOutHandler.CreateFactory(),
		cartCheckedOutHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register CartCheckedOut handler: %w", err)
	}

	// Keep identity cache current via ongoing CustomerCreated/CustomerUpdated events
	identityUpdateHandler := eventhandlers.NewOnCustomerIdentityUpdate(service.IdentityCache(), handlerLogger)
	logger.Debug("Registering handler", "event_type", identityUpdateHandler.EventType())

	if err := order.RegisterHandler(
		service,
		identityUpdateHandler.CreateFactory(),
		identityUpdateHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register CustomerIdentityUpdate handler: %w", err)
	}

	// Handle fallback verification responses from customer service
	identityRespHandler := eventhandlers.NewOnIdentityVerificationCompleted(service, handlerLogger)
	logger.Debug("Registering handler", "event_type", identityRespHandler.EventType())

	if err := order.RegisterHandler(
		service,
		identityRespHandler.CreateFactory(),
		identityRespHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register IdentityVerificationCompleted handler: %w", err)
	}

	logger.Debug("Event handler registration completed")

	return nil
}
