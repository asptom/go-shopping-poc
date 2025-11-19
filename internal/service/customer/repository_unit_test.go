package customer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	entity "go-shopping-poc/internal/entity/customer"
)

// TestErrorWrapping tests that repository methods properly wrap errors
func TestCustomerRepository_ErrorWrapping(t *testing.T) {
	// Test ErrInvalidUUID wrapping
	_, err := (&customerRepository{}).GetCustomerByID(context.Background(), "invalid-uuid")
	if err == nil {
		t.Error("Expected error for invalid UUID")
	}
	if !errors.Is(err, ErrInvalidUUID) {
		t.Errorf("Expected ErrInvalidUUID in error chain, got: %v", err)
	}

	// Test that error messages include context
	if !containsString(err.Error(), "invalid UUID format") {
		t.Errorf("Error message should include context, got: %s", err.Error())
	}
}

// TestCustomErrors tests that custom error types are properly defined
func TestCustomErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrCustomerNotFound", ErrCustomerNotFound, "customer not found"},
		{"ErrAddressNotFound", ErrAddressNotFound, "address not found"},
		{"ErrCreditCardNotFound", ErrCreditCardNotFound, "credit card not found"},
		{"ErrInvalidUUID", ErrInvalidUUID, "invalid UUID format"},
		{"ErrDatabaseOperation", ErrDatabaseOperation, "database operation failed"},
		{"ErrTransactionFailed", ErrTransactionFailed, "transaction failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("Error %s should not be nil", tt.name)
			}
			if tt.err.Error() != tt.expected {
				t.Errorf("Error %s should have message '%s', got '%s'", tt.name, tt.expected, tt.err.Error())
			}
		})
	}
}

// TestPatchCustomerRequest_Validation tests patch request validation
func TestPatchCustomerRequest_Validation(t *testing.T) {
	service := &CustomerService{}

	// Test nil patch data
	err := service.ValidatePatchData(nil)
	if err == nil {
		t.Error("Expected error for nil patch data")
	}

	// Test valid patch data
	validUUID := uuid.New().String()
	patchData := &PatchCustomerRequest{
		DefaultShippingAddressID: &validUUID,
		DefaultBillingAddressID:  &validUUID,
		DefaultCreditCardID:      &validUUID,
	}

	err = service.ValidatePatchData(patchData)
	if err != nil {
		t.Errorf("Expected no error for valid patch data, got: %v", err)
	}

	// Test invalid UUID
	invalidUUID := "not-a-uuid"
	patchData = &PatchCustomerRequest{
		DefaultShippingAddressID: &invalidUUID,
	}

	err = service.ValidatePatchData(patchData)
	if err == nil {
		t.Error("Expected error for invalid UUID")
	}
	// Check that the error message contains information about invalid UUID
	if !containsString(err.Error(), "invalid default_shipping_address_id") {
		t.Errorf("Expected error message to contain 'invalid default_shipping_address_id', got: %s", err.Error())
	}
}

// TestFieldUpdates tests the ApplyFieldUpdates method
func TestApplyFieldUpdates(t *testing.T) {
	service := &CustomerService{}

	customer := &entity.Customer{
		Username:       "olduser",
		Email:          "old@example.com",
		CustomerStatus: "inactive",
	}

	userName := "newuser"
	email := "new@example.com"
	status := "active"

	patchData := &PatchCustomerRequest{
		UserName:       &userName,
		Email:          &email,
		CustomerStatus: &status,
	}

	service.ApplyFieldUpdates(customer, patchData)

	if customer.Username != "newuser" {
		t.Errorf("Expected username 'newuser', got '%s'", customer.Username)
	}

	if customer.Email != "new@example.com" {
		t.Errorf("Expected email 'new@example.com', got '%s'", customer.Email)
	}

	if customer.CustomerStatus != "active" {
		t.Errorf("Expected status 'active', got '%s'", customer.CustomerStatus)
	}
}

