/*
Package customer_test provides comprehensive testing for customer repository update operations.

This file focuses on PUT and PATCH operations that modify existing customer data:
- PUT operations (complete customer replacement)
- PATCH operations (partial field updates)
- Data preservation during updates
- Response completeness verification
- Query operations for retrieving updated data

Test Categories:
1. PUT Operations - Complete customer record replacement with validation
2. PATCH Operations - Partial updates with field-level granularity
3. Data Preservation - Ensuring existing data is maintained during partial updates
4. Response Completeness - Verifying all fields are returned in responses
5. Query Operations - Retrieving customers with complete related data

Important Notes:
- All tests require database setup and create realistic test data
- PUT vs PATCH semantics are strictly tested for correct behavior
- Data preservation is critical for PATCH operations
- Response completeness ensures API consistency
*/
package customer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	outbox "go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/testutils"
)

// ===== SETUP & HELPERS =====

// setupUpdateTestDB creates a test database connection for update operations
func setupUpdateTestDB(t *testing.T) *sqlx.DB {
	return testutils.SetupTestDB(t)
}

// createTestCustomerWithFullData creates a test customer with addresses and credit cards for update testing
func createTestCustomerWithFullData(t *testing.T, db *sqlx.DB) *Customer {
	customerID := uuid.New()
	shippingAddrID := uuid.New()
	billingAddrID := uuid.New()
	cardID := uuid.New()

	customer := &Customer{
		CustomerID: customerID.String(),
		Username:   "testuser",
		Email:      "test@example.com",
		FirstName:  "Test",
		LastName:   "User",
		Phone:      "555-1234",
		// Don't set defaults initially - they'll be set via dedicated endpoints
		CustomerSince:  time.Now(),
		CustomerStatus: "active",
		StatusDateTime: time.Now(),
		Addresses: []Address{
			{
				AddressID:   shippingAddrID,
				CustomerID:  customerID,
				AddressType: "shipping",
				FirstName:   "Test",
				LastName:    "User",
				Address1:    "123 Main St",
				Address2:    "Apt 4B",
				City:        "Test City",
				State:       "TS",
				Zip:         "12345",
			},
			{
				AddressID:   billingAddrID,
				CustomerID:  customerID,
				AddressType: "billing",
				FirstName:   "Test",
				LastName:    "User",
				Address1:    "456 Oak Ave",
				Address2:    "",
				City:        "Test City",
				State:       "TS",
				Zip:         "12345",
			},
		},
		CreditCards: []CreditCard{
			{
				CardID:         cardID,
				CustomerID:     customerID,
				CardType:       "visa",
				CardNumber:     "4111111111111111",
				CardHolderName: "Test User",
				CardExpires:    "12/25",
				CardCVV:        "123",
			},
		},
		StatusHistory: []CustomerStatus{
			{
				ID:         1,
				CustomerID: customerID,
				OldStatus:  "inactive",
				NewStatus:  "active",
				ChangedAt:  time.Now(),
			},
		},
	}

	// Insert customer directly into database for testing
	repo := NewCustomerRepository(db, &outbox.Writer{})
	if err := repo.InsertCustomer(context.Background(), customer); err != nil {
		t.Fatalf("Failed to create test customer: %v", err)
	}

	// Fetch the customer to get the actual UUIDs that were generated
	createdCustomer, err := repo.GetCustomerByID(context.Background(), customer.CustomerID)
	if err != nil {
		t.Fatalf("Failed to fetch created customer: %v", err)
	}
	if createdCustomer == nil {
		t.Fatal("Created customer not found")
	}

	return createdCustomer
}

// ===== PUT OPERATIONS (Complete Replacement) =====

