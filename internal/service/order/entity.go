package order

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Order struct {
	OrderID       uuid.UUID  `json:"order_id" db:"order_id"`
	OrderNumber   string     `json:"order_number" db:"order_number"`
	CartID        uuid.UUID  `json:"cart_id" db:"cart_id"`
	CustomerID    *uuid.UUID `json:"customer_id,omitempty" db:"customer_id"`
	ContactID     int64      `json:"-" db:"contact_id"`
	CreditCardID  int64      `json:"-" db:"credit_card_id"`
	Currency      string     `json:"currency" db:"currency"`
	NetPrice      float64    `json:"net_price" db:"net_price"`
	Tax           float64    `json:"tax" db:"tax"`
	Shipping      float64    `json:"shipping" db:"shipping"`
	TotalPrice    float64    `json:"total_price" db:"total_price"`
	CurrentStatus string     `json:"current_status" db:"current_status"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`

	Contact       *Contact      `json:"contact,omitempty"`
	CreditCard    *CreditCard   `json:"credit_card,omitempty"`
	Addresses     []Address     `json:"addresses,omitempty"`
	Items         []OrderItem   `json:"items,omitempty"`
	StatusHistory []OrderStatus `json:"status_history,omitempty"`
}

func (o *Order) Validate() error {
	validStatuses := []string{"created", "confirmed", "processing", "shipped", "delivered", "cancelled", "refunded"}
	if !contains(validStatuses, o.CurrentStatus) {
		return errors.New("invalid order status")
	}
	if o.Currency == "" {
		return errors.New("currency is required")
	}
	return nil
}

func (o *Order) CanCancel() error {
	nonCancellableStatuses := []string{"shipped", "delivered", "cancelled", "refunded"}
	if contains(nonCancellableStatuses, o.CurrentStatus) {
		return fmt.Errorf("order cannot be cancelled in current status: %s", o.CurrentStatus)
	}
	return nil
}

func (o *Order) SetStatus(newStatus string) error {
	validTransitions := map[string][]string{
		"created":    {"confirmed", "cancelled"},
		"confirmed":  {"processing", "cancelled"},
		"processing": {"shipped", "cancelled"},
		"shipped":    {"delivered"},
		"delivered":  {"refunded"},
		"cancelled":  {},
		"refunded":   {},
	}

	allowed, ok := validTransitions[o.CurrentStatus]
	if !ok {
		return fmt.Errorf("unknown current status: %s", o.CurrentStatus)
	}

	if !contains(allowed, newStatus) {
		return fmt.Errorf("invalid status transition from %s to %s", o.CurrentStatus, newStatus)
	}

	o.CurrentStatus = newStatus
	o.StatusHistory = append(o.StatusHistory, OrderStatus{
		OrderID:   o.OrderID,
		Status:    newStatus,
		ChangedAt: time.Now(),
	})
	return nil
}

type OrderItem struct {
	ID             int64     `json:"id" db:"id"`
	OrderID        uuid.UUID `json:"order_id" db:"order_id"`
	LineNumber     int       `json:"line_number" db:"line_number"`
	ProductID      string    `json:"product_id" db:"product_id"`
	ProductName    string    `json:"product_name" db:"product_name"`
	UnitPrice      float64   `json:"unit_price" db:"unit_price"`
	Quantity       int       `json:"quantity" db:"quantity"`
	TotalPrice     float64   `json:"total_price" db:"total_price"`
	ItemStatus     string    `json:"item_status" db:"item_status"`
	ItemStatusDate time.Time `json:"item_status_date_time" db:"item_status_date_time"`
}

func (oi *OrderItem) CalculateLineTotal() {
	oi.TotalPrice = float64(oi.Quantity) * oi.UnitPrice
}

type Contact struct {
	ID        int64     `json:"id" db:"id"`
	OrderID   uuid.UUID `json:"order_id" db:"order_id"`
	Email     string    `json:"email" db:"email"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	Phone     string    `json:"phone" db:"phone"`
}

type Address struct {
	ID          int64     `json:"id" db:"id"`
	OrderID     uuid.UUID `json:"order_id" db:"order_id"`
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
	ID             int64     `json:"id" db:"id"`
	OrderID        uuid.UUID `json:"order_id" db:"order_id"`
	CardType       string    `json:"card_type" db:"card_type"`
	CardNumber     string    `json:"card_number" db:"card_number"`
	CardHolderName string    `json:"card_holder_name" db:"card_holder_name"`
	CardExpires    string    `json:"card_expires" db:"card_expires"`
	CardCVV        string    `json:"card_cvv" db:"card_cvv"`
}

func (cc *CreditCard) MaskedNumber() string {
	if len(cc.CardNumber) < 4 {
		return cc.CardNumber
	}
	return "****-****-****-" + cc.CardNumber[len(cc.CardNumber)-4:]
}

type OrderStatus struct {
	ID        int64     `json:"id" db:"id"`
	OrderID   uuid.UUID `json:"order_id" db:"order_id"`
	Status    string    `json:"order_status" db:"order_status"`
	ChangedAt time.Time `json:"status_date_time" db:"status_date_time"`
	Notes     string    `json:"notes,omitempty" db:"notes"`
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

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

	Contact    *Contact    `json:"contact,omitempty"`
	Addresses  []Address   `json:"addresses,omitempty"`
	CreditCard *CreditCard `json:"credit_card,omitempty"`
	Items      []CartItem  `json:"items,omitempty"`
}

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
