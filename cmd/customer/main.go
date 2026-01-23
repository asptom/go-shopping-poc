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

	"go-shopping-poc/internal/service/customer"

	"github.com/go-chi/chi/v5"
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

	// Get database URL from service config
	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		log.Fatalf("Customer: Database URL is required in service config")
	}

	// Create database provider
	log.Printf("[DEBUG] Customer: Creating database provider")
	dbProvider, err := database.NewDatabaseProvider(dbURL)
	if err != nil {
		log.Fatalf("Customer: Failed to create database provider: %v", err)
	}
	db := dbProvider.GetDatabase()
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("[ERROR] Customer: Error closing database connection: %v", err)
		}
	}()

	// Create event bus provider
	log.Printf("[DEBUG] Customer: Creating event bus provider")
	eventBusConfig := event.EventBusConfig{
		WriteTopic: cfg.WriteTopic,
		GroupID:    cfg.Group,
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
	if err != nil {
		log.Fatalf("Customer: Failed to create event bus provider: %v", err)
	}
	eventBus := eventBusProvider.GetEventBus()

	// Create outbox provider
	log.Printf("[DEBUG] Customer: Creating outbox provider")
	outboxProvider, err := outbox.NewOutboxProvider(db, eventBus)
	if err != nil {
		log.Fatalf("Customer: Failed to create outbox provider: %v", err)
	}
	outboxPublisher := outboxProvider.GetOutboxPublisher()
	outboxPublisher.Start()
	defer outboxPublisher.Stop()
	outboxWriter := outboxProvider.GetOutboxWriter()

	// Create CORS provider
	log.Printf("[DEBUG] Customer: Creating CORS provider")
	corsProvider, err := cors.NewCORSProvider()
	if err != nil {
		log.Fatalf("Customer: Failed to create CORS provider: %v", err)
	}
	corsHandler := corsProvider.GetCORSHandler()

	// Create customer infrastructure
	log.Printf("[DEBUG] Customer: Creating customer infrastructure")
	infrastructure := customer.NewCustomerInfrastructure(db, eventBus, outboxWriter, outboxPublisher, corsHandler)
	log.Printf("[DEBUG] Customer: Infrastructure created successfully")

	log.Printf("[DEBUG] Customer: Creating customer service")
	service := customer.NewCustomerService(infrastructure, cfg)
	log.Printf("[DEBUG] Customer: Service created successfully")

	log.Printf("[DEBUG] Customer: Creating customer handler")
	handler := customer.NewCustomerHandler(service)
	log.Printf("[DEBUG] Customer: Handler created successfully")

	// Set up router
	log.Printf("[DEBUG] Customer: Setting up HTTP router")
	router := chi.NewRouter()
	log.Printf("[DEBUG] Customer: Router setup completed")

	// Apply CORS middleware using service infrastructure
	router.Use(corsHandler)

	// Health check endpoint
	router.Get("/health", healthHandler)

	// Define routes
	customerRouter := chi.NewRouter()
	customerRouter.Post("/customers", handler.CreateCustomer)
	customerRouter.Get("/customers/{email}", handler.GetCustomerByEmailPath)
	customerRouter.Put("/customers", handler.UpdateCustomer)
	customerRouter.Patch("/customers/{id}", handler.PatchCustomer)

	// Address endpoints
	customerRouter.Post("/customers/{id}/addresses", handler.AddAddress)
	customerRouter.Put("/customers/addresses/{addressId}", handler.UpdateAddress)
	customerRouter.Delete("/customers/addresses/{addressId}", handler.DeleteAddress)

	// Credit card endpoints
	customerRouter.Post("/customers/{id}/credit-cards", handler.AddCreditCard)
	customerRouter.Put("/customers/credit-cards/{cardId}", handler.UpdateCreditCard)
	customerRouter.Delete("/customers/credit-cards/{cardId}", handler.DeleteCreditCard)

	// Default address endpoints
	customerRouter.Put("/customers/{id}/default-shipping-address/{addressId}", handler.SetDefaultShippingAddress)
	customerRouter.Put("/customers/{id}/default-billing-address/{addressId}", handler.SetDefaultBillingAddress)
	customerRouter.Delete("/customers/{id}/default-shipping-address", handler.ClearDefaultShippingAddress)
	customerRouter.Delete("/customers/{id}/default-billing-address", handler.ClearDefaultBillingAddress)

	// Default credit card endpoints
	customerRouter.Put("/customers/{id}/default-credit-card/{cardId}", handler.SetDefaultCreditCard)
	customerRouter.Delete("/customers/{id}/default-credit-card", handler.ClearDefaultCreditCard)

	router.Mount("/api/v1", customerRouter)
	// Start HTTP server with graceful shutdown
	serverAddr := "0.0.0.0" + cfg.ServicePort

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
		log.Printf("[INFO] Customer: Starting HTTP server on %s (Traefik will handle TLS)", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Customer: Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Printf("[INFO] Customer: Shutting down server...")

	// Create a deadline for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Customer: Server forced to shutdown: %v", err)
	}

	close(done)
	log.Printf("[INFO] Customer: Server exited")
}

// healthHandler returns a simple health check response
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
