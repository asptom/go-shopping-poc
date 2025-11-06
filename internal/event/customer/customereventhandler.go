package events

import (
	"context"

	"go-shopping-poc/pkg/logging"
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
		event.Event_Payload.CustomerID, event.Event_Payload.EventType, event.Event_Payload.ResourceID)

	// Call the custom callback if set
	if h.Callback != nil {
		return h.Callback(ctx, event.Event_Payload)
	}
	return nil
}
