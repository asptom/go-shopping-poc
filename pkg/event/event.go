package event

import (
	"encoding/json"
	"time"
)

// Event represents a generic event structure with a type and payload.

type Event[T any] struct {
	ID        string    `json:"id"`        // Unique identifier for the event
	Type      string    `json:"type"`      // Type of the event
	TimeStamp time.Time `json:"timestamp"` // Unix timestamp of the event
	Payload   T         `json:"payload"`   // Payload of the event
}

// EventInterface defines methods for event type and topic

type EventInterface interface {
	GetType() string
	GetTopic() string
	ToJSON() ([]byte, error)
}

// ToJSON serializes the event to JSON

func (e Event[T]) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON deserializes JSON into an Event

func FromJSON[T any](data []byte) (Event[T], error) {
	var event Event[T]
	err := json.Unmarshal(data, &event)
	return event, err
}
