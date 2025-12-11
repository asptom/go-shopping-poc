package events

import (
	"context"

	"go-shopping-poc/internal/platform/logging"
)

// Define the callback type
type CustomerEventCallback func(ctx context.Context, payload CustomerEventPayload) error

// CustomerEventHandler handles CustomerEvent and also allows for custom callbacks
type CustomerEventHandler struct {
	Callback CustomerEventCallback
}

// Handle processes a CustomerEvent
func (h *CustomerEventHandler) Handle(ctx context.Context, event CustomerEvent) error {
	logging.Debug("CustomerEventHandler: Handling event of type: %s", event.Type())

	logging.Debug("CustomerEventHandler: Payload information: CustomerId=%s, EventType=%s, ResourceID=%s",
		event.EventPayload.CustomerID, event.EventPayload.EventType, event.EventPayload.ResourceID)

	// Call the custom callback if set
	if h.Callback != nil {
		return h.Callback(ctx, event.EventPayload)
	}
	return nil
}
