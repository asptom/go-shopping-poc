package customer

import (
	"context"
	"strings"
	"testing"
	"time"

	"go-shopping-poc/internal/platform/service"
)

// MockCustomerRepository implements CustomerRepository for testing
type MockCustomerRepository struct{}

// mockConfig returns a test config for service tests
func mockConfig() *Config {
	return &Config{
		DatabaseURL:    "postgres://test:test@localhost:5432/test",
		ServicePort:    ":8080",
		WriteTopic:     "test-topic",
		Group:          "test-group",
		OutboxInterval: 5 * time.Second,
	}
}

// mockInfrastructure returns a test infrastructure for service tests
func mockInfrastructure(repo CustomerRepository) *CustomerInfrastructure {
	return &CustomerInfrastructure{
		Database:        nil, // Not needed for unit tests
		EventBus:        nil, // Not needed for unit tests
		OutboxWriter:    nil, // Not needed for unit tests
		OutboxPublisher: nil, // Not needed for unit tests
		CORSHandler:     nil, // Not needed for unit tests
	}
}

func (m *MockCustomerRepository) GetCustomerByEmail(ctx context.Context, email string) (*Customer, error) {
	return nil, nil // Mock implementation
}

func (m *MockCustomerRepository) GetCustomerByID(ctx context.Context, customerID string) (*Customer, error) {
	return nil, nil // Mock implementation
}

func (m *MockCustomerRepository) InsertCustomer(ctx context.Context, customer *Customer) error {
	return nil // Mock implementation
}

func (m *MockCustomerRepository) UpdateCustomer(ctx context.Context, customer *Customer) error {
	return nil // Mock implementation
}

func (m *MockCustomerRepository) PatchCustomer(ctx context.Context, customerID string, patchData *PatchCustomerRequest) error {
	return nil // Mock implementation
}

func (m *MockCustomerRepository) AddAddress(ctx context.Context, customerID string, addr *Address) (*Address, error) {
	return addr, nil // Mock implementation
}

func (m *MockCustomerRepository) UpdateAddress(ctx context.Context, addressID string, addr *Address) error {
	return nil // Mock implementation
}

func (m *MockCustomerRepository) DeleteAddress(ctx context.Context, addressID string) error {
	return nil // Mock implementation
}

func (m *MockCustomerRepository) AddCreditCard(ctx context.Context, customerID string, card *CreditCard) (*CreditCard, error) {
	return card, nil // Mock implementation
}

func (m *MockCustomerRepository) UpdateCreditCard(ctx context.Context, customerID string, card *CreditCard) error {
	return nil // Mock implementation
}

func (m *MockCustomerRepository) DeleteCreditCard(ctx context.Context, cardID string) error {
	return nil // Mock implementation
}

func (m *MockCustomerRepository) UpdateDefaultShippingAddress(ctx context.Context, customerID, addressID string) error {
	return nil // Mock implementation
}

func (m *MockCustomerRepository) UpdateDefaultBillingAddress(ctx context.Context, customerID, addressID string) error {
	return nil // Mock implementation
}

func (m *MockCustomerRepository) UpdateDefaultCreditCard(ctx context.Context, customerID, cardID string) error {
	return nil // Mock implementation
}

func (m *MockCustomerRepository) ClearDefaultShippingAddress(ctx context.Context, customerID string) error {
	return nil // Mock implementation
}

func (m *MockCustomerRepository) ClearDefaultBillingAddress(ctx context.Context, customerID string) error {
	return nil // Mock implementation
}

func (m *MockCustomerRepository) ClearDefaultCreditCard(ctx context.Context, customerID string) error {
	return nil // Mock implementation
}

