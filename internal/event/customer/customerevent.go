package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CustomerEventFactory implements EventFactory for CustomerEvent
type CustomerEventFactory struct{}

func (f CustomerEventFactory) FromJSON(data []byte) (CustomerEvent, error) {
	var event CustomerEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

// EventType is a typed alias for well-known customer events
type EventType string

const (
	CustomerCreated EventType = "customer.created"
	CustomerUpdated EventType = "customer.updated"

	AddressAdded   EventType = "address.add"
	AddressUpdated EventType = "address.update"
	AddressDeleted EventType = "address.delete"

	CardAdded   EventType = "card.add"
	CardUpdated EventType = "card.update"
	CardDeleted EventType = "card.delete"
)

// CustomerEventPayload represents the data structure for customer events
type CustomerEventPayload struct {
	CustomerID string            `json:"customer_id,omitempty"`
	EventType  EventType         `json:"event_type"`
	ResourceID string            `json:"resource_id,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
}

// CustomerEvent represents a customer-related event
type CustomerEvent struct {
	ID           string               `json:"id"`
	EventType    EventType            `json:"type"`
	Timestamp    time.Time            `json:"timestamp"`
	EventPayload CustomerEventPayload `json:"payload"`
}

// Convenience constructor for CustomerEvent
func NewCustomerEvent(customerID string, t EventType, resourceID string, details map[string]string) *CustomerEvent {
	payload := CustomerEventPayload{
		CustomerID: customerID,
		EventType:  t,
		ResourceID: resourceID,
		Details:    details,
	}

	return &CustomerEvent{
		ID:           uuid.New().String(),
		EventType:    t,
		Timestamp:    time.Now(),
		EventPayload: payload,
	}
}

// Implement Event Interface
func (e CustomerEvent) Type() string {
	return string(e.EventType)
}

func (e CustomerEvent) Topic() string {
	return "CustomerEvents"
}

func (e CustomerEvent) Payload() any {
	return e.EventPayload
}

func (e CustomerEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// Convenience constructors

func NewCustomerCreatedEvent(customerID string, details map[string]string) *CustomerEvent {
	return NewCustomerEvent(customerID, CustomerCreated, customerID, details)
}

func NewCustomerUpdatedEvent(customerID string, details map[string]string) *CustomerEvent {
	return NewCustomerEvent(customerID, CustomerUpdated, customerID, details)
}

func NewAddressAddedEvent(customerID, addressID string, details map[string]string) *CustomerEvent {
	return NewCustomerEvent(customerID, AddressAdded, addressID, details)
}

func NewAddressUpdatedEvent(customerID, addressID string, details map[string]string) *CustomerEvent {
	return NewCustomerEvent(customerID, AddressUpdated, addressID, details)
}

func NewAddressDeletedEvent(customerID, addressID string, details map[string]string) *CustomerEvent {
	return NewCustomerEvent(customerID, AddressDeleted, addressID, details)
}

func NewCardAddedEvent(customerID, cardID string, details map[string]string) *CustomerEvent {
	return NewCustomerEvent(customerID, CardAdded, cardID, details)
}

func NewCardUpdatedEvent(customerID, cardID string, details map[string]string) *CustomerEvent {
	return NewCustomerEvent(customerID, CardUpdated, cardID, details)
}

func NewCardDeletedEvent(customerID, cardID string, details map[string]string) *CustomerEvent {
	return NewCustomerEvent(customerID, CardDeleted, cardID, details)
}