// TestDefaultFieldHandling tests UUID pointer field handling
func TestDefaultFieldHandling(t *testing.T) {
	service := &CustomerService{}

	customer := &entity.Customer{}

	// Test setting UUID
	validUUID := uuid.New().String()
	patchData := &PatchCustomerRequest{
		DefaultShippingAddressID: &validUUID,
	}

	service.ApplyFieldUpdates(customer, patchData)

	if customer.DefaultShippingAddressID == nil {
		t.Error("Expected DefaultShippingAddressID to be set")
	}
	if customer.DefaultShippingAddressID.String() != validUUID {
		t.Errorf("Expected UUID %s, got %s", validUUID, customer.DefaultShippingAddressID.String())
	}

	// Test clearing UUID (empty string)
	emptyString := ""
	patchData = &PatchCustomerRequest{
		DefaultShippingAddressID: &emptyString,
	}

	service.ApplyFieldUpdates(customer, patchData)

	if customer.DefaultShippingAddressID != nil {
		t.Error("Expected DefaultShippingAddressID to be nil after clearing")
	}
}

// TestAddressTransformation tests address transformation
func TestAddressTransformation(t *testing.T) {
	service := &CustomerService{}

	patchAddresses := []PatchAddressRequest{
		{
			AddressType: "shipping",
			FirstName:   "John",
			LastName:    "Doe",
			Address1:    "123 Main St",
			City:        "Test City",
			State:       "TS",
			Zip:         "12345",
		},
	}

	addresses := service.TransformAddressesFromPatch(patchAddresses)

	if len(addresses) != 1 {
		t.Errorf("Expected 1 address, got %d", len(addresses))
	}

	addr := addresses[0]
	if addr.AddressType != "shipping" {
		t.Errorf("Expected address type 'shipping', got '%s'", addr.AddressType)
	}

	if addr.FirstName != "John" {
		t.Errorf("Expected first name 'John', got '%s'", addr.FirstName)
	}

	if addr.Address1 != "123 Main St" {
		t.Errorf("Expected address1 '123 Main St', got '%s'", addr.Address1)
	}
}

// TestCreditCardTransformation tests credit card transformation
func TestCreditCardTransformation(t *testing.T) {
	service := &CustomerService{}

	patchCards := []PatchCreditCardRequest{
		{
			CardType:       "visa",
			CardNumber:     "4111111111111111",
			CardHolderName: "John Doe",
			CardExpires:    "12/25",
			CardCVV:        "123",
		},
	}

	cards := service.TransformCreditCardsFromPatch(patchCards)

	if len(cards) != 1 {
		t.Errorf("Expected 1 credit card, got %d", len(cards))
	}

	card := cards[0]
	if card.CardType != "visa" {
		t.Errorf("Expected card type 'visa', got '%s'", card.CardType)
	}

	if card.CardHolderName != "John Doe" {
		t.Errorf("Expected card holder 'John Doe', got '%s'", card.CardHolderName)
	}

	if card.CardNumber != "4111111111111111" {
		t.Errorf("Expected card number '4111111111111111', got '%s'", card.CardNumber)
	}
}

// TestCustomerDefaults tests the PrepareCustomerDefaults method
func TestCustomerDefaults(t *testing.T) {
	repo := &customerRepository{}

	customer := &entity.Customer{
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
	}

	repo.PrepareCustomerDefaults(customer)

	if customer.CustomerID == "" {
		t.Error("CustomerID should be set")
	}

	if customer.CustomerStatus != "active" {
		t.Errorf("Expected status 'active', got '%s'", customer.CustomerStatus)
	}

	if customer.CustomerSince.IsZero() {
		t.Error("CustomerSince should be set")
	}

	if customer.StatusDateTime.IsZero() {
		t.Error("StatusDateTime should be set")
	}

	// Test that existing values are preserved
	customer2 := &entity.Customer{
		Username:       "testuser",
		Email:          "test@example.com",
		CustomerStatus: "inactive",
		CustomerSince:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		StatusDateTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	repo.PrepareCustomerDefaults(customer2)

	if customer2.CustomerStatus != "inactive" {
		t.Error("Existing CustomerStatus should be preserved")
	}

	if !customer2.CustomerSince.Equal(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Error("Existing CustomerSince should be preserved")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsString(s[1:len(s)-1], substr)))
}