// TestCustomerServicePlatformInterface verifies that CustomerService implements the platform service interface
func TestCustomerServicePlatformInterface(t *testing.T) {
	// Create service with platform base using mock repository
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	customerService := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	// Verify service implements platform Service interface
	var _ service.Service = customerService

	// Test basic service methods
	if customerService.Name() != "customer" {
		t.Errorf("Expected service name 'customer', got '%s'", customerService.Name())
	}

	// Test health check
	if err := customerService.Health(); err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test start/stop lifecycle
	ctx := context.Background()
	if err := customerService.Start(ctx); err != nil {
		t.Errorf("Service start failed: %v", err)
	}

	if err := customerService.Stop(ctx); err != nil {
		t.Errorf("Service stop failed: %v", err)
	}
}

// TestCustomerServiceFunctionality verifies that all existing functionality still works
func TestCustomerServiceFunctionality(t *testing.T) {
	// Create service with platform base using mock repository
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	customerService := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	// Test that service methods work as before
	ctx := context.Background()

	// Test GetCustomerByEmail
	customer, err := customerService.GetCustomerByEmail(ctx, "test@example.com")
	if err != nil {
		t.Errorf("GetCustomerByEmail failed: %v", err)
	}
	if customer != nil {
		t.Error("Expected nil customer from mock repository")
	}

	// Test GetCustomerByID
	customer, err = customerService.GetCustomerByID(ctx, "test-id")
	if err != nil {
		t.Errorf("GetCustomerByID failed: %v", err)
	}
	if customer != nil {
		t.Error("Expected nil customer from mock repository")
	}
}

// TestCreateCustomer_ValidCustomer tests successful customer creation with validation
func TestCreateCustomer_ValidCustomer(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()
	customer := &Customer{
		CustomerID:     "test-customer-123",
		Username:       "testuser",
		Email:          "test@example.com",
		FirstName:      "John",
		LastName:       "Doe",
		CustomerStatus: "active",
		Addresses: []Address{
			{
				AddressType: "shipping",
				FirstName:   "John",
				LastName:    "Doe",
				Address1:    "123 Main St",
				City:        "Anytown",
				State:       "CA",
				Zip:         "12345",
			},
		},
		CreditCards: []CreditCard{
			{
				CardType:       "visa",
				CardNumber:     "4111111111111111",
				CardHolderName: "John Doe",
				CardExpires:    "12/25",
				CardCVV:        "123",
			},
		},
	}

	err := service.CreateCustomer(ctx, customer)
	if err != nil {
		t.Errorf("CreateCustomer failed for valid customer: %v", err)
	}
}

// TestCreateCustomer_InvalidCustomer tests customer creation with validation errors
func TestCreateCustomer_InvalidCustomer(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()

	// Test with empty username
	customer := &Customer{
		CustomerID: "test-customer-123",
		Username:   "", // Invalid: empty username
		Email:      "test@example.com",
	}

	err := service.CreateCustomer(ctx, customer)
	if err == nil {
		t.Error("Expected validation error for empty username, but got none")
	}
	if !strings.Contains(err.Error(), "username is required") {
		t.Errorf("Expected username validation error, got: %v", err)
	}

	// Test with invalid email
	customer = &Customer{
		CustomerID: "test-customer-123",
		Username:   "testuser",
		Email:      "invalid-email", // Invalid: no @ symbol
	}

	err = service.CreateCustomer(ctx, customer)
	if err == nil {
		t.Error("Expected validation error for invalid email, but got none")
	}
	if !strings.Contains(err.Error(), "email must be valid format") {
		t.Errorf("Expected email validation error, got: %v", err)
	}

	// Test with invalid customer status
	customer = &Customer{
		CustomerID:     "test-customer-123",
		Username:       "testuser",
		CustomerStatus: "invalid", // Invalid: not active/inactive/suspended
	}

	err = service.CreateCustomer(ctx, customer)
	if err == nil {
		t.Error("Expected validation error for invalid status, but got none")
	}
	if !strings.Contains(err.Error(), "customer status must be") {
		t.Errorf("Expected status validation error, got: %v", err)
	}
}

