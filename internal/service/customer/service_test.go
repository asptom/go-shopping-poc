package customer

import (
	"context"
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
		ReadTopics:     []string{},
		Group:          "test-group",
		OutboxInterval: 5 * time.Second,
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
	customerService := NewCustomerService(mockRepo, mockConfig())

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
	customerService := NewCustomerService(mockRepo, mockConfig())

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
