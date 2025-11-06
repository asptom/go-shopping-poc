package main

import (
	"net/http"
	"os"

	"go-shopping-poc/pkg/config"
	"go-shopping-poc/pkg/cors"
	bus "go-shopping-poc/pkg/eventbus"
	"go-shopping-poc/pkg/logging"
	"go-shopping-poc/pkg/outbox"

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
		logging.Error("Customer:  DATABASE_URL not set")
	}

	logging.Debug("Customer: Connecting to database at %s", dbURL)
	db, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		logging.Error("Customer:  Failed to connect to DB: %v", err)
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

	// Address endpoints
	router.Post("/customers/{id}/addresses", handler.AddAddress)
	router.Put("/customers/addresses/{addressId}", handler.UpdateAddress)
	router.Delete("/customers/addresses/{addressId}", handler.DeleteAddress)

	// Credit card endpoints
	router.Post("/customers/{id}/credit-cards", handler.AddCreditCard)
	router.Put("/customers/credit-cards/{cardId}", handler.UpdateCreditCard)
	router.Delete("/customers/credit-cards/{cardId}", handler.DeleteCreditCard)

	// Start HTTP server (listen on 80; Traefik will terminate TLS)
	serverAddr := cfg.GetCustomerServicePort()
	if serverAddr == "" {
		serverAddr = ":8080"
	}
	logging.Info("Customer:  Starting HTTP server on %s (Traefik will handle TLS)", serverAddr)
	if err := http.ListenAndServe(serverAddr, router); err != nil {
		logging.Error("Customer:  Failed to start HTTP server: %v", err)
	}
	logging.Info("Customer:  Customer service stopped")
}