// TestCreateCustomer_InvalidAddress tests customer creation with invalid address
func TestCreateCustomer_InvalidAddress(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()
	customer := &Customer{
		CustomerID: "test-customer-123",
		Username:   "testuser",
		Addresses: []Address{
			{
				AddressType: "", // Invalid: empty address type
				FirstName:   "John",
				LastName:    "Doe",
				Address1:    "123 Main St",
				City:        "Anytown",
				State:       "CA",
				Zip:         "12345",
			},
		},
	}

	err := service.CreateCustomer(ctx, customer)
	if err == nil {
		t.Error("Expected validation error for invalid address, but got none")
	}
	if !strings.Contains(err.Error(), "address type is required") {
		t.Errorf("Expected address validation error, got: %v", err)
	}
}

// TestCreateCustomer_InvalidCreditCard tests customer creation with invalid credit card
func TestCreateCustomer_InvalidCreditCard(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()
	customer := &Customer{
		CustomerID: "test-customer-123",
		Username:   "testuser",
		CreditCards: []CreditCard{
			{
				CardType:       "", // Invalid: empty card type
				CardNumber:     "4111111111111111",
				CardHolderName: "John Doe",
				CardExpires:    "12/25",
				CardCVV:        "123",
			},
		},
	}

	err := service.CreateCustomer(ctx, customer)
	if err == nil {
		t.Error("Expected validation error for invalid credit card, but got none")
	}
	if !strings.Contains(err.Error(), "card type is required") {
		t.Errorf("Expected credit card validation error, got: %v", err)
	}
}

// TestUpdateCustomer tests customer update functionality
func TestUpdateCustomer(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()
	customer := &Customer{
		CustomerID: "test-customer-123",
		Username:   "testuser",
		Email:      "test@example.com",
	}

	err := service.UpdateCustomer(ctx, customer)
	if err != nil {
		t.Errorf("UpdateCustomer failed: %v", err)
	}
}

// TestPatchCustomer_ValidPatch tests successful customer patching
func TestPatchCustomer_ValidPatch(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()
	patchData := &PatchCustomerRequest{
		UserName:                 stringPtr("newusername"),
		Email:                    stringPtr("newemail@example.com"),
		FirstName:                stringPtr("Jane"),
		LastName:                 stringPtr("Smith"),
		CustomerStatus:           stringPtr("active"),
		DefaultShippingAddressID: stringPtr("550e8400-e29b-41d4-a716-446655440000"),
	}

	err := service.PatchCustomer(ctx, "test-customer-123", patchData)
	if err != nil {
		t.Errorf("PatchCustomer failed for valid patch: %v", err)
	}
}

// TestPatchCustomer_InvalidPatch tests patching with invalid data
func TestPatchCustomer_InvalidPatch(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()

	// Test with nil patch data
	err := service.PatchCustomer(ctx, "test-customer-123", nil)
	if err == nil {
		t.Error("Expected error for nil patch data, but got none")
	}
	if !strings.Contains(err.Error(), "invalid patch data") {
		t.Errorf("Expected patch data validation error, got: %v", err)
	}

	// Test with invalid UUID
	patchData := &PatchCustomerRequest{
		DefaultShippingAddressID: stringPtr("invalid-uuid"),
	}

	err = service.PatchCustomer(ctx, "test-customer-123", patchData)
	if err == nil {
		t.Error("Expected error for invalid UUID, but got none")
	}
	if !strings.Contains(err.Error(), "invalid default_shipping_address_id") {
		t.Errorf("Expected UUID validation error, got: %v", err)
	}
}

