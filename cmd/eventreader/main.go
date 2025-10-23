package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	events "go-shopping-poc/internal/event/customer"
	"go-shopping-poc/pkg/config"
	"go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
)

func main() {

	logging.SetLevel("DEBUG")
	logging.Info("EventReader service started")

	// Load configuration

	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	logging.Info("Configuration loaded from %s", envFile)
	logging.Info("Config: %v", cfg)

	broker := cfg.GetEventBroker()
	readTopics := cfg.GetEventReaderReadTopics()
	writeTopic := cfg.GetEventReaderWriteTopic()
	group := cfg.GetEventReaderGroup()

	logging.Info("Event Broker: %s, Read Topics: %v, Write Topic: %v, Group: %s", broker, readTopics, writeTopic, group)

	bus := event.NewEventBus(broker, readTopics, writeTopic, group)

	handler := &events.CustomerCreatedHandler{
		Callback: func(ctx context.Context, payload events.CustomerCreatedPayload) error {
			logging.Debug("Hooray - the callback worked")
			logging.Debug("Data in callback: %s, UserName: %s, Email: %s",
				payload.Customer.CustomerID, payload.Customer.Username, payload.Customer.Email)
			return nil
		},
	}
	var custEvent events.CustomerCreatedEvent
	bus.Subscribe(custEvent.GetType(), handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logging.Info("Starting event consumer...")
	// Start consuming in a goroutine
	go func() {
		logging.Debug("Event consumer started")
		if err := bus.StartConsuming(ctx); err != nil {
			logging.Error("Event consumer stopped:", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logging.Info("Received shutdown signal, shutting down...")
}
