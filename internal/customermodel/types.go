package customermodel

type Customer struct {
	CustomerID  string           `json:"customerId"` // UUID as string
	Username    string           `json:"username"`
	Email       string           `json:"email,omitempty"`
	FirstName   string           `json:"firstName,omitempty"`
	LastName    string           `json:"lastName,omitempty"`
	Phone       string           `json:"phone,omitempty"`
	Addresses   []Address        `json:"addresses,omitempty"`
	CreditCards []CreditCard     `json:"creditCards,omitempty"`
	Statuses    []CustomerStatus `json:"customerStatus,omitempty"`
}

// Address represents the API request/response for a customer address.
type Address struct {
	AddressType string `json:"addressType"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Address1    string `json:"address1"`
	Address2    string `json:"address2,omitempty"`
	City        string `json:"city"`
	State       string `json:"state"`
	Zip         string `json:"zip"`
	IsDefault   bool   `json:"isDefault"`
}

// CreditCard represents the API request/response for a customer credit card.
type CreditCard struct {
	CardType       string `json:"cardType"`
	CardNumber     string `json:"cardNumber"`
	CardHolderName string `json:"cardHolderName"`
	CardExpires    string `json:"cardExpires"`
	CardCVV        string `json:"cardCVV"`
	IsDefault      bool   `json:"isDefault"`
}

// CustomerStatus represents the API request/response for a customer's status.
type CustomerStatus struct {
	Status         string `json:"status"`
	StatusDateTime string `json:"statusDateTime"` // RFC3339 string
}