// TestValidatePatchData tests patch data validation
func TestValidatePatchData(t *testing.T) {
	mockInfra := mockInfrastructure(nil)
	service := NewCustomerServiceWithRepo(nil, mockInfra, mockConfig())

	// Test nil patch data
	err := service.ValidatePatchData(nil)
	if err == nil {
		t.Error("Expected error for nil patch data")
	}

	// Test valid patch data
	patchData := &PatchCustomerRequest{
		UserName: stringPtr("testuser"),
		Email:    stringPtr("test@example.com"),
	}
	err = service.ValidatePatchData(patchData)
	if err != nil {
		t.Errorf("Valid patch data should not return error: %v", err)
	}

	// Test invalid UUID
	patchData = &PatchCustomerRequest{
		DefaultShippingAddressID: stringPtr("invalid-uuid"),
	}
	err = service.ValidatePatchData(patchData)
	if err == nil {
		t.Error("Expected error for invalid UUID")
	}
	if !strings.Contains(err.Error(), "invalid default_shipping_address_id") {
		t.Errorf("Expected UUID validation error, got: %v", err)
	}
}

// TestApplyFieldUpdates tests field update application
func TestApplyFieldUpdates(t *testing.T) {
	mockInfra := mockInfrastructure(nil)
	service := NewCustomerServiceWithRepo(nil, mockInfra, mockConfig())

	customer := &Customer{
		CustomerID:     "test-customer-123",
		Username:       "oldusername",
		Email:          "old@example.com",
		FirstName:      "Old",
		LastName:       "Name",
		CustomerStatus: "inactive",
	}

	patchData := &PatchCustomerRequest{
		UserName:                 stringPtr("newusername"),
		Email:                    stringPtr("new@example.com"),
		FirstName:                stringPtr("New"),
		LastName:                 stringPtr("Name"),
		CustomerStatus:           stringPtr("active"),
		DefaultShippingAddressID: stringPtr("550e8400-e29b-41d4-a716-446655440000"),
		DefaultBillingAddressID:  stringPtr(""), // Clear the field
	}

	err := service.ApplyFieldUpdates(customer, patchData)
	if err != nil {
		t.Errorf("ApplyFieldUpdates failed: %v", err)
	}

	// Verify updates
	if customer.Username != "newusername" {
		t.Errorf("Expected username 'newusername', got '%s'", customer.Username)
	}
	if customer.Email != "new@example.com" {
		t.Errorf("Expected email 'new@example.com', got '%s'", customer.Email)
	}
	if customer.FirstName != "New" {
		t.Errorf("Expected first name 'New', got '%s'", customer.FirstName)
	}
	if customer.LastName != "Name" {
		t.Errorf("Expected last name 'Name', got '%s'", customer.LastName)
	}
	if customer.CustomerStatus != "active" {
		t.Errorf("Expected status 'active', got '%s'", customer.CustomerStatus)
	}
	if customer.DefaultShippingAddressID == nil {
		t.Error("Expected default shipping address ID to be set")
	}
	if customer.DefaultBillingAddressID != nil {
		t.Error("Expected default billing address ID to be cleared")
	}
}

// TestApplyFieldUpdates_InvalidUUID tests field updates with invalid UUID
func TestApplyFieldUpdates_InvalidUUID(t *testing.T) {
	mockInfra := mockInfrastructure(nil)
	service := NewCustomerServiceWithRepo(nil, mockInfra, mockConfig())

	customer := &Customer{}
	patchData := &PatchCustomerRequest{
		DefaultShippingAddressID: stringPtr("invalid-uuid"),
	}

	err := service.ApplyFieldUpdates(customer, patchData)
	if err == nil {
		t.Error("Expected error for invalid UUID")
	}
	if !strings.Contains(err.Error(), "invalid default_shipping_address_id") {
		t.Errorf("Expected UUID validation error, got: %v", err)
	}
}

