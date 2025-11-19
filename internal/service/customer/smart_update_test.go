package customer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	entity "go-shopping-poc/internal/entity/customer"
	outbox "go-shopping-poc/pkg/outbox"
	"go-shopping-poc/pkg/testutils"
)

// setupSmartUpdateTestDB creates a test database connection
func setupSmartUpdateTestDB(t *testing.T) *sqlx.DB {
	return testutils.SetupTestDB(t)
}

// createTestCustomerWithFullData creates a test customer with addresses and credit cards
func createTestCustomerWithFullData(t *testing.T, db *sqlx.DB) *entity.Customer {
	customerID := uuid.New()
	shippingAddrID := uuid.New()
	billingAddrID := uuid.New()
	cardID := uuid.New()

	customer := &entity.Customer{
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
		Addresses: []entity.Address{
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
		CreditCards: []entity.CreditCard{
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
		StatusHistory: []entity.CustomerStatus{
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
	repo := NewCustomerRepository(db, outbox.Writer{})
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

func TestUpdateCustomer_BasicInfoOnly(t *testing.T) {
	db := setupSmartUpdateTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
	ctx := context.Background()

	// Create test customer
	existing := createTestCustomerWithFullData(t, db)

	// Update only basic info - PUT requires complete record
	updateCustomer := &entity.Customer{
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

func TestPatchCustomer_BasicInfoOnly(t *testing.T) {
	db := setupSmartUpdateTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
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
}

func TestPatchCustomer_UpdateDefault(t *testing.T) {
	db := setupSmartUpdateTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
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

func TestGetCustomerByID(t *testing.T) {
	db := setupSmartUpdateTestDB(t)
	defer db.Close()

	repo := NewCustomerRepository(db, outbox.Writer{})
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
