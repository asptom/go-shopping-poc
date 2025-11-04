package events

import (
	"context"
	"encoding/json"
	"fmt"
	event "go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
)

// Define the callback type
type CustomerUpdatedCallback func(ctx context.Context, payload CustomerUpdatedPayload) error

// CustomerUpdatedHandler handles CustomerUpdatedEvent and also allows for custom callbacks
type CustomerUpdatedHandler struct {
	Callback CustomerUpdatedCallback
}

// Handle processes a CustomerUpdatedEvent
func (h *CustomerUpdatedHandler) Handle(ctx context.Context, event event.Event[any]) error {
	logging.Debug("CustomerUpdatedHandler: Handling event of type: %s", event.Type)

	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to convert event to JSON: %w", err)
	}

	var customerPayload CustomerUpdatedPayload
	if err := json.Unmarshal(payload, &customerPayload); err != nil {
		return err
	}
	logging.Info("CustomerUpdatedHandler: Handling CustomerUpdated with data: CustomerID=%s, UserName=%s, Email=%s",
		customerPayload.Customer.CustomerID, customerPayload.Customer.Username, customerPayload.Customer.Email)

	// Call the custom callback if set
	if h.Callback != nil {
		return h.Callback(ctx, customerPayload)
	}
	return nil
}
