package outbox

import (
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID             uuid.UUID `db:"id"`
	CreatedAt      time.Time `db:"created_at"`
	EventType      string    `db:"event_type"`
	EventPayload   []byte    `db:"event_payload"`
	TimesAttempted int       `db:"times_attempted"`
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
