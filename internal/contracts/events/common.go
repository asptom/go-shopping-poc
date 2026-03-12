// Package events defines the contract interfaces and data structures for events.
// This package contains only pure data structures and interfaces - no business logic.
//
// Key interfaces:
//   - Event: Common event interface (Type, Topic, Payload, ToJSON)
//   - EventFactory[T]: Type-safe event reconstruction from JSON
package events

// Event defines methods for event type and topic
type Event interface {
	Type() string
	Topic() string
	Payload() any
	ToJSON() ([]byte, error)
	GetEntityID() string
	GetResourceID() string
}

// EventFactory defines interface for reconstructing events from JSON
type EventFactory[T Event] interface {
	FromJSON([]byte) (T, error)
}