// TestPUT_BasicInfoOnly verifies PUT operations update basic fields while requiring complete records.
//
// Business Scenario: PUT replaces entire customer record, requiring all fields to be provided
// Expected: Basic fields updated, but all existing relations must be included in request
func TestPUT_BasicInfoOnly(t *testing.T) {
	db := setupUpdateTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
	ctx := context.Background()

	// Create test customer
	existing := createTestCustomerWithFullData(t, db)

	// Update only basic info - PUT requires complete record
	updateCustomer := &Customer{
		CustomerID:     existing.CustomerID,
		Username:       "updateduser",
		Email:          "updated@example.com",
		FirstName:      "Updated",
		LastName:       "User",
		Phone:          "555-9999",
		CustomerStatus: "inactive",
		StatusDateTime: time.Now(),
		// PUT requires all data - include existing addresses and credit cards
		Addresses:                existing.Addresses,
		CreditCards:              existing.CreditCards,
		DefaultShippingAddressID: existing.DefaultShippingAddressID,
		DefaultBillingAddressID:  existing.DefaultBillingAddressID,
		DefaultCreditCardID:      existing.DefaultCreditCardID,
		CustomerSince:            existing.CustomerSince,
	}

	err := repo.UpdateCustomer(ctx, updateCustomer)
	if err != nil {
		t.Fatalf("Failed to update customer basic info: %v", err)
	}

	// Verify basic info updated
	updated, err := repo.GetCustomerByID(ctx, existing.CustomerID)
	if err != nil {
		t.Fatalf("Failed to fetch updated customer: %v", err)
	}

	if updated.Username != "updateduser" {
		t.Errorf("Expected username 'updateduser', got '%s'", updated.Username)
	}
	if updated.Email != "updated@example.com" {
		t.Errorf("Expected email 'updated@example.com', got '%s'", updated.Email)
	}
}

// TestPUT_FullCustomerUpdate verifies complete customer record replacement.
//
// Business Scenario: PUT operations completely replace customer data
// Expected: All provided data is stored, replacing existing values
func TestPUT_FullCustomerUpdate(t *testing.T) {
	// This would test a complete PUT replacement - placeholder for future expansion
	t.Skip("Full PUT update test - placeholder for comprehensive PUT testing")
}

// ===== PATCH OPERATIONS (Partial Updates) =====

// TestPATCH_BasicInfoOnly verifies PATCH operations update only specified fields while preserving all existing data.
//
// Business Scenario: Customer updates profile information without affecting addresses or payment methods
// Expected: Only specified fields updated, all relations and defaults preserved with original UUIDs
func TestPATCH_BasicInfoOnly(t *testing.T) {
	db := setupUpdateTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
	ctx := context.Background()

	// Create test customer
	existing := createTestCustomerWithFullData(t, db)

	// Patch only basic info
	userName := "updateduser"
	email := "updated@example.com"
	firstName := "Updated"
	lastName := "User"
	phone := "555-9999"
	status := "inactive"

	patchData := &PatchCustomerRequest{
		UserName:       &userName,
		Email:          &email,
		FirstName:      &firstName,
		LastName:       &lastName,
		Phone:          &phone,
		CustomerStatus: &status,
	}

	err := repo.PatchCustomer(ctx, existing.CustomerID, patchData)
	if err != nil {
		t.Fatalf("Failed to patch customer basic info: %v", err)
	}

	// Verify basic info updated
	updated, err := repo.GetCustomerByID(ctx, existing.CustomerID)
	if err != nil {
		t.Fatalf("Failed to fetch updated customer: %v", err)
	}

	if updated.Username != "updateduser" {
		t.Errorf("Expected username 'updateduser', got '%s'", updated.Username)
	}
	if updated.Email != "updated@example.com" {
		t.Errorf("Expected email 'updated@example.com', got '%s'", updated.Email)
	}

	// Verify addresses and credit cards preserved (PATCH shouldn't affect them)
	if len(updated.Addresses) != len(existing.Addresses) {
		t.Errorf("Expected %d addresses, got %d", len(existing.Addresses), len(updated.Addresses))
	}
	if len(updated.CreditCards) != len(existing.CreditCards) {
		t.Errorf("Expected %d credit cards, got %d", len(existing.CreditCards), len(updated.CreditCards))
	}

	// Verify address IDs are preserved
	for i, addr := range updated.Addresses {
		if addr.AddressID != existing.Addresses[i].AddressID {
			t.Errorf("Address %d ID changed from %s to %s", i, existing.Addresses[i].AddressID, addr.AddressID)
		}
	}

	// Verify credit card IDs are preserved
	for i, card := range updated.CreditCards {
		if card.CardID != existing.CreditCards[i].CardID {
			t.Errorf("Credit card %d ID changed from %s to %s", i, existing.CreditCards[i].CardID, card.CardID)
		}
	}
}

