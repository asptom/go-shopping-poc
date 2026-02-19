package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CartValidationEventType defines validation-specific event types for cart-product decoupling
type CartValidationEventType string

const (
	// CartItemValidationRequested is emitted when a cart item needs product validation
	CartItemValidationRequested CartValidationEventType = "cart.item.validation.requested"
	// CartItemValidationCompleted is emitted when product validation is complete
	CartItemValidationCompleted CartValidationEventType = "cart.item.validation.completed"
)

// CartValidationPayload contains the validation request data
// This is sent from cart service to product service
type CartValidationPayload struct {
	CorrelationID string `json:"correlation_id"` // Links request to response
	CartID        string `json:"cart_id"`
	ProductID     string `json:"product_id"`
	Quantity      int    `json:"quantity"`
}

// CartValidationResultPayload contains the validation result data
// This is sent from product service back to cart service
type CartValidationResultPayload struct {
	CorrelationID string  `json:"correlation_id"`
	IsValid       bool    `json:"is_valid"` // Product exists
	InStock       bool    `json:"in_stock"` // Product has inventory
	ProductName   string  `json:"product_name,omitempty"`
	UnitPrice     float64 `json:"unit_price,omitempty"`
	Reason        string  `json:"reason,omitempty"` // "out_of_stock", "product_not_found", "invalid_product_id"
}

// CartValidationEvent represents cart validation lifecycle events
// Used for both request and result events
type CartValidationEvent struct {
	ID           string                  `json:"id"`
	EventType    CartValidationEventType `json:"type"`
	Timestamp    time.Time               `json:"timestamp"`
	EventPayload interface{}             `json:"payload"` // CartValidationPayload or CartValidationResultPayload
}

// CartValidationEventFactory implements EventFactory for deserialization
type CartValidationEventFactory struct{}

// FromJSON reconstructs a CartValidationEvent from JSON
func (f CartValidationEventFactory) FromJSON(data []byte) (CartValidationEvent, error) {
	// First unmarshal to get the event type
	var rawEvent struct {
		ID        string          `json:"id"`
		EventType string          `json:"type"`
		Timestamp time.Time       `json:"timestamp"`
		Payload   json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal(data, &rawEvent); err != nil {
		return CartValidationEvent{}, err
	}

	event := CartValidationEvent{
		ID:        rawEvent.ID,
		EventType: CartValidationEventType(rawEvent.EventType),
		Timestamp: rawEvent.Timestamp,
	}

	// Unmarshal payload based on event type
	switch event.EventType {
	case CartItemValidationRequested:
		var payload CartValidationPayload
		if err := json.Unmarshal(rawEvent.Payload, &payload); err != nil {
			return CartValidationEvent{}, err
		}
		event.EventPayload = payload
	case CartItemValidationCompleted:
		var payload CartValidationResultPayload
		if err := json.Unmarshal(rawEvent.Payload, &payload); err != nil {
			return CartValidationEvent{}, err
		}
		event.EventPayload = payload
	default:
		// Unknown type - leave as raw message for debugging
		event.EventPayload = rawEvent.Payload
	}

	return event, nil
}

// Event interface implementations
func (e CartValidationEvent) Type() string  { return string(e.EventType) }
func (e CartValidationEvent) Topic() string { return "CartEvents" }
func (e CartValidationEvent) Payload() any  { return e.EventPayload }

// ToJSON serializes the event to JSON
func (e CartValidationEvent) ToJSON() ([]byte, error) { return json.Marshal(e) }

// GetEntityID returns the cart ID for request events, correlation ID for result events
func (e CartValidationEvent) GetEntityID() string {
	if p, ok := e.EventPayload.(CartValidationPayload); ok {
		return p.CartID
	}
	if p, ok := e.EventPayload.(CartValidationResultPayload); ok {
		return "validation_" + p.CorrelationID
	}
	return ""
}

// GetResourceID returns the event ID
func (e CartValidationEvent) GetResourceID() string { return e.ID }

// NewCartItemValidationRequestedEvent creates a validation request event
func NewCartItemValidationRequestedEvent(cartID, productID string, quantity int, correlationID string) *CartValidationEvent {
	return &CartValidationEvent{
		ID:        uuid.New().String(),
		EventType: CartItemValidationRequested,
		Timestamp: time.Now(),
		EventPayload: CartValidationPayload{
			CorrelationID: correlationID,
			CartID:        cartID,
			ProductID:     productID,
			Quantity:      quantity,
		},
	}
}

// NewCartItemValidationCompletedEvent creates a validation result event
func NewCartItemValidationCompletedEvent(correlationID string, isValid, inStock bool, productName string, unitPrice float64, reason string) *CartValidationEvent {
	return &CartValidationEvent{
		ID:        uuid.New().String(),
		EventType: CartItemValidationCompleted,
		Timestamp: time.Now(),
		EventPayload: CartValidationResultPayload{
			CorrelationID: correlationID,
			IsValid:       isValid,
			InStock:       inStock,
			ProductName:   productName,
			UnitPrice:     unitPrice,
			Reason:        reason,
		},
	}
}
