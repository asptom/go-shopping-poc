package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	events "go-shopping-poc/internal/event/customer"
	"go-shopping-poc/pkg/config"
	"go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
)

// eventFactory creates an Event from Kafka message data.
func eventFactory(name string, payload []byte) (event.Event, error) {
	logging.Debug("eventFactory - received event of type %s", name)
	switch name {
	case "CustomerCreated":
		return events.CustomerCreatedEventFactory(name, payload)
	default:
		logging.Error("Unknown event type: %s", name)
		return nil, fmt.Errorf("unknown event type: %s", name)
	}
}

func customerCreatedHandler(e event.Event) {
	logging.Debug("customerCreatedHandler - received event %s", e.Name())
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
	var custEvent events.CustomerCreatedEvent
	bus.Subscribe(custEvent.Name(), customerCreatedHandler)

	// Subscribe to CustomerEvent events
	//logging.Info("Subscribing to CustomerEvent on topics: %v", readTopics)
	// bus.Subscribe("CustomerEvent", func(e event.Event) {
	// 	data, ok := e.Payload().([]byte)
	// 	if !ok {
	// 		logging.Error("Payload is not []byte")
	// 		return
	// 	}
	// 	var payload events.CustomerCreatedEvent
	// 	if err := json.Unmarshal(data, &payload); err != nil {
	// 		logging.Error("Failed to unmarshal payload: %v", err)
	// 		return
	// 	}
	// 	logging.Info("Received event: %s with payload: %s", e.Name(), string(data))
	// })
	//bus.Subscribe("CustomerCreated", eventFactory)
	//logging.Debug("Subscribed and received CustomerCreated event: %s", e.Name())
	// if _, err := eventFactory(e.Name(), e.Payload().([]byte)); err != nil {
	// 	logging.Error("Failed to process event: %v", err)
	// }
	//})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logging.Info("Starting Kafka consumer...")
	// Start consuming in a goroutine
	go func() {
		logging.Debug("Kafka consumer started")
		if err := bus.StartConsuming(ctx, eventFactory); err != nil {
			logging.Error("Kafka consumer stopped:", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logging.Info("Received shutdown signal, shutting down...")
}
