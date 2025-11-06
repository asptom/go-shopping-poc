package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	events "go-shopping-poc/internal/event/customer"
	"go-shopping-poc/pkg/config"
	evpkg "go-shopping-poc/pkg/event"
	bus "go-shopping-poc/pkg/eventbus"
	"go-shopping-poc/pkg/logging"
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

	bus := bus.NewEventBus(broker, readTopics, writeTopic, group)
	handler := events.CustomerEventHandler{
		Callback: events.CustomerEventCallback(func(ctx context.Context, payload events.CustomerEventPayload) error {
			logging.Debug("Eventreader: Hooray - the callback worked")
			logging.Debug("Eventreader: Data in callback: CustomerID=%s, EventType=%s, ResourceID=%s",
				payload.CustomerID, payload.EventType, payload.ResourceID)
			return nil
		}),
	}

	// subscribe adapter (not the typed handler directly)
	bus.Subscribe(string(events.CustomerCreated), &customerHandlerAdapter{inner: &handler})
	bus.Subscribe(string(events.CustomerUpdated), &customerHandlerAdapter{inner: &handler})
	bus.Subscribe(string(events.AddressAdded), &customerHandlerAdapter{inner: &handler})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logging.Debug("Eventreader: Starting event consumer...")
	// Start consuming in a goroutine
	go func() {
		logging.Debug("Eventreader: Event consumer started")
		if err := bus.StartConsuming(ctx); err != nil {
			logging.Error("Eventreader: Event consumer stopped:", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logging.Debug("Eventreader: Received shutdown signal, shutting down...")
}

// adapter to convert generic event.Event into events.CustomerEvent and delegate
type customerHandlerAdapter struct {
	inner *events.CustomerEventHandler
}

func (a *customerHandlerAdapter) Handle(ctx context.Context, ev evpkg.Event) error {
	// try concrete types: value or pointer
	switch v := ev.(type) {
	case *events.CustomerEvent:
		return a.inner.Handle(ctx, *v)
	default:
		return fmt.Errorf("unexpected event type: %T", ev)
	}
}