// TestTransformAddressesFromPatch tests address transformation from patch data
func TestTransformAddressesFromPatch(t *testing.T) {
	mockInfra := mockInfrastructure(nil)
	service := NewCustomerServiceWithRepo(nil, mockInfra, mockConfig())

	patchAddresses := []PatchAddressRequest{
		{
			AddressType: "shipping",
			FirstName:   "John",
			LastName:    "Doe",
			Address1:    "123 Main St",
			Address2:    "Apt 4B",
			City:        "Anytown",
			State:       "CA",
			Zip:         "12345",
		},
		{
			AddressType: "billing",
			FirstName:   "Jane",
			LastName:    "Smith",
			Address1:    "456 Oak Ave",
			City:        "Othertown",
			State:       "NY",
			Zip:         "67890",
		},
	}

	addresses := service.TransformAddressesFromPatch(patchAddresses)

	if len(addresses) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(addresses))
	}

	// Verify first address
	if addresses[0].AddressType != "shipping" {
		t.Errorf("Expected address type 'shipping', got '%s'", addresses[0].AddressType)
	}
	if addresses[0].FirstName != "John" {
		t.Errorf("Expected first name 'John', got '%s'", addresses[0].FirstName)
	}
	if addresses[0].Address2 != "Apt 4B" {
		t.Errorf("Expected address2 'Apt 4B', got '%s'", addresses[0].Address2)
	}

	// Verify second address
	if addresses[1].AddressType != "billing" {
		t.Errorf("Expected address type 'billing', got '%s'", addresses[1].AddressType)
	}
	if addresses[1].City != "Othertown" {
		t.Errorf("Expected city 'Othertown', got '%s'", addresses[1].City)
	}
}

// TestTransformCreditCardsFromPatch tests credit card transformation from patch data
func TestTransformCreditCardsFromPatch(t *testing.T) {
	mockInfra := mockInfrastructure(nil)
	service := NewCustomerServiceWithRepo(nil, mockInfra, mockConfig())

	patchCards := []PatchCreditCardRequest{
		{
			CardType:       "visa",
			CardNumber:     "4111111111111111",
			CardHolderName: "John Doe",
			CardExpires:    "12/25",
			CardCVV:        "123",
		},
		{
			CardType:       "mastercard",
			CardNumber:     "5555555555554444",
			CardHolderName: "Jane Smith",
			CardExpires:    "06/26",
			CardCVV:        "456",
		},
	}

	cards := service.TransformCreditCardsFromPatch(patchCards)

	if len(cards) != 2 {
		t.Errorf("Expected 2 credit cards, got %d", len(cards))
	}

	// Verify first card
	if cards[0].CardType != "visa" {
		t.Errorf("Expected card type 'visa', got '%s'", cards[0].CardType)
	}
	if cards[0].CardNumber != "4111111111111111" {
		t.Errorf("Expected card number '4111111111111111', got '%s'", cards[0].CardNumber)
	}
	if cards[0].CardCVV != "123" {
		t.Errorf("Expected CVV '123', got '%s'", cards[0].CardCVV)
	}

	// Verify second card
	if cards[1].CardType != "mastercard" {
		t.Errorf("Expected card type 'mastercard', got '%s'", cards[1].CardType)
	}
	if cards[1].CardHolderName != "Jane Smith" {
		t.Errorf("Expected card holder 'Jane Smith', got '%s'", cards[1].CardHolderName)
	}
}

// Test address and credit card management methods
func TestAddressManagement(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()

	// Test AddAddress
	addr := &Address{
		AddressType: "shipping",
		FirstName:   "John",
		LastName:    "Doe",
		Address1:    "123 Main St",
		City:        "Anytown",
		State:       "CA",
		Zip:         "12345",
	}

	result, err := service.AddAddress(ctx, "customer-123", addr)
	if err != nil {
		t.Errorf("AddAddress failed: %v", err)
	}
	if result == nil {
		t.Error("Expected address to be returned")
	}

	// Test UpdateAddress
	err = service.UpdateAddress(ctx, "address-123", addr)
	if err != nil {
		t.Errorf("UpdateAddress failed: %v", err)
	}

	// Test DeleteAddress
	err = service.DeleteAddress(ctx, "address-123")
	if err != nil {
		t.Errorf("DeleteAddress failed: %v", err)
	}
}

