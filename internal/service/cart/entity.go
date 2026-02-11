package cart

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Cart represents a shopping cart entity
type Cart struct {
	CartID        uuid.UUID  `json:"cart_id" db:"cart_id"`
	CustomerID    *uuid.UUID `json:"customer_id,omitempty" db:"customer_id"`
	ContactID     *int64     `json:"-" db:"contact_id"`
	CreditCardID  *int64     `json:"-" db:"credit_card_id"`
	CurrentStatus string     `json:"current_status" db:"current_status"`
	Currency      string     `json:"currency" db:"currency"`
	NetPrice      float64    `json:"net_price" db:"net_price"`
	Tax           float64    `json:"tax" db:"tax"`
	Shipping      float64    `json:"shipping" db:"shipping"`
	TotalPrice    float64    `json:"total_price" db:"total_price"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	Version       int        `json:"version" db:"version"`

	Contact       *Contact     `json:"contact,omitempty"`
	Addresses     []Address    `json:"addresses,omitempty"`
	CreditCard    *CreditCard  `json:"credit_card,omitempty"`
	Items         []CartItem   `json:"items,omitempty"`
	StatusHistory []CartStatus `json:"status_history,omitempty"`
}

// Validate performs domain validation
func (c *Cart) Validate() error {
	if c.Currency == "" {
		return errors.New("currency is required")
	}
	validStatuses := []string{"active", "checked_out", "completed", "cancelled"}
	if !contains(validStatuses, c.CurrentStatus) {
		return errors.New("invalid cart status")
	}
	return nil
}

// CalculateTotals computes all cart totals (3% tax, $0 shipping)
func (c *Cart) CalculateTotals() {
	c.NetPrice = 0
	for i := range c.Items {
		c.Items[i].CalculateLineTotal()
		c.NetPrice += c.Items[i].TotalPrice
	}
	c.Tax = c.calculateTax()
	c.Shipping = c.calculateShipping()
	c.TotalPrice = c.NetPrice + c.Tax + c.Shipping
}

func (c *Cart) calculateTax() float64 {
	return c.NetPrice * 0.03
}

func (c *Cart) calculateShipping() float64 {
	return 0.0
}

// CanCheckout validates cart is ready for checkout
func (c *Cart) CanCheckout() error {
	if c.CurrentStatus != "active" {
		return errors.New("cart must be active to checkout")
	}
	if len(c.Items) == 0 {
		return errors.New("cart must have at least one item")
	}
	if c.Contact == nil {
		return errors.New("contact information required")
	}
	if c.CreditCard == nil {
		return errors.New("payment method required")
	}
	return nil
}

// SetStatus updates cart status with validation
func (c *Cart) SetStatus(newStatus string) error {
	validTransitions := map[string][]string{
		"active":      {"checked_out", "cancelled"},
		"checked_out": {"completed", "cancelled"},
		"completed":   {},
		"cancelled":   {},
	}

	allowed, ok := validTransitions[c.CurrentStatus]
	if !ok {
		return fmt.Errorf("unknown current status: %s", c.CurrentStatus)
	}

	if !contains(allowed, newStatus) {
		return fmt.Errorf("invalid status transition from %s to %s", c.CurrentStatus, newStatus)
	}

	c.CurrentStatus = newStatus
	c.StatusHistory = append(c.StatusHistory, CartStatus{
		CartID:    c.CartID,
		Status:    newStatus,
		ChangedAt: time.Now(),
	})
	return nil
}

// CartItem represents an item in the cart
type CartItem struct {
	ID          int64     `json:"id" db:"id"`
	CartID      uuid.UUID `json:"cart_id" db:"cart_id"`
	LineNumber  string    `json:"line_number" db:"line_number"`
	ProductID   string    `json:"product_id" db:"product_id"`
	ProductName string    `json:"product_name" db:"product_name"`
	UnitPrice   float64   `json:"unit_price" db:"unit_price"`
	Quantity    int       `json:"quantity" db:"quantity"`
	TotalPrice  float64   `json:"total_price" db:"total_price"`
}

func (ci *CartItem) CalculateLineTotal() {
	ci.TotalPrice = float64(ci.Quantity) * ci.UnitPrice
}

func (ci *CartItem) Validate() error {
	if ci.ProductID == "" {
		return errors.New("product_id is required")
	}
	if ci.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	if ci.UnitPrice < 0 {
		return errors.New("unit_price cannot be negative")
	}
	return nil
}

// Contact represents contact information
type Contact struct {
	ID        int64     `json:"id" db:"id"`
	CartID    uuid.UUID `json:"cart_id" db:"cart_id"`
	Email     string    `json:"email" db:"email"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	Phone     string    `json:"phone" db:"phone"`
}

