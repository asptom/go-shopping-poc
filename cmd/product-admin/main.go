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
	"go-shopping-poc/internal/platform/downloader"
	"go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/platform/outbox/providers"
	"go-shopping-poc/internal/platform/storage"
	"go-shopping-poc/internal/service/product"

	"github.com/go-chi/chi/v5"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Panic recovered in product-admin service: %v", r)
		}
	}()
	log.SetFlags(log.LstdFlags)
	log.Printf("[INFO] Product-Admin: Product admin service started...")

	cfg, err := product.LoadAdminConfig()
	if err != nil {
		log.Fatalf("Product-Admin: Failed to load config: %v", err)
	}

	log.Printf("[DEBUG] Product-Admin: Configuration loaded successfully")

	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		log.Fatalf("Product-Admin: Database URL is required in service config")
	}

	log.Printf("[DEBUG] Product-Admin: Creating database provider")
	dbProvider, err := database.NewDatabaseProvider(dbURL)
	if err != nil {
		log.Fatalf("Product-Admin: Failed to create database provider: %v", err)
	}
	platformDB := dbProvider.GetDatabase()
	defer func() {
		if err := platformDB.Close(); err != nil {
			log.Printf("[ERROR] Product-Admin: Error closing database connection: %v", err)
		}
	}()

	log.Printf("[DEBUG] Product-Admin: Creating event bus provider")
	eventBusConfig := event.EventBusConfig{
		WriteTopic: cfg.WriteTopic,
		GroupID:    cfg.Group,
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
	if err != nil {
		log.Fatalf("Product-Admin: Failed to create event bus provider: %v", err)
	}
	eventBus := eventBusProvider.GetEventBus()

	log.Printf("[DEBUG] Product-Admin: Creating storage provider")
	storageProvider, err := storage.NewStorageProvider()
	if err != nil {
		log.Fatalf("Product-Admin: Failed to create storage provider: %v", err)
	}
	minioStorage := storageProvider.GetObjectStorage()

	log.Printf("[DEBUG] Product-Admin: Creating outbox providers")
	writerProvider := providers.NewWriterProvider(platformDB)
	publisherProvider := providers.NewPublisherProvider(platformDB, eventBus)
	if publisherProvider == nil {
		log.Fatalf("Product-Admin: Failed to create publisher provider")
	}
	outboxPublisher := publisherProvider.GetPublisher()
	outboxPublisher.Start()
	defer outboxPublisher.Stop()
	outboxWriter := writerProvider.GetWriter()

	log.Printf("[DEBUG] Product-Admin: Creating downloader provider")
	downloaderConfig := downloader.DownloaderProviderConfig{
		CacheDir:     cfg.CacheDir,
		CacheMaxAge:  cfg.CacheMaxAge,
		CacheMaxSize: cfg.CacheMaxSize,
	}
	downloaderProvider, err := downloader.NewDownloaderProvider(downloaderConfig)
	if err != nil {
		log.Fatalf("Product-Admin: Failed to create downloader provider: %v", err)
	}
	httpDownloader := downloaderProvider.GetHTTPDownloader()

	log.Printf("[DEBUG] Product-Admin: Creating Keycloak validator")
	keycloakValidator := auth.NewKeycloakValidator(cfg.KeycloakIssuer, cfg.KeycloakJWKSURL)

	log.Printf("[DEBUG] Product-Admin: Creating product repository")
	repo := product.NewProductRepository(platformDB.DB())
	log.Printf("[DEBUG] Product-Admin: Repository created successfully")

	log.Printf("[DEBUG] Product-Admin: Creating admin service")
	adminInfra := &product.AdminInfrastructure{
		Database:       platformDB,
		ObjectStorage:  minioStorage,
		OutboxWriter:   outboxWriter,
		HTTPDownloader: httpDownloader,
	}
	adminService := product.NewAdminService(repo, cfg, adminInfra)
	log.Printf("[DEBUG] Product-Admin: Service created successfully")

	log.Printf("[DEBUG] Product-Admin: Creating admin handler")
	adminHandler := product.NewAdminHandler(adminService)
	log.Printf("[DEBUG] Product-Admin: Handler created successfully")

	log.Printf("[DEBUG] Product-Admin: Setting up HTTP router")
	router := chi.NewRouter()
	log.Printf("[DEBUG] Product-Admin: Router setup completed")

	log.Printf("[DEBUG] Product-Admin: Creating CORS provider")
	corsProvider, err := cors.NewCORSProvider()
	if err != nil {
		log.Fatalf("Product-Admin: Failed to create CORS provider: %v", err)
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
		log.Printf("[INFO] Product-Admin: Starting HTTP server on %s", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Product-Admin: Failed to start HTTP server: %v", err)
		}
	}()

	<-quit
	log.Printf("[INFO] Product-Admin: Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Product-Admin: Server forced to shutdown: %v", err)
	}

	close(done)
	log.Printf("[INFO] Product-Admin: Server exited")
}
