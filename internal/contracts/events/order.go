package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OrderEventType string

const (
	OrderCreated OrderEventType = "order.created"
	OrderUpdated OrderEventType = "order.updated"
	OrderDeleted OrderEventType = "order.deleted"
)

type OrderEventPayload struct {
	OrderID     string  `json:"order_id"`
	OrderNumber string  `json:"order_number"`
	CartID      string  `json:"cart_id"`
	CustomerID  *string `json:"customer_id,omitempty"`
	Total       float64 `json:"total"`
}

type OrderEvent struct {
	ID        string            `json:"id"`
	EventType OrderEventType    `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Data      OrderEventPayload `json:"payload"`
}

type OrderEventFactory struct{}

func (f OrderEventFactory) FromJSON(data []byte) (OrderEvent, error) {
	var event OrderEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

func NewOrderEvent(orderID string, orderNumber string, t OrderEventType, cartID string, customerID *string, total float64) *OrderEvent {
	payload := OrderEventPayload{
		OrderID:     orderID,
		OrderNumber: orderNumber,
		CartID:      cartID,
		CustomerID:  customerID,
		Total:       total,
	}

	return &OrderEvent{
		ID:        uuid.New().String(),
		EventType: t,
		Timestamp: time.Now(),
		Data:      payload,
	}
}

func (e OrderEvent) Type() string            { return string(e.EventType) }
func (e OrderEvent) Topic() string           { return "OrderEvents" }
func (e OrderEvent) Payload() any            { return e.Data }
func (e OrderEvent) ToJSON() ([]byte, error) { return json.Marshal(e) }
func (e OrderEvent) GetEntityID() string     { return e.Data.CartID }
func (e OrderEvent) GetResourceID() string   { return e.ID }

// Convenience constructor for OrderEvents

func NewOrderCreatedEvent(orderID string, orderNumber string, cartID string, customerID *string, total float64) *OrderEvent {
	return NewOrderEvent(orderID, orderNumber, OrderCreated, cartID, customerID, total)
}

func NewOrderUpdatedEvent(orderID string, orderNumber string, cartID string, customerID *string, total float64) *OrderEvent {
	return NewOrderEvent(orderID, orderNumber, OrderUpdated, cartID, customerID, total)
}

func NewOrderDeletedEvent(orderID string, orderNumber string, cartID string, customerID *string, total float64) *OrderEvent {
	return NewOrderEvent(orderID, orderNumber, OrderDeleted, cartID, customerID, total)
}
