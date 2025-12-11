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
	bus "go-shopping-poc/internal/platform/eventbus"
	"go-shopping-poc/internal/platform/logging"
	"go-shopping-poc/internal/platform/outbox"

	"go-shopping-poc/internal/service/customer"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func main() {
	logging.SetLevel("INFO")
	logging.Info("Customer:  Customer service started...")

	// Load configuration
	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	logging.Debug("Customer:  Configuration loaded from %s", envFile)
	logging.Debug("Customer:  Config: %v", cfg)

	// Connect to Postgres
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" && cfg.GetCustomerDBURL() != "" {
		dbURL = cfg.GetCustomerDBURL()
	}
	if dbURL == "" {
		log.Fatal("Customer: DATABASE_URL not set")
	}

	logging.Debug("Customer: Connecting to database at %s", dbURL)
	db, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		log.Fatalf("Customer: Failed to connect to DB: %v", err)
	}
	defer db.Close()

	// Connect to Kafka
	broker := cfg.GetEventBroker()
	writeTopic := cfg.GetCustomerWriteTopic()
	readTopics := cfg.GetCustomerReadTopics()
	groupID := cfg.GetCustomerGroup()
	outboxInterval := cfg.GetCustomerOutboxInterval()

	bus := bus.NewEventBus(broker, readTopics, writeTopic, groupID)

	// Initialize
	logging.Debug("Customer:  Initializing outbox reader and writer")
	outboxPublisher := outbox.NewPublisher(db, bus, 1, 1, outboxInterval)
	outboxPublisher.Start()
	logging.Debug("Customer:  Outbox publisher started")
	defer outboxPublisher.Stop()
	outboxWriter := *outbox.NewWriter(db)
	repo := customer.NewCustomerRepository(db, outboxWriter)
	service := customer.NewCustomerService(repo)
	handler := customer.NewCustomerHandler(service)

	// Set up router
	router := chi.NewRouter()
	// Apply CORS middleware
	router.Use(cors.New(cfg))

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
	serverAddr := cfg.GetCustomerServicePort()
	if serverAddr == "" {
		serverAddr = ":8080"
	}

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
		logging.Info("Customer: Starting HTTP server on %s (Traefik will handle TLS)", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Customer: Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	logging.Info("Customer: Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Customer: Server forced to shutdown: %v", err)
	}

	close(done)
	logging.Info("Customer: Server exited")
}
