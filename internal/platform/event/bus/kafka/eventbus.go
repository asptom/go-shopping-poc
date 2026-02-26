package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	kafkaconfig "go-shopping-poc/internal/platform/event/kafka"

	"github.com/segmentio/kafka-go"
)

// EventBus enables subscribing to and publishing events via Kafka.
// It supports multiple topics for reading and one for writing.
type EventBus struct {
	writer        *kafka.Writer
	readers       map[string]*kafka.Reader
	typedHandlers map[string][]func(ctx context.Context, data []byte) error
	kafkaCfg      *kafkaconfig.Config
	mu            sync.RWMutex
	logger        *slog.Logger
}

func init() {
	logger = Logger()
}

// NewEventBus creates a Kafka event bus using the provided configuration.
func NewEventBus(kafkaCfg *kafkaconfig.Config) *EventBus {
	if kafkaCfg == nil {
		// Return a minimal eventbus that won't panic but won't work
		return &EventBus{
			readers:       make(map[string]*kafka.Reader),
			typedHandlers: make(map[string][]func(ctx context.Context, data []byte) error),
			kafkaCfg:      nil,
			logger:        logger,
		}
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaCfg.Brokers...),
		Topic:    kafkaCfg.Topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &EventBus{
		writer:        writer,
		readers:       make(map[string]*kafka.Reader),
		typedHandlers: make(map[string][]func(ctx context.Context, data []byte) error),
		kafkaCfg:      kafkaCfg,
		logger:        logger,
	}
}

func (eb *EventBus) WriteTopic() string {
	if eb.writer == nil {
		return ""
	}
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
// The topic is automatically determined from event's Topic() method.
// If no reader exists for the topic, a new Kafka reader is created.
func SubscribeTyped[T events.Event](eb *EventBus, factory events.EventFactory[T], handler bus.HandlerFunc[T]) {
	// Create a dummy event to get topic
	dummy := *new(T)
	topic := dummy.Topic()

	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Check if reader exists for this topic, create if not
	if _, exists := eb.readers[topic]; !exists {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers: eb.kafkaCfg.Brokers,
			Topic:   topic,
			GroupID: eb.kafkaCfg.GroupID,
		})
		eb.readers[topic] = reader
		eb.logger.Debug("created new Kafka reader for topic",
			"topic", topic)
	}

	typedHandler := NewTypedHandler(factory, handler)
	eb.typedHandlers[topic] = append(eb.typedHandlers[topic], typedHandler.Handle)
	eb.logger.Info("subscribed to topic",
		"topic", topic)
}

// Publish sends an event to a specified Kafka topic.
func (eb *EventBus) Publish(ctx context.Context, topic string, event events.Event) error {
	eb.logger.Debug("Publishing event",
		"topic", topic,
		"event_type", event.Type(),
	)

	value, err := json.Marshal(event)
	if err != nil {
		eb.logger.Error("Failed to convert event to JSON",
			"error", err,
		)
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
	eb.logger.Debug("Publishing raw event",
		"topic", topic,
		"event_type", eventType,
	)

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

				eb.logger.Debug("Received message",
					"topic", topic,
					"key", string(m.Key),
				)

				// Handle new typed handlers
				eb.mu.RLock()
				typedHandlers := eb.typedHandlers[topic]
				eb.mu.RUnlock()

				if len(typedHandlers) > 0 {
					eb.logger.Debug("Processing with typed handlers",
						"topic", topic,
					)

					for _, handler := range typedHandlers {
						h := handler
						data := m.Value // Raw JSON bytes
						go func() {
							if err := h(ctx, data); err != nil {
								eb.logger.Error("typed handler error",
									"topic", topic,
									"error", err,
								)
							}
						}()
					}
				}

				// If no handlers found, log and continue
				if len(typedHandlers) == 0 {
					eb.logger.Debug("No handlers found",
						"topic", topic,
						"key", string(m.Key),
					)
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
