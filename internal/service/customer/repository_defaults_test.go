/*
Package customer_test provides comprehensive testing for customer repository default field operations.

This file focuses on dedicated default field management endpoints and includes:
- Default address management (shipping/billing)
- Default credit card management
- Service layer integration testing
- Error handling for invalid inputs
- Integration tests for customer creation with relations

Test Categories:
1. Default Address Management - Setting and clearing default shipping/billing addresses
2. Default Credit Card Management - Setting and clearing default credit cards
3. Service Layer Integration - Testing service methods that delegate to repository
4. Error Handling - Invalid UUIDs, non-existent customers
5. Integration Tests - Full customer creation with addresses and credit cards

Important Notes:
- All tests require database setup and use testutils for consistent test data
- Default operations are tested both individually and through service layer
- Error cases verify proper validation and error propagation
*/
package customer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/testutils"
)

// ===== SETUP & HELPERS =====

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) database.Database {
	return testutils.SetupTestDB(t)
}

// createTestCustomer creates a test customer in the database
func createTestCustomer(t *testing.T, db database.Database) string {
	return testutils.CreateTestCustomer(t, db)
}

// createTestAddress creates a test address for a customer
func createTestAddress(t *testing.T, db database.Database, customerID string) string {
	return testutils.CreateTestAddress(t, db, customerID)
}

// createTestCreditCard creates a test credit card for a customer
func createTestCreditCard(t *testing.T, db database.Database, customerID string) string {
	cardID := uuid.New()
	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		t.Fatalf("Invalid customer ID: %v", err)
	}

	query := `INSERT INTO customers.CreditCard (card_id, customer_id, card_type, card_number, card_holder_name, card_expires, card_cvv)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = db.Exec(context.Background(), query, cardID, custUUID, "visa", "4111111111111111", "Test User", "12/25", "123")
	if err != nil {
		t.Fatalf("Failed to create test credit card: %v", err)
	}

	return cardID.String()
}

// ===== DEFAULT ADDRESS MANAGEMENT =====

// TestDefaultShippingAddress_Set verifies setting a default shipping address for a customer.
//
// Business Scenario: Customer selects one of their addresses as the default for shipping
// Expected: Default shipping address ID is stored in customer record
func TestDefaultShippingAddress_Set(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
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
	err = db.GetContext(context.Background(), &defaultAddrID, query, customerID)
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

// TestDefaultBillingAddress_Set verifies setting a default billing address for a customer.
//
// Business Scenario: Customer selects one of their addresses as the default for billing
// Expected: Default billing address ID is stored in customer record
func TestDefaultBillingAddress_Set(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
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
	err = db.GetContext(context.Background(), &defaultAddrID, query, customerID)
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

// ===== DEFAULT CREDIT CARD MANAGEMENT =====

// TestDefaultCreditCard_Set verifies setting a default credit card for a customer.
//
// Business Scenario: Customer selects one of their credit cards as the default for payments
// Expected: Default credit card ID is stored in customer record
func TestDefaultCreditCard_Set(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
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
	err = db.GetContext(context.Background(), &defaultCardID, query, customerID)
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

// TestDefaultShippingAddress_Clear verifies clearing a default shipping address for a customer.
//
// Business Scenario: Customer removes their default shipping address selection
// Expected: Default shipping address ID is set to NULL in customer record
func TestDefaultShippingAddress_Clear(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
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
	err = db.GetContext(context.Background(), &defaultAddrID, query, customerID)
	if err != nil {
		t.Fatalf("Failed to query default shipping address: %v", err)
	}

	if defaultAddrID != nil {
		t.Errorf("Expected default shipping address to be nil, got %s", defaultAddrID.String())
	}
}

// TestDefaultBillingAddress_Clear verifies clearing a default billing address for a customer.
//
// Business Scenario: Customer removes their default billing address selection
// Expected: Default billing address ID is set to NULL in customer record
func TestDefaultBillingAddress_Clear(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
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
	err = db.GetContext(context.Background(), &defaultAddrID, query, customerID)
	if err != nil {
		t.Fatalf("Failed to query default billing address: %v", err)
	}

	if defaultAddrID != nil {
		t.Errorf("Expected default billing address to be nil, got %s", defaultAddrID.String())
	}
}

// TestDefaultCreditCard_Clear verifies clearing a default credit card for a customer.
//
// Business Scenario: Customer removes their default credit card selection
// Expected: Default credit card ID is set to NULL in customer record
func TestDefaultCreditCard_Clear(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
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
	err = db.GetContext(context.Background(), &defaultCardID, query, customerID)
	if err != nil {
		t.Fatalf("Failed to query default credit card: %v", err)
	}

	if defaultCardID != nil {
		t.Errorf("Expected default credit card to be nil, got %s", defaultCardID.String())
	}
}

// ===== SERVICE LAYER INTEGRATION =====

// TestServiceLayer_DefaultMethods verifies that service layer methods correctly delegate to repository.
//
// Business Scenario: Service layer provides high-level API that delegates to repository operations
// Expected: Service methods work correctly and maintain data integrity
func TestServiceLayer_DefaultMethods(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create infrastructure for integration test
	infrastructure := &CustomerInfrastructure{
		Database:        db,
		EventBus:        nil, // Not needed for this test
		OutboxWriter:    &outbox.Writer{},
		OutboxPublisher: nil, // Not needed for this test
		CORSHandler:     nil, // Not needed for this test
	}

	// Create minimal config for testing
	config := &Config{
		DatabaseURL:    "postgres://test:test@localhost:5432/test",
		ServicePort:    ":8080",
		WriteTopic:     "test-topic",
		OutboxInterval: 5 * time.Second,
	}
	service := NewCustomerService(infrastructure, config)
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
	err = db.GetContext(context.Background(), &defaultAddrID, query, customerID)
	if err != nil {
		t.Fatalf("Failed to query default shipping address: %v", err)
	}

	if defaultAddrID == nil || defaultAddrID.String() != addressID {
		t.Errorf("Service layer did not correctly set default shipping address")
	}
}

// ===== ERROR HANDLING =====

// TestDefaultOperations_InvalidUUID verifies error handling for invalid UUID inputs.
//
// Business Scenario: Invalid UUIDs should be rejected with appropriate error messages
// Expected: Operations fail gracefully with validation errors for malformed UUIDs
func TestDefaultOperations_InvalidUUID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
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

// ===== QUERY OPERATIONS =====

// TestGetCustomerByEmail verifies retrieving customers by email address.
//
// Business Scenario: Authentication and customer lookup by email
// Expected: Returns complete customer data for valid email, nil for non-existent email
func TestGetCustomerByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
	ctx := context.Background()

	// Create a test customer
	testEmail := "test.email@example.com"
	customerID := uuid.New()
	query := `INSERT INTO customers.Customer (customer_id, user_name, email, first_name, last_name, phone, customer_since, customer_status, status_date_time)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := db.Exec(context.Background(), query, customerID, "testuser", testEmail, "Test", "User", "555-1234", time.Now(), "active", time.Now())
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

