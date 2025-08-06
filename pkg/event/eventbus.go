package event

import (
	"context"
	"encoding/json"
	"go-shopping-poc/pkg/logging"
	"sync"

	"github.com/segmentio/kafka-go"
)

// Handler is a function that handles/processes an event.

type Handler func(Event[any])

// EventHandler defines the interface for handling events
type EventHandler interface {
	Handle(ctx context.Context, event Event[any]) error
}

// KafkaEventBus allows subscribing to and publishing events via Kafka.
// It supports multiple topics for reading and writing.

type EventBus struct {
	writers  map[string]*kafka.Writer
	readers  map[string]*kafka.Reader
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

// NewEventBus creates a Kafka event bus without authentication (PLAINTEXT).

func NewEventBus(broker string, readTopics []string, writeTopics []string, groupID string) *EventBus {

	writers := make(map[string]*kafka.Writer)
	for _, topic := range writeTopics {
		logging.Info("Creating Kafka writer for topic: %s", topic)
		writers[topic] = &kafka.Writer{
			Addr:     kafka.TCP(broker),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		}
	}

	readers := make(map[string]*kafka.Reader)
	for _, topic := range readTopics {
		logging.Info("Creating Kafka reader for topic: %s", topic)
		readers[topic] = kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{broker},
			Topic:   topic,
			GroupID: groupID,
			Dialer:  &kafka.Dialer{},
		})
	}

	return &EventBus{
		writers:  writers,
		readers:  readers,
		handlers: make(map[string][]EventHandler),
	}
}

func (eb *EventBus) WriteTopics() []string {
	var topics []string
	for topic := range eb.writers {
		topics = append(topics, topic)
	}
	return topics
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
	logging.Info("EventBus - subscribed to event: %s", eventType)
}

// Publish sends an event to a specified Kafka topic.

func (eb *EventBus) Publish(ctx context.Context, topic string, event *Event[any]) error {
	logging.Debug("EventBus - Publishing event to topic: %s, event type: %s", topic, event.Type)
	writer, ok := eb.writers[topic]
	if !ok {
		logging.Debug("EventBus - No writer found for topic: %s", topic)
		return ErrUnknownTopic(topic)
	}

	value, err := event.ToJSON()
	if err != nil {
		logging.Error("EventBus - Failed to convert event to JSON: %v", err)
		return err
	}
	return writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.Type),
		Value: value,
	})
}

// StartConsuming starts consuming messages from all configured Kafka topics and dispatches them to handlers.

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
				// Transform the message into an Event

				// evt, err := eventFactory(string(m.Key), m.Value)
				// if err != nil {
				// 	continue
				// }

				logging.Debug("EventBus - Atttempting to handle event of type: %s with payload: %s", string(m.Key), string(m.Value))

				var event Event[any]
				if err := json.Unmarshal(m.Value, &event); err != nil {
					logging.Error("Failed to unmarshal event: %v", err)
					continue
				}

				logging.Debug("EventBus - Received event of type: %s from topic: %s", event.Type, topic)

				for _, handler := range handlers {
					go handler.Handle(ctx, event) // Call handler in a goroutine
					logging.Debug("EventBus - Dispatched event of type: %s to handler", event.Type)
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
	return "unknown topic: " + string(e)
}

// ErrInvalidPayloadType is returned when the event payload is not a []byte.
type ErrInvalidPayloadType string

func (e ErrInvalidPayloadType) Error() string {
	return "invalid payload type for event: " + string(e)
}
