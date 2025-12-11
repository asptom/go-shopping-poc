package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	events "go-shopping-poc/internal/event/customer"
	"go-shopping-poc/internal/platform/config"
	"go-shopping-poc/internal/platform/eventbus"
	"go-shopping-poc/internal/platform/logging"
)

func main() {

	logging.SetLevel("INFO")
	logging.Info("Eventreader: EventReader service started")

	// Load configuration

	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	logging.Debug("Eventreader: Configuration loaded from %s", envFile)
	logging.Debug("Eventreader: Config: %v", cfg)

	broker := cfg.GetEventBroker()
	readTopics := cfg.GetEventReaderReadTopics()
	writeTopic := cfg.GetEventReaderWriteTopic()
	group := cfg.GetEventReaderGroup()

	logging.Debug("Eventreader: Event Broker: %s, Read Topics: %v, Write Topic: %v, Group: %s", broker, readTopics, writeTopic, group)

	eventBus := eventbus.NewEventBus(broker, readTopics, writeTopic, group)

	// Create factory and handler using the new typed system
	factory := events.CustomerEventFactory{}
	handler := eventbus.HandlerFunc[events.CustomerEvent](func(ctx context.Context, evt events.CustomerEvent) error {
		logging.Info("Eventreader: Processing customer event: %s", evt.Type())
		logging.Info("Eventreader: Data in event: CustomerID=%s, EventType=%s, ResourceID=%s",
			evt.EventPayload.CustomerID, evt.EventPayload.EventType, evt.EventPayload.ResourceID)
		return nil
	})

	// Subscribe using the new typed system - no adapter needed!
	eventbus.SubscribeTyped(eventBus, factory, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logging.Debug("Eventreader: Starting event consumer...")
	// Start consuming in a goroutine
	go func() {
		logging.Debug("Eventreader: Event consumer started")
		if err := eventBus.StartConsuming(ctx); err != nil {
			logging.Error("Eventreader: Event consumer stopped:", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logging.Debug("Eventreader: Received shutdown signal, shutting down...")
}
