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
	bus "go-shopping-poc/internal/platform/event/bus/kafka"
	"go-shopping-poc/internal/platform/event/kafka"
	"go-shopping-poc/internal/platform/outbox"

	"go-shopping-poc/internal/service/customer"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Panic recovered in customer service: %v", r)
		}
	}()
	log.SetFlags(log.LstdFlags)
	log.Printf("[INFO] Customer: Customer service started...")

	// Load service-specific configuration
	cfg, err := customer.LoadConfig()
	if err != nil {
		log.Fatalf("Customer: Failed to load config: %v", err)
	}

	log.Printf("[DEBUG] Customer: Configuration loaded successfully")

	// Connect to Postgres (maintain DATABASE_URL backward compatibility)
	dbURL := cfg.DatabaseURL
	if envURL := os.Getenv("DATABASE_URL"); envURL != "" {
		dbURL = envURL
		log.Printf("[INFO] Customer: Using DATABASE_URL from environment")
	}
	if dbURL == "" {
		log.Fatal("Customer: Database URL not set")
	}

	log.Printf("[DEBUG] Customer: Connecting to database at %s", dbURL)
	db, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		log.Fatalf("Customer: Failed to connect to DB: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("[ERROR] Customer: Error closing database connection: %v", err)
		}
	}()

	// Connect to Kafka
	kafkaCfg, err := kafka.LoadConfig()
	if err != nil {
		log.Fatalf("Customer: Failed to load Kafka config: %v", err)
	}

	kafkaCfg.Topic = cfg.WriteTopic
	kafkaCfg.GroupID = cfg.Group
	bus := bus.NewEventBus(kafkaCfg)

	// Initialize outbox
	log.Printf("[DEBUG] Customer: Initializing outbox")
	outboxCfg, err := outbox.LoadConfig()
	if err != nil {
		log.Fatalf("Customer: Failed to load outbox config: %v", err)
	}

	outboxPublisher := outbox.NewPublisher(db, bus, outboxCfg.BatchSize, outboxCfg.DeleteBatchSize, outboxCfg.ProcessInterval)
	outboxPublisher.Start()
	// log.Printf("[DEBUG] Customer: Outbox publisher started")
	defer outboxPublisher.Stop()
	log.Printf("[DEBUG] Customer: Creating outbox writer")
	// outboxWriter := outbox.NewWriter(db)
	outboxWriter := (*outbox.Writer)(nil)
	// log.Printf("[DEBUG] Customer: Outbox writer created successfully")

	// Create service with dependency injection
	log.Printf("[DEBUG] Customer: Creating customer repository")
	repo := customer.NewCustomerRepository(db, outboxWriter)
	log.Printf("[DEBUG] Customer: Repository created successfully")

	log.Printf("[DEBUG] Customer: Creating customer service")
	service := customer.NewCustomerService(repo, cfg)
	log.Printf("[DEBUG] Customer: Service created successfully")

	log.Printf("[DEBUG] Customer: Creating customer handler")
	handler := customer.NewCustomerHandler(service)
	log.Printf("[DEBUG] Customer: Handler created successfully")

	// Set up router
	log.Printf("[DEBUG] Customer: Setting up HTTP router")
	router := chi.NewRouter()
	log.Printf("[DEBUG] Customer: Router setup completed")

	// Apply CORS middleware using service config (required)
	log.Printf("[DEBUG] Customer: Loading CORS configuration...")
	corsCfg, err := cors.LoadConfig()
	if err != nil {
		log.Fatalf("Customer: Failed to load CORS config: %v", err)
	}
	log.Printf("[DEBUG] Customer: CORS configuration loaded successfully")
	router.Use(cors.NewFromConfig(corsCfg))

	// Health check endpoint
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Define routes
	router.Post("/customers", handler.CreateCustomer)
	router.Get("/customers/{email}", handler.GetCustomerByEmailPath)
	router.Put("/customers", handler.UpdateCustomer)
	router.Patch("/customers/{id}", handler.PatchCustomer)

	// Address endpoints
	router.Post("/customers/{id}/addresses", handler.AddAddress)
	router.Put("/customers/addresses/{addressId}", handler.UpdateAddress)
	router.Delete("/customers/addresses/{addressId}", handler.DeleteAddress)

	// Credit card endpoints
	router.Post("/customers/{id}/credit-cards", handler.AddCreditCard)
	router.Put("/customers/credit-cards/{cardId}", handler.UpdateCreditCard)
	router.Delete("/customers/credit-cards/{cardId}", handler.DeleteCreditCard)

	// Default address endpoints
	router.Put("/customers/{id}/default-shipping-address/{addressId}", handler.SetDefaultShippingAddress)
	router.Put("/customers/{id}/default-billing-address/{addressId}", handler.SetDefaultBillingAddress)
	router.Delete("/customers/{id}/default-shipping-address", handler.ClearDefaultShippingAddress)
	router.Delete("/customers/{id}/default-billing-address", handler.ClearDefaultBillingAddress)

	// Default credit card endpoints
	router.Put("/customers/{id}/default-credit-card/{cardId}", handler.SetDefaultCreditCard)
	router.Delete("/customers/{id}/default-credit-card", handler.ClearDefaultCreditCard)

	// Start HTTP server with graceful shutdown
	serverAddr := "0.0.0.0:8080"

	server := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	// Channel to listen for interrupt signal
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	// Register interrupt signals
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("[INFO] Customer: Starting HTTP server on %s (Traefik will handle TLS)", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Customer: Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Printf("[INFO] Customer: Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Customer: Server forced to shutdown: %v", err)
	}

	close(done)
	log.Printf("[INFO] Customer: Server exited")
}