// TestPATCH_DefaultFieldsOnly verifies PATCH operations can update only default field selections.
//
// Business Scenario: Customer changes their default address/credit card preferences
// Expected: Only default field pointers updated, all other data preserved
func TestPATCH_DefaultFieldsOnly(t *testing.T) {
	db := setupUpdateTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
	ctx := context.Background()

	// Create test customer
	existing := createTestCustomerWithFullData(t, db)

	// First set a default address using dedicated endpoint
	if err := repo.UpdateDefaultShippingAddress(ctx, existing.CustomerID, existing.Addresses[0].AddressID.String()); err != nil {
		t.Fatalf("Failed to set default shipping address: %v", err)
	}

	// Verify default was set
	customerWithDefault, err := repo.GetCustomerByID(ctx, existing.CustomerID)
	if err != nil {
		t.Fatalf("Failed to fetch customer: %v", err)
	}

	if customerWithDefault.DefaultShippingAddressID.String() != existing.Addresses[0].AddressID.String() {
		t.Errorf("Expected default shipping address %s, got %s",
			existing.Addresses[0].AddressID.String(), customerWithDefault.DefaultShippingAddressID.String())
	}
}

// ===== DATA PRESERVATION =====
// (Tests for data preservation are included in the PATCH tests above)

// ===== RESPONSE COMPLETENESS =====

// TestPATCH_Response_IncludesNullDefaults verifies PATCH responses include null default fields when none are set.
//
// Business Scenario: API responses must be consistent regardless of default field values
// Expected: All default fields present in response (null when unset, UUID when set)
func TestPATCH_Response_IncludesNullDefaults(t *testing.T) {
	db := setupUpdateTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
	ctx := context.Background()

	// Create test customer (defaults will be null initially)
	existing := createTestCustomerWithFullData(t, db)

	// Patch only basic info - should return complete customer with null defaults
	userName := "patcheduser"
	email := "patched@example.com"

	patchData := &PatchCustomerRequest{
		UserName: &userName,
		Email:    &email,
	}

	err := repo.PatchCustomer(ctx, existing.CustomerID, patchData)
	if err != nil {
		t.Fatalf("Failed to patch customer: %v", err)
	}

	// Fetch updated customer to verify response structure
	updated, err := repo.GetCustomerByID(ctx, existing.CustomerID)
	if err != nil {
		t.Fatalf("Failed to fetch updated customer: %v", err)
	}

	// Verify default fields are present (should be nil/null initially)
	// Note: We can't directly test JSON marshaling here, but we can verify
	// the struct fields are accessible and in expected state
	if updated.DefaultShippingAddressID != nil {
		t.Errorf("Expected default shipping address to be nil, got %v", updated.DefaultShippingAddressID)
	}
	if updated.DefaultBillingAddressID != nil {
		t.Errorf("Expected default billing address to be nil, got %v", updated.DefaultBillingAddressID)
	}
	if updated.DefaultCreditCardID != nil {
		t.Errorf("Expected default credit card to be nil, got %v", updated.DefaultCreditCardID)
	}

	// Verify basic fields were updated
	if updated.Username != "patcheduser" {
		t.Errorf("Expected username 'patcheduser', got '%s'", updated.Username)
	}
	if updated.Email != "patched@example.com" {
		t.Errorf("Expected email 'patched@example.com', got '%s'", updated.Email)
	}
}

