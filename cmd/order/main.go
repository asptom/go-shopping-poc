package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-shopping-poc/internal/platform/cors"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/platform/outbox/providers"
	"go-shopping-poc/internal/service/order"
	"go-shopping-poc/internal/service/order/eventhandlers"

	"github.com/go-chi/chi/v5"
)

func main() {

	log.SetFlags(log.LstdFlags)
	log.Printf("[INFO] Order: Order service started...")

	cfg, err := order.LoadConfig()
	if err != nil {
		log.Fatalf("Order: Failed to load config: %v", err)
	}

	log.Printf("[DEBUG] Order: Configuration loaded successfully")
	log.Printf("[DEBUG] Order: Read Topics: %v, Write Topic: %v, Group: %s",
		cfg.ReadTopics, cfg.WriteTopic, cfg.Group)

	// Database setup
	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		log.Fatalf("Order: Database URL is required")
	}

	log.Printf("[DEBUG] Order: Creating database provider")
	dbProvider, err := database.NewDatabaseProvider(dbURL)
	if err != nil {
		log.Fatalf("Order: Failed to create database provider: %v", err)
	}
	db := dbProvider.GetDatabase()
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("[ERROR] Order: Error closing database: %v", err)
		}
	}()

	// Event bus setup
	log.Printf("[DEBUG] Order: Creating event bus provider")
	eventBusConfig := event.EventBusConfig{
		WriteTopic: cfg.WriteTopic,
		GroupID:    cfg.Group,
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
	if err != nil {
		log.Fatalf("Order: Failed to create event bus provider: %v", err)
	}
	eventBus := eventBusProvider.GetEventBus()

	// Outbox setup
	log.Printf("[DEBUG] Order: Creating outbox providers")
	writerProvider := providers.NewWriterProvider(db)
	publisherProvider := providers.NewPublisherProvider(db, eventBus)
	if publisherProvider == nil {
		log.Fatalf("Order: Failed to create publisher provider")
	}
	outboxPublisher := publisherProvider.GetPublisher()
	outboxPublisher.Start()
	defer outboxPublisher.Stop()
	outboxWriter := writerProvider.GetWriter()

	// CORS setup
	log.Printf("[DEBUG] Order: Creating CORS provider")
	corsProvider, err := cors.NewCORSProvider()
	if err != nil {
		log.Fatalf("Order: Failed to create CORS provider: %v", err)
	}
	corsHandler := corsProvider.GetCORSHandler()

	// Infrastructure and service setup
	log.Printf("[DEBUG] Order: Creating order infrastructure")
	infrastructure := order.NewOrderInfrastructure(
		db, eventBus, outboxWriter, outboxPublisher, corsHandler,
	)

	// Service setup
	log.Printf("[DEBUG] Order: Creating order service")
	service := order.NewOrderService(infrastructure, cfg)

	// Register event handlers
	log.Printf("[DEBUG] Order: Registering event handlers")
	if err := registerEventHandlers(service); err != nil {
		log.Fatalf("Order: Failed to register event handlers: %v", err)
		os.Exit(1)
	}

	// Start consuming events from Kafka (in a goroutine so it doesn't block)
	log.Printf("[INFO] Order: Starting event consumer...")
	log.Printf("[INFO] Order: Subscribed to topics: %v", service.EventBus().ReadTopics())
	go func() {
		ctx := context.Background()
		if err := service.Start(ctx); err != nil {
			log.Printf("[ERROR] Order: Event consumer error: %v", err)
		}
	}()

	log.Printf("[DEBUG] Order: Creating order handler")
	handler := order.NewOrderHandler(service)

	log.Printf("[DEBUG] Order: Setting up HTTP router")
	router := chi.NewRouter()
	router.Use(corsHandler)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	orderRouter := chi.NewRouter()
	orderRouter.Get("/orders/{id}", handler.GetOrder)
	orderRouter.Get("/orders", handler.GetOrdersByCustomer)
	orderRouter.Post("/orders/{id}/cancel", handler.CancelOrder)
	orderRouter.Patch("/orders/{id}/status", handler.UpdateOrderStatus)

	router.Mount("/api/v1", orderRouter)

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
		log.Printf("[INFO] Order: Starting HTTP server on %s", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Order: Failed to start HTTP server: %v", err)
		}
	}()

	<-quit
	log.Printf("[INFO] Order: Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Order: Server forced to shutdown: %v", err)
	}

	close(done)
	log.Printf("[INFO] Order: Server exited")
}

func registerEventHandlers(service *order.OrderService) error {
	log.Printf("[INFO] Order: Registering event handlers...")

	cartCheckedOutHandler := eventhandlers.NewOnCartCheckedOut(service)

	log.Printf("[INFO] Order: Registering handler for event type: %s", cartCheckedOutHandler.EventType())

	if err := order.RegisterHandler(
		service,
		cartCheckedOutHandler.CreateFactory(),
		cartCheckedOutHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register CartCheckedOut handler: %w", err)
	}

	log.Printf("[INFO] Order: Successfully registered CartCheckedOut handler")

	log.Printf("[INFO] Order: Event handler registration completed")
	return nil
}
