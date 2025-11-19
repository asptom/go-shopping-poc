package entity

import (
	"time"

	"github.com/google/uuid"
)

// Customer represents a logical customer entity in the system.

type Customer struct {
	CustomerID               string           `json:"customer_id" db:"customer_id"` // UUID as string
	Username                 string           `json:"user_name" db:"user_name"`
	Email                    string           `json:"email,omitempty" db:"email"`
	FirstName                string           `json:"first_name,omitempty" db:"first_name"`
	LastName                 string           `json:"last_name,omitempty" db:"last_name"`
	Phone                    string           `json:"phone,omitempty" db:"phone"`
	DefaultShippingAddressID *uuid.UUID       `json:"default_shipping_address_id,omitempty" db:"default_shipping_address_id"`
	DefaultBillingAddressID  *uuid.UUID       `json:"default_billing_address_id,omitempty" db:"default_billing_address_id"`
	DefaultCreditCardID      *uuid.UUID       `json:"default_credit_card_id,omitempty" db:"default_credit_card_id"`
	CustomerSince            time.Time        `json:"customer_since" db:"customer_since"`
	CustomerStatus           string           `json:"customer_status" db:"customer_status"`
	StatusDateTime           time.Time        `json:"status_date_time" db:"status_date_time"`
	Addresses                []Address        `json:"addresses,omitempty"`
	CreditCards              []CreditCard     `json:"credit_cards,omitempty"`
	StatusHistory            []CustomerStatus `json:"status_history,omitempty"`
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
}

type CreditCard struct {
	CardID         uuid.UUID `json:"card_id" db:"card_id"`
	CustomerID     uuid.UUID `json:"customer_id" db:"customer_id"`
	CardType       string    `json:"card_type" db:"card_type"`
	CardNumber     string    `json:"card_number" db:"card_number"`
	CardHolderName string    `json:"card_holder_name" db:"card_holder_name"`
	CardExpires    string    `json:"card_expires" db:"card_expires"`
	CardCVV        string    `json:"card_cvv" db:"card_cvv"`
}

type CustomerStatus struct {
	ID         int64     `json:"-" db:"id"`
	CustomerID uuid.UUID `json:"-" db:"customer_id"`
	OldStatus  string    `json:"old_status" db:"old_status"`
	NewStatus  string    `json:"new_status" db:"new_status"`
	ChangedAt  time.Time `json:"changed_at" db:"changed_at"` // RFC3339 string
}
