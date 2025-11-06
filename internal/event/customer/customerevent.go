package events

import (
	"encoding/json"
	"time"

	ev "go-shopping-poc/pkg/event"

	"github.com/google/uuid"
)

// register both topic and explicit change-type keys so payloads can be unmarshaled
// regardless of whether the outbox stored the topic or the change-type string.
func init() {
	unmarshal := func(b []byte) (ev.Event, error) {
		var e CustomerEvent
		if err := json.Unmarshal(b, &e); err != nil {
			return nil, err
		}
		return &e, nil
	}

	// topic-based registration
	ev.Register("customer.changes", unmarshal)

	// register individual change type strings for compatibility
	ev.Register(string(CustomerCreated), unmarshal)
	ev.Register(string(CustomerUpdated), unmarshal)

	ev.Register(string(AddressAdded), unmarshal)
	ev.Register(string(AddressUpdated), unmarshal)
	ev.Register(string(AddressDeleted), unmarshal)

	ev.Register(string(CardAdded), unmarshal)
	ev.Register(string(CardUpdated), unmarshal)
	ev.Register(string(CardDeleted), unmarshal)
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

// CustomerEvent wraps the generic Event type with our specific payload
type CustomerEvent struct {
	Event_ID        string
	Event_Type      string
	Event_TimeStamp time.Time
	Event_Payload   CustomerEventPayload
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
		Event_ID:        uuid.New().String(),
		Event_Type:      string(t),
		Event_TimeStamp: time.Now(),
		Event_Payload:   payload,
	}
}

// Implement Event Interface
func (e *CustomerEvent) Type() string {
	return e.Event_Type
}

func (e *CustomerEvent) Topic() string {
	return "CustomerEvents"
}

func (e *CustomerEvent) Payload() any {
	return e.Event_Payload
}

func (e *CustomerEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

func (e *CustomerEvent) FromJSON(data []byte) error {
	return json.Unmarshal(data, e)
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
