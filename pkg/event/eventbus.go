package event

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/segmentio/kafka-go"
)

// Handler is a function that handles an Event.
type Handler func(Event)

// KafkaEventBus allows subscribing to and publishing events via Kafka.
// It supports multiple topics for reading and writing.
type KafkaEventBus struct {
	writers  map[string]*kafka.Writer
	readers  map[string]*kafka.Reader
	handlers map[string][]Handler
	mu       sync.RWMutex
}

// NewKafkaEventBus creates a new KafkaEventBus.
// Accepts a single broker as a string, and slices of topic names to read from and write to.
func NewKafkaEventBus(broker string, readTopics []string, writeTopics []string, groupID string) *KafkaEventBus {
	writers := make(map[string]*kafka.Writer)
	for _, topic := range writeTopics {
		writers[topic] = &kafka.Writer{
			Addr:     kafka.TCP(broker),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		}
	}

	readers := make(map[string]*kafka.Reader)
	for _, topic := range readTopics {
		readers[topic] = kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{broker},
			Topic:   topic,
			GroupID: groupID,
		})
	}

	return &KafkaEventBus{
		writers:  writers,
		readers:  readers,
		handlers: make(map[string][]Handler),
	}
}

// Subscribe adds a handler for a specific event name.
func (eb *KafkaEventBus) Subscribe(eventName string, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventName] = append(eb.handlers[eventName], handler)
}

// Publish sends an event to a specified Kafka topic.
func (eb *KafkaEventBus) Publish(ctx context.Context, topic string, event Event) error {
	writer, ok := eb.writers[topic]
	if !ok {
		return ErrUnknownTopic(topic)
	}
	payload, err := json.Marshal(event.Payload())
	if err != nil {
		return err
	}
	return writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.Name()),
		Value: payload,
	})
}

// StartConsuming starts consuming messages from all configured Kafka topics and dispatches them to handlers.
func (eb *KafkaEventBus) StartConsuming(ctx context.Context, eventFactory func(name string, payload []byte) (Event, error)) error {
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
				evt, err := eventFactory(string(m.Key), m.Value)
				if err != nil {
					continue
				}
				for _, handler := range handlers {
					go handler(evt)
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
