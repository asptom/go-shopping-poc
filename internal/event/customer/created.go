package events

import (
	"encoding/json"
	entity "go-shopping-poc/internal/entity/customer"
)

type CustomerCreatedEvent struct {
	EventType    string
	EventPayload entity.Customer
}

func NewCustomerCreatedEvent(customer entity.Customer) CustomerCreatedEvent {
	return CustomerCreatedEvent{
		EventType:    "CustomerCreated",
		EventPayload: customer,
	}
}

// Implement the event.Event interface

func (e CustomerCreatedEvent) Name() string { return "CustomerCreated" }
func (e CustomerCreatedEvent) Payload() any {
	b, err := json.Marshal(e.EventPayload)
	if err != nil {
		return nil
	}
	return b
}
