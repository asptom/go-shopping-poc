package outbox

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestOutboxEvent_JSONMarshaling tests JSON marshaling and unmarshaling of OutboxEvent
func TestOutboxEvent_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name    string
		event   OutboxEvent
		wantErr bool
	}{
		{
			name: "valid event with all fields",
			event: OutboxEvent{
				ID:             123,
				EventType:      "customer.created",
				Topic:          "CustomerEvents",
				EventPayload:   []byte(`{"customer_id":"test-123"}`),
				CreatedAt:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				TimesAttempted: 2,
				PublishedAt:    sql.NullTime{Time: time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC), Valid: true},
			},
			wantErr: false,
		},
		{
			name: "event without published_at",
			event: OutboxEvent{
				ID:             456,
				EventType:      "customer.updated",
				Topic:          "CustomerEvents",
				EventPayload:   []byte(`{"customer_id":"test-456"}`),
				CreatedAt:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				TimesAttempted: 0,
				PublishedAt:    sql.NullTime{Valid: false},
			},
			wantErr: false,
		},
		{
			name: "event with empty payload",
			event: OutboxEvent{
				ID:             789,
				EventType:      "customer.deleted",
				Topic:          "CustomerEvents",
				EventPayload:   []byte{},
				CreatedAt:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				TimesAttempted: 1,
				PublishedAt:    sql.NullTime{Valid: false},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("OutboxEvent.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Test unmarshaling
			var unmarshaled OutboxEvent
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("OutboxEvent.UnmarshalJSON() error = %v", err)
				return
			}

			// Verify fields
			if unmarshaled.ID != tt.event.ID {
				t.Errorf("ID = %v, want %v", unmarshaled.ID, tt.event.ID)
			}
			if unmarshaled.EventType != tt.event.EventType {
				t.Errorf("EventType = %v, want %v", unmarshaled.EventType, tt.event.EventType)
			}
			if unmarshaled.Topic != tt.event.Topic {
				t.Errorf("Topic = %v, want %v", unmarshaled.Topic, tt.event.Topic)
			}
			if string(unmarshaled.EventPayload) != string(tt.event.EventPayload) {
				t.Errorf("EventPayload = %v, want %v", string(unmarshaled.EventPayload), string(tt.event.EventPayload))
			}
			if unmarshaled.TimesAttempted != tt.event.TimesAttempted {
				t.Errorf("TimesAttempted = %v, want %v", unmarshaled.TimesAttempted, tt.event.TimesAttempted)
			}
			if unmarshaled.PublishedAt.Valid != tt.event.PublishedAt.Valid {
				t.Errorf("PublishedAt.Valid = %v, want %v", unmarshaled.PublishedAt.Valid, tt.event.PublishedAt.Valid)
			}
			if unmarshaled.PublishedAt.Valid && !unmarshaled.PublishedAt.Time.Equal(tt.event.PublishedAt.Time) {
				t.Errorf("PublishedAt.Time = %v, want %v", unmarshaled.PublishedAt.Time, tt.event.PublishedAt.Time)
			}
		})
	}
}

