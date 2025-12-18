/*
Package customer_test provides comprehensive testing for customer repository validation and transformation logic.

This file focuses on unit tests that validate business logic without requiring database setup:
- Error type definitions and error wrapping
- Request validation for PATCH operations
- Field application and transformation logic
- Entity preparation and defaults setting

Test Categories:
1. Error Types & Wrapping - Custom error definitions and proper error chaining
2. Request Validation - PATCH request validation and UUID format checking
3. Field Application - Applying PATCH updates to customer entities
4. Data Transformation - Converting PATCH requests to entity structures
5. Entity Preparation - Setting defaults and preparing entities for database operations

Important Notes:
- All tests are pure unit tests that don't require database connections
- Tests validate business logic and data transformation rules
- Error handling and validation are critical for API reliability
*/
package customer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

)

// ===== ERROR TYPES & WRAPPING =====

// TestCustomErrors_Definitions verifies that all custom error types are properly defined.
//
// Business Scenario: Repository operations must return consistent, identifiable error types
// Expected: All error constants are defined with appropriate messages
func TestCustomErrors_Definitions(t *testing.T) {
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

// TestRepository_ErrorWrapping verifies that repository methods properly wrap and chain errors.
//
// Business Scenario: Errors should be wrapped with context while maintaining error chain
// Expected: Invalid inputs return wrapped errors that can be identified with errors.Is()
func TestRepository_ErrorWrapping(t *testing.T) {
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

// ===== REQUEST VALIDATION =====

// TestPatchRequest_Validation verifies PATCH request validation logic.
//
// Business Scenario: PATCH requests must be validated before processing
// Expected: Valid requests pass, invalid requests fail with descriptive errors
func TestPatchRequest_Validation(t *testing.T) {
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
}

// TestPatchRequest_InvalidUUID verifies UUID validation in PATCH requests.
//
// Business Scenario: Invalid UUIDs in PATCH requests should be rejected
// Expected: Malformed UUIDs return validation errors with field-specific messages
func TestPatchRequest_InvalidUUID(t *testing.T) {
	service := &CustomerService{}

	// Test invalid UUID
	invalidUUID := "not-a-uuid"
	patchData := &PatchCustomerRequest{
		DefaultShippingAddressID: &invalidUUID,
	}

	err := service.ValidatePatchData(patchData)
	if err == nil {
		t.Error("Expected error for invalid UUID")
	}
	// Check that the error message contains information about invalid UUID
	if !containsString(err.Error(), "invalid default_shipping_address_id") {
		t.Errorf("Expected error message to contain 'invalid default_shipping_address_id', got: %s", err.Error())
	}
}

// ===== FIELD APPLICATION =====

// TestFieldUpdates_BasicFields verifies applying basic field updates from PATCH requests.
//
// Business Scenario: PATCH requests should update only specified fields
// Expected: Basic fields (username, email, status) are updated correctly
func TestFieldUpdates_BasicFields(t *testing.T) {
	service := &CustomerService{}

	customer := &Customer{
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

// TestFieldUpdates_DefaultFields verifies applying default field updates from PATCH requests.
//
// Business Scenario: Default field pointers should be set/cleared correctly
// Expected: UUID pointers are set for valid values, cleared for empty strings
func TestFieldUpdates_DefaultFields(t *testing.T) {
	service := &CustomerService{}

	customer := &Customer{}

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
}

// TestFieldUpdates_DefaultFieldClearing verifies clearing default fields with empty strings.
//
// Business Scenario: Empty string values should clear default field pointers
// Expected: Default field pointers are set to nil when empty string provided
func TestFieldUpdates_DefaultFieldClearing(t *testing.T) {
	service := &CustomerService{}

	customer := &Customer{}

	// First set a UUID
	validUUID := uuid.New().String()
	patchData := &PatchCustomerRequest{
		DefaultShippingAddressID: &validUUID,
	}
	service.ApplyFieldUpdates(customer, patchData)

	// Then clear it with empty string
	emptyString := ""
	patchData = &PatchCustomerRequest{
		DefaultShippingAddressID: &emptyString,
	}
	service.ApplyFieldUpdates(customer, patchData)

	if customer.DefaultShippingAddressID != nil {
		t.Error("Expected DefaultShippingAddressID to be nil after clearing")
	}
}

// ===== DATA TRANSFORMATION =====

// TestAddressTransformation_FromPatch verifies transforming PATCH address requests to entity addresses.
//
// Business Scenario: PATCH requests with address arrays need conversion to entity structures
// Expected: Address fields are correctly mapped from request to entity format
func TestAddressTransformation_FromPatch(t *testing.T) {
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

// TestCreditCardTransformation_FromPatch verifies transforming PATCH credit card requests to entity cards.
//
// Business Scenario: PATCH requests with credit card arrays need conversion to entity structures
// Expected: Credit card fields are correctly mapped from request to entity format
func TestCreditCardTransformation_FromPatch(t *testing.T) {
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

// ===== ENTITY PREPARATION =====

// TestCustomerDefaults_NewCustomer verifies setting defaults for new customers.
//
// Business Scenario: New customer entities need default values for database insertion
// Expected: CustomerID is generated, status set to 'active', timestamps set to current time
func TestCustomerDefaults_NewCustomer(t *testing.T) {
	repo := &customerRepository{}

	customer := &Customer{
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
}

// TestCustomerDefaults_ExistingValuesPreserved verifies that existing values are not overwritten.
//
// Business Scenario: When updating existing customers, preserve their current values
// Expected: Only nil/zero values are set, existing values remain unchanged
func TestCustomerDefaults_ExistingValuesPreserved(t *testing.T) {
	repo := &customerRepository{}

	customer := &Customer{
		Username:       "testuser",
		Email:          "test@example.com",
		CustomerStatus: "inactive",
		CustomerSince:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		StatusDateTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	repo.PrepareCustomerDefaults(customer)

	if customer.CustomerStatus != "inactive" {
		t.Error("Existing CustomerStatus should be preserved")
	}

	if !customer.CustomerSince.Equal(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Error("Existing CustomerSince should be preserved")
	}
}

// ===== HELPERS =====

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsString(s[1:len(s)-1], substr)))
}
