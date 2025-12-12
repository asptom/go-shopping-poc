package bus

import (
	"context"
	"go-shopping-poc/internal/contracts/events"
)

// Bus defines the interface for event transport mechanisms
type Bus interface {
	// Publish sends an event to a specified topic
	Publish(ctx context.Context, topic string, event events.Event) error

	// PublishRaw sends raw JSON data to a specified topic
	PublishRaw(ctx context.Context, topic string, eventType string, data []byte) error

	// StartConsuming reads messages from all configured topics and dispatches them to handlers
	StartConsuming(ctx context.Context) error

	// RegisterHandler registers a typed event handler for any event type
	RegisterHandler(factory any, handler any) error

	// WriteTopic returns the topic used for writing
	WriteTopic() string

	// ReadTopics returns the list of topics used for reading
	ReadTopics() []string
}

// HandlerFunc defines a function type for handling typed events
type HandlerFunc[T events.Event] func(ctx context.Context, event T) error