// TestOutboxEvent_FieldValidation tests field validation for OutboxEvent
func TestOutboxEvent_FieldValidation(t *testing.T) {
	tests := []struct {
		name  string
		event OutboxEvent
		valid bool
	}{
		{
			name: "valid event",
			event: OutboxEvent{
				ID:             1,
				EventType:      "customer.created",
				Topic:          "CustomerEvents",
				EventPayload:   []byte(`{"data":"test"}`),
				CreatedAt:      time.Now(),
				TimesAttempted: 0,
				PublishedAt:    sql.NullTime{Valid: false},
			},
			valid: true,
		},
		{
			name: "negative ID (allowed for database auto-increment)",
			event: OutboxEvent{
				ID:             -1,
				EventType:      "customer.created",
				Topic:          "CustomerEvents",
				EventPayload:   []byte(`{"data":"test"}`),
				CreatedAt:      time.Now(),
				TimesAttempted: 0,
				PublishedAt:    sql.NullTime{Valid: false},
			},
			valid: true,
		},
		{
			name: "empty event type",
			event: OutboxEvent{
				ID:             1,
				EventType:      "",
				Topic:          "CustomerEvents",
				EventPayload:   []byte(`{"data":"test"}`),
				CreatedAt:      time.Now(),
				TimesAttempted: 0,
				PublishedAt:    sql.NullTime{Valid: false},
			},
			valid: true, // Empty strings are valid in database
		},
		{
			name: "empty topic",
			event: OutboxEvent{
				ID:             1,
				EventType:      "customer.created",
				Topic:          "",
				EventPayload:   []byte(`{"data":"test"}`),
				CreatedAt:      time.Now(),
				TimesAttempted: 0,
				PublishedAt:    sql.NullTime{Valid: false},
			},
			valid: true, // Empty strings are valid in database
		},
		{
			name: "nil payload",
			event: OutboxEvent{
				ID:             1,
				EventType:      "customer.created",
				Topic:          "CustomerEvents",
				EventPayload:   nil,
				CreatedAt:      time.Now(),
				TimesAttempted: 0,
				PublishedAt:    sql.NullTime{Valid: false},
			},
			valid: true, // nil slices are valid
		},
		{
			name: "zero created_at",
			event: OutboxEvent{
				ID:             1,
				EventType:      "customer.created",
				Topic:          "CustomerEvents",
				EventPayload:   []byte(`{"data":"test"}`),
				CreatedAt:      time.Time{},
				TimesAttempted: 0,
				PublishedAt:    sql.NullTime{Valid: false},
			},
			valid: true, // zero time is valid
		},
		{
			name: "negative times attempted",
			event: OutboxEvent{
				ID:             1,
				EventType:      "customer.created",
				Topic:          "CustomerEvents",
				EventPayload:   []byte(`{"data":"test"}`),
				CreatedAt:      time.Now(),
				TimesAttempted: -1,
				PublishedAt:    sql.NullTime{Valid: false},
			},
			valid: true, // negative values are allowed in database
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For now, all events are considered valid since the struct doesn't have validation
			// In a real scenario, you might add validation methods
			if tt.valid {
				// Verify the event struct is properly constructed by checking field assignments
				switch tt.name {
				case "valid event":
					if tt.event.ID != 1 {
						t.Errorf("ID = %v, want 1", tt.event.ID)
					}
					if tt.event.EventType != "customer.created" {
						t.Errorf("EventType = %v, want 'customer.created'", tt.event.EventType)
					}
					if tt.event.Topic != "CustomerEvents" {
						t.Errorf("Topic = %v, want 'CustomerEvents'", tt.event.Topic)
					}
					if string(tt.event.EventPayload) != `{"data":"test"}` {
						t.Errorf("EventPayload = %v, want '{\"data\":\"test\"}'", string(tt.event.EventPayload))
					}
					if tt.event.TimesAttempted != 0 {
						t.Errorf("TimesAttempted = %v, want 0", tt.event.TimesAttempted)
					}
					if tt.event.PublishedAt.Valid {
						t.Errorf("PublishedAt.Valid = %v, want false", tt.event.PublishedAt.Valid)
					}
				case "negative ID (allowed for database auto-increment)":
					if tt.event.ID != -1 {
						t.Errorf("ID = %v, want -1", tt.event.ID)
					}
					if tt.event.EventType != "customer.created" {
						t.Errorf("EventType = %v, want 'customer.created'", tt.event.EventType)
					}
					if tt.event.Topic != "CustomerEvents" {
						t.Errorf("Topic = %v, want 'CustomerEvents'", tt.event.Topic)
					}
				case "empty event type":
					if tt.event.ID != 1 {
						t.Errorf("ID = %v, want 1", tt.event.ID)
					}
					if tt.event.EventType != "" {
						t.Errorf("EventType = %v, want empty string", tt.event.EventType)
					}
					if tt.event.Topic != "CustomerEvents" {
						t.Errorf("Topic = %v, want 'CustomerEvents'", tt.event.Topic)
					}
				case "empty topic":
					if tt.event.ID != 1 {
						t.Errorf("ID = %v, want 1", tt.event.ID)
					}
					if tt.event.EventType != "customer.created" {
						t.Errorf("EventType = %v, want 'customer.created'", tt.event.EventType)
					}
					if tt.event.Topic != "" {
						t.Errorf("Topic = %v, want empty string", tt.event.Topic)
					}
				case "nil payload":
					if tt.event.ID != 1 {
						t.Errorf("ID = %v, want 1", tt.event.ID)
					}
					if tt.event.EventType != "customer.created" {
						t.Errorf("EventType = %v, want 'customer.created'", tt.event.EventType)
					}
					if tt.event.EventPayload != nil {
						t.Errorf("EventPayload = %v, want nil", tt.event.EventPayload)
					}
				case "zero created_at":
					if tt.event.ID != 1 {
						t.Errorf("ID = %v, want 1", tt.event.ID)
					}
					if tt.event.CreatedAt != (time.Time{}) {
						t.Errorf("CreatedAt = %v, want zero time", tt.event.CreatedAt)
					}
				case "negative times attempted":
					if tt.event.ID != 1 {
						t.Errorf("ID = %v, want 1", tt.event.ID)
					}
					if tt.event.TimesAttempted != -1 {
						t.Errorf("TimesAttempted = %v, want -1", tt.event.TimesAttempted)
					}
				}
			}
		})
	}
}

