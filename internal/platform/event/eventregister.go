package event

import (
	"fmt"
)

// UnmarshalFunc converts raw JSON bytes into a concrete event that implements event.Event
type UnmarshalFunc func([]byte) (Event, error)

var registry = map[string]UnmarshalFunc{}

// Register registers an unmarshal function for an event type (call from init() in each event package).
func Register(eventType string, fn UnmarshalFunc) {
	registry[eventType] = fn
}

// UnmarshalEvent builds an event.Event for the given stored eventType and JSON payload.
// If no constructor is registered, returns a RawEvent that carries the payload.
func UnmarshalEvent(eventType string, data []byte) (Event, error) {
	if fn, ok := registry[eventType]; ok {
		return fn(data)
	}
	// fallback: wrap raw payload so publisher can still publish it
	return &RawEvent{
		eventType: eventType,
		payload:   data,
	}, nil
}

// RawEvent is a generic wrapper implementing event.Event for unknown types.
type RawEvent struct {
	eventType string
	payload   []byte
}

func (r *RawEvent) Type() string  { return r.eventType }
func (r *RawEvent) Topic() string { return r.eventType }
func (r *RawEvent) Payload() any  { return r.payload }

// Ensure RawEvent implements the package Event interface by providing FromJSON (and a ToJSON helper).
func (r *RawEvent) FromJSON(b []byte) error {
	// store a copy of the payload
	if b == nil {
		r.payload = nil
		return nil
	}
	r.payload = make([]byte, len(b))
	copy(r.payload, b)
	return nil
}

func (r *RawEvent) ToJSON() ([]byte, error) {
	if r.payload == nil {
		return nil, nil
	}
	out := make([]byte, len(r.payload))
	copy(out, r.payload)
	return out, nil
}

// Optional helper for debug
func MustUnmarshalEvent(eventType string, data []byte) Event {
	e, err := UnmarshalEvent(eventType, data)
	if err != nil {
		panic(fmt.Sprintf("unmarshal event %s: %v", eventType, err))
	}
	return e
}