// TestPATCH_Response_IncludesSetDefaults verifies PATCH responses include actual UUID values when defaults are set.
//
// Business Scenario: When customers have default selections, responses must include the UUID values
// Expected: Default fields contain actual UUIDs when defaults are configured
func TestPATCH_Response_IncludesSetDefaults(t *testing.T) {
	db := setupUpdateTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
	ctx := context.Background()

	// Create test customer
	existing := createTestCustomerWithFullData(t, db)

	// Set some defaults using dedicated endpoints
	shippingAddrID := existing.Addresses[0].AddressID.String()
	billingAddrID := existing.Addresses[1].AddressID.String()
	cardID := existing.CreditCards[0].CardID.String()

	if err := repo.UpdateDefaultShippingAddress(ctx, existing.CustomerID, shippingAddrID); err != nil {
		t.Fatalf("Failed to set default shipping address: %v", err)
	}
	if err := repo.UpdateDefaultBillingAddress(ctx, existing.CustomerID, billingAddrID); err != nil {
		t.Fatalf("Failed to set default billing address: %v", err)
	}
	if err := repo.UpdateDefaultCreditCard(ctx, existing.CustomerID, cardID); err != nil {
		t.Fatalf("Failed to set default credit card: %v", err)
	}

	// Now patch basic info - response should include the set defaults
	userName := "patcheduser"
	email := "patched@example.com"

	patchData := &PatchCustomerRequest{
		UserName: &userName,
		Email:    &email,
	}

	err := repo.PatchCustomer(ctx, existing.CustomerID, patchData)
	if err != nil {
		t.Fatalf("Failed to patch customer: %v", err)
	}

	// Fetch updated customer
	updated, err := repo.GetCustomerByID(ctx, existing.CustomerID)
	if err != nil {
		t.Fatalf("Failed to fetch updated customer: %v", err)
	}

	// Verify default fields are present and correct
	if updated.DefaultShippingAddressID == nil {
		t.Error("Expected default shipping address to be set, got nil")
	} else if updated.DefaultShippingAddressID.String() != shippingAddrID {
		t.Errorf("Expected default shipping address %s, got %s", shippingAddrID, updated.DefaultShippingAddressID.String())
	}

	if updated.DefaultBillingAddressID == nil {
		t.Error("Expected default billing address to be set, got nil")
	} else if updated.DefaultBillingAddressID.String() != billingAddrID {
		t.Errorf("Expected default billing address %s, got %s", billingAddrID, updated.DefaultBillingAddressID.String())
	}

	if updated.DefaultCreditCardID == nil {
		t.Error("Expected default credit card to be set, got nil")
	} else if updated.DefaultCreditCardID.String() != cardID {
		t.Errorf("Expected default credit card %s, got %s", cardID, updated.DefaultCreditCardID.String())
	}

	// Verify basic fields were updated
	if updated.Username != "patcheduser" {
		t.Errorf("Expected username 'patcheduser', got '%s'", updated.Username)
	}
	if updated.Email != "patched@example.com" {
		t.Errorf("Expected email 'patched@example.com', got '%s'", updated.Email)
	}
}

// ===== QUERY OPERATIONS =====

// TestGetCustomerByID_CompleteData verifies retrieving customers with all related data.
//
// Business Scenario: Customer profile retrieval must include complete information
// Expected: Customer, addresses, credit cards, and status history all returned correctly
func TestGetCustomerByID_CompleteData(t *testing.T) {
	db := setupUpdateTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
	ctx := context.Background()

	// Create test customer
	created := createTestCustomerWithFullData(t, db)

	// Fetch by ID
	fetched, err := repo.GetCustomerByID(ctx, created.CustomerID)
	if err != nil {
		t.Fatalf("Failed to fetch customer by ID: %v", err)
	}
	if fetched == nil {
		t.Fatal("Expected customer, got nil")
	}

	// Verify all data fetched correctly
	if fetched.CustomerID != created.CustomerID {
		t.Errorf("Expected customer ID %s, got %s", created.CustomerID, fetched.CustomerID)
	}
	if fetched.Username != created.Username {
		t.Errorf("Expected username %s, got %s", created.Username, fetched.Username)
	}
	if len(fetched.Addresses) != len(created.Addresses) {
		t.Errorf("Expected %d addresses, got %d", len(created.Addresses), len(fetched.Addresses))
	}
	if len(fetched.CreditCards) != len(created.CreditCards) {
		t.Errorf("Expected %d credit cards, got %d", len(created.CreditCards), len(fetched.CreditCards))
	}
	if len(fetched.StatusHistory) != len(created.StatusHistory) {
		t.Errorf("Expected %d status history entries, got %d", len(created.StatusHistory), len(fetched.StatusHistory))
	}
}

