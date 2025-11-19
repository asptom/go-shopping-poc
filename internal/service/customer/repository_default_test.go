package customer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"go-shopping-poc/pkg/config"
	outbox "go-shopping-poc/pkg/outbox"
)

// Load configuration
func initConfig() *config.Config {
	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)
	return cfg
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sqlx.DB {
	// Use environment variable or default test database
	cfg := initConfig()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" && cfg.GetCustomerDBURLLocal() != "" {
		dbURL = cfg.GetCustomerDBURLLocal()
	}
	db, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
	}
	return db
}

// createTestCustomer creates a test customer in the database
func createTestCustomer(t *testing.T, db *sqlx.DB) string {
	customerID := uuid.New()
	query := `INSERT INTO customers.Customer (customer_id, user_name, email, first_name, last_name, phone, customer_since, customer_status, status_date_time) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := db.Exec(query, customerID, "testuser", "test@example.com", "Test", "User", "555-1234", time.Now(), "active", time.Now())
	if err != nil {
		t.Fatalf("Failed to create test customer: %v", err)
	}

	return customerID.String()
}

// createTestAddress creates a test address for a customer
func createTestAddress(t *testing.T, db *sqlx.DB, customerID string) string {
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

// createTestCreditCard creates a test credit card for a customer
func createTestCreditCard(t *testing.T, db *sqlx.DB, customerID string) string {
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

func TestUpdateDefaultShippingAddress(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create mock outbox writer (nil for testing)
	repo := NewCustomerRepository(db, outbox.Writer{})
	ctx := context.Background()

	// Create test customer and address
	customerID := createTestCustomer(t, db)
	addressID := createTestAddress(t, db, customerID)

	// Test setting default shipping address
	err := repo.UpdateDefaultShippingAddress(ctx, customerID, addressID)
	if err != nil {
		t.Fatalf("Failed to set default shipping address: %v", err)
	}

	// Verify the default was set
	var defaultAddrID *uuid.UUID
	query := `SELECT default_shipping_address_id FROM customers.Customer WHERE customer_id = $1`
	err = db.Get(&defaultAddrID, query, customerID)
	if err != nil {
		t.Fatalf("Failed to query default shipping address: %v", err)
	}

	if defaultAddrID == nil {
		t.Error("Default shipping address was not set")
		return
	}

	if defaultAddrID.String() != addressID {
		t.Errorf("Expected default shipping address %s, got %s", addressID, defaultAddrID.String())
	}
}

func TestUpdateDefaultBillingAddress(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
	ctx := context.Background()

	// Create test customer and address
	customerID := createTestCustomer(t, db)
	addressID := createTestAddress(t, db, customerID)

	// Test setting default billing address
	err := repo.UpdateDefaultBillingAddress(ctx, customerID, addressID)
	if err != nil {
		t.Fatalf("Failed to set default billing address: %v", err)
	}

	// Verify the default was set
	var defaultAddrID *uuid.UUID
	query := `SELECT default_billing_address_id FROM customers.Customer WHERE customer_id = $1`
	err = db.Get(&defaultAddrID, query, customerID)
	if err != nil {
		t.Fatalf("Failed to query default billing address: %v", err)
	}

	if defaultAddrID == nil {
		t.Error("Default billing address was not set")
		return
	}

	if defaultAddrID.String() != addressID {
		t.Errorf("Expected default billing address %s, got %s", addressID, defaultAddrID.String())
	}
}

func TestUpdateDefaultCreditCard(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
	ctx := context.Background()

	// Create test customer and credit card
	customerID := createTestCustomer(t, db)
	cardID := createTestCreditCard(t, db, customerID)

	// Test setting default credit card
	err := repo.UpdateDefaultCreditCard(ctx, customerID, cardID)
	if err != nil {
		t.Fatalf("Failed to set default credit card: %v", err)
	}

	// Verify the default was set
	var defaultCardID *uuid.UUID
	query := `SELECT default_credit_card_id FROM customers.Customer WHERE customer_id = $1`
	err = db.Get(&defaultCardID, query, customerID)
	if err != nil {
		t.Fatalf("Failed to query default credit card: %v", err)
	}

	if defaultCardID == nil {
		t.Error("Default credit card was not set")
		return
	}

	if defaultCardID.String() != cardID {
		t.Errorf("Expected default credit card %s, got %s", cardID, defaultCardID.String())
	}
}

func TestClearDefaultShippingAddress(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
	ctx := context.Background()

	// Create test customer and address
	customerID := createTestCustomer(t, db)
	addressID := createTestAddress(t, db, customerID)

	// First set a default
	err := repo.UpdateDefaultShippingAddress(ctx, customerID, addressID)
	if err != nil {
		t.Fatalf("Failed to set default shipping address: %v", err)
	}

	// Then clear it
	err = repo.ClearDefaultShippingAddress(ctx, customerID)
	if err != nil {
		t.Fatalf("Failed to clear default shipping address: %v", err)
	}

	// Verify the default was cleared
	var defaultAddrID *uuid.UUID
	query := `SELECT default_shipping_address_id FROM customers.Customer WHERE customer_id = $1`
	err = db.Get(&defaultAddrID, query, customerID)
	if err != nil {
		t.Fatalf("Failed to query default shipping address: %v", err)
	}

	if defaultAddrID != nil {
		t.Errorf("Expected default shipping address to be nil, got %s", defaultAddrID.String())
	}
}

func TestClearDefaultBillingAddress(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
	ctx := context.Background()

	// Create test customer and address
	customerID := createTestCustomer(t, db)
	addressID := createTestAddress(t, db, customerID)

	// First set a default
	err := repo.UpdateDefaultBillingAddress(ctx, customerID, addressID)
	if err != nil {
		t.Fatalf("Failed to set default billing address: %v", err)
	}

	// Then clear it
	err = repo.ClearDefaultBillingAddress(ctx, customerID)
	if err != nil {
		t.Fatalf("Failed to clear default billing address: %v", err)
	}

	// Verify the default was cleared
	var defaultAddrID *uuid.UUID
	query := `SELECT default_billing_address_id FROM customers.Customer WHERE customer_id = $1`
	err = db.Get(&defaultAddrID, query, customerID)
	if err != nil {
		t.Fatalf("Failed to query default billing address: %v", err)
	}

	if defaultAddrID != nil {
		t.Errorf("Expected default billing address to be nil, got %s", defaultAddrID.String())
	}
}

func TestClearDefaultCreditCard(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
	ctx := context.Background()

	// Create test customer and credit card
	customerID := createTestCustomer(t, db)
	cardID := createTestCreditCard(t, db, customerID)

	// First set a default
	err := repo.UpdateDefaultCreditCard(ctx, customerID, cardID)
	if err != nil {
		t.Fatalf("Failed to set default credit card: %v", err)
	}

	// Then clear it
	err = repo.ClearDefaultCreditCard(ctx, customerID)
	if err != nil {
		t.Fatalf("Failed to clear default credit card: %v", err)
	}

	// Verify the default was cleared
	var defaultCardID *uuid.UUID
	query := `SELECT default_credit_card_id FROM customers.Customer WHERE customer_id = $1`
	err = db.Get(&defaultCardID, query, customerID)
	if err != nil {
		t.Fatalf("Failed to query default credit card: %v", err)
	}

	if defaultCardID != nil {
		t.Errorf("Expected default credit card to be nil, got %s", defaultCardID.String())
	}
}

func TestServiceLayerDefaultMethods(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
	service := NewCustomerService(repo)
	ctx := context.Background()

	// Create test customer and address
	customerID := createTestCustomer(t, db)
	addressID := createTestAddress(t, db, customerID)

	// Test service layer method
	err := service.SetDefaultShippingAddress(ctx, customerID, addressID)
	if err != nil {
		t.Fatalf("Service failed to set default shipping address: %v", err)
	}

	// Verify through repository
	var defaultAddrID *uuid.UUID
	query := `SELECT default_shipping_address_id FROM customers.Customer WHERE customer_id = $1`
	err = db.Get(&defaultAddrID, query, customerID)
	if err != nil {
		t.Fatalf("Failed to query default shipping address: %v", err)
	}

	if defaultAddrID == nil || defaultAddrID.String() != addressID {
		t.Errorf("Service layer did not correctly set default shipping address")
	}
}

func TestInvalidUUIDHandling(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
	ctx := context.Background()

	// Test with invalid customer ID
	err := repo.UpdateDefaultShippingAddress(ctx, "invalid-uuid", uuid.New().String())
	if err == nil {
		t.Error("Expected error for invalid customer UUID")
	}

	// Test with invalid address ID
	customerID := createTestCustomer(t, db)
	err = repo.UpdateDefaultShippingAddress(ctx, customerID, "invalid-uuid")
	if err == nil {
		t.Error("Expected error for invalid address UUID")
	}
}

func TestGetCustomerByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
	ctx := context.Background()

	// Create a test customer
	testEmail := "test.email@example.com"
	customerID := uuid.New()
	query := `INSERT INTO customers.Customer (customer_id, user_name, email, first_name, last_name, phone, customer_since, customer_status, status_date_time)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := db.Exec(query, customerID, "testuser", testEmail, "Test", "User", "555-1234", time.Now(), "active", time.Now())
	if err != nil {
		t.Fatalf("Failed to create test customer: %v", err)
	}

	// Test successful retrieval by email
	customer, err := repo.GetCustomerByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("Failed to get customer by email: %v", err)
	}
	if customer == nil {
		t.Fatal("Expected customer, got nil")
	}
	if customer.Email != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, customer.Email)
	}
	if customer.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", customer.Username)
	}

	// Test with non-existent email
	nonExistentCustomer, err := repo.GetCustomerByEmail(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("Unexpected error for non-existent email: %v", err)
	}
	if nonExistentCustomer != nil {
		t.Error("Expected nil for non-existent email, got customer")
	}
}

func TestNonExistentCustomer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
	ctx := context.Background()

	// Test with non-existent customer
	nonExistentID := uuid.New().String()
	addressID := uuid.New().String()

	err := repo.UpdateDefaultShippingAddress(ctx, nonExistentID, addressID)
	if err == nil {
		t.Error("Expected error for non-existent customer")
	}
}
