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

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/cors"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/platform/outbox/providers"
	"go-shopping-poc/internal/platform/sse"
	"go-shopping-poc/internal/service/cart"
	"go-shopping-poc/internal/service/cart/eventhandlers"

	"github.com/go-chi/chi/v5"
)

func main() {

	log.SetFlags(log.LstdFlags)
	log.Printf("[INFO] Cart: Cart service started...")

	cfg, err := cart.LoadConfig()
	if err != nil {
		log.Fatalf("Cart: Failed to load config: %v", err)
	}

	log.Printf("[DEBUG] Cart: Configuration loaded successfully")
	log.Printf("[DEBUG] Cart: Read Topics: %v, Write Topic: %v, Group: %s",
		cfg.ReadTopics, cfg.WriteTopic, cfg.Group)

	// Database setup
	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		log.Fatalf("Cart: Database URL is required")
	}

	log.Printf("[DEBUG] Cart: Creating database provider")
	dbProvider, err := database.NewDatabaseProvider(dbURL)
	if err != nil {
		log.Fatalf("Cart: Failed to create database provider: %v", err)
	}
	db := dbProvider.GetDatabase()
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("[ERROR] Cart: Error closing database: %v", err)
		}
	}()

	//Event bus setup

	log.Printf("[DEBUG] Cart: Creating event bus provider")
	eventBusConfig := event.EventBusConfig{
		WriteTopic: cfg.WriteTopic,
		GroupID:    cfg.Group,
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
	if err != nil {
		log.Fatalf("Cart: Failed to create event bus provider: %v", err)
	}
	eventBus := eventBusProvider.GetEventBus()

	// Outbox setup
	log.Printf("[DEBUG] Cart: Creating outbox providers")
	writerProvider := providers.NewWriterProvider(db)
	publisherProvider := providers.NewPublisherProvider(db, eventBus)
	if publisherProvider == nil {
		log.Fatalf("Cart: Failed to create publisher provider")
	}
	outboxPublisher := publisherProvider.GetPublisher()
	outboxPublisher.Start()
	defer outboxPublisher.Stop()
	outboxWriter := writerProvider.GetWriter()

	// CORS setup
	log.Printf("[DEBUG] Cart: Creating CORS provider")
	corsProvider, err := cors.NewCORSProvider()
	if err != nil {
		log.Fatalf("Cart: Failed to create CORS provider: %v", err)
	}
	corsHandler := corsProvider.GetCORSHandler()

	// Product service client setup
	productServiceURL := cfg.ProductServiceURL
	if productServiceURL == "" {
		productServiceURL = "http://product-svc.shopping.svc.cluster.local:80"
	}
	log.Printf("[DEBUG] Cart: Product service URL: %s", productServiceURL)
	productClient := cart.NewProductClient(productServiceURL)

	// SSE provider setup
	log.Printf("[DEBUG] Cart: Creating SSE provider")
	sseProvider := sse.NewProvider()

	// Infrastructure and service setup
	log.Printf("[DEBUG] Cart: Creating cart infrastructure")
	infrastructure := cart.NewCartInfrastructure(
		db, eventBus, outboxWriter, outboxPublisher, productClient, corsHandler, sseProvider,
	)

	// Service setup
	log.Printf("[DEBUG] Cart: Creating cart service")
	service := cart.NewCartService(infrastructure, cfg)

	// Register event handlers with validation

	log.Printf("[DEBUG] Cart: Registering event handlers")
	if err := registerEventHandlers(service, sseProvider.GetHub()); err != nil {
		log.Fatalf("Cart: Failed to register event handlers: %v", err)
		os.Exit(1)
	}

	log.Printf("[DEBUG] Cart: Creating cart handler")
	handler := cart.NewCartHandler(service)

	log.Printf("[DEBUG] Cart: Setting up HTTP router")
	router := chi.NewRouter()
	router.Use(corsHandler)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	cartRouter := chi.NewRouter()
	cartRouter.Post("/carts", handler.CreateCart)
	cartRouter.Get("/carts/{id}", handler.GetCart)
	cartRouter.Delete("/carts/{id}", handler.DeleteCart)

	cartRouter.Post("/carts/{id}/items", handler.AddItem)
	cartRouter.Put("/carts/{id}/items/{line}", handler.UpdateItem)
	cartRouter.Delete("/carts/{id}/items/{line}", handler.RemoveItem)

	cartRouter.Put("/carts/{id}/contact", handler.SetContact)

	cartRouter.Post("/carts/{id}/addresses", handler.AddAddress)

	cartRouter.Put("/carts/{id}/payment", handler.SetPayment)

	cartRouter.Post("/carts/{id}/checkout", handler.Checkout)

	// SSE route for real-time order updates
	cartRouter.Get("/carts/{id}/stream", sseProvider.GetHandler().ServeHTTP)

	router.Mount("/api/v1", cartRouter)

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
		log.Printf("[INFO] Cart: Starting HTTP server on %s", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Cart: Failed to start HTTP server: %v", err)
		}
	}()

	<-quit
	log.Printf("[INFO] Cart: Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Cart: Server forced to shutdown: %v", err)
	}

	close(done)
	log.Printf("[INFO] Cart: Server exited")
}

// registerEventHandlers registers all event handlers with the service and validates registration
func registerEventHandlers(service *cart.CartService, sseHub *sse.Hub) error {
	log.Printf("[INFO] Cart: Registering event handlers...")

	// Register OrderCreated handler using the clean generic method
	orderCreatedHandler := eventhandlers.NewOnOrderCreated(sseHub)

	// Log handler registration details
	log.Printf("[INFO] Cart: Registering handler for event type: %s", orderCreatedHandler.EventType())
	log.Printf("[INFO] Cart: Handler will process events from topic: %s", events.OrderEvent{}.Topic())

	if err := cart.RegisterHandler(
		service,
		orderCreatedHandler.CreateFactory(),
		orderCreatedHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register OrderCreated handler: %w", err)
	}

	log.Printf("[INFO] Cart: Successfully registered OrderCreated handler")

	// Future handlers can be registered here using the same generic method:
	// customerUpdatedHandler := eventhandlers.NewOnCustomerUpdated()
	// if err := eventreader.RegisterHandler(
	//     service,
	//     customerUpdatedHandler.CreateFactory(),
	//     customerUpdatedHandler.CreateHandler(),
	// ); err != nil {
	//     return fmt.Errorf("failed to register CustomerUpdated handler: %w", err)
	// }

	log.Printf("[INFO] Cart: Event handler registration completed")
	return nil
}
