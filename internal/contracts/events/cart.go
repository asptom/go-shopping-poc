package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CartEventType string

const (
	CartCreated    CartEventType = "cart.created"
	CartDeleted    CartEventType = "cart.deleted"
	CartCheckedOut CartEventType = "cart.checked_out"
)

// CartItemEventType defines cart item-specific event types
type CartItemEventType string

const (
	CartItemAdded     CartItemEventType = "cart.item.added"
	CartItemConfirmed CartItemEventType = "cart.item.confirmed"
	CartItemRejected  CartItemEventType = "cart.item.rejected"
)

// CartItemPayload contains cart item event data
type CartItemPayload struct {
	CartID       string  `json:"cart_id"`
	LineNumber   string  `json:"line_number"`
	ProductID    string  `json:"product_id"`
	Quantity     int     `json:"quantity"`
	ProductName  string  `json:"product_name,omitempty"`
	UnitPrice    float64 `json:"unit_price,omitempty"`
	ValidationID string  `json:"validation_id,omitempty"`
}

// CartItemEvent represents cart item lifecycle events
type CartItemEvent struct {
	ID        string            `json:"id"`
	EventType CartItemEventType `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Data      CartItemPayload   `json:"payload"`
}

// CartItemEventFactory implements EventFactory
type CartItemEventFactory struct{}

func (f CartItemEventFactory) FromJSON(data []byte) (CartItemEvent, error) {
	var event CartItemEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

// Event interface implementations
func (e CartItemEvent) Type() string            { return string(e.EventType) }
func (e CartItemEvent) Topic() string           { return "CartEvents" }
func (e CartItemEvent) Payload() any            { return e.Data }
func (e CartItemEvent) ToJSON() ([]byte, error) { return json.Marshal(e) }
func (e CartItemEvent) GetEntityID() string     { return e.Data.CartID }
func (e CartItemEvent) GetResourceID() string   { return e.ID }

func NewCartItemAddedEvent(cartID, lineNumber, productID string, quantity int, validationID string) *CartItemEvent {
	return &CartItemEvent{
		ID:        uuid.New().String(),
		EventType: CartItemAdded,
		Timestamp: time.Now(),
		Data: CartItemPayload{
			CartID:       cartID,
			LineNumber:   lineNumber,
			ProductID:    productID,
			Quantity:     quantity,
			ValidationID: validationID,
		},
	}
}

func NewCartItemConfirmedEvent(cartID, lineNumber, productID, productName string, unitPrice float64, quantity int) *CartItemEvent {
	return &CartItemEvent{
		ID:        uuid.New().String(),
		EventType: CartItemConfirmed,
		Timestamp: time.Now(),
		Data: CartItemPayload{
			CartID:      cartID,
			LineNumber:  lineNumber,
			ProductID:   productID,
			ProductName: productName,
			UnitPrice:   unitPrice,
			Quantity:    quantity,
		},
	}
}

func NewCartItemRejectedEvent(cartID, lineNumber, productID, reason string) *CartItemEvent {
	return &CartItemEvent{
		ID:        uuid.New().String(),
		EventType: CartItemRejected,
		Timestamp: time.Now(),
		Data: CartItemPayload{
			CartID:     cartID,
			LineNumber: lineNumber,
			ProductID:  productID,
		},
	}
}

type CartEventPayload struct {
	CartID     string            `json:"cart_id"`
	CustomerID *string           `json:"customer_id,omitempty"`
	TotalPrice float64           `json:"total_price,omitempty"`
	ItemCount  int               `json:"item_count,omitempty"`
	Details    map[string]string `json:"details,omitempty"`

	CartSnapshot *CartSnapshot `json:"cart_snapshot,omitempty"`
}

type CartSnapshot struct {
	Currency   string            `json:"currency"`
	NetPrice   float64           `json:"net_price"`
	Tax        float64           `json:"tax"`
	Shipping   float64           `json:"shipping"`
	TotalPrice float64           `json:"total_price"`
	CustomerID *string           `json:"customer_id,omitempty"`
	Contact    *SnapshotContact  `json:"contact,omitempty"`
	CreditCard *SnapshotPayment  `json:"credit_card,omitempty"`
	Addresses  []SnapshotAddress `json:"addresses,omitempty"`
	Items      []SnapshotItem    `json:"items,omitempty"`
}

type SnapshotContact struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
}

type SnapshotPayment struct {
	CardType       string `json:"card_type"`
	CardNumber     string `json:"card_number"`
	CardHolderName string `json:"card_holder_name"`
	CardExpires    string `json:"card_expires"`
	CardCVV        string `json:"card_cvv"`
}

type SnapshotAddress struct {
	AddressType string `json:"address_type"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Address1    string `json:"address_1"`
	Address2    string `json:"address_2"`
	City        string `json:"city"`
	State       string `json:"state"`
	Zip         string `json:"zip"`
}

type SnapshotItem struct {
	LineNumber  string  `json:"line_number"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	UnitPrice   float64 `json:"unit_price"`
	Quantity    int     `json:"quantity"`
	TotalPrice  float64 `json:"total_price"`
}

type CartEvent struct {
	ID           string           `json:"id"`
	EventType    CartEventType    `json:"type"`
	Timestamp    time.Time        `json:"timestamp"`
	EventPayload CartEventPayload `json:"payload"`
}

type CartEventFactory struct{}

func (f CartEventFactory) FromJSON(data []byte) (CartEvent, error) {
	var event CartEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

func (e CartEvent) Type() string            { return string(e.EventType) }
func (e CartEvent) Topic() string           { return "CartEvents" }
func (e CartEvent) Payload() any            { return e.EventPayload }
func (e CartEvent) ToJSON() ([]byte, error) { return json.Marshal(e) }
func (e CartEvent) GetEntityID() string     { return e.EventPayload.CartID }
func (e CartEvent) GetResourceID() string   { return e.ID }

func NewCartEvent(cartID string, eventType CartEventType, customerID *string, totalPrice float64, itemCount int, details map[string]string) *CartEvent {
	payload := CartEventPayload{
		CartID:     cartID,
		CustomerID: customerID,
		TotalPrice: totalPrice,
		ItemCount:  itemCount,
		Details:    details,
	}

	return &CartEvent{
		ID:           uuid.New().String(),
		EventType:    eventType,
		Timestamp:    time.Now(),
		EventPayload: payload,
	}
}

func NewCartCreatedEvent(cartID string, customerID *string) *CartEvent {
	return NewCartEvent(cartID, CartCreated, customerID, 0, 0, nil)
}

func NewCartDeletedEvent(cartID string, customerID *string) *CartEvent {
	return NewCartEvent(cartID, CartDeleted, customerID, 0, 0, nil)
}

func NewCartCheckedOutEvent(cartID string, customerID *string, totalPrice float64, itemCount int) *CartEvent {
	return NewCartEvent(cartID, CartCheckedOut, customerID, totalPrice, itemCount, nil)
}

func NewCartCheckedOutEventWithSnapshot(cartID string, customerID *string, snapshot *CartSnapshot) *CartEvent {
	payload := CartEventPayload{
		CartID:       cartID,
		CustomerID:   customerID,
		TotalPrice:   snapshot.TotalPrice,
		ItemCount:    len(snapshot.Items),
		CartSnapshot: snapshot,
	}

	return &CartEvent{
		ID:           uuid.New().String(),
		EventType:    CartCheckedOut,
		Timestamp:    time.Now(),
		EventPayload: payload,
	}
}
