// Package events defines the contract interfaces and data structures for events.
// This package contains only pure data structures and interfaces - no business logic.
//
// Key interfaces:
//   - Event: Common event interface (Type, Topic, Payload, ToJSON)
//   - EventFactory[T]: Type-safe event reconstruction from JSON
//
// Event types:
//   - CustomerEvent: Customer domain events
//   - OrderEvent: Order domain events (future)
//   - ProductEvent: Product domain events (future)
//
// Usage example:
//
//	// Create a customer event
//	event := events.NewCustomerCreated(customerID, customerData)
//
//	// Serialize to JSON
//	jsonData, err := event.ToJSON()
//
//	// Deserialize with type safety
//	factory := events.CustomerEventFactory{}
//	restoredEvent, err := factory.FromJSON(jsonData)
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
