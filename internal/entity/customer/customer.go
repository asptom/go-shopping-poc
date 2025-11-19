// Package entity defines the domain entities for the customer bounded context.
//
// This package contains the core business objects (entities) that represent
// customers, addresses, credit cards, and related domain concepts. Entities
// include validation methods and business logic.
package entity

import (
	"errors"
	"strings"
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

// Validate performs domain validation on the Customer entity
func (c *Customer) Validate() error {
	if strings.TrimSpace(c.Username) == "" {
		return errors.New("username is required")
	}
	if len(c.Username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if c.Email != "" && !strings.Contains(c.Email, "@") {
		return errors.New("email must be valid format")
	}
	if c.CustomerStatus != "" && c.CustomerStatus != "active" && c.CustomerStatus != "inactive" && c.CustomerStatus != "suspended" {
		return errors.New("customer status must be active, inactive, or suspended")
	}
	return nil
}

// IsActive returns true if the customer is in active status
func (c *Customer) IsActive() bool {
	return c.CustomerStatus == "active"
}

// FullName returns the customer's full name
func (c *Customer) FullName() string {
	parts := []string{}
	if c.FirstName != "" {
		parts = append(parts, c.FirstName)
	}
	if c.LastName != "" {
		parts = append(parts, c.LastName)
	}
	return strings.Join(parts, " ")
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

// Validate performs domain validation on the Address entity
func (a *Address) Validate() error {
	if strings.TrimSpace(a.AddressType) == "" {
		return errors.New("address type is required")
	}
	if a.AddressType != "shipping" && a.AddressType != "billing" {
		return errors.New("address type must be shipping or billing")
	}
	if strings.TrimSpace(a.Address1) == "" {
		return errors.New("address line 1 is required")
	}
	if strings.TrimSpace(a.City) == "" {
		return errors.New("city is required")
	}
	if strings.TrimSpace(a.State) == "" {
		return errors.New("state is required")
	}
	if strings.TrimSpace(a.Zip) == "" {
		return errors.New("zip code is required")
	}
	return nil
}

// FullAddress returns a formatted full address string
func (a *Address) FullAddress() string {
	parts := []string{a.Address1}
	if a.Address2 != "" {
		parts = append(parts, a.Address2)
	}
	parts = append(parts, a.City+",", a.State, a.Zip)
	return strings.Join(parts, " ")
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

// Validate performs domain validation on the CreditCard entity
func (c *CreditCard) Validate() error {
	if strings.TrimSpace(c.CardType) == "" {
		return errors.New("card type is required")
	}
	validTypes := []string{"visa", "mastercard", "amex", "discover"}
	isValidType := false
	for _, t := range validTypes {
		if c.CardType == t {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("card type must be visa, mastercard, amex, or discover")
	}
	if strings.TrimSpace(c.CardNumber) == "" {
		return errors.New("card number is required")
	}
	if strings.TrimSpace(c.CardHolderName) == "" {
		return errors.New("card holder name is required")
	}
	if strings.TrimSpace(c.CardExpires) == "" {
		return errors.New("card expiration is required")
	}
	if strings.TrimSpace(c.CardCVV) == "" {
		return errors.New("card CVV is required")
	}
	return nil
}

// MaskedNumber returns a masked version of the card number for display
func (c *CreditCard) MaskedNumber() string {
	if len(c.CardNumber) < 4 {
		return c.CardNumber
	}
	return "****-****-****-" + c.CardNumber[len(c.CardNumber)-4:]
}

type CustomerStatus struct {
	ID         int64     `json:"-" db:"id"`
	CustomerID uuid.UUID `json:"-" db:"customer_id"`
	OldStatus  string    `json:"old_status" db:"old_status"`
	NewStatus  string    `json:"new_status" db:"new_status"`
	ChangedAt  time.Time `json:"changed_at" db:"changed_at"` // RFC3339 string
}

// Validate performs domain validation on the CustomerStatus entity
func (cs *CustomerStatus) Validate() error {
	if cs.OldStatus == "" && cs.NewStatus == "" {
		return errors.New("at least one of old_status or new_status must be provided")
	}
	validStatuses := []string{"active", "inactive", "suspended"}
	if cs.OldStatus != "" {
		if !contains(validStatuses, cs.OldStatus) {
			return errors.New("old_status must be active, inactive, or suspended")
		}
	}
	if cs.NewStatus != "" {
		if !contains(validStatuses, cs.NewStatus) {
			return errors.New("new_status must be active, inactive, or suspended")
		}
	}
	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
