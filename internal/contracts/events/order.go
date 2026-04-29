package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OrderEventType string

const (
	OrderCreated                             OrderEventType = "order.created"
	OrderUpdated                             OrderEventType = "order.updated"
	OrderDeleted                             OrderEventType = "order.deleted"
	CustomerIdentityVerificationRequested   OrderEventType = "order.customer.identity_verification_requested"
	CustomerIdentityVerificationCompleted   OrderEventType = "order.customer.identity_verification_completed"
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

func NewOrderCreatedEvent(orderID string, orderNumber string, cartID string, customerID *string, total float64) *OrderEvent {
	return NewOrderEvent(orderID, orderNumber, OrderCreated, cartID, customerID, total)
}

func NewOrderUpdatedEvent(orderID string, orderNumber string, cartID string, customerID *string, total float64) *OrderEvent {
	return NewOrderEvent(orderID, orderNumber, OrderUpdated, cartID, customerID, total)
}

func NewOrderDeletedEvent(orderID string, orderNumber string, cartID string, customerID *string, total float64) *OrderEvent {
	return NewOrderEvent(orderID, orderNumber, OrderDeleted, cartID, customerID, total)
}

// CustomerIdentityVerificationRequestPayload carries a verification request from the order service
type CustomerIdentityVerificationRequestPayload struct {
	RequestID   string `json:"request_id"`
	Email       string `json:"email"`
	KeycloakSub string `json:"keycloak_sub"`
}

// CustomerIdentityVerificationRequestEvent is published to OrderEvents by the order service
type CustomerIdentityVerificationRequestEvent struct {
	ID        string                                              `json:"id"`
	EventType OrderEventType                                      `json:"type"`
	Timestamp time.Time                                           `json:"timestamp"`
	Data      CustomerIdentityVerificationRequestPayload          `json:"payload"`
}

func (e CustomerIdentityVerificationRequestEvent) Type() string                      { return string(e.EventType) }
func (e CustomerIdentityVerificationRequestEvent) Topic() string                     { return "OrderEvents" }
func (e CustomerIdentityVerificationRequestEvent) Payload() any                      { return e.Data }
func (e CustomerIdentityVerificationRequestEvent) ToJSON() ([]byte, error)           { return json.Marshal(e) }
func (e CustomerIdentityVerificationRequestEvent) GetEntityID() string               { return e.Data.KeycloakSub }
func (e CustomerIdentityVerificationRequestEvent) GetResourceID() string             { return e.ID }

// CustomerIdentityVerificationRequestEventFactory implements EventFactory
type CustomerIdentityVerificationRequestEventFactory struct{}

func (f CustomerIdentityVerificationRequestEventFactory) FromJSON(data []byte) (CustomerIdentityVerificationRequestEvent, error) {
	var event CustomerIdentityVerificationRequestEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

func NewCustomerIdentityVerificationRequestedEvent(requestID, email, keycloakSub string) *CustomerIdentityVerificationRequestEvent {
	return &CustomerIdentityVerificationRequestEvent{
		ID:        uuid.New().String(),
		EventType: CustomerIdentityVerificationRequested,
		Timestamp: time.Now(),
		Data: CustomerIdentityVerificationRequestPayload{
			RequestID:   requestID,
			Email:       email,
			KeycloakSub: keycloakSub,
		},
	}
}

// CustomerIdentityVerificationResultPayload carries a verification response from the customer service
type CustomerIdentityVerificationResultPayload struct {
	RequestID     string `json:"request_id"`
	Authorized    bool   `json:"authorized"`
	CustomerID    string `json:"customer_id,omitempty"`
	ResolvedEmail string `json:"email,omitempty"`
	Error         string `json:"error,omitempty"`
}

// CustomerIdentityVerificationCompletedEvent is published to CustomerEvents by the customer service
type CustomerIdentityVerificationCompletedEvent struct {
	ID        string                                               `json:"id"`
	EventType OrderEventType                                       `json:"type"`
	Timestamp time.Time                                            `json:"timestamp"`
	Data      CustomerIdentityVerificationResultPayload            `json:"payload"`
}

func (e CustomerIdentityVerificationCompletedEvent) Type() string                     { return string(e.EventType) }
func (e CustomerIdentityVerificationCompletedEvent) Topic() string                    { return "CustomerEvents" }
func (e CustomerIdentityVerificationCompletedEvent) Payload() any                     { return e.Data }
func (e CustomerIdentityVerificationCompletedEvent) ToJSON() ([]byte, error)          { return json.Marshal(e) }
func (e CustomerIdentityVerificationCompletedEvent) GetEntityID() string              { return e.Data.CustomerID }
func (e CustomerIdentityVerificationCompletedEvent) GetResourceID() string            { return e.ID }

// CustomerIdentityVerificationCompletedEventFactory implements EventFactory
type CustomerIdentityVerificationCompletedEventFactory struct{}

func (f CustomerIdentityVerificationCompletedEventFactory) FromJSON(data []byte) (CustomerIdentityVerificationCompletedEvent, error) {
	var event CustomerIdentityVerificationCompletedEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

func NewCustomerIdentityVerificationCompletedEvent(requestID string, authorized bool, customerID, email, errStr string) *CustomerIdentityVerificationCompletedEvent {
	return &CustomerIdentityVerificationCompletedEvent{
		ID:        uuid.New().String(),
		EventType: CustomerIdentityVerificationCompleted,
		Timestamp: time.Now(),
		Data: CustomerIdentityVerificationResultPayload{
			RequestID:     requestID,
			Authorized:    authorized,
			CustomerID:    customerID,
			ResolvedEmail: email,
			Error:         errStr,
		},
	}
}
