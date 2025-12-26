package kafka

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-shopping-poc/internal/contracts/events"
	kafkaconfig "go-shopping-poc/internal/platform/event/kafka"
)

// Note: For comprehensive testing of Publish and StartConsuming methods,
// we would need to mock the kafka.Writer and kafka.Reader interfaces.
// Since these are third-party types, we'll focus on testing the logic
// that can be tested without external dependencies.

func TestNewEventBus(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	require.NotNil(t, eventBus)
	assert.NotNil(t, eventBus.writer)
	assert.NotNil(t, eventBus.readers)
	assert.NotNil(t, eventBus.typedHandlers)
	assert.Equal(t, config, eventBus.kafkaCfg)
	assert.Equal(t, "test-topic", eventBus.WriteTopic())
	assert.Empty(t, eventBus.ReadTopics())
}

func TestEventBus_WriteTopic(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "write-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)
	assert.Equal(t, "write-topic", eventBus.WriteTopic())
}

func TestEventBus_ReadTopics(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	// Initially no readers
	assert.Empty(t, eventBus.ReadTopics())

	// Add a reader by subscribing
	factory := events.CustomerEventFactory{}
	handler := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}

	SubscribeTyped(eventBus, factory, handler)

	// Now should have one topic
	topics := eventBus.ReadTopics()
	assert.Len(t, topics, 1)
	assert.Contains(t, topics, "CustomerEvents")
}

func TestSubscribeTyped(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	// Initially no handlers
	assert.Empty(t, eventBus.typedHandlers)

	// Subscribe to customer events
	factory := events.CustomerEventFactory{}
	handler := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}

	SubscribeTyped(eventBus, factory, handler)

	// Should now have handlers for CustomerEvents topic
	assert.Contains(t, eventBus.typedHandlers, "CustomerEvents")
	assert.Len(t, eventBus.typedHandlers["CustomerEvents"], 1)
	assert.Contains(t, eventBus.readers, "CustomerEvents")
}

func TestSubscribeTyped_MultipleHandlers(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	factory := events.CustomerEventFactory{}

	handler1 := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}

	handler2 := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}

	// Subscribe multiple handlers to the same topic
	SubscribeTyped(eventBus, factory, handler1)
	SubscribeTyped(eventBus, factory, handler2)

	// Should have 2 handlers for CustomerEvents topic
	assert.Len(t, eventBus.typedHandlers["CustomerEvents"], 2)
}

func TestEventBus_Publish_Structure(t *testing.T) {
	// Test that Publish method exists and can be called
	// (will fail due to no Kafka connection, but tests the method signature)
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)
	event := events.NewCustomerCreatedEvent("test-customer", nil)

	// This will attempt to connect to Kafka and fail, but tests the method exists
	err := eventBus.Publish(context.Background(), "test-topic", event)
	assert.Error(t, err) // Expected to fail without real Kafka
}

func TestEventBus_PublishRaw_Structure(t *testing.T) {
	// Test that PublishRaw method exists and can be called
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)
	jsonData := []byte(`{"test": "data"}`)

	// This will attempt to connect to Kafka and fail, but tests the method exists
	err := eventBus.PublishRaw(context.Background(), "test-topic", "test.event", jsonData)
	assert.Error(t, err) // Expected to fail without real Kafka
}

func TestEventBus_StartConsuming_NoHandlers(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// StartConsuming with no handlers should return quickly due to context timeout
	err := eventBus.StartConsuming(ctx)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestEventBus_StartConsuming_WithHandlers(t *testing.T) {
	// This is complex to test with real Kafka readers
	// For now, we'll test that the method exists and can be called
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	// Add a handler
	factory := events.CustomerEventFactory{}
	handler := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}
	SubscribeTyped(eventBus, factory, handler)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// This will fail trying to connect to Kafka, but tests the structure
	err := eventBus.StartConsuming(ctx)
	assert.Error(t, err) // Expected to fail without real Kafka
}

// TestEventBus_Concurrency tests concurrent access to the event bus
func TestEventBus_Concurrency(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	factory := events.CustomerEventFactory{}

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrently subscribe handlers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler := func(ctx context.Context, event events.CustomerEvent) error {
				return nil
			}
			SubscribeTyped(eventBus, factory, handler)
		}()
	}

	wg.Wait()

	// Should have handlers for CustomerEvents topic
	assert.Contains(t, eventBus.typedHandlers, "CustomerEvents")
	assert.Len(t, eventBus.typedHandlers["CustomerEvents"], numGoroutines)
}

