package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go-shopping-poc/pkg/config"
	"go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
)

// ExampleEvent is a custom event with a payload.
type ExampleEvent struct {
	ExampleID   string
	ExampleData string
}

func (e ExampleEvent) Name() string { return "CustomerEvent" }
func (e ExampleEvent) Payload() interface{} {
	b, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	return b
}

// eventFactory creates an Event from Kafka message data.
func eventFactory(name string, payload []byte) (event.Event, error) {
	switch name {
	case "CustomerEvent":
		var e ExampleEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			return nil, err
		}
		return e, nil
	default:
		return nil, fmt.Errorf("unknown event type: %s", name)
	}
}

func main() {

	logging.SetLevel("DEBUG")
	logging.Info("EventReader service started")

	// Load configuration

	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	logging.Info("Configuration loaded from %s", envFile)
	logging.Info("Config: %v", cfg)

	broker := cfg.KafkaBroker
	readTopics := cfg.GetEventReaderKafkaReadTopics()
	writeTopics := cfg.GetEventReaderKafkaWriteTopics()
	groupID := cfg.GetKafkaGroupEventExample()

	logging.Info("Kafka Broker: %s, ReadTopics: %v, Write Topics: %v, Group ID: %s", broker, readTopics, writeTopics, groupID)

	bus := event.NewKafkaEventBus(broker, readTopics, writeTopics, groupID)

	// Subscribe to CustomerEvent events
	logging.Info("Subscribing to CustomerEvent on topics: %v", readTopics)
	bus.Subscribe("CustomerEvent", func(e event.Event) {
		data, ok := e.Payload().([]byte)
		if !ok {
			logging.Error("Payload is not []byte")
			return
		}
		var payload ExampleEvent
		if err := json.Unmarshal(data, &payload); err != nil {
			logging.Error("Failed to unmarshal payload: %v", err)
			return
		}
		logging.Info("Received event: %s with payload: %s", e.Name(), string(data))
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consuming in a goroutine
	go func() {
		if err := bus.StartConsuming(ctx, eventFactory); err != nil {
			logging.Error("Kafka consumer stopped:", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logging.Info("Received shutdown signal, shutting down...")
}