// ===== INTEGRATION TESTS =====

// TestInsertCustomerWithRelations verifies creating a customer with addresses and credit cards.
//
// Business Scenario: New customer registration with complete profile information
// Expected: Customer, addresses, credit cards, and status history all created with proper relationships
func TestInsertCustomerWithRelations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
	ctx := context.Background()

	// Create customer with addresses and credit cards
	customer := &Customer{
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Phone:     "555-1234",
		Addresses: []Address{
			{
				AddressType: "shipping",
				FirstName:   "Test",
				LastName:    "User",
				Address1:    "123 Main St",
				City:        "Test City",
				State:       "TS",
				Zip:         "12345",
			},
		},
		CreditCards: []CreditCard{
			{
				CardType:       "visa",
				CardNumber:     "4111111111111111",
				CardHolderName: "Test User",
				CardExpires:    "12/25",
				CardCVV:        "123",
			},
		},
	}

	// Insert customer
	err := repo.InsertCustomer(ctx, customer)
	if err != nil {
		t.Fatalf("Failed to insert customer with relations: %v", err)
	}

	// Verify customer was created
	if customer.CustomerID == "" {
		t.Error("Customer ID should be set")
	}

	// Verify addresses were created
	if len(customer.Addresses) != 1 {
		t.Errorf("Expected 1 address, got %d", len(customer.Addresses))
	}
	if customer.Addresses[0].AddressID == uuid.Nil {
		t.Error("Address ID should be set")
	}
	if customer.Addresses[0].CustomerID.String() != customer.CustomerID {
		t.Error("Address customer ID should match customer ID")
	}

	// Verify credit cards were created
	if len(customer.CreditCards) != 1 {
		t.Errorf("Expected 1 credit card, got %d", len(customer.CreditCards))
	}
	if customer.CreditCards[0].CardID == uuid.Nil {
		t.Error("Credit card ID should be set")
	}
	if customer.CreditCards[0].CustomerID.String() != customer.CustomerID {
		t.Error("Credit card customer ID should match customer ID")
	}

	// Verify status history was created
	if len(customer.StatusHistory) != 1 {
		t.Errorf("Expected 1 status history record, got %d", len(customer.StatusHistory))
	}
	if customer.StatusHistory[0].NewStatus != "active" {
		t.Errorf("Expected status 'active', got '%s'", customer.StatusHistory[0].NewStatus)
	}

	// Cleanup
	cleanupTestData(t, db, customer.CustomerID)
}

// cleanupTestData removes test data from the database
func cleanupTestData(t *testing.T, db database.Database, customerID string) {
	testutils.CleanupTestData(t, db, customerID)
}

// TestDefaultOperations_NonExistentCustomer verifies error handling for operations on non-existent customers.
//
// Business Scenario: Attempting to set defaults for customers that don't exist
// Expected: Operations fail gracefully with appropriate error messages
func TestDefaultOperations_NonExistentCustomer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
	ctx := context.Background()

	// Test with non-existent customer
	nonExistentID := uuid.New().String()
	addressID := uuid.New().String()

	err := repo.UpdateDefaultShippingAddress(ctx, nonExistentID, addressID)
	if err == nil {
		t.Error("Expected error for non-existent customer")
	}
}