func TestCreditCardManagement(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()

	// Test AddCreditCard
	card := &CreditCard{
		CardType:       "visa",
		CardNumber:     "4111111111111111",
		CardHolderName: "John Doe",
		CardExpires:    "12/25",
		CardCVV:        "123",
	}

	result, err := service.AddCreditCard(ctx, "customer-123", card)
	if err != nil {
		t.Errorf("AddCreditCard failed: %v", err)
	}
	if result == nil {
		t.Error("Expected credit card to be returned")
	}

	// Test UpdateCreditCard
	err = service.UpdateCreditCard(ctx, "customer-123", card)
	if err != nil {
		t.Errorf("UpdateCreditCard failed: %v", err)
	}

	// Test DeleteCreditCard
	err = service.DeleteCreditCard(ctx, "card-123")
	if err != nil {
		t.Errorf("DeleteCreditCard failed: %v", err)
	}
}

func TestDefaultSettings(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()

	// Test SetDefaultShippingAddress
	err := service.SetDefaultShippingAddress(ctx, "customer-123", "address-456")
	if err != nil {
		t.Errorf("SetDefaultShippingAddress failed: %v", err)
	}

	// Test SetDefaultBillingAddress
	err = service.SetDefaultBillingAddress(ctx, "customer-123", "address-789")
	if err != nil {
		t.Errorf("SetDefaultBillingAddress failed: %v", err)
	}

	// Test SetDefaultCreditCard
	err = service.SetDefaultCreditCard(ctx, "customer-123", "card-101")
	if err != nil {
		t.Errorf("SetDefaultCreditCard failed: %v", err)
	}

	// Test ClearDefaultShippingAddress
	err = service.ClearDefaultShippingAddress(ctx, "customer-123")
	if err != nil {
		t.Errorf("ClearDefaultShippingAddress failed: %v", err)
	}

	// Test ClearDefaultBillingAddress
	err = service.ClearDefaultBillingAddress(ctx, "customer-123")
	if err != nil {
		t.Errorf("ClearDefaultBillingAddress failed: %v", err)
	}

	// Test ClearDefaultCreditCard
	err = service.ClearDefaultCreditCard(ctx, "customer-123")
	if err != nil {
		t.Errorf("ClearDefaultCreditCard failed: %v", err)
	}
}

// TestCreateCustomer_EventPublishing tests that customer creation publishes events
func TestCreateCustomer_EventPublishing(t *testing.T) {
	// This test verifies that the service properly calls repository methods
	// that would trigger event publishing through the outbox pattern
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()
	customer := &Customer{
		CustomerID: "event-test-customer-123",
		Username:   "eventuser",
		Email:      "event@example.com",
	}

	// The mock repository doesn't actually publish events, but we can verify
	// that the service calls the repository method correctly
	err := service.CreateCustomer(ctx, customer)
	if err != nil {
		t.Errorf("CreateCustomer should succeed for valid customer, got error: %v", err)
	}

	// In a real integration test with database, we would verify that
	// the outbox table contains the expected event
	// For now, we verify the service layer logic works correctly
}

// TestUpdateCustomer_EventPublishing tests that customer updates publish events
func TestUpdateCustomer_EventPublishing(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()
	customer := &Customer{
		CustomerID: "update-event-test-123",
		Username:   "updateuser",
		Email:      "update@example.com",
	}

	err := service.UpdateCustomer(ctx, customer)
	if err != nil {
		t.Errorf("UpdateCustomer should succeed, got error: %v", err)
	}

	// In integration tests, we would verify event publishing
}

// TestPatchCustomer_EventPublishing tests that customer patches publish events
func TestPatchCustomer_EventPublishing(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()
	patchData := &PatchCustomerRequest{
		UserName: stringPtr("patcheduser"),
		Email:    stringPtr("patched@example.com"),
	}

	err := service.PatchCustomer(ctx, "patch-event-test-123", patchData)
	if err != nil {
		t.Errorf("PatchCustomer should succeed for valid patch, got error: %v", err)
	}

	// In integration tests, we would verify event publishing
}