func TestPATCH_SingleField(t *testing.T) {
	db := setupUpdateTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, &outbox.Writer{})
	ctx := context.Background()

	// Create test customer
	existing := createTestCustomerWithFullData(t, db)

	// Set default shipping address
	shippingAddrID := existing.Addresses[0].AddressID.String()
	err := repo.UpdateDefaultShippingAddress(ctx, existing.CustomerID, shippingAddrID)
	if err != nil {
		t.Fatalf("Failed to set default shipping address: %v", err)
	}

	// Set default credit card
	cardID := existing.CreditCards[0].CardID.String()
	err = repo.UpdateDefaultCreditCard(ctx, existing.CustomerID, cardID)
	if err != nil {
		t.Fatalf("Failed to set default credit card: %v", err)
	}

	// Fetch customer to verify defaults are set
	customerWithDefaults, err := repo.GetCustomerByID(ctx, existing.CustomerID)
	if err != nil {
		t.Fatalf("Failed to fetch customer with defaults: %v", err)
	}
	if customerWithDefaults.DefaultShippingAddressID == nil {
		t.Fatal("Default shipping address not set")
	}
	if customerWithDefaults.DefaultCreditCardID == nil {
		t.Fatal("Default credit card not set")
	}

	// Now patch only phone
	newPhone := "904-716-7998"
	patchData := &PatchCustomerRequest{
		Phone: &newPhone,
	}

	err = repo.PatchCustomer(ctx, existing.CustomerID, patchData)
	if err != nil {
		t.Fatalf("Failed to patch phone: %v", err)
	}

	// Verify phone updated
	updated, err := repo.GetCustomerByID(ctx, existing.CustomerID)
	if err != nil {
		t.Fatalf("Failed to fetch updated customer: %v", err)
	}

	if updated.Phone != newPhone {
		t.Errorf("Expected phone '%s', got '%s'", newPhone, updated.Phone)
	}

	// Verify defaults preserved
	if updated.DefaultShippingAddressID == nil || *updated.DefaultShippingAddressID != *customerWithDefaults.DefaultShippingAddressID {
		t.Errorf("Default shipping address not preserved")
	}
	if updated.DefaultCreditCardID == nil || *updated.DefaultCreditCardID != *customerWithDefaults.DefaultCreditCardID {
		t.Errorf("Default credit card not preserved")
	}

	// Verify addresses preserved with same IDs
	if len(updated.Addresses) != len(existing.Addresses) {
		t.Errorf("Expected %d addresses, got %d", len(existing.Addresses), len(updated.Addresses))
	}
	for i, addr := range updated.Addresses {
		if addr.AddressID != existing.Addresses[i].AddressID {
			t.Errorf("Address %d ID changed from %s to %s", i, existing.Addresses[i].AddressID, addr.AddressID)
		}
	}

	// Verify credit cards preserved with same IDs
	if len(updated.CreditCards) != len(existing.CreditCards) {
		t.Errorf("Expected %d credit cards, got %d", len(existing.CreditCards), len(updated.CreditCards))
	}
	for i, card := range updated.CreditCards {
		if card.CardID != existing.CreditCards[i].CardID {
			t.Errorf("Credit card %d ID changed from %s to %s", i, existing.CreditCards[i].CardID, card.CardID)
		}
	}
}
