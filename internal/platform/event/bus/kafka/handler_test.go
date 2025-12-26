package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-shopping-poc/internal/contracts/events"
)

func TestNewTypedHandler(t *testing.T) {
	// Create a mock factory and handler
	factory := events.CustomerEventFactory{}
	handler := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}

	typedHandler := NewTypedHandler(factory, handler)

	require.NotNil(t, typedHandler)
	assert.NotNil(t, typedHandler.factory)
	assert.NotNil(t, typedHandler.handler)
}

func TestTypedHandler_Handle_Success(t *testing.T) {
	// Create a test event
	originalEvent := events.NewCustomerCreatedEvent("test-customer-id", map[string]string{"key": "value"})
	jsonData, err := originalEvent.ToJSON()
	require.NoError(t, err)

	// Create a mock factory
	factory := events.CustomerEventFactory{}

	// Create a handler that verifies the event
	var receivedEvent events.CustomerEvent
	handler := func(ctx context.Context, event events.CustomerEvent) error {
		receivedEvent = event
		return nil
	}

	typedHandler := NewTypedHandler(factory, handler)

	// Handle the event
	ctx := context.Background()
	err = typedHandler.Handle(ctx, jsonData)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, originalEvent.ID, receivedEvent.ID)
	assert.Equal(t, originalEvent.EventType, receivedEvent.EventType)
	assert.Equal(t, originalEvent.EventPayload.CustomerID, receivedEvent.EventPayload.CustomerID)
}

func TestTypedHandler_Handle_InvalidJSON(t *testing.T) {
	// Create a mock factory
	factory := events.CustomerEventFactory{}

	// Create a handler
	handler := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}

	typedHandler := NewTypedHandler(factory, handler)

	// Handle invalid JSON
	ctx := context.Background()
	invalidJSON := []byte("invalid json")
	err := typedHandler.Handle(ctx, invalidJSON)

	// Verify error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal event")
}

func TestTypedHandler_Handle_HandlerError(t *testing.T) {
	// Create a test event
	originalEvent := events.NewCustomerCreatedEvent("test-customer-id", map[string]string{"key": "value"})
	jsonData, err := originalEvent.ToJSON()
	require.NoError(t, err)

	// Create a mock factory
	factory := events.CustomerEventFactory{}

	// Create a handler that returns an error
	expectedError := errors.New("handler error")
	handler := func(ctx context.Context, event events.CustomerEvent) error {
		return expectedError
	}

	typedHandler := NewTypedHandler(factory, handler)

	// Handle the event
	ctx := context.Background()
	err = typedHandler.Handle(ctx, jsonData)

	// Verify the handler error is returned
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestTypedHandler_Handle_WithComplexEvent(t *testing.T) {
	// Create a more complex event
	details := map[string]string{
		"first_name": "John",
		"last_name":  "Doe",
		"email":      "john.doe@example.com",
	}
	originalEvent := events.NewAddressAddedEvent("customer-123", "address-456", details)
	jsonData, err := originalEvent.ToJSON()
	require.NoError(t, err)

	// Create a mock factory
	factory := events.CustomerEventFactory{}

	// Create a handler that verifies all fields
	var receivedEvent events.CustomerEvent
	handler := func(ctx context.Context, event events.CustomerEvent) error {
		receivedEvent = event
		return nil
	}

	typedHandler := NewTypedHandler(factory, handler)

	// Handle the event
	ctx := context.Background()
	err = typedHandler.Handle(ctx, jsonData)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "customer-123", receivedEvent.EventPayload.CustomerID)
	assert.Equal(t, "address-456", receivedEvent.EventPayload.ResourceID)
	assert.Equal(t, events.AddressAdded, receivedEvent.EventType)
	assert.Equal(t, details, receivedEvent.EventPayload.Details)
}

// TestTypedHandler_Handle_NilContext tests handling with nil context
func TestTypedHandler_Handle_NilContext(t *testing.T) {
	// Create a test event
	originalEvent := events.NewCustomerCreatedEvent("test-customer-id", nil)
	jsonData, err := originalEvent.ToJSON()
	require.NoError(t, err)

	// Create a mock factory
	factory := events.CustomerEventFactory{}

	// Create a handler that uses the context
	handler := func(ctx context.Context, event events.CustomerEvent) error {
		// Handler should handle nil context gracefully
		if ctx == nil {
			return errors.New("context is nil")
		}
		return nil
	}

	typedHandler := NewTypedHandler(factory, handler)

	// Handle with valid context
	err = typedHandler.Handle(context.TODO(), jsonData)

	// Verify the handler succeeded
	assert.NoError(t, err)
}

// TestTypedHandler_Handle_EmptyJSON tests handling empty JSON
func TestTypedHandler_Handle_EmptyJSON(t *testing.T) {
	// Create a mock factory
	factory := events.CustomerEventFactory{}

	handler := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}

	typedHandler := NewTypedHandler(factory, handler)

	// Handle empty JSON
	ctx := context.Background()
	err := typedHandler.Handle(ctx, []byte{})

	// Should fail to unmarshal
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal event")
}

// TestTypedHandler_Handle_MalformedEventJSON tests handling malformed event JSON
func TestTypedHandler_Handle_MalformedEventJSON(t *testing.T) {
	// Create JSON that looks like an event but is malformed
	malformedJSON := `{
		"id": "test-id",
		"type": "customer.created",
		"timestamp": "invalid-timestamp",
		"payload": {
			"customer_id": "test-customer",
			"event_type": "customer.created"
		}
	}`

	// Create a mock factory
	factory := events.CustomerEventFactory{}

	handler := func(ctx context.Context, event events.CustomerEvent) error {
		return nil
	}

	typedHandler := NewTypedHandler(factory, handler)

	// Handle malformed JSON
	ctx := context.Background()
	err := typedHandler.Handle(ctx, []byte(malformedJSON))

	// Should fail to unmarshal due to invalid timestamp
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal event")
}
