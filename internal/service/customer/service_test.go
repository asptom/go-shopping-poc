package customer_test

import (
	"context"
	"testing"

	"go-shopping-poc/internal/service/customer"
)

// Create mock repository for testing
type mockCustomerRepository struct {
	createCustomerFunc  func(ctx context.Context, customer *customer.Customer) error
	getCustomerByIDFunc func(ctx context.Context, customerID string) (*customer.Customer, error)
	patchCustomerFunc   func(ctx context.Context, customerID string, patchData *customer.PatchCustomerRequest) error
}

func (m *mockCustomerRepository) InsertCustomer(ctx context.Context, cust *customer.Customer) error {
	return m.createCustomerFunc(ctx, cust)
}

func (m *mockCustomerRepository) GetCustomerByEmail(ctx context.Context, email string) (*customer.Customer, error) {
	return nil, nil
}

func (m *mockCustomerRepository) GetCustomerByID(ctx context.Context, customerID string) (*customer.Customer, error) {
	return m.getCustomerByIDFunc(ctx, customerID)
}

func (m *mockCustomerRepository) UpdateCustomer(ctx context.Context, customer *customer.Customer) error {
	return nil
}

func (m *mockCustomerRepository) PatchCustomer(ctx context.Context, customerID string, patchData *customer.PatchCustomerRequest) error {
	return m.patchCustomerFunc(ctx, customerID, patchData)
}

func (m *mockCustomerRepository) AddAddress(ctx context.Context, customerID string, addr *customer.Address) (*customer.Address, error) {
	return nil, nil
}

func (m *mockCustomerRepository) UpdateAddress(ctx context.Context, addressID string, addr *customer.Address) error {
	return nil
}

func (m *mockCustomerRepository) DeleteAddress(ctx context.Context, addressID string) error {
	return nil
}

func (m *mockCustomerRepository) AddCreditCard(ctx context.Context, customerID string, card *customer.CreditCard) (*customer.CreditCard, error) {
	return nil, nil
}

func (m *mockCustomerRepository) UpdateCreditCard(ctx context.Context, cardID string, card *customer.CreditCard) error {
	return nil
}

func (m *mockCustomerRepository) DeleteCreditCard(ctx context.Context, cardID string) error {
	return nil
}

func (m *mockCustomerRepository) UpdateDefaultShippingAddress(ctx context.Context, customerID string, addressID string) error {
	return nil
}

func (m *mockCustomerRepository) UpdateDefaultBillingAddress(ctx context.Context, customerID string, addressID string) error {
	return nil
}

func (m *mockCustomerRepository) UpdateDefaultCreditCard(ctx context.Context, customerID string, cardID string) error {
	return nil
}

func (m *mockCustomerRepository) ClearDefaultShippingAddress(ctx context.Context, customerID string) error {
	return nil
}

func (m *mockCustomerRepository) ClearDefaultBillingAddress(ctx context.Context, customerID string) error {
	return nil
}

func (m *mockCustomerRepository) ClearDefaultCreditCard(ctx context.Context, customerID string) error {
	return nil
}

func TestCustomerServiceCreateCustomerSuccess(t *testing.T) {
	t.Parallel()

	mockRepo := &mockCustomerRepository{
		createCustomerFunc: func(ctx context.Context, cust *customer.Customer) error {
			return nil
		},
	}

	mockInfra := &customer.CustomerInfrastructure{
		// Mock infrastructure components
	}

	svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

	testCustomer := &customer.Customer{
		Username: "testuser",
		Email:    "test@example.com",
	}

	err := svc.CreateCustomer(context.Background(), testCustomer)
	if err != nil {
		t.Errorf("CreateCustomer() failed: %v", err)
	}
}

func TestCustomerServicePatchCustomerSuccess(t *testing.T) {
	t.Parallel()

	existingCustomer := &customer.Customer{
		CustomerID: "123e4567-e89b-12d3-a456-426614174000",
		Username:   "oldusername",
		Email:      "old@example.com",
	}

	mockRepo := &mockCustomerRepository{
		getCustomerByIDFunc: func(ctx context.Context, customerID string) (*customer.Customer, error) {
			return existingCustomer, nil
		},
		patchCustomerFunc: func(ctx context.Context, customerID string, patchData *customer.PatchCustomerRequest) error {
			return nil
		},
	}

	mockInfra := &customer.CustomerInfrastructure{}
	svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

	patchData := &customer.PatchCustomerRequest{
		UserName: strPtr("newusername"),
	}

	err := svc.PatchCustomer(context.Background(), existingCustomer.CustomerID, patchData)
	if err != nil {
		t.Errorf("PatchCustomer() failed: %v", err)
	}
}

func TestCustomerServiceValidatePatchData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		patchData *customer.PatchCustomerRequest
		wantError bool
	}{
		{
			name:      "nil patch data",
			patchData: nil,
			wantError: true,
		},
		{
			name: "valid patch",
			patchData: &customer.PatchCustomerRequest{
				UserName: strPtr("newusername"),
			},
			wantError: false,
		},
		{
			name: "invalid UUID in address ID",
			patchData: &customer.PatchCustomerRequest{
				DefaultShippingAddressID: strPtr("invalid-uuid"),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockCustomerRepository{}
			mockInfra := &customer.CustomerInfrastructure{}
			svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

			err := svc.ValidatePatchData(tt.patchData)

			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePatchData() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCustomerServiceTransformAddressesFromPatch(t *testing.T) {
	t.Parallel()

	mockRepo := &mockCustomerRepository{}
	mockInfra := &customer.CustomerInfrastructure{}
	svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

	patchAddresses := []customer.PatchAddressRequest{
		{
			AddressType: "shipping",
			FirstName:   "John",
			LastName:    "Doe",
			Address1:    "123 Main St",
			City:        "Springfield",
			State:       "IL",
			Zip:         "62701",
		},
	}

	addresses := svc.TransformAddressesFromPatch(patchAddresses)

	if len(addresses) != 1 {
		t.Fatalf("expected 1 address, got %d", len(addresses))
	}

	if addresses[0].AddressType != "shipping" {
		t.Errorf("expected address_type shipping, got %s", addresses[0].AddressType)
	}
}

func TestCustomerServiceTransformCreditCardsFromPatch(t *testing.T) {
	t.Parallel()

	mockRepo := &mockCustomerRepository{}
	mockInfra := &customer.CustomerInfrastructure{}
	svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

	patchCards := []customer.PatchCreditCardRequest{
		{
			CardType:       "visa",
			CardNumber:     "4111111111111111",
			CardHolderName: "John Doe",
			CardExpires:    "12/25",
			CardCVV:        "123",
		},
	}

	cards := svc.TransformCreditCardsFromPatch(patchCards)

	if len(cards) != 1 {
		t.Fatalf("expected 1 credit card, got %d", len(cards))
	}

	if cards[0].CardType != "visa" {
		t.Errorf("expected card_type visa, got %s", cards[0].CardType)
	}
}

// Helper function to create string pointers
func strPtr(s string) *string {
	return &s
}
