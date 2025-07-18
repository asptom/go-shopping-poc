package event

// Event is a generic interface for events with a payload.

type Event interface {
	Name() string
	Payload() any
}