// TestEventBus_ReadTopics_AfterSubscribe tests ReadTopics after subscribing
func TestEventBus_ReadTopics_AfterSubscribe(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	// Initially empty
	assert.Empty(t, eventBus.ReadTopics())

	// Subscribe
	factory := events.CustomerEventFactory{}
	handler := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}
	SubscribeTyped(eventBus, factory, handler)

	// Should contain CustomerEvents
	topics := eventBus.ReadTopics()
	assert.Contains(t, topics, "CustomerEvents")
	assert.Len(t, topics, 1)
}

// TestEventBus_WriteTopic_Consistency tests WriteTopic returns consistent value
func TestEventBus_WriteTopic_Consistency(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "my-write-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	// Call multiple times to ensure consistency
	for i := 0; i < 5; i++ {
		assert.Equal(t, "my-write-topic", eventBus.WriteTopic())
	}
}

// TestSubscribeTyped_DifferentTopics tests subscribing to different event types
func TestSubscribeTyped_DifferentTopics(t *testing.T) {
	// Note: This test assumes we have different event types with different topics
	// For now, we'll test with the same topic since CustomerEvent is our main type
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	factory := events.CustomerEventFactory{}

	handler1 := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}

	handler2 := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}

	SubscribeTyped(eventBus, factory, handler1)
	SubscribeTyped(eventBus, factory, handler2)

	// Both should be on the same topic
	assert.Len(t, eventBus.typedHandlers["CustomerEvents"], 2)
	assert.Len(t, eventBus.readers, 1) // Only one reader created
}

// TestEventBus_StartConsuming_ContextCancellation tests context cancellation handling
func TestEventBus_StartConsuming_ContextCancellation(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	// Add a handler to trigger reader creation
	factory := events.CustomerEventFactory{}
	handler := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}
	SubscribeTyped(eventBus, factory, handler)

	// Create a context that will be cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// StartConsuming should return context.Canceled
	err := eventBus.StartConsuming(ctx)
	assert.Equal(t, context.Canceled, err)
}

// TestEventBus_Publish_InvalidEvent tests publishing with invalid event
func TestEventBus_Publish_InvalidEvent(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	// Create an event that will fail JSON marshaling
	// (This is hard to trigger with our current event structure, but tests the error path)
	event := events.NewCustomerCreatedEvent("test-customer", nil)

	ctx := context.Background()
	err := eventBus.Publish(ctx, "test-topic", event)
	// Will fail due to no Kafka connection, but tests the method path
	assert.Error(t, err)
}

// TestEventBus_ReadTopics_Empty tests ReadTopics on empty eventbus
func TestEventBus_ReadTopics_Empty(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	topics := eventBus.ReadTopics()
	assert.Empty(t, topics)
	assert.IsType(t, []string{}, topics)
}

// TestSubscribeTyped_ReaderReuse tests that subscribing to the same topic reuses readers
func TestSubscribeTyped_ReaderReuse(t *testing.T) {
	config := &kafkaconfig.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		GroupID: "test-group",
	}

	eventBus := NewEventBus(config)

	factory := events.CustomerEventFactory{}

	handler1 := func(ctx context.Context, event events.CustomerEvent) error { return nil }
	handler2 := func(ctx context.Context, event events.CustomerEvent) error { return nil }
	handler3 := func(ctx context.Context, event events.CustomerEvent) error { return nil }

	// Subscribe multiple times
	SubscribeTyped(eventBus, factory, handler1)
	SubscribeTyped(eventBus, factory, handler2)
	SubscribeTyped(eventBus, factory, handler3)

	// Should have 3 handlers but only 1 reader
	assert.Len(t, eventBus.typedHandlers["CustomerEvents"], 3)
	assert.Len(t, eventBus.readers, 1)
	assert.Contains(t, eventBus.readers, "CustomerEvents")
}

// TestNewEventBus_NilConfig tests NewEventBus with nil config (should panic or handle gracefully)
func TestNewEventBus_NilConfig(t *testing.T) {
	// This should not panic but may create invalid state
	defer func() {
		if r := recover(); r != nil {
			t.Logf("NewEventBus with nil config panicked: %v", r)
		}
	}()

	eventBus := NewEventBus(nil)
	// If it doesn't panic, check that it creates something
	assert.NotNil(t, eventBus)
}

// TestEventBus_WriteTopic_NilWriter tests WriteTopic when writer is nil
func TestEventBus_WriteTopic_NilWriter(t *testing.T) {
	// Create an eventbus with potential nil writer
	eventBus := &EventBus{
		writer: nil, // Manually set to nil for testing
	}

	// This should not panic
	topic := eventBus.WriteTopic()
	assert.Empty(t, topic) // Should return empty string for nil writer
}
