package event

// Event represents a generic event structure with a type and payload.

// type Event[T any] struct {
// 	Event_ID        string    `json:"id"`        // Unique identifier for the event
// 	Event_Type      string    `json:"type"`      // Type of the event
// 	Event_TimeStamp time.Time `json:"timestamp"` // Unix timestamp of the event
// 	Event_Payload   T         `json:"payload"`   // Payload of the event
// }

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

// // ToJSON serializes the event to JSON

// func (e Event[T]) ToJSON() ([]byte, error) {
// 	return json.Marshal(e)
// }

// // FromJSON deserializes JSON into an Event

// func FromJSON[T any](data []byte) (Event[T], error) {
// 	var event Event[T]
// 	err := json.Unmarshal(data, &event)
// 	return event, err
// }
