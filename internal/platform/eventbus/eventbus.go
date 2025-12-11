package eventbus

import (
	"context"
	"encoding/json"
	event "go-shopping-poc/internal/platform/event"
	"go-shopping-poc/internal/platform/logging"
	"sync"

	"github.com/segmentio/kafka-go"
)

// Handler is a function that handles/processes an event.

// Handler func(Event[any])

// EventBus enables subscribing to and publishing events via Kafka.
// It supports multiple topics for reading and one for writing.

type EventBus struct {
	writer        *kafka.Writer
	readers       map[string]*kafka.Reader
	typedHandlers map[string][]func(ctx context.Context, data []byte) error
	mu            sync.RWMutex
}

// NewEventBus creates a Kafka event bus without authentication (PLAINTEXT).

func NewEventBus(broker string, readTopics []string, writeTopic string, group string) *EventBus {

	writer := &kafka.Writer{
		Addr:     kafka.TCP(broker),
		Topic:    writeTopic,
		Balancer: &kafka.LeastBytes{},
	}

	readers := make(map[string]*kafka.Reader)
	for _, topic := range readTopics {
		logging.Debug("Eventbus - Creating Kafka reader for topic: %s", topic)
		readers[topic] = kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{broker},
			Topic:   topic,
			GroupID: group,
			Dialer:  &kafka.Dialer{},
		})
	}

	return &EventBus{
		writer:        writer,
		readers:       readers,
		typedHandlers: make(map[string][]func(ctx context.Context, data []byte) error),
	}
}

func (eb *EventBus) WriteTopic() string {
	return eb.writer.Topic
}

func (eb *EventBus) ReadTopics() []string {
	var topics []string
	for topic := range eb.readers {
		topics = append(topics, topic)
	}
	return topics
}

// SubscribeTyped adds a type-safe handler for events of type T.
// The topic is automatically determined from the event's Topic() method.

func SubscribeTyped[T event.Event](eb *EventBus, factory event.EventFactory[T], handler HandlerFunc[T]) {
	// Create a dummy event to get the topic
	dummy := *new(T)
	topic := dummy.Topic()

	typedHandler := NewTypedHandler(factory, handler)

	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.typedHandlers[topic] = append(eb.typedHandlers[topic], typedHandler.Handle)
	logging.Info("Eventbus: subscribed to topic: %s with typed handler", topic)
}

// Publish sends an event to a specified Kafka topic.

func (eb *EventBus) Publish(ctx context.Context, topic string, event event.Event) error {
	logging.Debug("Eventbus: Publishing event to topic: %s, event type: %s", topic, event.Type())

	value, err := json.Marshal(event)
	if err != nil {
		logging.Error("Eventbus: Failed to convert event to JSON: %v", err)
		return err
	}
	return eb.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.Type()),
		Value: value,
	})
}

// PublishRaw sends raw JSON data to a specified Kafka topic.
// Used by the outbox publisher to avoid double marshaling.

func (eb *EventBus) PublishRaw(ctx context.Context, topic string, eventType string, data []byte) error {
	logging.Debug("Eventbus: Publishing raw event to topic: %s, event type: %s", topic, eventType)

	return eb.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(eventType),
		Value: data,
	})
}

// StartConsuming reads messages from all configured Kafka read topics and dispatches them to handlers.

func (eb *EventBus) StartConsuming(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(eb.readers))

	for topic, reader := range eb.readers {
		wg.Add(1)
		go func(topic string, reader *kafka.Reader) {
			defer wg.Done()
			for {
				m, err := reader.ReadMessage(ctx)
				if err != nil {
					errCh <- err
					return
				}

				logging.Debug("Eventbus: Received message from topic: %s, key: %s", topic, string(m.Key))

				// Handle new typed handlers
				eb.mu.RLock()
				typedHandlers := eb.typedHandlers[topic]
				eb.mu.RUnlock()

				if len(typedHandlers) > 0 {
					logging.Debug("Eventbus: Processing with typed handlers for topic: %s", topic)

					for _, handler := range typedHandlers {
						h := handler
						data := m.Value // Raw JSON bytes
						go func() {
							if err := h(ctx, data); err != nil {
								logging.Error("Eventbus: typed handler error for topic %s: %v", topic, err)
							}
						}()
					}
				}

				// If no handlers found, log and continue
				if len(typedHandlers) == 0 {
					logging.Debug("Eventbus: No handlers found for topic: %s, key: %s", topic, string(m.Key))
				}
			}
		}(topic, reader)
	}

	// Wait for any error or context cancellation
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ErrUnknownTopic is returned when a topic is not configured for writing.
type ErrUnknownTopic string

func (e ErrUnknownTopic) Error() string {
	return "Eventbus - Unknown topic: " + string(e)
}

// ErrInvalidPayloadType is returned when the event payload is not a []byte.
type ErrInvalidPayloadType string

func (e ErrInvalidPayloadType) Error() string {
	return "Eventbus - Invalid payload type for event: " + string(e)
}
