package main

import (
	"net/http"
	"os"

	"go-shopping-poc/pkg/config"
	"go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
	"go-shopping-poc/pkg/outbox"

	"go-shopping-poc/internal/service/customer"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func main() {
	logging.SetLevel("DEBUG")
	logging.Info("Customer service started...")

	// Load configuration

	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	logging.Info("Configuration loaded from %s", envFile)
	logging.Info("Config: %v", cfg)

	// Connect to Postgres
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" && cfg.GetCustomerDBURL() != "" {
		dbURL = cfg.GetCustomerDBURL()
	}
	if dbURL == "" {
		logging.Error("DATABASE_URL not set")
	}

	logging.Info("Connecting to database at %s", dbURL)
	db, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		logging.Error("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	// Connect to Kafka
	broker := cfg.KafkaBroker
	writeTopics := cfg.GetCustomerKafkaWriteTopics()
	groupID := cfg.GetCustomerKafkaGroupID()

	bus := event.NewKafkaEventBus(broker, nil, writeTopics, groupID)

	// Initialize
	logging.Debug("Initializing outbox reader and writer")
	outboxReader := outbox.NewReader(db, bus, 1, 1)
	outboxReader.Start()
	logging.Debug("Outbox reader started")
	defer outboxReader.Stop()
	outboxWriter := outbox.NewWriter(db)
	repo := customer.NewCustomerRepository(db, outboxWriter)
	service := customer.NewCustomerService(repo)
	handler := customer.NewCustomerHandler(service)

	// Set up router
	router := chi.NewRouter()
	router.Post("/customers", handler.CreateCustomer)
	router.Get("/customers/{id}", handler.GetCustomerByID)

	// Start HTTP server
	serverAddr := cfg.GetCustomerServicePort()
	if serverAddr == "" {
		serverAddr = ":80"
	}
	logging.Info("Starting HTTP server on %s", serverAddr)
	if err := http.ListenAndServe(serverAddr, router); err != nil {
		logging.Error("Failed to start HTTP server: %v", err)
	}
	logging.Info("Customer service stopped")
}
