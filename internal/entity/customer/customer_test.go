package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCustomerEntityMatchesSchema(t *testing.T) {
	// Test that Customer entity has all required fields from schema
	customerID := uuid.New().String()
	now := time.Now()
	shippingAddrID := uuid.New()
	billingAddrID := uuid.New()
	cardID := uuid.New()

	customer := &Customer{
		CustomerID:               customerID,
		Username:                 "testuser",
		Email:                    "test@example.com",
		FirstName:                "Test",
		LastName:                 "User",
		Phone:                    "555-1234",
		DefaultShippingAddressID: &shippingAddrID,
		DefaultBillingAddressID:  &billingAddrID,
		DefaultCreditCardID:      &cardID,
		CustomerSince:            now,
		CustomerStatus:           "active",
		StatusDateTime:           now,
		Addresses:                []Address{},
		CreditCards:              []CreditCard{},
		StatusHistory:            []CustomerStatus{},
	}

	// Verify all fields are set correctly
	if customer.CustomerID != customerID {
		t.Errorf("Expected CustomerID %s, got %s", customerID, customer.CustomerID)
	}
	if customer.CustomerStatus != "active" {
		t.Errorf("Expected CustomerStatus 'active', got %s", customer.CustomerStatus)
	}
	if customer.DefaultShippingAddressID == nil {
		t.Error("DefaultShippingAddressID should not be nil")
	} else if *customer.DefaultShippingAddressID != shippingAddrID {
		t.Errorf("Expected DefaultShippingAddressID %s, got %s", shippingAddrID, *customer.DefaultShippingAddressID)
	}
	if customer.DefaultBillingAddressID == nil {
		t.Error("DefaultBillingAddressID should not be nil")
	} else if *customer.DefaultBillingAddressID != billingAddrID {
		t.Errorf("Expected DefaultBillingAddressID %s, got %s", billingAddrID, *customer.DefaultBillingAddressID)
	}
	if customer.DefaultCreditCardID == nil {
		t.Error("DefaultCreditCardID should not be nil")
	} else if *customer.DefaultCreditCardID != cardID {
		t.Errorf("Expected DefaultCreditCardID %s, got %s", cardID, *customer.DefaultCreditCardID)
	}
}

func TestCustomerEntityNullableFields(t *testing.T) {
	// Test that nullable UUID fields can be nil (representing NULL in database)
	customerID := uuid.New().String()
	now := time.Now()

	customer := &Customer{
		CustomerID: customerID,
		Username:   "testuser",
		Email:      "test@example.com",
		FirstName:  "Test",
		LastName:   "User",
		Phone:      "555-1234",
		// Nullable fields set to nil to represent NULL values
		DefaultShippingAddressID: nil,
		DefaultBillingAddressID:  nil,
		DefaultCreditCardID:      nil,
		CustomerSince:            now,
		CustomerStatus:           "active",
		StatusDateTime:           now,
		Addresses:                []Address{},
		CreditCards:              []CreditCard{},
		StatusHistory:            []CustomerStatus{},
	}

	// Verify nullable fields can be nil
	if customer.DefaultShippingAddressID != nil {
		t.Error("DefaultShippingAddressID should be nil")
	}
	if customer.DefaultBillingAddressID != nil {
		t.Error("DefaultBillingAddressID should be nil")
	}
	if customer.DefaultCreditCardID != nil {
		t.Error("DefaultCreditCardID should be nil")
	}
}

func TestAddressEntityMatchesSchema(t *testing.T) {
	customerID := uuid.New()
	addressID := uuid.New()

	address := &Address{
		AddressID:   addressID,
		CustomerID:  customerID,
		AddressType: "shipping",
		FirstName:   "Test",
		LastName:    "User",
		Address1:    "123 Main St",
		Address2:    "Apt 4",
		City:        "Test City",
		State:       "TS",
		Zip:         "12345",
	}

	// Verify all fields are set correctly
	if address.AddressID != addressID {
		t.Errorf("Expected AddressID %s, got %s", addressID, address.AddressID)
	}
	if address.AddressType != "shipping" {
		t.Errorf("Expected AddressType 'shipping', got %s", address.AddressType)
	}
}

func TestCreditCardEntityMatchesSchema(t *testing.T) {
	customerID := uuid.New()
	cardID := uuid.New()

	card := &CreditCard{
		CardID:         cardID,
		CustomerID:     customerID,
		CardType:       "visa",
		CardNumber:     "4111111111111111",
		CardHolderName: "Test User",
		CardExpires:    "12/25",
		CardCVV:        "123",
	}

	// Verify all fields are set correctly
	if card.CardID != cardID {
		t.Errorf("Expected CardID %s, got %s", cardID, card.CardID)
	}
	if card.CardType != "visa" {
		t.Errorf("Expected CardType 'visa', got %s", card.CardType)
	}
}

func TestCustomerStatusEntityMatchesSchema(t *testing.T) {
	customerID := uuid.New()
	now := time.Now()

	status := &CustomerStatus{
		ID:         1,
		CustomerID: customerID,
		OldStatus:  "inactive",
		NewStatus:  "active",
		ChangedAt:  now,
	}

	// Verify all fields are set correctly
	if status.CustomerID != customerID {
		t.Errorf("Expected CustomerID %s, got %s", customerID, status.CustomerID)
	}
	if status.OldStatus != "inactive" {
		t.Errorf("Expected OldStatus 'inactive', got %s", status.OldStatus)
	}
	if status.NewStatus != "active" {
		t.Errorf("Expected NewStatus 'active', got %s", status.NewStatus)
	}
}

// Helper function for UUID pointers
func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}
