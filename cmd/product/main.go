package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-shopping-poc/internal/platform/cors"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/downloader"
	"go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/storage"

	"go-shopping-poc/internal/service/product"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Panic recovered in product service: %v", r)
		}
	}()
	log.SetFlags(log.LstdFlags)
	log.Printf("[INFO] Product: Product service started...")

	// Load service-specific configuration
	cfg, err := product.LoadConfig()
	if err != nil {
		log.Fatalf("Product: Failed to load config: %v", err)
	}

	log.Printf("[DEBUG] Product: Configuration loaded successfully")

	// Get database URL from service config
	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		log.Fatalf("Product: Database URL is required in service config")
	}

	// Create database provider
	log.Printf("[DEBUG] Product: Creating database provider")
	dbProvider, err := database.NewDatabaseProvider(dbURL)
	if err != nil {
		log.Fatalf("Product: Failed to create database provider: %v", err)
	}
	platformDB := dbProvider.GetDatabase()
	defer func() {
		if err := platformDB.Close(); err != nil {
			log.Printf("[ERROR] Product: Error closing database connection: %v", err)
		}
	}()

	// Create event bus provider
	log.Printf("[DEBUG] Product: Creating event bus provider")
	eventBusConfig := event.EventBusConfig{
		WriteTopic: "product-events",  // Service-specific topic
		GroupID:    "product-service", // Service-specific group
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
	if err != nil {
		log.Fatalf("Product: Failed to create event bus provider: %v", err)
	}
	eventBus := eventBusProvider.GetEventBus()

	// Create storage provider
	log.Printf("[DEBUG] Product: Creating storage provider")
	storageProvider, err := storage.NewStorageProvider()
	if err != nil {
		log.Fatalf("Product: Failed to create storage provider: %v", err)
	}
	minioStorage := storageProvider.GetObjectStorage()

	// Create outbox provider
	log.Printf("[DEBUG] Product: Creating outbox provider")
	outboxProvider, err := outbox.NewOutboxProvider(platformDB, eventBus)
	if err != nil {
		log.Fatalf("Product: Failed to create outbox provider: %v", err)
	}
	outboxPublisher := outboxProvider.GetOutboxPublisher()
	outboxPublisher.Start()
	defer outboxPublisher.Stop()
	outboxWriter := outboxProvider.GetOutboxWriter()

	// Create service with dependency injection
	log.Printf("[DEBUG] Product: Creating product repository")
	repo := product.NewProductRepository(platformDB.DB())
	log.Printf("[DEBUG] Product: Repository created successfully")

	// Create downloader provider
	log.Printf("[DEBUG] Product: Creating downloader provider")
	downloaderConfig := downloader.DownloaderProviderConfig{
		CacheDir:     cfg.CacheDir,
		CacheMaxAge:  cfg.CacheMaxAge,
		CacheMaxSize: cfg.CacheMaxSize,
	}
	downloaderProvider, err := downloader.NewDownloaderProvider(downloaderConfig)
	if err != nil {
		log.Fatalf("Product: Failed to create downloader provider: %v", err)
	}
	httpDownloader := downloaderProvider.GetHTTPDownloader()

	// Create infrastructure struct
	infrastructure := &product.ProductInfrastructure{
		Database:       platformDB,
		ObjectStorage:  minioStorage,
		OutboxWriter:   outboxWriter,
		HTTPDownloader: httpDownloader,
	}

	service := product.NewProductService(repo, cfg, infrastructure)
	log.Printf("[DEBUG] Product: Service created successfully")

	log.Printf("[DEBUG] Product: Creating product handler")
	handler := product.NewProductHandler(service)
	log.Printf("[DEBUG] Product: Handler created successfully")

	// Set up router
	log.Printf("[DEBUG] Product: Setting up HTTP router")
	router := chi.NewRouter()
	log.Printf("[DEBUG] Product: Router setup completed")

	// Create CORS provider
	log.Printf("[DEBUG] Product: Creating CORS provider")
	corsProvider, err := cors.NewCORSProvider()
	if err != nil {
		log.Fatalf("Product: Failed to create CORS provider: %v", err)
	}
	corsHandler := corsProvider.GetCORSHandler()
	router.Use(corsHandler)

	// Health check endpoint
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Define routes
	router.Post("/products", handler.CreateProduct)
	router.Get("/products/{id}", handler.GetProduct)
	router.Put("/products/{id}", handler.UpdateProduct)
	router.Delete("/products/{id}", handler.DeleteProduct)

	// Search and filter endpoints
	router.Get("/products/search", handler.SearchProducts)
	router.Get("/products/category/{category}", handler.GetProductsByCategory)
	router.Get("/products/brand/{brand}", handler.GetProductsByBrand)
	router.Get("/products/in-stock", handler.GetProductsInStock)

	// Ingestion endpoint
	router.Post("/products/ingest", handler.IngestProducts)

	// Start HTTP server with graceful shutdown
	serverAddr := "0.0.0.0:" + cfg.ServicePort

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Channel to listen for interrupt signal
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	// Register interrupt signals
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("[INFO] Product: Starting HTTP server on %s (Traefik will handle TLS)", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Product: Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Printf("[INFO] Product: Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Product: Server forced to shutdown: %v", err)
	}

	close(done)
	log.Printf("[INFO] Product: Server exited")
}
