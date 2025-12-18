// Package testutils provides shared testing utilities and helpers.
//
// This package contains common test setup functions, data creation helpers,
// and cleanup utilities used across multiple test files to reduce duplication
// and ensure consistent test data management.
package testutils

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"go-shopping-poc/internal/service/eventreader"
)

// SetupTestDB creates a test database connection
func SetupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	// Get current working directory for debugging
	cwd, _ := os.Getwd()
	t.Logf("Test working directory: %s", cwd)

	// Use DATABASE_URL environment variable directly
	// This avoids import cycles with service packages
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping test, DATABASE_URL not set")
	}

	t.Logf("Using DATABASE_URL environment variable: %s", dbURL)

	db, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
	}
	return db
}

// CreateTestCustomer creates a test customer in the database
func CreateTestCustomer(t *testing.T, db *sqlx.DB) string {
	t.Helper()

	customerID := uuid.New()
	query := `INSERT INTO customers.Customer (customer_id, user_name, email, first_name, last_name, phone, customer_since, customer_status, status_date_time)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := db.Exec(query, customerID, "testuser", "test@example.com", "Test", "User", "555-1234", time.Now(), "active", time.Now())
	if err != nil {
		t.Fatalf("Failed to create test customer: %v", err)
	}

	return customerID.String()
}

// CreateTestAddress creates a test address for a customer
func CreateTestAddress(t *testing.T, db *sqlx.DB, customerID string) string {
	t.Helper()

	addressID := uuid.New()
	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		t.Fatalf("Invalid customer ID: %v", err)
	}

	query := `INSERT INTO customers.Address (address_id, customer_id, address_type, first_name, last_name, address_1, city, state, zip)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = db.Exec(query, addressID, custUUID, "shipping", "Test", "User", "123 Main St", "Test City", "TS", "12345")
	if err != nil {
		t.Fatalf("Failed to create test address: %v", err)
	}

	return addressID.String()
}

// CreateTestCreditCard creates a test credit card for a customer
func CreateTestCreditCard(t *testing.T, db *sqlx.DB, customerID string) string {
	t.Helper()

	cardID := uuid.New()
	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		t.Fatalf("Invalid customer ID: %v", err)
	}

	query := `INSERT INTO customers.CreditCard (card_id, customer_id, card_type, card_number, card_holder_name, card_expires, card_cvv)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = db.Exec(query, cardID, custUUID, "visa", "4111111111111111", "Test User", "12/25", "123")
	if err != nil {
		t.Fatalf("Failed to create test credit card: %v", err)
	}

	return cardID.String()
}

// CleanupTestData removes test data from the database
func CleanupTestData(t *testing.T, db *sqlx.DB, customerID string) {
	t.Helper()

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		t.Fatalf("Invalid customer ID for cleanup: %v", err)
	}

	// Clean up in reverse order of dependencies
	queries := []string{
		`DELETE FROM customers.CreditCard WHERE customer_id = $1`,
		`DELETE FROM customers.Address WHERE customer_id = $1`,
		`DELETE FROM customers.CustomerStatusHistory WHERE customer_id = $1`,
		`DELETE FROM customers.Customer WHERE customer_id = $1`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query, custUUID); err != nil {
			t.Logf("Warning: Failed to cleanup test data: %v", err)
		}
	}
}

// SetupTestEnvironment prepares the test environment for integration tests
func SetupTestEnvironment(t *testing.T) {
	t.Helper()

	// Load eventreader configuration for Kafka checks
	cfg, err := eventreader.LoadConfig()
	if err != nil {
		t.Skipf("Skipping integration test: Failed to load eventreader config: %v", err)
	}

	// Check if Kafka configuration is available, skip if not

	if len(cfg.ReadTopics) == 0 {
		t.Skip("Skipping integration test: Event reader topics configuration is missing")
	}

	if cfg.WriteTopic == "" {
		t.Skip("Skipping integration test: Event reader write topic configuration is missing")
	}

	if cfg.Group == "" {
		t.Skip("Skipping integration test: Event reader group configuration is missing")
	}

	t.Log("Test environment setup completed successfully")
}
