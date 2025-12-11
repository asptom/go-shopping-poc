package events

// Event defines methods for event type and topic
type Event interface {
	Type() string
	Topic() string
	Payload() any
	ToJSON() ([]byte, error)
}

// EventFactory defines interface for reconstructing events from JSON
type EventFactory[T Event] interface {
	FromJSON([]byte) (T, error)
}
