package entity

import (
	"testing"
)

func TestCustomer_Validate(t *testing.T) {
	tests := []struct {
		name     string
		customer *Customer
		wantErr  bool
	}{
		{
			name: "valid customer",
			customer: &Customer{
				Username:       "testuser",
				Email:          "test@example.com",
				CustomerStatus: "active",
			},
			wantErr: false,
		},
		{
			name: "missing username",
			customer: &Customer{
				Email:          "test@example.com",
				CustomerStatus: "active",
			},
			wantErr: true,
		},
		{
			name: "username too short",
			customer: &Customer{
				Username:       "ab",
				Email:          "test@example.com",
				CustomerStatus: "active",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			customer: &Customer{
				Username:       "testuser",
				Email:          "invalid-email",
				CustomerStatus: "active",
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			customer: &Customer{
				Username:       "testuser",
				Email:          "test@example.com",
				CustomerStatus: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.customer.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Customer.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCustomer_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"active customer", "active", true},
		{"inactive customer", "inactive", false},
		{"suspended customer", "suspended", false},
		{"empty status", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Customer{CustomerStatus: tt.status}
			if got := c.IsActive(); got != tt.expected {
				t.Errorf("Customer.IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCustomer_FullName(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		lastName  string
		expected  string
	}{
		{"both names", "John", "Doe", "John Doe"},
		{"first name only", "John", "", "John"},
		{"last name only", "", "Doe", "Doe"},
		{"no names", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Customer{FirstName: tt.firstName, LastName: tt.lastName}
			if got := c.FullName(); got != tt.expected {
				t.Errorf("Customer.FullName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAddress_Validate(t *testing.T) {
	tests := []struct {
		name    string
		address *Address
		wantErr bool
	}{
		{
			name: "valid address",
			address: &Address{
				AddressType: "shipping",
				Address1:    "123 Main St",
				City:        "Test City",
				State:       "TS",
				Zip:         "12345",
			},
			wantErr: false,
		},
		{
			name: "missing address type",
			address: &Address{
				Address1: "123 Main St",
				City:     "Test City",
				State:    "TS",
				Zip:      "12345",
			},
			wantErr: true,
		},
		{
			name: "invalid address type",
			address: &Address{
				AddressType: "invalid",
				Address1:    "123 Main St",
				City:        "Test City",
				State:       "TS",
				Zip:         "12345",
			},
			wantErr: true,
		},
		{
			name: "missing address1",
			address: &Address{
				AddressType: "shipping",
				City:        "Test City",
				State:       "TS",
				Zip:         "12345",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.address.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Address.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddress_FullAddress(t *testing.T) {
	addr := &Address{
		Address1: "123 Main St",
		Address2: "Apt 4B",
		City:     "Test City",
		State:    "TS",
		Zip:      "12345",
	}

	expected := "123 Main St Apt 4B Test City, TS 12345"
	if got := addr.FullAddress(); got != expected {
		t.Errorf("Address.FullAddress() = %v, want %v", got, expected)
	}
}

func TestCreditCard_Validate(t *testing.T) {
	tests := []struct {
		name       string
		creditCard *CreditCard
		wantErr    bool
	}{
		{
			name: "valid credit card",
			creditCard: &CreditCard{
				CardType:       "visa",
				CardNumber:     "4111111111111111",
				CardHolderName: "John Doe",
				CardExpires:    "12/25",
				CardCVV:        "123",
			},
			wantErr: false,
		},
		{
			name: "invalid card type",
			creditCard: &CreditCard{
				CardType:       "invalid",
				CardNumber:     "4111111111111111",
				CardHolderName: "John Doe",
				CardExpires:    "12/25",
				CardCVV:        "123",
			},
			wantErr: true,
		},
		{
			name: "missing card number",
			creditCard: &CreditCard{
				CardType:       "visa",
				CardHolderName: "John Doe",
				CardExpires:    "12/25",
				CardCVV:        "123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creditCard.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreditCard.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreditCard_MaskedNumber(t *testing.T) {
	tests := []struct {
		name     string
		number   string
		expected string
	}{
		{"full card number", "4111111111111111", "****-****-****-1111"},
		{"short number", "123", "123"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &CreditCard{CardNumber: tt.number}
			if got := cc.MaskedNumber(); got != tt.expected {
				t.Errorf("CreditCard.MaskedNumber() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCustomerStatus_Validate(t *testing.T) {
	tests := []struct {
		name           string
		customerStatus *CustomerStatus
		wantErr        bool
	}{
		{
			name: "valid status change",
			customerStatus: &CustomerStatus{
				OldStatus: "active",
				NewStatus: "inactive",
			},
			wantErr: false,
		},
		{
			name: "invalid old status",
			customerStatus: &CustomerStatus{
				OldStatus: "invalid",
				NewStatus: "active",
			},
			wantErr: true,
		},
		{
			name: "invalid new status",
			customerStatus: &CustomerStatus{
				OldStatus: "active",
				NewStatus: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.customerStatus.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CustomerStatus.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
