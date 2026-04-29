package eventhandlers

import (
	"context"
	"log/slog"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/service/order"
)

// OnCustomerIdentityUpdate keeps the identity cache current by processing
// CustomerCreated and CustomerUpdated events.
type OnCustomerIdentityUpdate struct {
	cache  *order.IdentityCache
	logger *slog.Logger
}

func NewOnCustomerIdentityUpdate(cache *order.IdentityCache, logger *slog.Logger) *OnCustomerIdentityUpdate {
	if logger == nil {
		logger = slog.Default()
	}
	return &OnCustomerIdentityUpdate{
		cache:  cache,
		logger: logger.With("component", "on_customer_identity_update"),
	}
}

func (h *OnCustomerIdentityUpdate) Handle(ctx context.Context, event events.Event) error {
	switch e := event.(type) {
	case events.CustomerEvent:
		if e.EventType != events.CustomerCreated && e.EventType != events.CustomerUpdated {
			return nil
			}
		keycloakSub := e.EventPayload.Details["keycloak_sub"]
		if keycloakSub == "" {
			return nil // not linked to Keycloak
			}
		h.cache.Set(keycloakSub, order.CustomerIdentity{
			CustomerID:  e.EventPayload.CustomerID,
			Email:       e.EventPayload.Details["email"],
			KeycloakSub: keycloakSub,
			})
		h.logger.Debug("Identity cache updated",
			"customer_id", e.EventPayload.CustomerID,
			"event_type", string(e.EventType))
	case *events.CustomerEvent:
		return h.Handle(ctx, *e)
	default:
		return nil
	}
	return nil
}

func (h *OnCustomerIdentityUpdate) EventType() string {
	return string(events.CustomerCreated)
}

func (h *OnCustomerIdentityUpdate) CreateFactory() events.EventFactory[events.CustomerEvent] {
	return events.CustomerEventFactory{}
}

func (h *OnCustomerIdentityUpdate) CreateHandler() bus.HandlerFunc[events.CustomerEvent] {
	return func(ctx context.Context, event events.CustomerEvent) error {
		return h.Handle(ctx, event)
	}
}
