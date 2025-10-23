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

	broker := cfg.GetEventBroker()
	readTopics := cfg.GetEventWriterReadTopics()
	writeTopic := cfg.GetEventWriterWriteTopic()
	groupID := cfg.GetEventWriterGroup()

	logging.Info("Event Broker: %s, Read Topics: %v, Write Topic: %v, Group: %s", broker, readTopics, writeTopic, groupID)

	bus := event.NewEventBus(broker, readTopics, writeTopic, groupID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(writeTopic) > 0 {
		logging.Info("Configured write topic: %v", writeTopic)
	} else {
		logging.Error("No write topic configured")
		return
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// Start writing events to write topic
loop:
	for {
		select {
		case <-sig:
			logging.Info("Received shutdown signal, shutting down...")
			break loop
		default:

			logging.Info("Publishing event to topic: %s", writeTopic)
			err := bus.Publish(ctx, writeTopic, &event.Event[any]{
				ID:        "example-id",
				Type:      "ExampleEvent",
				TimeStamp: time.Now(),
				Payload:   ExampleEvent{ExampleID: "123", ExampleData: "Example Data"},
			})
			if err != nil {
				logging.Error("Error publishing event to topic %s: %v", writeTopic, err)
				continue
			}
			logging.Info("Event published to topic: %s", writeTopic)

			logging.Info("Sleeping...")
			time.Sleep(time.Second * 20)
		}
	}
}
