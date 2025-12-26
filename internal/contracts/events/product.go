// Package events defines the contract interfaces and data structures for events.
// This package contains only pure data structures and interfaces - no business logic.
//
// Key interfaces:
//   - Event: Common event interface (Type, Topic, Payload, ToJSON)
//   - EventFactory[T]: Type-safe event reconstruction from JSON
//
// Event types:
//   - CustomerEvent: Customer domain events
//   - OrderEvent: Order domain events (future)
//   - ProductEvent: Product domain events
//
// Usage example:
//
//	// Create a product event
//	event := events.NewProductCreated(productID, productData)
//
//	// Serialize to JSON
//	jsonData, err := event.ToJSON()
//
//	// Deserialize with type safety
//	factory := events.ProductEventFactory{}
//	restoredEvent, err := factory.FromJSON(jsonData)
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ProductEventType is a typed alias for well-known product events
type ProductEventType string

const (
	ProductCreated ProductEventType = "product.created"
	ProductUpdated ProductEventType = "product.updated"
	ProductDeleted ProductEventType = "product.deleted"

	ProductImageAdded   ProductEventType = "product.image.added"
	ProductImageUpdated ProductEventType = "product.image.updated"
	ProductImageDeleted ProductEventType = "product.image.deleted"

	ProductIngestionStarted   ProductEventType = "product.ingestion.started"
	ProductIngestionCompleted ProductEventType = "product.ingestion.completed"
	ProductIngestionFailed    ProductEventType = "product.ingestion.failed"
)

// ProductEventPayload represents the data structure for product events
type ProductEventPayload struct {
	ProductID  string            `json:"product_id,omitempty"`
	EventType  ProductEventType  `json:"event_type"`
	ResourceID string            `json:"resource_id,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
}

// ProductEvent represents a product-related event
type ProductEvent struct {
	ID           string              `json:"id"`
	EventType    ProductEventType    `json:"type"`
	Timestamp    time.Time           `json:"timestamp"`
	EventPayload ProductEventPayload `json:"payload"`
}

// ProductEventFactory implements EventFactory for ProductEvent
type ProductEventFactory struct{}

func (f ProductEventFactory) FromJSON(data []byte) (ProductEvent, error) {
	var event ProductEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

// Convenience constructor for ProductEvent
func NewProductEvent(productID string, t ProductEventType, resourceID string, details map[string]string) *ProductEvent {
	payload := ProductEventPayload{
		ProductID:  productID,
		EventType:  t,
		ResourceID: resourceID,
		Details:    details,
	}

	return &ProductEvent{
		ID:           uuid.New().String(),
		EventType:    t,
		Timestamp:    time.Now(),
		EventPayload: payload,
	}
}

// Implement Event Interface
func (e ProductEvent) Type() string {
	return string(e.EventType)
}

func (e ProductEvent) Topic() string {
	return "ProductEvents"
}

func (e ProductEvent) Payload() any {
	return e.EventPayload
}

func (e ProductEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

func (e ProductEvent) GetEntityID() string {
	return e.EventPayload.ProductID
}

func (e ProductEvent) GetResourceID() string {
	return e.EventPayload.ResourceID
}

// Convenience constructors

func NewProductCreatedEvent(productID string, details map[string]string) *ProductEvent {
	return NewProductEvent(productID, ProductCreated, productID, details)
}

func NewProductUpdatedEvent(productID string, details map[string]string) *ProductEvent {
	return NewProductEvent(productID, ProductUpdated, productID, details)
}

func NewProductDeletedEvent(productID string, details map[string]string) *ProductEvent {
	return NewProductEvent(productID, ProductDeleted, productID, details)
}

func NewProductImageAddedEvent(productID, imageID string, details map[string]string) *ProductEvent {
	return NewProductEvent(productID, ProductImageAdded, imageID, details)
}

func NewProductImageUpdatedEvent(productID, imageID string, details map[string]string) *ProductEvent {
	return NewProductEvent(productID, ProductImageUpdated, imageID, details)
}

func NewProductImageDeletedEvent(productID, imageID string, details map[string]string) *ProductEvent {
	return NewProductEvent(productID, ProductImageDeleted, imageID, details)
}

func NewProductIngestionStartedEvent(batchID string, details map[string]string) *ProductEvent {
	return NewProductEvent("", ProductIngestionStarted, batchID, details)
}

func NewProductIngestionCompletedEvent(batchID string, details map[string]string) *ProductEvent {
	return NewProductEvent("", ProductIngestionCompleted, batchID, details)
}

func NewProductIngestionFailedEvent(batchID string, details map[string]string) *ProductEvent {
	return NewProductEvent("", ProductIngestionFailed, batchID, details)
}