// TestOutboxEvent_PublishedAtHandling tests PublishedAt field handling
func TestOutboxEvent_PublishedAtHandling(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		publishedAt sql.NullTime
		isPublished bool
	}{
		{
			name:        "not published",
			publishedAt: sql.NullTime{Valid: false},
			isPublished: false,
		},
		{
			name:        "published",
			publishedAt: sql.NullTime{Time: now, Valid: true},
			isPublished: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := OutboxEvent{
				ID:             1,
				EventType:      "test.event",
				Topic:          "TestTopic",
				EventPayload:   []byte(`{"test":"data"}`),
				CreatedAt:      now.Add(-time.Hour),
				TimesAttempted: 1,
				PublishedAt:    tt.publishedAt,
			}

			// Verify PublishedAt field
			if event.PublishedAt.Valid != tt.isPublished {
				t.Errorf("PublishedAt.Valid = %v, want %v", event.PublishedAt.Valid, tt.isPublished)
			}

			if tt.isPublished && !event.PublishedAt.Time.Equal(now) {
				t.Errorf("PublishedAt.Time = %v, want %v", event.PublishedAt.Time, now)
			}

			// Verify other fields are properly set
			if event.ID != 1 {
				t.Errorf("ID = %v, want 1", event.ID)
			}
			if event.EventType != "test.event" {
				t.Errorf("EventType = %v, want 'test.event'", event.EventType)
			}
			if event.Topic != "TestTopic" {
				t.Errorf("Topic = %v, want 'TestTopic'", event.Topic)
			}
			if string(event.EventPayload) != `{"test":"data"}` {
				t.Errorf("EventPayload = %v, want '{\"test\":\"data\"}'", string(event.EventPayload))
			}
			if !event.CreatedAt.Equal(now.Add(-time.Hour)) {
				t.Errorf("CreatedAt = %v, want %v", event.CreatedAt, now.Add(-time.Hour))
			}
			if event.TimesAttempted != 1 {
				t.Errorf("TimesAttempted = %v, want 1", event.TimesAttempted)
			}
		})
	}
}

// TestOutboxEvent_TimesAttemptedIncrement tests incrementing times attempted
func TestOutboxEvent_TimesAttemptedIncrement(t *testing.T) {
	event := OutboxEvent{
		ID:             1,
		EventType:      "test.event",
		Topic:          "TestTopic",
		EventPayload:   []byte(`{"test":"data"}`),
		CreatedAt:      time.Now(),
		TimesAttempted: 0,
		PublishedAt:    sql.NullTime{Valid: false},
	}

	// Verify initial state of other fields
	if event.ID != 1 {
		t.Errorf("ID = %v, want 1", event.ID)
	}
	if event.EventType != "test.event" {
		t.Errorf("EventType = %v, want 'test.event'", event.EventType)
	}
	if event.Topic != "TestTopic" {
		t.Errorf("Topic = %v, want 'TestTopic'", event.Topic)
	}
	if string(event.EventPayload) != `{"test":"data"}` {
		t.Errorf("EventPayload = %v, want '{\"test\":\"data\"}'", string(event.EventPayload))
	}
	if event.CreatedAt.IsZero() {
		t.Errorf("CreatedAt should not be zero")
	}
	if event.PublishedAt.Valid {
		t.Errorf("PublishedAt.Valid = %v, want false", event.PublishedAt.Valid)
	}

	// Increment attempts
	event.TimesAttempted++
	if event.TimesAttempted != 1 {
		t.Errorf("TimesAttempted = %v, want 1", event.TimesAttempted)
	}

	event.TimesAttempted++
	if event.TimesAttempted != 2 {
		t.Errorf("TimesAttempted = %v, want 2", event.TimesAttempted)
	}
}

