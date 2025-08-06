package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-shopping-poc/pkg/config"
	"go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
)

// ExampleEvent is a custom event with a payload.
type ExampleEvent struct {
	ExampleID   string
	ExampleData string
}

func (e ExampleEvent) Name() string { return "ExampleEvent" }
func (e ExampleEvent) Payload() interface{} {
	b, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	return b
}

func main() {

	logging.SetLevel("DEBUG")
	logging.Info("EventWriter service started")

	// Load configuration

	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	logging.Info("Configuration loaded from %s", envFile)
	logging.Info("Config: %v", cfg)

	broker := cfg.KafkaBroker
	readTopics := cfg.GetEventWriterKafkaReadTopics()
	writeTopics := cfg.GetEventWriterKafkaWriteTopics()
	groupID := cfg.GetKafkaGroupEventExample()

	logging.Info("Kafka Broker: %s, ReadTopics: %v, Write Topics: %v, Group ID: %s", broker, readTopics, writeTopics, groupID)

	bus := event.NewEventBus(broker, readTopics, writeTopics, groupID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(writeTopics) > 0 {
		logging.Info("Configured topics: %v", writeTopics)
	} else {
		logging.Error("No write topics configured")
		return
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// Start writing events to write topics
loop:
	for {
		select {
		case <-sig:
			logging.Info("Received shutdown signal, shutting down...")
			break loop
		default:
			for _, topic := range writeTopics {
				logging.Info("Publishing event to topic: %s", topic)
				err := bus.Publish(ctx, topic, &event.Event[any]{
					ID:        "example-id",
					Type:      "ExampleEvent",
					TimeStamp: time.Now(),
					Payload:   ExampleEvent{ExampleID: "123", ExampleData: "Example Data"},
				})
				if err != nil {
					logging.Error("Error publishing event to topic %s: %v", topic, err)
					continue
				}
				logging.Info("Event published to topic: %s", topic)
			}
			logging.Info("Sleeping...")
			time.Sleep(time.Second * 20)
		}
	}
}
