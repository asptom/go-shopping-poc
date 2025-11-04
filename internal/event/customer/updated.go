package events

import (
	entity "go-shopping-poc/internal/entity/customer"
	event "go-shopping-poc/pkg/event"
	"time"

	"github.com/google/uuid"
)

type CustomerUpdatedPayload struct {
	Customer entity.Customer `json:"customer"`
}

// CustomerUpdatedEvent is a concrete event type
type CustomerUpdatedEvent struct {
	event.Event[CustomerUpdatedPayload]
}

const EventTypeCustomerUpdated = "CustomerUpdated"

// NewCustomerUpdatedEvent creates a new CustomerUpdatedEvent
func NewCustomerUpdatedEvent(customer entity.Customer) CustomerUpdatedEvent {
	return CustomerUpdatedEvent{
		Event: event.Event[CustomerUpdatedPayload]{
			ID:        uuid.New().String(),
			Type:      EventTypeCustomerUpdated,
			TimeStamp: time.Now(),
			Payload: CustomerUpdatedPayload{
				Customer: customer,
			},
		},
	}
}

// GetType returns the event type
func (e CustomerUpdatedEvent) GetType() string {
	return EventTypeCustomerUpdated
}

// GetTopic returns the Kafka topic
func (e CustomerUpdatedEvent) GetTopic() string {
	return "CustomerEvents"
}
