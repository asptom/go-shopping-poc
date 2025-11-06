package eventbus

import (
	"context"
	"encoding/json"
	event "go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
	"sync"

	"github.com/segmentio/kafka-go"
)

// Handler is a function that handles/processes an event.

// Handler func(Event[any])

// EventHandler defines the interface for handling events
type EventHandler interface {
	Handle(ctx context.Context, event event.Event) error
}

// EventBus enables subscribing to and publishing events via Kafka.
// It supports multiple topics for reading and one for writing.

type EventBus struct {
	writer   *kafka.Writer
	readers  map[string]*kafka.Reader
	handlers map[string][]EventHandler
	mu       sync.RWMutex
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
		writer:   writer,
		readers:  readers,
		handlers: make(map[string][]EventHandler),
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

// Subscribe adds a handler for a specific event type.

func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	logging.Info("Eventbus: subscribed to event: %s", eventType)
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
				eb.mu.RLock()
				handlers := eb.handlers[string(m.Key)]
				eb.mu.RUnlock()
				if len(handlers) == 0 {
					continue
				}

				logging.Debug("Eventbus: Atttempting to handle event of type: %s with payload: %s", string(m.Key), string(m.Value))

				// Use the event registry to obtain a concrete event.Event from stored payload
				evt, err := event.UnmarshalEvent(string(m.Key), m.Value)
				if err != nil {
					logging.Error("Eventbus - Failed to unmarshal event payload for key %s: %v", string(m.Key), err)
					continue
				}
				logging.Info("Eventbus: Received event of type: %s from topic: %s", evt.Type(), topic)

				for _, handler := range handlers {
					// capture handler and evt for the goroutine
					h := handler
					e := evt
					go func() {
						if err := h.Handle(ctx, e); err != nil {
							logging.Error("Eventbus: handler error for event %s: %v", e.Type(), err)
						}
					}()
					logging.Debug("Eventbus: Dispatched event of type: %s to handler", evt.Type())
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
