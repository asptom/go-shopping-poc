package entity

import (
	"time"

	"github.com/google/uuid"
)

// Customer represents a logical customer entity in the system.

type Customer struct {
	CustomerID  string           `json:"customerid"` // UUID as string
	Username    string           `json:"username"`
	Email       string           `json:"email,omitempty"`
	FirstName   string           `json:"firstName,omitempty"`
	LastName    string           `json:"lastName,omitempty"`
	Phone       string           `json:"phone,omitempty"`
	Addresses   []Address        `json:"addresses,omitempty"`
	CreditCards []CreditCard     `json:"creditCards,omitempty"`
	Statuses    []CustomerStatus `json:"customerStatus,omitempty"`
}

type CustomerBase struct {
	CustomerID uuid.UUID `db:"customer_id"`
	Username   string    `db:"user_name"`
	Email      string    `db:"email"`
	FirstName  string    `db:"first_name"`
	LastName   string    `db:"last_name"`
	Phone      string    `db:"phone"`
}

type Address struct {
	ID          int64     `json:"id"`
	CustomerID  uuid.UUID `json:"customerId"`
	AddressType string    `json:"addressType"`
	FirstName   string    `json:"firstName"`
	LastName    string    `json:"lastName"`
	Address1    string    `json:"address_1"`
	Address2    string    `json:"address_2"`
	City        string    `json:"city"`
	State       string    `json:"state"`
	Zip         string    `json:"zip"`
	Isdefault   bool      `json:"isdefault"`
}

type CreditCard struct {
	ID             int64     `json:"id"`
	CustomerID     uuid.UUID `json:"customerId"`
	CardType       string    `json:"cardType"`
	CardNumber     string    `json:"cardNumber"`
	CardHolderName string    `json:"cardHolderName"`
	CardExpires    string    `json:"cardExpires"`
	CardCVV        string    `json:"cardCVV"`
	Isdefault      bool      `json:"isdefault"`
}

type CustomerStatus struct {
	ID             int64     `json:"id"`
	CustomerID     uuid.UUID `json:"customerId"`
	CustomerStatus string    `json:"customerStatus"`
	StatusDateTime time.Time `json:"statusDateTime" db:"statusDateTime"` // RFC3339 string
}
