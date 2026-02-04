package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-shopping-poc/internal/platform/config"
	"go-shopping-poc/internal/platform/cors"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/outbox/providers"
	"go-shopping-poc/internal/platform/storage/minio"
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

	log.Printf("[DEBUG] Product: Creating outbox writer provider")
	writerProvider := providers.NewWriterProvider(platformDB)

	log.Printf("[DEBUG] Product: Creating catalog service")
	catalogInfra := &product.CatalogInfrastructure{
		Database:     platformDB,
		OutboxWriter: writerProvider.GetWriter(),
	}
	catalogService := product.NewCatalogService(catalogInfra, cfg)
	log.Printf("[DEBUG] Product: Service created successfully")

	log.Printf("[DEBUG] Product: Loading MinIO configuration")
	minioCfg, err := config.LoadConfig[minio.PlatformConfig]("platform-minio")
	if err != nil {
		log.Fatalf("Product: Failed to load MinIO config: %v", err)
	}

	log.Printf("[DEBUG] Product: Creating MinIO storage client")
	// Choose MinIO endpoint based on environment
	minioEndpoint := minioCfg.EndpointLocal
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		minioEndpoint = minioCfg.EndpointKubernetes
		log.Printf("[DEBUG] Product: Using Kubernetes MinIO endpoint for connections: %s", minioEndpoint)
	} else {
		log.Printf("[DEBUG] Product: Using local MinIO endpoint: %s", minioEndpoint)
	}

	minioStorage, err := minio.NewClient(&minio.Config{
		Endpoint:  minioEndpoint,
		AccessKey: minioCfg.AccessKey,
		SecretKey: minioCfg.SecretKey,
		Secure:    minioCfg.TLSVerify,
	})
	if err != nil {
		log.Fatalf("Product: Failed to create MinIO storage: %v", err)
	}
	log.Printf("[DEBUG] Product: MinIO storage initialized")

	log.Printf("[DEBUG] Product: Creating catalog handler")
	catalogHandler := product.NewCatalogHandler(catalogService, minioStorage, cfg.MinIOBucket)
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
	productRouter.Get("/products/{id}/images", catalogHandler.GetProductImages)
	productRouter.Get("/products/{id}/main-image", catalogHandler.GetProductMainImage)
	// Direct image access: /api/v1/products/{id}/images/{imageName:.+}
	// Example: /api/v1/products/40121298/images/image_0.jpg
	productRouter.Get("/products/{id}/images/{imageName:.+}", catalogHandler.GetDirectImage)
	log.Printf("[DEBUG] Product: Registered /products/{id}/images/{imageName:.+} route for direct image access")

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
