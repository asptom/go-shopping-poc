package event

// Event is a generic interface for events with a payload.

type Event interface {
	Name() string
	Payload() any
}

// Process Event is an implementation of the Event interface
// that can be leveraged to process events

type ProcessEvent struct {
	EventType    string
	EventPayload any
}

func NewProcessEvent(eventType string, eventPayload any) ProcessEvent {
	return ProcessEvent{
		EventType:    eventType,
		EventPayload: eventPayload,
	}
}

func (e ProcessEvent) Name() string { return e.EventType }
func (e ProcessEvent) Payload() any {
	return e.EventPayload
}
