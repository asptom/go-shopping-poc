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
	"go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/service/product"

	"github.com/go-chi/chi/v5"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Panic recovered in product service: %v", r)
		}
	}()
	log.SetFlags(log.LstdFlags)
	log.Printf("[INFO] Product: Product catalog service started...")

	cfg, err := product.LoadConfig()
	if err != nil {
		log.Fatalf("Product: Failed to load config: %v", err)
	}

	log.Printf("[DEBUG] Product: Configuration loaded successfully")

	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		log.Fatalf("Product: Database URL is required in service config")
	}

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

	log.Printf("[DEBUG] Product: Creating event bus provider")
	eventBusConfig := event.EventBusConfig{
		WriteTopic: "ProductEvents",
		GroupID:    "ProductCatalogGroup",
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
	if err != nil {
		log.Fatalf("Product: Failed to create event bus provider: %v", err)
	}
	eventBus := eventBusProvider.GetEventBus()

	log.Printf("[DEBUG] Product: Creating outbox provider")
	outboxProvider, err := outbox.NewOutboxProvider(platformDB, eventBus)
	if err != nil {
		log.Fatalf("Product: Failed to create outbox provider: %v", err)
	}
	outboxPublisher := outboxProvider.GetOutboxPublisher()
	outboxPublisher.Start()
	defer outboxPublisher.Stop()

	log.Printf("[DEBUG] Product: Creating product repository")
	repo := product.NewProductRepository(platformDB.DB())
	log.Printf("[DEBUG] Product: Repository created successfully")

	log.Printf("[DEBUG] Product: Creating catalog service")
	catalogInfra := &product.CatalogInfrastructure{
		Database:     platformDB,
		OutboxWriter: outbox.NewWriter(platformDB),
	}
	catalogService := product.NewCatalogService(repo, catalogInfra, cfg)
	log.Printf("[DEBUG] Product: Service created successfully")

	log.Printf("[DEBUG] Product: Creating catalog handler")
	catalogHandler := product.NewCatalogHandler(catalogService)
	log.Printf("[DEBUG] Product: Handler created successfully")

	log.Printf("[DEBUG] Product: Setting up HTTP router")
	router := chi.NewRouter()
	log.Printf("[DEBUG] Product: Router setup completed")

	log.Printf("[DEBUG] Product: Creating CORS provider")
	corsProvider, err := cors.NewCORSProvider()
	if err != nil {
		log.Fatalf("Product: Failed to create CORS provider: %v", err)
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
		log.Printf("[INFO] Product: Starting HTTP server on %s", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Product: Failed to start HTTP server: %v", err)
		}
	}()

	<-quit
	log.Printf("[INFO] Product: Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Product: Server forced to shutdown: %v", err)
	}

	close(done)
	log.Printf("[INFO] Product: Server exited")
}