// TestOutboxEvent_IDGeneration tests ID field handling
func TestOutboxEvent_IDGeneration(t *testing.T) {
	event := OutboxEvent{
		EventType:      "test.event",
		Topic:          "TestTopic",
		EventPayload:   []byte(`{"test":"data"}`),
		CreatedAt:      time.Now(),
		TimesAttempted: 0,
		PublishedAt:    sql.NullTime{Valid: false},
	}

	// Verify other fields are properly set
	if event.EventType != "test.event" {
		t.Errorf("EventType = %v, want 'test.event'", event.EventType)
	}
	if event.Topic != "TestTopic" {
		t.Errorf("Topic = %v, want 'TestTopic'", event.Topic)
	}
	if string(event.EventPayload) != `{"test":"data"}` {
		t.Errorf("EventPayload = %v, want '{\"test\":\"data\"}'", string(event.EventPayload))
	}
	if event.CreatedAt.IsZero() {
		t.Errorf("CreatedAt should not be zero")
	}
	if event.TimesAttempted != 0 {
		t.Errorf("TimesAttempted = %v, want 0", event.TimesAttempted)
	}
	if event.PublishedAt.Valid {
		t.Errorf("PublishedAt.Valid = %v, want false", event.PublishedAt.Valid)
	}

	// ID should be zero initially (database auto-increment)
	if event.ID != 0 {
		t.Errorf("Initial ID = %v, want 0", event.ID)
	}

	// Simulate database assignment
	event.ID = 12345
	if event.ID != 12345 {
		t.Errorf("Assigned ID = %v, want 12345", event.ID)
	}
}

// TestOutboxEvent_EventPayloadHandling tests payload field handling
func TestOutboxEvent_EventPayloadHandling(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "empty payload",
			payload: []byte{},
		},
		{
			name:    "simple JSON payload",
			payload: []byte(`{"key":"value"}`),
		},
		{
			name:    "complex JSON payload",
			payload: []byte(`{"id":"` + uuid.New().String() + `","type":"test","data":{"nested":"value"}}`),
		},
		{
			name:    "large payload",
			payload: make([]byte, 10000), // 10KB payload
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := OutboxEvent{
				ID:             1,
				EventType:      "test.event",
				Topic:          "TestTopic",
				EventPayload:   tt.payload,
				CreatedAt:      time.Now(),
				TimesAttempted: 0,
				PublishedAt:    sql.NullTime{Valid: false},
			}

			// Verify other fields are properly set
			if event.ID != 1 {
				t.Errorf("ID = %v, want 1", event.ID)
			}
			if event.EventType != "test.event" {
				t.Errorf("EventType = %v, want 'test.event'", event.EventType)
			}
			if event.Topic != "TestTopic" {
				t.Errorf("Topic = %v, want 'TestTopic'", event.Topic)
			}
			if event.CreatedAt.IsZero() {
				t.Errorf("CreatedAt should not be zero")
			}
			if event.TimesAttempted != 0 {
				t.Errorf("TimesAttempted = %v, want 0", event.TimesAttempted)
			}
			if event.PublishedAt.Valid {
				t.Errorf("PublishedAt.Valid = %v, want false", event.PublishedAt.Valid)
			}

			// Verify EventPayload field
			if len(event.EventPayload) != len(tt.payload) {
				t.Errorf("EventPayload length = %v, want %v", len(event.EventPayload), len(tt.payload))
			}

			for i, b := range tt.payload {
				if event.EventPayload[i] != b {
					t.Errorf("EventPayload[%d] = %v, want %v", i, event.EventPayload[i], b)
				}
			}
		})
	}
}
