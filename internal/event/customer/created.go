package events

import (
	entity "go-shopping-poc/internal/entity/customer"
	event "go-shopping-poc/pkg/event"
	"time"

	"github.com/google/uuid"
)

type CustomerCreatedPayload struct {
	Customer entity.Customer `json:"customer"`
}

// CustomerCreatedEvent is a concrete event type
type CustomerCreatedEvent struct {
	event.Event[CustomerCreatedPayload]
}

const EventTypeCustomerCreated = "CustomerCreated"

// NewCustomerCreatedEvent creates a new CustomerCreatedEvent
func NewCustomerCreatedEvent(customer entity.Customer) CustomerCreatedEvent {
	return CustomerCreatedEvent{
		Event: event.Event[CustomerCreatedPayload]{
			ID:        uuid.New().String(),
			Type:      EventTypeCustomerCreated,
			TimeStamp: time.Now(),
			Payload: CustomerCreatedPayload{
				Customer: customer,
			},
		},
	}
}

// GetType returns the event type
func (e CustomerCreatedEvent) GetType() string {
	return EventTypeCustomerCreated
}

// GetTopic returns the Kafka topic
func (e CustomerCreatedEvent) GetTopic() string {
	return "CustomerEvents"
}
