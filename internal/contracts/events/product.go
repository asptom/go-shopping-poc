package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

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

	ProductViewed         ProductEventType = "product.viewed"
	ProductSearchExecuted ProductEventType = "product.search.executed"
	ProductCategoryViewed ProductEventType = "product.category.viewed"

	ProductValidated   ProductEventType = "product.validated"
	ProductUnavailable ProductEventType = "product.unavailable"
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

func NewProductViewedEvent(productID string, details map[string]string) *ProductEvent {
	return NewProductEvent(productID, ProductViewed, productID, details)
}

func NewProductSearchExecutedEvent(query string, details map[string]string) *ProductEvent {
	return NewProductEvent("", ProductSearchExecuted, query, details)
}

func NewProductCategoryViewedEvent(category string, details map[string]string) *ProductEvent {
	return NewProductEvent("", ProductCategoryViewed, category, details)
}

// ProductValidationPayload contains product validation results
type ProductValidationPayload struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name,omitempty"`
	UnitPrice   float64 `json:"unit_price,omitempty"`
	IsAvailable bool    `json:"is_available"`
	Reason      string  `json:"reason,omitempty"`

	// Context information (optional, for correlation)
	CartID     string `json:"cart_id,omitempty"`
	LineNumber string `json:"line_number,omitempty"`
}

// NewProductValidatedEvent creates a product validation success event
func NewProductValidatedEvent(productID, productName string, unitPrice float64, cartID, lineNumber, validationID string) *ProductEvent {
	details := map[string]string{
		"cart_id":       cartID,
		"line_number":   lineNumber,
		"unit_price":    fmt.Sprintf("%.2f", unitPrice),
		"product_name":  productName,
		"validation_id": validationID,
	}

	return &ProductEvent{
		ID:        uuid.New().String(),
		EventType: ProductValidated,
		Timestamp: time.Now(),
		EventPayload: ProductEventPayload{
			ProductID:  productID,
			EventType:  ProductValidated,
			ResourceID: productID,
			Details:    details,
		},
	}
}

// NewProductUnavailableEvent creates a product validation failure event
func NewProductUnavailableEvent(productID, reason string, cartID, lineNumber, validationID string) *ProductEvent {
	details := map[string]string{
		"cart_id":       cartID,
		"line_number":   lineNumber,
		"reason":        reason,
		"validation_id": validationID,
	}

	return &ProductEvent{
		ID:        uuid.New().String(),
		EventType: ProductUnavailable,
		Timestamp: time.Now(),
		EventPayload: ProductEventPayload{
			ProductID:  productID,
			EventType:  ProductUnavailable,
			ResourceID: productID,
			Details:    details,
		},
	}
}
