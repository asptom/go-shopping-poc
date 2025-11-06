package outbox

import (
	"database/sql"
	"time"
)

// OutboxEvent represents an event stored in the outbox table.
type OutboxEvent struct {
	ID             int64        `json:"event_id" db:"id"`
	EventType      string       `json:"event_type" db:"event_type"`
	Topic          string       `json:"topic" db:"topic"`
	EventPayload   []byte       `json:"event_payload" db:"event_payload"`
	CreatedAt      time.Time    `json:"created_at" db:"created_at"`
	TimesAttempted int          `json:"times_attempted" db:"times_attempted"`
	PublishedAt    sql.NullTime `json:"published_at" db:"published_at"`
}
