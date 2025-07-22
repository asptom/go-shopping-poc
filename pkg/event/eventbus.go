package event

import (
	"context"
	"go-shopping-poc/pkg/logging"
	"sync"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
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

// NewKafkaEventBus creates a new KafkaEventBus without authentication (PLAINTEXT).
func NewKafkaEventBus(broker string, readTopics []string, writeTopics []string, groupID string) *KafkaEventBus {
	return newKafkaEventBusInternal(broker, readTopics, writeTopics, groupID, nil)
}

// NewKafkaEventBusWithAuth creates a new KafkaEventBus with SASL/PLAIN authentication.
func NewKafkaEventBusWithAuth(broker string, readTopics []string, writeTopics []string, groupID string, username, password string) *KafkaEventBus {
	var dialer *kafka.Dialer
	if username != "" && password != "" {
		dialer = &kafka.Dialer{
			SASLMechanism: plain.Mechanism{
				Username: username,
				Password: password,
			},
			// Uncomment if your broker requires TLS (SASL_SSL)
			// TLS: &tls.Config{},
		}
	}
	return newKafkaEventBusInternal(broker, readTopics, writeTopics, groupID, dialer)
}

// newKafkaEventBusInternal is a helper to create a KafkaEventBus with or without a custom dialer.
func newKafkaEventBusInternal(broker string, readTopics []string, writeTopics []string, groupID string, dialer *kafka.Dialer) *KafkaEventBus {
	if dialer == nil {
		dialer = &kafka.Dialer{}
	}

	writers := make(map[string]*kafka.Writer)
	for _, topic := range writeTopics {
		logging.Info("Creating Kafka writer for topic: %s", topic)
		writers[topic] = &kafka.Writer{
			Addr:     kafka.TCP(broker),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
			// Transport: &kafka.Transport{
			// 	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			// 		return dialer.DialContext(ctx, network, address)
			// 	},
			// },
		}
	}

	readers := make(map[string]*kafka.Reader)
	for _, topic := range readTopics {
		logging.Info("Creating Kafka reader for topic: %s", topic)
		readers[topic] = kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{broker},
			Topic:   topic,
			GroupID: groupID,
			Dialer:  dialer,
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
	logging.Info("KafkaEventBus - subscribed to event: %s", eventName)
}

// Publish sends an event to a specified Kafka topic.
func (eb *KafkaEventBus) Publish(ctx context.Context, topic string, event Event) error {
	logging.Debug("KafkaEventBus - Publishing event to topic: %s, event name: %s", topic, event.Name())
	writer, ok := eb.writers[topic]
	if !ok {
		logging.Debug("KafkaEventBus - No writer found for topic: %s", topic)
		return ErrUnknownTopic(topic)
	}

	// Handle payload conversion
	var value []byte
	switch payload := event.Payload().(type) {
	case []byte:
		value = payload
	case string:
		value = []byte(payload)
	default:
		return ErrInvalidPayloadType(event.Name())
	}
	return writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.Name()),
		Value: value,
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

// ErrInvalidPayloadType is returned when the event payload is not a []byte.
type ErrInvalidPayloadType string

func (e ErrInvalidPayloadType) Error() string {
	return "invalid payload type for event: " + string(e)
}
