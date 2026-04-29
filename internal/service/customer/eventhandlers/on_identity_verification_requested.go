package eventhandlers

import (
	"context"
	"fmt"
	"log/slog"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/service/customer"
)

// OnIdentityVerificationRequested handles identity verification requests from
// the order service. It verifies that the given email/keycloak_sub pair
// corresponds to a known customer and publishes the result back via events.
type OnIdentityVerificationRequested struct {
	service customer.CustomerService
	logger  *slog.Logger
}

func NewOnIdentityVerificationRequested(service customer.CustomerService, logger *slog.Logger) *OnIdentityVerificationRequested {
	if logger == nil {
		logger = slog.Default()
	}
	return &OnIdentityVerificationRequested{
		service: service,
		logger:  logger.With("component", "on_identity_verification_requested"),
	}
}

func (h *OnIdentityVerificationRequested) Handle(ctx context.Context, event events.Event) error {
	var reqEvent events.CustomerIdentityVerificationRequestEvent
	switch e := event.(type) {
	case events.CustomerIdentityVerificationRequestEvent:
		reqEvent = e
	case *events.CustomerIdentityVerificationRequestEvent:
		reqEvent = *e
	default:
		return nil
	}

	if reqEvent.EventType != events.CustomerIdentityVerificationRequested {
		h.logger.Debug("Ignoring non-verification event", "event_type", string(reqEvent.EventType))
		return nil
	}

	cust, err := h.service.GetCustomerByEmail(ctx, reqEvent.Data.Email)
	if err != nil {
		h.publishResult(ctx, reqEvent.Data.RequestID, false, "", "", fmt.Sprintf("customer lookup error: %v", err))
		return nil
	}

	if cust == nil {
		h.publishResult(ctx, reqEvent.Data.RequestID, false, "", "", "customer not found")
		return nil
	}

	if cust.KeycloakSub == "" {
		h.publishResult(ctx, reqEvent.Data.RequestID, false, "", "", "customer missing keycloak_sub linkage")
		return nil
	}
	if cust.KeycloakSub != reqEvent.Data.KeycloakSub {
		h.publishResult(ctx, reqEvent.Data.RequestID, false, "", "", "token subject does not match customer")
		return nil
	}

	h.publishResult(ctx, reqEvent.Data.RequestID, true, cust.CustomerID, cust.Email, "")
	return nil
}

func (h *OnIdentityVerificationRequested) publishResult(ctx context.Context, requestID string, authorized bool, customerID, email, errStr string) {
	respEvent := events.NewCustomerIdentityVerificationCompletedEvent(
		requestID, authorized, customerID, email, errStr,
	)
	if err := h.service.EventBus().Publish(ctx, respEvent.Topic(), respEvent); err != nil {
		h.logger.Error("Failed to publish verification result", "request_id", requestID, "error", err)
	}
}

func (h *OnIdentityVerificationRequested) EventType() string {
	return string(events.CustomerIdentityVerificationRequested)
}

func (h *OnIdentityVerificationRequested) CreateFactory() events.EventFactory[events.CustomerIdentityVerificationRequestEvent] {
	return events.CustomerIdentityVerificationRequestEventFactory{}
}

func (h *OnIdentityVerificationRequested) CreateHandler() bus.HandlerFunc[events.CustomerIdentityVerificationRequestEvent] {
	return func(ctx context.Context, event events.CustomerIdentityVerificationRequestEvent) error {
		return h.Handle(ctx, event)
	}
}