func (c *Contact) Validate() error {
	if strings.TrimSpace(c.Email) == "" {
		return errors.New("email is required")
	}
	if !strings.Contains(c.Email, "@") {
		return errors.New("invalid email format")
	}
	if strings.TrimSpace(c.FirstName) == "" {
		return errors.New("first_name is required")
	}
	if strings.TrimSpace(c.LastName) == "" {
		return errors.New("last_name is required")
	}
	if strings.TrimSpace(c.Phone) == "" {
		return errors.New("phone is required")
	}
	return nil
}

// Address represents a shipping or billing address
type Address struct {
	ID          int64     `json:"id" db:"id"`
	CartID      uuid.UUID `json:"cart_id" db:"cart_id"`
	AddressType string    `json:"address_type" db:"address_type"`
	FirstName   string    `json:"first_name" db:"first_name"`
	LastName    string    `json:"last_name" db:"last_name"`
	Address1    string    `json:"address_1" db:"address_1"`
	Address2    string    `json:"address_2" db:"address_2"`
	City        string    `json:"city" db:"city"`
	State       string    `json:"state" db:"state"`
	Zip         string    `json:"zip" db:"zip"`
}

func (a *Address) Validate() error {
	validTypes := []string{"shipping", "billing"}
	if !contains(validTypes, a.AddressType) {
		return errors.New("address_type must be shipping or billing")
	}
	if strings.TrimSpace(a.Address1) == "" {
		return errors.New("address_1 is required")
	}
	if strings.TrimSpace(a.City) == "" {
		return errors.New("city is required")
	}
	if strings.TrimSpace(a.State) == "" {
		return errors.New("state is required")
	}
	if strings.TrimSpace(a.Zip) == "" {
		return errors.New("zip is required")
	}
	return nil
}

// CreditCard represents payment information
type CreditCard struct {
	ID             int64     `json:"id" db:"id"`
	CartID         uuid.UUID `json:"cart_id" db:"cart_id"`
	CardType       string    `json:"card_type" db:"card_type"`
	CardNumber     string    `json:"card_number" db:"card_number"`
	CardHolderName string    `json:"card_holder_name" db:"card_holder_name"`
	CardExpires    string    `json:"card_expires" db:"card_expires"`
	CardCVV        string    `json:"card_cvv" db:"card_cvv"`
}

func (cc *CreditCard) Validate() error {
	if strings.TrimSpace(cc.CardNumber) == "" {
		return errors.New("card_number is required")
	}
	if strings.TrimSpace(cc.CardHolderName) == "" {
		return errors.New("card_holder_name is required")
	}
	if strings.TrimSpace(cc.CardExpires) == "" {
		return errors.New("card_expires is required")
	}
	if strings.TrimSpace(cc.CardCVV) == "" {
		return errors.New("card_cvv is required")
	}
	return nil
}

func (cc *CreditCard) MaskedNumber() string {
	if len(cc.CardNumber) < 4 {
		return cc.CardNumber
	}
	return "****-****-****-" + cc.CardNumber[len(cc.CardNumber)-4:]
}

// CartStatus represents a cart status history entry
type CartStatus struct {
	ID        int64     `json:"id" db:"id"`
	CartID    uuid.UUID `json:"cart_id" db:"cart_id"`
	Status    string    `json:"cart_status" db:"cart_status"`
	ChangedAt time.Time `json:"status_date_time" db:"status_date_time"`
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
