package outbox

import (
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID             uuid.UUID `json:"event_id" db:"id"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	EventType      string    `json:"event_type" db:"event_type"`
	EventPayload   []byte    `json:"event_payload" db:"event_payload"`
	TimesAttempted int       `json:"times_attempted" db:"times_attempted"`
}

func NewOutboxEvent(eventID uuid.UUID, eventType string, eventPayload any) *OutboxEvent {
	return &OutboxEvent{
		ID:             uuid.New(),
		CreatedAt:      time.Now(),
		EventType:      eventType,
		EventPayload:   eventPayload.([]byte),
		TimesAttempted: 0,
	}
}
