package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-shopping-poc/pkg/config"
	"go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
)

// OrderCreatedEvent is a custom event with a payload.
type OrderCreatedEvent struct {
	OrderID string
	UserID  string
}

func (e OrderCreatedEvent) Name() string { return "OrderCreated" }
func (e OrderCreatedEvent) Payload() any { return e }

// eventFactory creates an Event from Kafka message data.
func eventFactory(name string, payload []byte) (event.Event, error) {
	switch name {
	case "OrderCreated":
		var e OrderCreatedEvent
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
	logging.Debug("This is a debug message: %v", 123)
	logging.Info("Service started")

	// Choose env file based on an ENV variable or default to development
	env := os.Getenv("APP_ENV")
	envFile := ".env.development"
	if env == "production" {
		envFile = ".env.production"
	}
	cfg := config.Load(envFile)

	broker := cfg.KafkaBroker               // Now a single broker string
	readTopics := []string{cfg.KafkaTopic}  // Add more topics if needed
	writeTopics := []string{cfg.KafkaTopic} // Add more topics if needed
	groupID := cfg.KafkaGroupID

	bus := event.NewKafkaEventBus(broker, readTopics, writeTopics, groupID)

	// Subscribe to OrderCreated events
	bus.Subscribe("OrderCreated", func(e event.Event) {
		payload := e.Payload().(OrderCreatedEvent)
		fmt.Printf("Order created: OrderID=%s, UserID=%s\n", payload.OrderID, payload.UserID)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consuming in a goroutine
	go func() {
		if err := bus.StartConsuming(ctx, eventFactory); err != nil {
			log.Println("Kafka consumer stopped:", err)
		}
	}()

	// Publish an event to the desired topic
	err := bus.Publish(ctx, cfg.KafkaTopic, OrderCreatedEvent{OrderID: "123", UserID: "u1"})
	if err != nil {
		log.Println("Publish error:", err)
		logging.Error("This is an error: %v", err)
	}

	// Wait for interrupt to exit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	fmt.Println("Shutting down...")
	logging.Warning("This is a warning")
}
