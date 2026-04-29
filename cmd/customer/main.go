package main

import (
	"context"
	"log"
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

	"go-shopping-poc/internal/service/customer"
	"go-shopping-poc/internal/service/customer/eventhandlers"

	"github.com/go-chi/chi/v5"
)

func main() {
	loggerProvider, err := logging.NewLoggerProvider(logging.DefaultLoggerConfig("customer"))
	if err != nil {
		log.Fatalf("Customer: Failed to create logger provider: %v", err)
	}
	logger := loggerProvider.Logger()

	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic recovered in customer service", "panic", r)
			}
	}()

	logger.Info("Customer service starting", "version", "1.0.0")

	cfg, err := customer.LoadConfig()
	if err != nil {
		logger.Error("Failed to load config", logging.ErrorAttr(err))
		os.Exit(1)
	}

	logger.Debug("Configuration loaded")

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
			logger.Error("Error closing database connection", logging.ErrorAttr(err))
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

	// Optional Keycloak auth middleware
	var authMiddleware func(http.Handler) http.Handler
	if cfg.KeycloakIssuer != "" && cfg.KeycloakJWKSURL != "" {
		validator := auth.NewKeycloakValidator(cfg.KeycloakIssuer, cfg.KeycloakJWKSURL)
		authMiddleware = auth.RequireAuth(validator, "")
		logger.Info("Keycloak auth middleware enabled")
	}

	logger.Debug("Creating customer infrastructure")
	infrastructure := customer.NewCustomerInfrastructure(db, eventBus, outboxWriter, outboxPublisher, corsHandler)
	logger.Debug("Infrastructure created successfully")

	logger.Debug("Creating customer service")
	service := customer.NewCustomerService(logger, infrastructure, cfg)
	logger.Debug("Service created successfully")

	logger.Debug("Creating customer handler")
	handler := customer.NewCustomerHandler(service)
	logger.Debug("Handler created successfully")

	// Register identity verification event handler (consumes from OrderEvents topic)
	identityVerificationHandler := eventhandlers.NewOnIdentityVerificationRequested(
		*service, logger.With("component", "identity_verification"),
	)
	if err := customer.RegisterHandler(
		service,
		identityVerificationHandler.CreateFactory(),
		identityVerificationHandler.CreateHandler(),
	); err != nil {
		logger.Error("Failed to register identity verification handler", logging.ErrorAttr(err))
		os.Exit(1)
	}

	// Start event consumer (handles verification requests from order service)
	go func() {
		if err := service.Start(context.Background()); err != nil {
			logger.Error("Event consumer error", logging.ErrorAttr(err))
		}
	}()

	logger.Debug("Setting up HTTP router")
	router := chi.NewRouter()
	logger.Debug("Router setup completed")

	router.Use(corsHandler)

	router.Get("/health", healthHandler)

	customerRouter := chi.NewRouter()

	// POST /customers — uses optional auth (claims extracted in handler if present)
	if authMiddleware != nil {
		protected := customerRouter.With(authMiddleware)
		protected.Post("/customers", handler.CreateCustomer)
	} else {
		customerRouter.Post("/customers", handler.CreateCustomer)
	}
	customerRouter.Get("/customers/{email}", handler.GetCustomerByEmailPath)

	// Protected routes — apply auth middleware (if configured)
	if authMiddleware != nil {
		protected := customerRouter.With(authMiddleware)
		protected.Put("/customers", handler.UpdateCustomer)
		protected.Patch("/customers/{id}", handler.PatchCustomer)
	} else {
		customerRouter.Put("/customers", handler.UpdateCustomer)
		customerRouter.Patch("/customers/{id}", handler.PatchCustomer)
	}

	customerRouter.Post("/customers/{id}/addresses", handler.AddAddress)
	customerRouter.Put("/customers/addresses/{addressId}", handler.UpdateAddress)
	customerRouter.Delete("/customers/addresses/{addressId}", handler.DeleteAddress)

	customerRouter.Post("/customers/{id}/credit-cards", handler.AddCreditCard)
	customerRouter.Put("/customers/credit-cards/{cardId}", handler.UpdateCreditCard)
	customerRouter.Delete("/customers/credit-cards/{cardId}", handler.DeleteCreditCard)

	customerRouter.Put("/customers/{id}/default-shipping-address/{addressId}", handler.SetDefaultShippingAddress)
	customerRouter.Put("/customers/{id}/default-billing-address/{addressId}", handler.SetDefaultBillingAddress)
	customerRouter.Delete("/customers/{id}/default-shipping-address", handler.ClearDefaultShippingAddress)
	customerRouter.Delete("/customers/{id}/default-billing-address", handler.ClearDefaultBillingAddress)

	customerRouter.Put("/customers/{id}/default-credit-card/{cardId}", handler.SetDefaultCreditCard)
	customerRouter.Delete("/customers/{id}/default-credit-card", handler.ClearDefaultCreditCard)

	router.Mount("/api/v1", customerRouter)
	serverAddr := "0.0.0.0" + cfg.ServicePort

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:   30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:   120 * time.Second,
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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", logging.ErrorAttr(err))
		}

	close(done)
	logger.Info("Server exited")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
