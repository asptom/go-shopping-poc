package entity

import (
	"time"

	"github.com/google/uuid"
)

// Customer represents a logical customer entity in the system.

type Customer struct {
	CustomerID  string           `json:"customer_id" db:"customer_id"` // UUID as string
	Username    string           `json:"user_name" db:"user_name"`
	Email       string           `json:"email,omitempty" db:"email"`
	FirstName   string           `json:"first_name,omitempty" db:"first_name"`
	LastName    string           `json:"last_name,omitempty" db:"last_name"`
	Phone       string           `json:"phone,omitempty" db:"phone"`
	Addresses   []Address        `json:"addresses,omitempty"`
	CreditCards []CreditCard     `json:"credit_cards,omitempty"`
	Statuses    []CustomerStatus `json:"customer_statuses,omitempty"`
}

type CustomerBase struct {
	CustomerID uuid.UUID `json:"-" db:"customer_id"`
	Username   string    `json:"user_name" db:"user_name"`
	Email      string    `json:"email" db:"email"`
	FirstName  string    `json:"first_name" db:"first_name"`
	LastName   string    `json:"last_name" db:"last_name"`
	Phone      string    `json:"phone" db:"phone"`
}

type Address struct {
	AddressID   uuid.UUID `json:"address_id" db:"address_id"`
	CustomerID  uuid.UUID `json:"customer_id" db:"customer_id"`
	AddressType string    `json:"address_type" db:"address_type"`
	FirstName   string    `json:"first_name" db:"first_name"`
	LastName    string    `json:"last_name" db:"last_name"`
	Address1    string    `json:"address_1" db:"address_1"`
	Address2    string    `json:"address_2" db:"address_2"`
	City        string    `json:"city" db:"city"`
	State       string    `json:"state" db:"state"`
	Zip         string    `json:"zip" db:"zip"`
	IsDefault   bool      `json:"is_default" db:"is_default"`
}

type CreditCard struct {
	CardID         uuid.UUID `json:"card_id" db:"card_id"`
	CustomerID     uuid.UUID `json:"customer_id" db:"customer_id"`
	CardType       string    `json:"card_type" db:"card_type"`
	CardNumber     string    `json:"card_number" db:"card_number"`
	CardHolderName string    `json:"card_holder_name" db:"card_holder_name"`
	CardExpires    string    `json:"card_expires" db:"card_expires"`
	CardCVV        string    `json:"card_cvv" db:"card_cvv"`
	IsDefault      bool      `json:"is_default" db:"is_default"`
}

type CustomerStatus struct {
	ID             int64     `json:"-" db:"id"`
	CustomerID     uuid.UUID `json:"-" db:"customer_id"`
	CustomerStatus string    `json:"customer_status" db:"customer_status"`
	StatusDateTime time.Time `json:"status_date_time" db:"status_date_time"` // RFC3339 string
}
