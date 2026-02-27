package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-shopping-poc/internal/platform/config"
	"go-shopping-poc/internal/platform/cors"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/platform/logging"
	"go-shopping-poc/internal/platform/outbox/providers"
	"go-shopping-poc/internal/platform/storage/minio"
	"go-shopping-poc/internal/service/product"
	"go-shopping-poc/internal/service/product/eventhandlers"

	"github.com/go-chi/chi/v5"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			slog.Default().Error("Panic recovered in product service", "panic", r)
		}
	}()

	loggerProvider, err := logging.NewLoggerProvider(logging.DefaultLoggerConfig("product"))
	if err != nil {
		log.Fatalf("Product: Failed to create logger provider: %v", err)
	}
	logger := loggerProvider.Logger()

	logger.Info("Product catalog service starting")

	cfg, err := product.LoadConfig()
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

	logger.Debug("Creating outbox writer provider")
	writerProvider := providers.NewWriterProvider(platformDB, providers.WithWriterLogger(logger))

	// Event bus setup for consuming cart validation requests
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

	// Create outbox publisher for immediate event processing
	logger.Debug("Creating outbox publisher")
	publisherProvider := providers.NewPublisherProvider(platformDB, eventBus, providers.WithPublisherLogger(logger))
	outboxPublisher := publisherProvider.GetPublisher()
	outboxPublisher.Start()
	defer outboxPublisher.Stop()

	logger.Debug("Creating catalog service")
	catalogInfra := &product.CatalogInfrastructure{
		Database:        platformDB,
		OutboxWriter:    writerProvider.GetWriter(),
		OutboxPublisher: outboxPublisher,
		EventBus:        eventBus,
	}
	catalogService := product.NewCatalogService(logger, catalogInfra, cfg)
	logger.Debug("Service created successfully")

	// Register event handlers
	logger.Debug("Registering event handlers")
	cartItemAddedHandler := eventhandlers.NewOnCartItemAdded(catalogService, logger)
	if err := product.RegisterHandler(
		catalogService,
		cartItemAddedHandler.CreateFactory(),
		cartItemAddedHandler.CreateHandler(),
	); err != nil {
		logger.Error("Failed to register CartItemAdded handler", "error", err.Error())
		os.Exit(1)
	}
	logger.Debug("Event handlers registered successfully")

	// Start event consumer in background
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()

	go func() {
		if err := catalogService.Start(consumerCtx); err != nil {
			logger.Error("Event consumer stopped", "error", err.Error())
		}
	}()

	logger.Debug("Loading MinIO configuration")
	minioCfg, err := config.LoadConfig[minio.PlatformConfig]("platform-minio")
	if err != nil {
		logger.Error("Failed to load MinIO config", "error", err.Error())
		os.Exit(1)
	}

	logger.Debug("Creating MinIO storage client")
	// Choose MinIO endpoint based on environment
	minioEndpoint := minioCfg.EndpointLocal
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		minioEndpoint = minioCfg.EndpointKubernetes
		logger.Debug("Using Kubernetes MinIO endpoint for connections", "endpoint", minioEndpoint)
	} else {
		logger.Debug("Using local MinIO endpoint", "endpoint", minioEndpoint)
	}

	minioStorage, err := minio.NewClient(&minio.Config{
		Endpoint:  minioEndpoint,
		AccessKey: minioCfg.AccessKey,
		SecretKey: minioCfg.SecretKey,
		Secure:    minioCfg.TLSVerify,
	})
	if err != nil {
		logger.Error("Failed to create MinIO storage", "error", err.Error())
		os.Exit(1)
	}
	logger.Debug("MinIO storage initialized")

	logger.Debug("Creating catalog handler")
	catalogHandler := product.NewCatalogHandler(logger, catalogService, minioStorage, cfg.MinIOBucket)
	logger.Debug("Handler created successfully")

	logger.Debug("Setting up HTTP router")
	router := chi.NewRouter()
	logger.Debug("Router setup completed")

	logger.Debug("Creating CORS provider")
	corsProvider, err := cors.NewCORSProvider()
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

	productRouter := chi.NewRouter()
	productRouter.Get("/products", catalogHandler.GetAllProducts)
	productRouter.Get("/products/{id}", catalogHandler.GetProduct)
	productRouter.Get("/products/search", catalogHandler.SearchProducts)
	productRouter.Get("/products/category/{category}", catalogHandler.GetProductsByCategory)
	productRouter.Get("/products/brand/{brand}", catalogHandler.GetProductsByBrand)
	productRouter.Get("/products/in-stock", catalogHandler.GetProductsInStock)
	productRouter.Get("/products/{id}/images", catalogHandler.GetProductImages)
	productRouter.Get("/products/{id}/main-image", catalogHandler.GetProductMainImage)
	// Direct image access: /api/v1/products/{id}/images/{imageName:.+}
	// Example: /api/v1/products/40121298/images/image_0.jpg
	productRouter.Get("/products/{id}/images/{imageName:.+}", catalogHandler.GetDirectImage)
	logger.Debug("Registered /products/{id}/images/{imageName:.+} route for direct image access")

	router.Mount("/api/v1", productRouter)

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
