package events

import (
	"context"
	"encoding/json"
	"fmt"
	event "go-shopping-poc/pkg/event"
	"go-shopping-poc/pkg/logging"
)

// Define the callback type
type CustomerCreatedCallback func(ctx context.Context, payload CustomerCreatedPayload) error

// CustomerCreatedHandler handles CustomerCreatedEvent and also allows for custom callbacks
type CustomerCreatedHandler struct {
	Callback CustomerCreatedCallback
}

// Handle processes a CustomerCreatedEvent
func (h *CustomerCreatedHandler) Handle(ctx context.Context, event event.Event[any]) error {
	logging.Debug("CustomerCreatedHandler: Handling event of type: %s", event.Type)

	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to convert event to JSON: %w", err)
	}

	var customerPayload CustomerCreatedPayload
	if err := json.Unmarshal(payload, &customerPayload); err != nil {
		return err
	}
	logging.Info("CustomerCreatedHandler: Handling CustomerCreated with data: CustomerID=%s, UserName=%s, Email=%s",
		customerPayload.Customer.CustomerID, customerPayload.Customer.Username, customerPayload.Customer.Email)

	// Call the custom callback if set
	if h.Callback != nil {
		return h.Callback(ctx, customerPayload)
	}
	return nil
}
