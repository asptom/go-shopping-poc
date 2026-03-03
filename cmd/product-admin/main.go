package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-shopping-poc/internal/platform/auth"
	"go-shopping-poc/internal/platform/cors"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/downloader"
	"go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/platform/logging"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/outbox/providers"
	"go-shopping-poc/internal/platform/storage"
	"go-shopping-poc/internal/service/product"

	"github.com/go-chi/chi/v5"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			slog.Default().Error("Panic recovered in product-admin service", "panic", r)
		}
	}()

	loggerProvider, err := logging.NewLoggerProvider(logging.DefaultLoggerConfig("product-admin"))
	if err != nil {
		slog.Default().Error("Failed to create logger provider", "error", err.Error())
		os.Exit(1)
	}
	logger := loggerProvider.Logger()

	logger.Info("Product admin service starting")

	cfg, err := product.LoadAdminConfig()
	if err != nil {
		logger.Error("Failed to load config", "error", err.Error())
		os.Exit(1)
	}

	logger.Debug("Configuration loaded successfully")

	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		logger.Error("Database URL is required in service config")
		os.Exit(1)
	}

	logger.Debug("Creating database provider")
	dbProvider, err := database.NewDatabaseProvider(dbURL, database.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to create database provider", "error", err.Error())
		os.Exit(1)
	}
	platformDB := dbProvider.GetDatabase()
	defer func() {
		if err := platformDB.Close(); err != nil {
			logger.Error("Error closing database connection", "error", err.Error())
		}
	}()

	logger.Debug("Creating event bus provider")
	eventBusConfig := event.EventBusConfig{
		WriteTopic: cfg.WriteTopic,
		GroupID:    cfg.Group,
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig, event.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to create event bus provider", "error", err.Error())
		os.Exit(1)
	}
	eventBus := eventBusProvider.GetEventBus()

	logger.Debug("Creating storage provider")
	storageProvider, err := storage.NewStorageProvider(storage.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to create storage provider", "error", err.Error())
		os.Exit(1)
	}
	minioStorage := storageProvider.GetObjectStorage()

	// Validate config to set defaults
	if err := cfg.Validate(); err != nil {
		logger.Error("Config validation failed", "error", err.Error())
		os.Exit(1)
	}

	logger.Debug("Creating outbox providers")
	writerProvider := providers.NewWriterProvider(platformDB, providers.WithWriterLogger(logger))

	// Create outbox publisher with service-specific fast interval for validation events
	outboxConfig := outbox.Config{
		BatchSize:       cfg.OutboxBatchSize,
		ProcessInterval: cfg.OutboxProcessInterval,
	}
	logger.Debug("Outbox publisher configured", "interval", outboxConfig.ProcessInterval, "batch_size", outboxConfig.BatchSize)
	outboxPublisher := outbox.NewPublisher(platformDB, eventBus, outboxConfig, outbox.WithPublisherLogger(logger))
	outboxPublisher.Start()
	defer outboxPublisher.Stop()

	outboxWriter := writerProvider.GetWriter()

	logger.Debug("Creating downloader provider")
	downloaderConfig := downloader.DownloaderProviderConfig{
		CacheDir:     cfg.CacheDir,
		CacheMaxAge:  cfg.CacheMaxAge,
		CacheMaxSize: cfg.CacheMaxSize,
	}
	downloaderProvider, err := downloader.NewDownloaderProvider(downloaderConfig, downloader.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to create downloader provider", "error", err.Error())
		os.Exit(1)
	}
	httpDownloader := downloaderProvider.GetHTTPDownloader()

	logger.Debug("Creating Keycloak validator")
	keycloakValidator := auth.NewKeycloakValidator(cfg.KeycloakIssuer, cfg.KeycloakJWKSURL)

	logger.Debug("Creating admin service")
	adminInfra := &product.AdminInfrastructure{
		Database:       platformDB,
		ObjectStorage:  minioStorage,
		OutboxWriter:   outboxWriter,
		HTTPDownloader: httpDownloader,
	}
	adminService := product.NewAdminService(logger, cfg, adminInfra)
	logger.Debug("Service created successfully")

	logger.Debug("Creating admin handler")
	adminHandler := product.NewAdminHandler(adminService)
	logger.Debug("Handler created successfully")

	logger.Debug("Setting up HTTP router")
	router := chi.NewRouter()
	logger.Debug("Router setup completed")

	logger.Debug("Creating CORS provider")
	corsProvider, err := cors.NewCORSProvider(cors.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to create CORS provider", "error", err.Error())
		os.Exit(1)
	}
	corsHandler := corsProvider.GetCORSHandler()
	router.Use(corsHandler)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	adminRouter := chi.NewRouter()
	adminRouter.Use(auth.RequireAuth(keycloakValidator, "product-admin"))
	adminRouter.Post("/products", adminHandler.CreateProduct)
	adminRouter.Put("/products/{id}", adminHandler.UpdateProduct)
	adminRouter.Delete("/products/{id}", adminHandler.DeleteProduct)
	adminRouter.Post("/products/{id}/images", adminHandler.AddProductImage)
	adminRouter.Put("/products/images/{id}", adminHandler.UpdateProductImage)
	adminRouter.Delete("/products/images/{id}", adminHandler.DeleteProductImage)
	adminRouter.Put("/products/{id}/main-image/{imgId}", adminHandler.SetMainImage)
	adminRouter.Post("/products/ingest", adminHandler.IngestProducts)

	router.Mount("/api/v1/admin", adminRouter)

	serverAddr := "0.0.0.0" + cfg.ServicePort

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Starting HTTP server", "address", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start HTTP server", "error", err.Error())
		}
	}()

	<-quit
	logger.Info("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err.Error())
	}

	close(done)
	logger.Info("Server exited")
}
