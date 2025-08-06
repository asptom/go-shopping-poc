package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	events "go-shopping-poc/internal/event/customer"
	"go-shopping-poc/pkg/config"
	"go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
)

// CustomerCreatedHandler handles CustomerCreatedEvent
type CustomerCreatedHandler struct{}

// Handle processes a CustomerCreatedEvent
func (h *CustomerCreatedHandler) Handle(ctx context.Context, event event.Event[any]) error {
	logging.Debug("CustomerCreatedHandler: Handling event of type: %s", event.Type)

	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to convert event to JSON: %w", err)
	}

	var customerPayload events.CustomerCreatedPayload
	if err := json.Unmarshal(payload, &customerPayload); err != nil {
		return err
	}
	logging.Info("CustomerCreatedHandler: Handling CustomerCreated with data: CustomerID=%s, UserName=%s, Email=%s",
		customerPayload.Customer.CustomerID, customerPayload.Customer.Username, customerPayload.Customer.Email)
	return nil
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

	bus := event.NewEventBus(broker, readTopics, writeTopics, groupID)
	var custEvent events.CustomerCreatedEvent
	bus.Subscribe(custEvent.GetType(), &CustomerCreatedHandler{})

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
		if err := bus.StartConsuming(ctx); err != nil {
			logging.Error("Kafka consumer stopped:", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logging.Info("Received shutdown signal, shutting down...")
}
