package events

import (
	"bytes"
	"encoding/json"
	entity "go-shopping-poc/internal/entity/customer"
	"go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
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

func CustomerCreatedEventFactory(name string, payload []byte) (event.Event, error) {
	logging.Debug("Factory for CustomerCreatedEvent - received event")
	var b = bytes.NewBuffer(payload)
	var p entity.Customer
	if err := json.NewDecoder(b).Decode(&p); err != nil {
		logging.Error("Failed to unmarshal CustomerCreated event payload: %v", err)
		return nil, err
	}
	logging.Debug("Factory for CustomerCreatedEvent - payload: %s", string(payload))
	return NewCustomerCreatedEvent(p), nil
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
