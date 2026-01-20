package customer_test

import (
	"testing"

	"go-shopping-poc/internal/service/customer"
)

func TestCustomerValidateSuccess(t *testing.T) {
	t.Parallel()

	cust := &customer.Customer{
		Username:       "testuser",
		Email:          "test@example.com",
		CustomerStatus: "active",
	}

	if err := cust.Validate(); err != nil {
		t.Errorf("valid customer should pass validation: %v", err)
	}
}

func TestCustomerValidateUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		username  string
		wantError bool
	}{
		{"empty username", "", true},
		{"whitespace only", "   ", true},
		{"too short", "ab", true},
		{"valid username", "testuser", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cust := &customer.Customer{Username: tt.username}
			err := cust.Validate()

			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCustomerValidateEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		email     string
		wantError bool
	}{
		{"missing @", "testexample.com", true},
		{"valid email", "test@example.com", false},
		{"empty email", "", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cust := &customer.Customer{
				Username: "testuser",
				Email:    tt.email,
			}
			err := cust.Validate()

			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCustomerValidateStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    string
		wantError bool
	}{
		{"active status", "active", false},
		{"inactive status", "inactive", false},
		{"suspended status", "suspended", false},
		{"invalid status", "pending", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cust := &customer.Customer{
				Username:       "testuser",
				CustomerStatus: tt.status,
			}
			err := cust.Validate()

			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCustomerIsActive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"active customer", "active", true},
		{"inactive customer", "inactive", false},
		{"suspended customer", "suspended", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cust := &customer.Customer{CustomerStatus: tt.status}

			if got := cust.IsActive(); got != tt.expected {
				t.Errorf("IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCustomerFullName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		first    string
		last     string
		expected string
	}{
		{"both names", "John", "Doe", "John Doe"},
		{"first only", "John", "", "John"},
		{"last only", "", "Doe", "Doe"},
		{"no names", "", "", ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cust := &customer.Customer{
				FirstName: tt.first,
				LastName:  tt.last,
			}

			if got := cust.FullName(); got != tt.expected {
				t.Errorf("FullName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAddressValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		address   *customer.Address
		wantError bool
	}{
		{
			name: "valid shipping address",
			address: &customer.Address{
				AddressType: "shipping",
				Address1:    "123 Main St",
				City:        "Springfield",
				State:       "IL",
				Zip:         "62701",
			},
			wantError: false,
		},
		{
			name: "valid billing address",
			address: &customer.Address{
				AddressType: "billing",
				Address1:    "456 Oak Ave",
				City:        "Springfield",
				State:       "IL",
				Zip:         "62702",
			},
			wantError: false,
		},
		{
			name: "invalid address type",
			address: &customer.Address{
				AddressType: "invalid",
				Address1:    "123 Main St",
				City:        "Springfield",
				State:       "IL",
				Zip:         "62701",
			},
			wantError: true,
		},
		{
			name: "missing address line 1",
			address: &customer.Address{
				AddressType: "shipping",
				City:        "Springfield",
				State:       "IL",
				Zip:         "62701",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.address.Validate()

			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCreditCardValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		card      *customer.CreditCard
		wantError bool
	}{
		{
			name: "valid visa card",
			card: &customer.CreditCard{
				CardType:       "visa",
				CardNumber:     "4111111111111111",
				CardHolderName: "John Doe",
				CardExpires:    "12/25",
				CardCVV:        "123",
			},
			wantError: false,
		},
		{
			name: "valid mastercard",
			card: &customer.CreditCard{
				CardType:       "mastercard",
				CardNumber:     "5555555555555555",
				CardHolderName: "Jane Smith",
				CardExpires:    "06/24",
				CardCVV:        "456",
			},
			wantError: false,
		},
		{
			name: "valid amex",
			card: &customer.CreditCard{
				CardType:       "amex",
				CardNumber:     "371449635398431",
				CardHolderName: "Bob Johnson",
				CardExpires:    "09/26",
				CardCVV:        "789",
			},
			wantError: false,
		},
		{
			name: "valid discover",
			card: &customer.CreditCard{
				CardType:       "discover",
				CardNumber:     "6011111111111117",
				CardHolderName: "Alice Brown",
				CardExpires:    "03/25",
				CardCVV:        "321",
			},
			wantError: false,
		},
		{
			name: "invalid card type",
			card: &customer.CreditCard{
				CardType:       "unionpay",
				CardNumber:     "6221260069260074",
				CardHolderName: "Test User",
				CardExpires:    "12/25",
				CardCVV:        "123",
			},
			wantError: true,
		},
		{
			name: "missing card type",
			card: &customer.CreditCard{
				CardNumber:     "4111111111111111",
				CardHolderName: "John Doe",
				CardExpires:    "12/25",
				CardCVV:        "123",
			},
			wantError: true,
		},
		{
			name: "missing card number",
			card: &customer.CreditCard{
				CardType:       "visa",
				CardHolderName: "John Doe",
				CardExpires:    "12/25",
				CardCVV:        "123",
			},
			wantError: true,
		},
		{
			name: "missing card holder name",
			card: &customer.CreditCard{
				CardType:    "visa",
				CardNumber:  "4111111111111111",
				CardExpires: "12/25",
				CardCVV:     "123",
			},
			wantError: true,
		},
		{
			name: "missing card expiration",
			card: &customer.CreditCard{
				CardType:       "visa",
				CardNumber:     "4111111111111111",
				CardHolderName: "John Doe",
				CardCVV:        "123",
			},
			wantError: true,
		},
		{
			name: "missing card CVV",
			card: &customer.CreditCard{
				CardType:       "visa",
				CardNumber:     "4111111111111111",
				CardHolderName: "John Doe",
				CardExpires:    "12/25",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.card.Validate()

			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCreditCardMaskedNumber(t *testing.T) {
	t.Parallel()

	card := &customer.CreditCard{
		CardNumber: "4111111111111111",
	}

	masked := card.MaskedNumber()
	expected := "****-****-****-1111"

	if masked != expected {
		t.Errorf("MaskedNumber() = %v, want %v", masked, expected)
	}
}

func TestAddressFullAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		address  *customer.Address
		expected string
	}{
		{
			name: "address with address2",
			address: &customer.Address{
				Address1: "123 Main St",
				Address2: "Apt 4B",
				City:     "Springfield",
				State:    "IL",
				Zip:      "62701",
			},
			expected: "123 Main St Apt 4B Springfield, IL 62701",
		},
		{
			name: "address without address2",
			address: &customer.Address{
				Address1: "456 Oak Ave",
				City:     "Springfield",
				State:    "IL",
				Zip:      "62702",
			},
			expected: "456 Oak Ave Springfield, IL 62702",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := tt.address.FullAddress()

			if got != tt.expected {
				t.Errorf("FullAddress() = %v, want %v", got, tt.expected)
			}
		})
	}
}
