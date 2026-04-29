package eventhandlers

import (
	"context"
	"fmt"
	"log/slog"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/service/order"
)

// OnIdentityVerificationCompleted dispatches Kafka verification
// responses back to the waiting VerifyCustomerIdentity call.
type OnIdentityVerificationCompleted struct {
	service *order.OrderService
	logger    *slog.Logger
}

func NewOnIdentityVerificationCompleted(service *order.OrderService, logger *slog.Logger) *OnIdentityVerificationCompleted {
	if logger == nil {
		logger = slog.Default()
	}
	return &OnIdentityVerificationCompleted{
		service: service,
		logger:  logger.With("component", "on_identity_verification_completed"),
	}
}

func (h *OnIdentityVerificationCompleted) Handle(ctx context.Context, event events.Event) error {
	var respEvent events.CustomerIdentityVerificationCompletedEvent
	switch e := event.(type) {
	case events.CustomerIdentityVerificationCompletedEvent:
		respEvent = e
	case *events.CustomerIdentityVerificationCompletedEvent:
		respEvent = *e
	default:
		return nil
	}

	if respEvent.EventType != events.CustomerIdentityVerificationCompleted {
		h.logger.Debug("Ignoring non-verification event", "event_type", string(respEvent.EventType))
		return nil
	}

	if respEvent.Data.RequestID == "" {
		h.logger.Warn("Received verification response with empty request ID")
		return nil
	}

	var err error
	if !respEvent.Data.Authorized && respEvent.Data.Error != "" {
		err = fmt.Errorf("%s", respEvent.Data.Error)
	}

	h.service.DispatchVerificationResult(
		respEvent.Data.RequestID,
		order.CustomerIdentity{
			CustomerID:  respEvent.Data.CustomerID,
			Email:       respEvent.Data.ResolvedEmail,
			},
		err,
	)
	return nil
}

func (h *OnIdentityVerificationCompleted) EventType() string {
	return string(events.CustomerIdentityVerificationCompleted)
}

func (h *OnIdentityVerificationCompleted) CreateFactory() events.EventFactory[events.CustomerIdentityVerificationCompletedEvent] {
	return events.CustomerIdentityVerificationCompletedEventFactory{}
}

func (h *OnIdentityVerificationCompleted) CreateHandler() bus.HandlerFunc[events.CustomerIdentityVerificationCompletedEvent] {
	return func(ctx context.Context, event events.CustomerIdentityVerificationCompletedEvent) error {
		return h.Handle(ctx, event)
	}
}
