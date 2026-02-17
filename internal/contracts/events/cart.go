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
