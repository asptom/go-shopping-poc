package events

import (
	"encoding/json"
	entity "go-shopping-poc/internal/entity/customer"
)

type CustomerCreatedEvent struct {
	EventPayload entity.Customer `json:"customer"`
}

// Implement the event.Event interface

func (e CustomerCreatedEvent) Name() string { return "CustomerCreated" }
func (e CustomerCreatedEvent) Payload() any {
	b, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	return b
}