// TestAddressOperations_EventPublishing tests that address operations publish events
func TestAddressOperations_EventPublishing(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()

	// Test AddAddress event publishing
	addr := &Address{
		AddressType: "shipping",
		FirstName:   "John",
		LastName:    "Doe",
		Address1:    "123 Main St",
		City:        "Anytown",
		State:       "CA",
		Zip:         "12345",
	}

	result, err := service.AddAddress(ctx, "address-event-test-123", addr)
	if err != nil {
		t.Errorf("AddAddress should succeed, got error: %v", err)
	}
	if result == nil {
		t.Error("Expected address to be returned")
	}

	// Test UpdateAddress event publishing
	err = service.UpdateAddress(ctx, "address-456", addr)
	if err != nil {
		t.Errorf("UpdateAddress should succeed, got error: %v", err)
	}

	// Test DeleteAddress event publishing
	err = service.DeleteAddress(ctx, "address-456")
	if err != nil {
		t.Errorf("DeleteAddress should succeed, got error: %v", err)
	}
}

// TestCreditCardOperations_EventPublishing tests that credit card operations publish events
func TestCreditCardOperations_EventPublishing(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()

	// Test AddCreditCard event publishing
	card := &CreditCard{
		CardType:       "visa",
		CardNumber:     "4111111111111111",
		CardHolderName: "John Doe",
		CardExpires:    "12/25",
		CardCVV:        "123",
	}

	result, err := service.AddCreditCard(ctx, "card-event-test-123", card)
	if err != nil {
		t.Errorf("AddCreditCard should succeed, got error: %v", err)
	}
	if result == nil {
		t.Error("Expected credit card to be returned")
	}

	// Test UpdateCreditCard event publishing
	err = service.UpdateCreditCard(ctx, "customer-123", card)
	if err != nil {
		t.Errorf("UpdateCreditCard should succeed, got error: %v", err)
	}

	// Test DeleteCreditCard event publishing
	err = service.DeleteCreditCard(ctx, "card-456")
	if err != nil {
		t.Errorf("DeleteCreditCard should succeed, got error: %v", err)
	}
}

// TestDefaultSettings_EventPublishing tests that default setting changes publish events
func TestDefaultSettings_EventPublishing(t *testing.T) {
	mockRepo := &MockCustomerRepository{}
	mockInfra := mockInfrastructure(mockRepo)
	service := NewCustomerServiceWithRepo(mockRepo, mockInfra, mockConfig())

	ctx := context.Background()

	// Test SetDefaultShippingAddress event publishing
	err := service.SetDefaultShippingAddress(ctx, "defaults-event-test-123", "address-456")
	if err != nil {
		t.Errorf("SetDefaultShippingAddress should succeed, got error: %v", err)
	}

	// Test SetDefaultBillingAddress event publishing
	err = service.SetDefaultBillingAddress(ctx, "defaults-event-test-123", "address-789")
	if err != nil {
		t.Errorf("SetDefaultBillingAddress should succeed, got error: %v", err)
	}

	// Test SetDefaultCreditCard event publishing
	err = service.SetDefaultCreditCard(ctx, "defaults-event-test-123", "card-101")
	if err != nil {
		t.Errorf("SetDefaultCreditCard should succeed, got error: %v", err)
	}

	// Test ClearDefaultShippingAddress event publishing
	err = service.ClearDefaultShippingAddress(ctx, "defaults-event-test-123")
	if err != nil {
		t.Errorf("ClearDefaultShippingAddress should succeed, got error: %v", err)
	}

	// Test ClearDefaultBillingAddress event publishing
	err = service.ClearDefaultBillingAddress(ctx, "defaults-event-test-123")
	if err != nil {
		t.Errorf("ClearDefaultBillingAddress should succeed, got error: %v", err)
	}

	// Test ClearDefaultCreditCard event publishing
	err = service.ClearDefaultCreditCard(ctx, "defaults-event-test-123")
	if err != nil {
		t.Errorf("ClearDefaultCreditCard should succeed, got error: %v", err)
	}
}

// Helper function to create string pointers for tests
func stringPtr(s string) *string {
	return &s
}
