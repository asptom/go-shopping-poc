# Cart-Product Decoupling Plan v2: Optimistic Add with Event-Driven Validation

## Executive Summary

This plan refines the cart-product decoupling approach to strictly adhere to the existing codebase architecture. The key improvements focus on:
- Using the **outbox pattern** for transactional event publishing (not direct event bus)
- Properly integrating with the **product service's existing BaseService** architecture
- Following established **event structure patterns** with factories
- Handling **edge cases** like duplicate items and service unavailability

---

## Current Architecture Context

### Event Publishing Pattern (Critical)
The codebase uses the **outbox pattern** for ALL event publishing:
- Events are written to the outbox table within the same database transaction as business data changes
- A separate outbox publisher polls and publishes to Kafka
- This ensures transactional consistency - no events are lost if the service crashes

**DO NOT publish directly to event bus in business logic** - always use outbox pattern.

### Service Base Types
- **CartService**: Uses `EventServiceBase` - can consume events and has event bus
- **CatalogService**: Uses `BaseService` - NO event consumption capability currently
- Event consumption requires either upgrading to `EventServiceBase` or creating a separate event reader service

### Existing Event Topics
- `CartEvents` - Cart lifecycle events (created, deleted, checked_out)
- `OrderEvents` - Order events (created, updated, deleted)
- `ProductEvents` - Product events (lifecycle, analytics)

### CartItem Entity Structure
Currently has: `ID`, `CartID`, `LineNumber`, `ProductID`, `ProductName`, `UnitPrice`, `Quantity`, `TotalPrice`

---

## Proposed Solution: Optimistic Add with Validation

### Event Flow

```
┌─────────────┐     Add Item (Async)     ┌─────────────┐
│   Frontend  │ ───────────────────────> │ Cart Service│
└─────────────┘                          └──────┬──────┘
     ▲                                          │
     │                                          │ 1. Add item with status "pending"
     │                                          │ 2. Write validation event to outbox
     │                                          │ 3. Return immediately to frontend
     │                                          │
     │    SSE: cart.item.validated              │ 4. Outbox publisher sends event
     │ ─────────────────────────────────────────┘
     │                                          
     │                                     ┌─────────────┐
     │                                     │   Kafka     │
     │                                     │  CartEvents │
     │                                     └──────┬──────┘
     │                                            │
     │                                     ┌──────┴──────┐
     │                                     │  Validation │
     │                                     │   Service   │
     │                                     └──────┬──────┘
     │                                            │
     │                                     ┌──────┴──────┐
     │                                     │   Kafka     │
     │                                     │ProductEvents│
     │                                     └──────┬──────┘
     │                                            │
     │    5. Process validation result            │
     └────────────────────────────────────────────┘
```

**Key Difference from v1:** Cart service publishes via outbox (not direct event bus), and product service needs event consumption capability.

---

## Implementation Plan

### Phase 0: Prerequisites

**Status:** Create new migration for CartItem schema updates

**New Columns:**
```sql
ALTER TABLE carts.CartItem ADD COLUMN status VARCHAR(20) DEFAULT 'confirmed';
ALTER TABLE carts.CartItem ADD COLUMN validation_id UUID;
ALTER TABLE carts.CartItem ADD COLUMN backorder_reason VARCHAR(100);

-- Index for validation ID lookups
CREATE INDEX idx_cart_items_validation_id ON carts.CartItem(validation_id);

-- Index for status filtering
CREATE INDEX idx_cart_items_status ON carts.CartItem(status);
```

**Status Values:**
- `confirmed` - Item validated and available (DEFAULT for existing items)
- `pending_validation` - Item added, waiting for validation
- `backorder` - Item validated but out of stock or not found

### Phase 1: Define Validation Events

**New file: `internal/contracts/events/cart_validation.go`**

```go
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CartValidationEventType defines validation-specific event types
type CartValidationEventType string

const (
	CartItemValidationRequested  CartValidationEventType = "cart.item.validation.requested"
	CartItemValidationCompleted  CartValidationEventType = "cart.item.validation.completed"
)

// CartValidationPayload contains validation request data
type CartValidationPayload struct {
	CorrelationID string `json:"correlation_id"`  // Links request to response
	CartID        string `json:"cart_id"`
	ProductID     string `json:"product_id"`
	Quantity      int    `json:"quantity"`
}

// CartValidationResultPayload contains validation result data
type CartValidationResultPayload struct {
	CorrelationID string  `json:"correlation_id"`
	IsValid       bool    `json:"is_valid"`
	InStock       bool    `json:"in_stock"`
	ProductName   string  `json:"product_name,omitempty"`
	UnitPrice     float64 `json:"unit_price,omitempty"`
	Reason        string  `json:"reason,omitempty"` // "out_of_stock", "product_not_found"
}

// CartValidationEvent represents cart validation events
type CartValidationEvent struct {
	ID           string                    `json:"id"`
	EventType    CartValidationEventType   `json:"type"`
	Timestamp    time.Time                 `json:"timestamp"`
	EventPayload interface{}               `json:"payload"` // CartValidationPayload or CartValidationResultPayload
}

// CartValidationEventFactory implements EventFactory
type CartValidationEventFactory struct{}

func (f CartValidationEventFactory) FromJSON(data []byte) (CartValidationEvent, error) {
	var event CartValidationEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

// Event interface implementations
func (e CartValidationEvent) Type() string            { return string(e.EventType) }
func (e CartValidationEvent) Topic() string           { return "CartEvents" }
func (e CartValidationEvent) Payload() any            { return e.EventPayload }
func (e CartValidationEvent) ToJSON() ([]byte, error) { return json.Marshal(e) }
func (e CartValidationEvent) GetEntityID() string     { 
	if p, ok := e.EventPayload.(CartValidationPayload); ok {
		return p.CartID
	}
	if p, ok := e.EventPayload.(CartValidationResultPayload); ok {
		return "validation_" + p.CorrelationID
	}
	return ""
}
func (e CartValidationEvent) GetResourceID() string   { return e.ID }

// Constructor for validation request
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

// Constructor for validation result
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
```

### Phase 2: Update CartItem Entity

**File: `internal/service/cart/entity.go`**

Add to CartItem struct:
```go
type CartItem struct {
	ID              int64     `json:"id" db:"id"`
	CartID          uuid.UUID `json:"cart_id" db:"cart_id"`
	LineNumber      string    `json:"line_number" db:"line_number"`
	ProductID       string    `json:"product_id" db:"product_id"`
	ProductName     string    `json:"product_name" db:"product_name"`
	UnitPrice       float64   `json:"unit_price" db:"unit_price"`
	Quantity        int       `json:"quantity" db:"quantity"`
	TotalPrice      float64   `json:"total_price" db:"total_price"`
	
	// New fields for validation
	Status          string    `json:"status" db:"status"`                           // "confirmed", "pending_validation", "backorder"
	ValidationID    *string   `json:"validation_id,omitempty" db:"validation_id"`   // correlation ID
	BackorderReason string    `json:"backorder_reason,omitempty" db:"backorder_reason"`
}
```

Add domain methods:
```go
// IsPendingValidation returns true if item is waiting for validation
func (ci *CartItem) IsPendingValidation() bool {
	return ci.Status == "pending_validation"
}

// IsBackorder returns true if item is on backorder
func (ci *CartItem) IsBackorder() bool {
	return ci.Status == "backorder"
}

// IsConfirmed returns true if item is confirmed
func (ci *CartItem) IsConfirmed() bool {
	return ci.Status == "confirmed"
}

// ConfirmItem updates item with validated product details
func (ci *CartItem) ConfirmItem(productName string, unitPrice float64) error {
	if ci.Status != "pending_validation" {
		return fmt.Errorf("cannot confirm item with status %s", ci.Status)
	}
	ci.Status = "confirmed"
	ci.ProductName = productName
	ci.UnitPrice = unitPrice
	ci.CalculateLineTotal()
	ci.BackorderReason = ""
	return nil
}

// MarkAsBackorder marks item as backorder with reason
func (ci *CartItem) MarkAsBackorder(reason string) error {
	if ci.Status != "pending_validation" {
		return fmt.Errorf("cannot mark as backorder: item has status %s", ci.Status)
	}
	ci.Status = "backorder"
	ci.BackorderReason = reason
	return nil
}
```

### Phase 3: Repository Updates

**File: `internal/service/cart/repository.go`**

Add interface methods:
```go
type CartRepository interface {
	// ... existing methods ...
	
	// Item lookup by validation ID
	GetItemByValidationID(ctx context.Context, validationID string) (*CartItem, error)
	
	// Get item by cart ID and product ID (for duplicate detection)
	GetItemByProductID(ctx context.Context, cartID, productID string) (*CartItem, error)
	
	// Update item status and details
	UpdateItemStatus(ctx context.Context, item *CartItem) error
}
```

**File: `internal/service/cart/repository_items.go`**

Add implementations:
```go
func (r *cartRepository) GetItemByValidationID(ctx context.Context, validationID string) (*CartItem, error) {
	var item CartItem
	err := r.db.GetContext(ctx, &item, `
		SELECT id, cart_id, line_number, product_id, product_name, unit_price, quantity, total_price, status, validation_id, backorder_reason
		FROM carts.CartItem
		WHERE validation_id = $1
	`, validationID)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCartItemNotFound
		}
		return nil, fmt.Errorf("%w: failed to get item by validation ID: %v", ErrDatabaseOperation, err)
	}
	
	return &item, nil
}

func (r *cartRepository) GetItemByProductID(ctx context.Context, cartID, productID string) (*CartItem, error) {
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
	}
	
	var item CartItem
	err = r.db.GetContext(ctx, &item, `
		SELECT id, cart_id, line_number, product_id, product_name, unit_price, quantity, total_price, status, validation_id, backorder_reason
		FROM carts.CartItem
		WHERE cart_id = $1 AND product_id = $2
	`, cartUUID, productID)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCartItemNotFound
		}
		return nil, fmt.Errorf("%w: failed to get item by product ID: %v", ErrDatabaseOperation, err)
	}
	
	return &item, nil
}

func (r *cartRepository) UpdateItemStatus(ctx context.Context, item *CartItem) error {
	_, err := r.db.Exec(ctx, `
		UPDATE carts.CartItem
		SET product_name = $1, unit_price = $2, total_price = $3, status = $4, backorder_reason = $5
		WHERE id = $6
	`, item.ProductName, item.UnitPrice, item.TotalPrice, item.Status, item.BackorderReason, item.ID)
	
	if err != nil {
		return fmt.Errorf("%w: failed to update item status: %v", ErrDatabaseOperation, err)
	}
	
	return nil
}
```

### Phase 4: Modify AddItem Service Method

**File: `internal/service/cart/service.go` - `AddItem`**

```go
func (s *CartService) AddItem(ctx context.Context, cartID string, productID string, quantity int) (*CartItem, error) {
	log.Printf("[DEBUG] CartService: Adding item to cart %s: product_id=%s, quantity=%d", cartID, productID, quantity)

	if quantity <= 0 {
		return nil, errors.New("quantity must be positive")
	}

	cart, err := s.repo.GetCartByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	if cart.CurrentStatus != "active" {
		return nil, errors.New("cannot add items to non-active cart")
	}

	// Check if product already exists in cart (prevent duplicates during validation)
	existingItem, err := s.repo.GetItemByProductID(ctx, cartID, productID)
	if err == nil && existingItem != nil {
		if existingItem.IsPendingValidation() {
			return nil, errors.New("product is already being added to cart, please wait for validation")
		}
		if existingItem.IsConfirmed() {
			return nil, errors.New("product already exists in cart, use update quantity instead")
		}
		// If backorder, allow adding again (will create new validation attempt)
	}

	// Generate correlation ID for this validation
	correlationID := uuid.New().String()

	// Create item with pending status
	item := &CartItem{
		ProductID:    productID,
		Quantity:     quantity,
		Status:       "pending_validation",
		ValidationID: &correlationID,
		// ProductName and UnitPrice will be updated after validation
	}

	// Begin transaction to add item and write validation event
	tx, err := s.infrastructure.Database.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Add item to cart (this will get line number from sequence)
	// Note: Need to modify AddItem to accept transaction or create AddItemTx
	if err := s.repo.AddItemTx(ctx, tx, cartID, item); err != nil {
		return nil, fmt.Errorf("failed to add item: %w", err)
	}

	// Write validation event to outbox (transactional)
	validationEvent := events.NewCartItemValidationRequestedEvent(cartID, productID, quantity, correlationID)
	if err := s.infrastructure.OutboxWriter.WriteEvent(ctx, tx, validationEvent); err != nil {
		return nil, fmt.Errorf("failed to write validation event: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	// Update cart totals with placeholder (will be recalculated after validation)
	cart.Items = append(cart.Items, *item)
	cart.CalculateTotals()
	
	log.Printf("[DEBUG] CartService: Updating cart totals for cart %s after adding pending item", cartID)
	if err := s.repo.UpdateCart(ctx, cart); err != nil {
		log.Printf("[WARN] CartService: failed to update cart totals for cart %s: %v", cartID, err)
		// Don't fail the request, cart totals will be updated on validation
	}

	log.Printf("[INFO] CartService: Added pending item to cart %s with validation ID %s", cartID, correlationID)
	return item, nil
}
```

**Note:** Requires adding `AddItemTx` method to repository that accepts a transaction.

### Phase 5: Product Service Event Consumption

The product service currently uses `BaseService` and has no event consumption. Two options:

#### Option A: Add Event Consumption to CatalogService (Recommended)

**File: `internal/service/product/config.go`**

Add event bus configuration:
```go
type Config struct {
	// ... existing fields ...
	
	// Event bus configuration for consuming CartEvents
	ReadTopics []string `mapstructure:"product_read_topics"`  // Should include "CartEvents"
	WriteTopic string   `mapstructure:"product_write_topic"` // Should be "ProductEvents"
	Group      string   `mapstructure:"product_group"`       // Consumer group ID
}
```

**File: `internal/service/product/service.go`**

Change from BaseService to EventServiceBase:
```go
type CatalogInfrastructure struct {
	Database     database.Database
	OutboxWriter *outbox.Writer
	EventBus     bus.Bus  // NEW: Event bus for consuming validation requests
}

type CatalogService struct {
	*service.EventServiceBase  // CHANGED: Now can consume events
	repo           ProductRepository
	infrastructure *CatalogInfrastructure
	config         *Config
}

func NewCatalogService(infrastructure *CatalogInfrastructure, config *Config) *CatalogService {
	repo := NewProductRepository(infrastructure.Database, infrastructure.OutboxWriter)

	return &CatalogService{
		EventServiceBase: service.NewEventServiceBase("product", infrastructure.EventBus),
		repo:             repo,
		infrastructure:   infrastructure,
		config:           config,
	}
}

// RegisterHandler adds a new event handler
func RegisterHandler[T events.Event](s Service, factory events.EventFactory[T], handler bus.HandlerFunc[T]) error {
	return service.RegisterHandler(s, factory, handler)
}

// Service interface for handler registration
type Service interface {
	service.Service
}
```

**File: `cmd/product/main.go`**

Add event bus setup:
```go
func main() {
	// ... existing setup code ...

	// Event bus setup (NEW)
	log.Printf("[DEBUG] Product: Creating event bus provider")
	eventBusConfig := event.EventBusConfig{
		WriteTopic: cfg.WriteTopic,
		GroupID:    cfg.Group,
	}
	eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
	if err != nil {
		log.Fatalf("Product: Failed to create event bus provider: %v", err)
	}
	eventBus := eventBusProvider.GetEventBus()

	// Create infrastructure with event bus
	catalogInfra := &product.CatalogInfrastructure{
		Database:     platformDB,
		OutboxWriter: writerProvider.GetWriter(),
		EventBus:     eventBus,  // NEW
	}
	catalogService := product.NewCatalogService(catalogInfra, cfg)

	// Register event handlers
	if err := registerEventHandlers(catalogService); err != nil {
		log.Fatalf("Product: Failed to register event handlers: %v", err)
	}

	// Start event consumer in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		if err := catalogService.Start(ctx); err != nil {
			log.Printf("[ERROR] Product: Event consumer stopped: %v", err)
		}
	}()

	// ... rest of HTTP server setup ...
}

func registerEventHandlers(service *product.CatalogService) error {
	log.Printf("[INFO] Product: Registering event handlers...")
	
	// Register CartItemValidationRequested handler
	validationHandler := eventhandlers.NewOnCartItemValidationRequested(service)
	
	if err := product.RegisterHandler(
		service,
		validationHandler.CreateFactory(),
		validationHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register CartItemValidationRequested handler: %w", err)
	}
	
	log.Printf("[INFO] Product: Successfully registered event handlers")
	return nil
}
```

#### Option B: Create Separate Validation Service

If adding event consumption to CatalogService is not desirable, create a separate lightweight service specifically for validation:

**New file: `cmd/validation/main.go`**

This service would only consume CartEvents and publish ProductEvents, keeping the catalog service pure.

### Phase 6: Add Product Validation Event Handler

**New file: `internal/service/product/eventhandlers/on_cart_item_validation_requested.go`**

```go
package eventhandlers

import (
	"context"
	"fmt"
	"log"
	"strconv"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/service/product"
)

type OnCartItemValidationRequested struct {
	service        *product.CatalogService
	outboxWriter   *outbox.Writer
	database       database.Database
}

func NewOnCartItemValidationRequested(service *product.CatalogService) *OnCartItemValidationRequested {
	return &OnCartItemValidationRequested{
		service:      service,
		outboxWriter: service.GetInfrastructure().OutboxWriter,
		database:     service.GetInfrastructure().Database,
	}
}

func (h *OnCartItemValidationRequested) Handle(ctx context.Context, event events.Event) error {
	validationEvent, ok := event.(events.CartValidationEvent)
	if !ok {
		log.Printf("[ERROR] Product: Expected CartValidationEvent, got %T", event)
		return nil
	}

	if validationEvent.EventType != events.CartItemValidationRequested {
		log.Printf("[DEBUG] Product: Ignoring event type: %s", validationEvent.EventType)
		return nil
	}

	payload, ok := validationEvent.EventPayload.(events.CartValidationPayload)
	if !ok {
		log.Printf("[ERROR] Product: Invalid payload type for validation request")
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(validationEvent.EventType),
		payload.ProductID,
		payload.CartID)

	log.Printf("[DEBUG] Product: Validating product %s for cart %s", payload.ProductID, payload.CartID)

	// Parse product ID
	productID, err := strconv.ParseInt(payload.ProductID, 10, 64)
	if err != nil {
		log.Printf("[ERROR] Product: Invalid product ID format: %s", payload.ProductID)
		return h.publishValidationResult(ctx, payload.CorrelationID, false, false, "", 0, "invalid_product_id")
	}

	// Validate product exists and is in stock
	product, err := h.service.GetProductByID(ctx, productID)
	
	result := events.CartValidationResultPayload{
		CorrelationID: payload.CorrelationID,
	}

	if err != nil {
		result.IsValid = false
		result.Reason = "product_not_found"
		log.Printf("[DEBUG] Product: Product %s not found: %v", payload.ProductID, err)
	} else if !product.InStock {
		result.IsValid = false
		result.Reason = "out_of_stock"
		result.InStock = false
		result.ProductName = product.Name
		log.Printf("[DEBUG] Product: Product %s is out of stock", payload.ProductID)
	} else {
		result.IsValid = true
		result.InStock = true
		result.ProductName = product.Name
		result.UnitPrice = product.FinalPrice
		log.Printf("[DEBUG] Product: Product %s validated successfully", payload.ProductID)
	}

	return h.publishValidationResult(ctx, result.CorrelationID, result.IsValid, result.InStock, 
		result.ProductName, result.UnitPrice, result.Reason)
}

func (h *OnCartItemValidationRequested) publishValidationResult(ctx context.Context, correlationID string, 
	isValid, inStock bool, productName string, unitPrice float64, reason string) error {
	
	// Write result to outbox within a transaction
	tx, err := h.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	resultEvent := events.NewCartItemValidationCompletedEvent(correlationID, isValid, inStock, productName, unitPrice, reason)
	
	if err := h.outboxWriter.WriteEvent(ctx, tx, resultEvent); err != nil {
		return fmt.Errorf("failed to write validation result to outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit validation result: %w", err)
	}
	committed = true

	return nil
}

func (h *OnCartItemValidationRequested) EventType() string {
	return string(events.CartItemValidationRequested)
}

func (h *OnCartItemValidationRequested) CreateHandler() bus.HandlerFunc[events.CartValidationEvent] {
	return func(ctx context.Context, event events.CartValidationEvent) error {
		return h.Handle(ctx, event)
	}
}

func (h *OnCartItemValidationRequested) CreateFactory() events.EventFactory[events.CartValidationEvent] {
	return events.CartValidationEventFactory{}
}

// Ensure implementation
var _ handler.EventHandler = (*OnCartItemValidationRequested)(nil)
var _ handler.HandlerFactory[events.CartValidationEvent] = (*OnCartItemValidationRequested)(nil)
```

### Phase 7: Add Cart Service Response Handler

**New file: `internal/service/cart/eventhandlers/on_cart_item_validation_completed.go`**

```go
package eventhandlers

import (
	"context"
	"log"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/platform/sse"
	"go-shopping-poc/internal/service/cart"
)

type OnCartItemValidationCompleted struct {
	repo   cart.CartRepository
	sseHub *sse.Hub
}

func NewOnCartItemValidationCompleted(repo cart.CartRepository, sseHub *sse.Hub) *OnCartItemValidationCompleted {
	return &OnCartItemValidationCompleted{
		repo:   repo,
		sseHub: sseHub,
	}
}

func (h *OnCartItemValidationCompleted) Handle(ctx context.Context, event events.Event) error {
	validationEvent, ok := event.(events.CartValidationEvent)
	if !ok {
		log.Printf("[ERROR] Cart: Expected CartValidationEvent, got %T", event)
		return nil
	}

	if validationEvent.EventType != events.CartItemValidationCompleted {
		log.Printf("[DEBUG] Cart: Ignoring event type: %s", validationEvent.EventType)
		return nil
	}

	payload, ok := validationEvent.EventPayload.(events.CartValidationResultPayload)
	if !ok {
		log.Printf("[ERROR] Cart: Invalid payload type for validation result")
		return nil
	}

	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(validationEvent.EventType),
		payload.CorrelationID,
		"")

	// Find cart item by validation ID
	item, err := h.repo.GetItemByValidationID(ctx, payload.CorrelationID)
	if err != nil {
		log.Printf("[DEBUG] Cart: Item not found for validation ID %s - may have been removed by user", payload.CorrelationID)
		return nil // Item not found - may have been removed by user before validation completed
	}

	// Get cart for totals recalculation
	cartObj, err := h.repo.GetCartByID(ctx, item.CartID.String())
	if err != nil {
		log.Printf("[ERROR] Cart: Failed to get cart %s for validation update: %v", item.CartID, err)
		return err
	}

	// Update item based on validation result
	if payload.IsValid && payload.InStock {
		// Confirm item with product details
		if err := item.ConfirmItem(payload.ProductName, payload.UnitPrice); err != nil {
			log.Printf("[ERROR] Cart: Failed to confirm item %s: %v", item.LineNumber, err)
			return err
		}
		log.Printf("[INFO] Cart: Item %s confirmed for cart %s", item.LineNumber, item.CartID)
	} else {
		// Mark as backorder
		reason := payload.Reason
		if reason == "" {
			reason = "validation_failed"
		}
		if err := item.MarkAsBackorder(reason); err != nil {
			log.Printf("[ERROR] Cart: Failed to mark item %s as backorder: %v", item.LineNumber, err)
			return err
		}
		log.Printf("[INFO] Cart: Item %s marked as backorder for cart %s: %s", item.LineNumber, item.CartID, reason)
	}

	// Update item in database
	if err := h.repo.UpdateItemStatus(ctx, item); err != nil {
		return err
	}

	// Recalculate cart totals
	for i := range cartObj.Items {
		if cartObj.Items[i].LineNumber == item.LineNumber {
			cartObj.Items[i] = *item
			break
		}
	}
	cartObj.CalculateTotals()
	
	if err := h.repo.UpdateCart(ctx, cartObj); err != nil {
		return err
	}

	// Push SSE notification to frontend
	if h.sseHub != nil {
		eventType := "cart.item.validated"
		if item.IsBackorder() {
			eventType = "cart.item.backorder"
		}
		
		h.sseHub.Publish(
			item.CartID.String(),
			eventType,
			map[string]interface{}{
				"lineNumber":      item.LineNumber,
				"productId":       item.ProductID,
				"status":          item.Status,
				"productName":     item.ProductName,
				"unitPrice":       item.UnitPrice,
				"quantity":        item.Quantity,
				"totalPrice":      item.TotalPrice,
				"backorderReason": item.BackorderReason,
			},
		)
	}

	return nil
}

func (h *OnCartItemValidationCompleted) EventType() string {
	return string(events.CartItemValidationCompleted)
}

func (h *OnCartItemValidationCompleted) CreateHandler() bus.HandlerFunc[events.CartValidationEvent] {
	return func(ctx context.Context, event events.CartValidationEvent) error {
		return h.Handle(ctx, event)
	}
}

func (h *OnCartItemValidationCompleted) CreateFactory() events.EventFactory[events.CartValidationEvent] {
	return events.CartValidationEventFactory{}
}

// Ensure implementation
var _ handler.EventHandler = (*OnCartItemValidationCompleted)(nil)
var _ handler.HandlerFactory[events.CartValidationEvent] = (*OnCartItemValidationCompleted)(nil)
```

### Phase 8: Register New Event Handler in Cart Service

**File: `cmd/cart/main.go`** - Update `registerEventHandlers`:

```go
func registerEventHandlers(service *cart.CartService, sseHub *sse.Hub) error {
	log.Printf("[INFO] Cart: Registering event handlers...")

	// Existing: OrderCreated handler
	orderCreatedHandler := eventhandlers.NewOnOrderCreated(sseHub)
	if err := cart.RegisterHandler(
		service,
		orderCreatedHandler.CreateFactory(),
		orderCreatedHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register OrderCreated handler: %w", err)
	}
	log.Printf("[INFO] Cart: Registered OrderCreated handler")

	// NEW: CartItemValidationCompleted handler
	validationHandler := eventhandlers.NewOnCartItemValidationCompleted(service.GetRepository(), sseHub)
	if err := cart.RegisterHandler(
		service,
		validationHandler.CreateFactory(),
		validationHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register CartItemValidationCompleted handler: %w", err)
	}
	log.Printf("[INFO] Cart: Registered CartItemValidationCompleted handler")

	log.Printf("[INFO] Cart: Successfully registered all event handlers")
	return nil
}
```

**File: `internal/service/cart/service.go`** - Add repository accessor:

```go
// GetRepository returns the cart repository for event handlers
func (s *CartService) GetRepository() CartRepository {
	return s.repo
}

// GetInfrastructure returns the infrastructure for event handlers
func (s *CartService) GetInfrastructure() *CartInfrastructure {
	return s.infrastructure
}
```

### Phase 9: Handle Checkout with Backorders

**File: `internal/service/cart/service.go` - `Checkout`**

```go
func (s *CartService) Checkout(ctx context.Context, cartID string) (*Cart, error) {
	cart, err := s.repo.GetCartByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	// Check for pending validation items
	for _, item := range cart.Items {
		if item.IsPendingValidation() {
			return nil, errors.New("cannot checkout: some items are still being validated, please wait")
		}
	}

	// Allow checkout with backorder items - frontend handles display
	// Calculate totals (backorder items are included in totals)
	cart.CalculateTotals()

	if err := s.repo.UpdateCart(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to update cart totals: %w", err)
	}

	checkedOutCart, err := s.repo.CheckoutCart(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("checkout failed: %w", err)
	}

	return checkedOutCart, nil
}
```

### Phase 9: Update Cart Snapshot to Include Status

**File: `internal/service/cart/repository_checkout.go`** - Update `CreateCartSnapshot`:

```go
func CreateCartSnapshot(cart *Cart) *events.CartSnapshot {
	snapshot := &events.CartSnapshot{
		Currency:   cart.Currency,
		NetPrice:   cart.NetPrice,
		Tax:        cart.Tax,
		Shipping:   cart.Shipping,
		TotalPrice: cart.TotalPrice,
	}

	// Include all items (including backorder) in snapshot
	for _, item := range cart.Items {
		snapshot.Items = append(snapshot.Items, events.SnapshotItem{
			LineNumber:  item.LineNumber,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    item.Quantity,
			TotalPrice:  item.TotalPrice,
			// Note: SnapshotItem may need status field added
		})
	}

	// ... rest of snapshot creation ...
}
```



---

## Configuration Changes

### Cart Service Config

```yaml
# Existing
CART_WRITE_TOPIC=cart
CART_READ_TOPICS=order,product
CART_GROUP=cart-consumer-group

# No changes needed - cart already consumes from product topic
```

### Product Service Config

```yaml
# NEW - Add to product service configuration
PRODUCT_WRITE_TOPIC=product
PRODUCT_READ_TOPICS=cart
PRODUCT_GROUP=product-consumer-group
```

### Environment Variables

```bash
# Cart Service
export CART_WRITE_TOPIC=cart
export CART_READ_TOPICS=order,product
export CART_GROUP=cart-consumer-group

# Product Service
export PRODUCT_WRITE_TOPIC=product
export PRODUCT_READ_TOPICS=cart
export PRODUCT_GROUP=product-consumer-group
```

---

## Frontend Integration Guide

### Updated Cart Display

Items should display differently based on status:

```
Item: Widget A          [✓ Available]     $10.00 x 2 = $20.00
Item: Widget B          [⟳ Validating...] $15.00 x 1 = $15.00 (pending)
Item: Widget C          [! Backorder]     $5.00 x 3 = $15.00 (Out of stock - ships when available)
```

### SSE Event Handling

```javascript
const eventSource = new EventSource(`/api/v1/carts/${cartId}/stream`);

// Handle item validation complete
eventSource.addEventListener('cart.item.validated', (e) => {
  const data = JSON.parse(e.data);
  updateCartItem(data.lineNumber, {
    status: data.status,
    productName: data.productName,
    unitPrice: data.unitPrice,
    totalPrice: data.totalPrice
  });
});

// Handle backorder
eventSource.addEventListener('cart.item.backorder', (e) => {
  const data = JSON.parse(e.data);
  updateCartItem(data.lineNumber, {
    status: 'backorder',
    backorderReason: data.backorderReason,
    // Price may still be set for backorder items
    unitPrice: data.unitPrice
  });
});
```

### Checkout Considerations

1. **Disable checkout button** if any items are `pending_validation`
2. **Show warning** if cart contains `backorder` items
3. **Allow user to remove** backorder items before checkout
4. **Display ETA** for backorder items if available

---

## Edge Cases & Error Handling

### 1. Duplicate Product Addition
If user adds same product twice before validation completes:
- Second request is rejected with "product is already being added"
- User must wait for first validation to complete
- After validation, user can update quantity

### 2. Item Removed Before Validation
If user removes item while validation is pending:
- Validation result arrives, but item not found in database
- Handler logs and ignores (idempotent)
- No error returned

### 3. Product Service Down
If product service is unavailable:
- Cart item remains in `pending_validation` state
- Outbox publisher retries sending validation request
- When product service recovers, processes backlog
- No data loss due to outbox pattern

### 4. Validation Timeout
No explicit timeout mechanism:
- Item stays pending until product service responds
- Frontend shows "Validating..." indefinitely
- User can remove item if tired of waiting
- Consider adding max pending time in future

### 5. Product Price Changes During Validation
- Price is captured at validation time
- If price changes after validation, already-confirmed items keep original price
- New additions get new price
- This is consistent with optimistic add pattern

---

## Testing Strategy

### Unit Tests

**Event Factory Tests:**
```go
func TestCartValidationEventFactory(t *testing.T) {
    factory := events.CartValidationEventFactory{}
    event := events.NewCartItemValidationRequestedEvent("cart-1", "prod-1", 2, "corr-1")
    
    jsonData, _ := event.ToJSON()
    restored, err := factory.FromJSON(jsonData)
    
    assert.NoError(t, err)
    assert.Equal(t, event.EventType, restored.EventType)
}
```

**Entity State Transition Tests:**
```go
func TestCartItem_ConfirmItem(t *testing.T) {
    item := &CartItem{Status: "pending_validation"}
    err := item.ConfirmItem("Widget", 10.00)
    
    assert.NoError(t, err)
    assert.Equal(t, "confirmed", item.Status)
    assert.Equal(t, "Widget", item.ProductName)
}
```

### Integration Tests

**Happy Path:**
1. Add item to cart
2. Verify item is `pending_validation`
3. Verify validation event in outbox
4. Simulate product service response
5. Verify item updated to `confirmed`
6. Verify SSE event sent

**Backorder Path:**
1. Add out-of-stock item
2. Simulate product service "out_of_stock" response
3. Verify item marked `backorder`
4. Verify reason stored

**Duplicate Prevention:**
1. Add item (pending)
2. Try to add same item again
3. Verify error returned

**Item Removal During Validation:**
1. Add item
2. Remove item before validation complete
3. Simulate validation response
4. Verify no error, item not found gracefully handled

---

## Performance Optimization

### Target Latency: 100-500ms End-to-End

To achieve sub-second validation notification to the frontend while maintaining reliability:

### 1. Outbox Processing Interval Configuration

**Default:** `OUTBOX_PROCESS_INTERVAL=5s` (too slow for cart validation)

**Recommended for Cart Service:**
```yaml
# Cart service-specific outbox configuration
OUTBOX_PROCESS_INTERVAL=200ms  # Poll outbox every 200ms
OUTBOX_BATCH_SIZE=10           # Process up to 10 events per poll
```

**Configuration Location:**
```go
// In cart service config
outboxConfig := outbox.Config{
    ProcessInterval: 200 * time.Millisecond,  // Fast polling for cart
    BatchSize:       10,
}
```

**Impact Analysis:**
- **DB Load:** ~5 queries/second (minimal with proper indexing)
- **Latency:** Outbox poll delay reduced from 5s to 200ms average (100ms best case)
- **Trade-off:** Slightly higher DB CPU usage vs much better UX

### 2. Optimistic UI Pattern (Frontend)

**Recommended Approach:** Combine fast outbox polling with optimistic UI updates

```javascript
// Frontend Add to Cart Flow
async function addToCart(productId, quantity) {
  // 1. Optimistically add item to UI immediately
  const tempItem = {
    productId,
    quantity,
    status: 'pending_validation',
    lineNumber: generateTempLineNumber()
  };
  
  // Show item immediately with "Validating..." spinner
  cartUI.addItem(tempItem);
  
  // 2. Make API call
  try {
    const response = await api.post(`/carts/${cartId}/items`, {
      productId,
      quantity
    });
    
    // 3. API returns immediately with lineNumber
    // Update temp item with real line number
    cartUI.updateLineNumber(tempItem.tempId, response.lineNumber);
    
  } catch (error) {
    // Remove optimistic item on error
    cartUI.removeItem(tempItem.tempId);
    showError(error.message);
  }
}

// SSE handler updates the item when validation completes
eventSource.addEventListener('cart.item.validated', (e) => {
  const data = JSON.parse(e.data);
  cartUI.updateItem(data.lineNumber, {
    status: 'confirmed',
    productName: data.productName,
    unitPrice: data.unitPrice,
    totalPrice: data.totalPrice
  });
  // UI transitions from "Validating..." to "✓ Available"
});

eventSource.addEventListener('cart.item.backorder', (e) => {
  const data = JSON.parse(e.data);
  cartUI.updateItem(data.lineNumber, {
    status: 'backorder',
    backorderReason: data.backorderReason
  });
  // UI transitions from "Validating..." to "! Backorder"
});
```

**Expected User Experience:**
1. User clicks "Add to Cart" 
2. Item appears immediately with "Validating..." spinner (< 50ms)
3. Within 200-500ms, item updates to:
   - "✓ Available" with price (success)
   - "! Backorder" with reason (out of stock)

### 3. Performance Monitoring

**Key Metrics to Track:**
```prometheus
# Outbox processing latency
outbox_processing_duration_seconds{topic="CartEvents"}

# End-to-end validation time (from AddItem API to SSE)
histogram_quantile(0.95, 
  sum(rate(validation_completion_duration_seconds_bucket[5m])) by (le)
)

# Pending validation item count (alert if > threshold)
cart_items_pending_validation_count
```

**Alert Thresholds:**
- Validation time P95 > 1s: Warning
- Validation time P95 > 2s: Critical
- Pending items > 100 for > 5min: Product service may be down

### 4. Alternative: Trigger-Based Immediate Processing (Future Enhancement)

If 200ms is still too slow, consider:

```go
// After writing to outbox in AddItem, trigger immediate processing
func (s *CartService) AddItem(...) {
    // ... write item and event to outbox ...
    
    // Trigger immediate outbox processing for this event
    go s.infrastructure.OutboxPublisher.TriggerImmediateProcess()
}
```

**Pros:** Near-zero latency for validation events
**Cons:** More complex, could overwhelm if burst of events

**Recommendation:** Start with 200ms interval + optimistic UI. Only implement trigger-based if metrics show it's needed.

---

## Trade-offs

| Aspect | Before (HTTP Sync) | After (Event-Driven) |
|--------|-------------------|---------------------|
| **Coupling** | Tight - direct HTTP dependency | Loose - event-based communication |
| **Availability** | Cart fails if product service down | Cart works independently (pending state) |
| **Latency** | Synchronous wait (100-500ms) | Optimized to 200-500ms via fast outbox + optimistic UI |
| **Consistency** | Strong (immediate) | Eventual (seconds to minutes) |
| **Complexity** | Simple HTTP client | Higher - events, handlers, outbox |
| **Reliability** | At-least-once (HTTP retry) | At-least-once (outbox + Kafka) |
| **Data Loss Risk** | Low (immediate feedback) | Very Low (outbox persistence) |

---

## Success Metrics

1. **Availability**: Cart service uptime > 99.9% (independent of product service)
2. **Latency**: AddItem response time < 50ms (P95)
3. **Validation Time**: 95% of items validated within 500ms (end-to-end from AddItem to SSE)
4. **Error Rate**: < 0.1% validation failures
5. **Backorder Rate**: Track percentage of items on backorder

---

## Future Enhancements

1. **Validation Timeout**: Add max pending time (e.g., 30 seconds) with auto-backorder
2. **Price Refresh**: Allow users to refresh prices for backorder items
3. **Inventory Reservation**: Reserve stock during validation window
4. **Batch Validation**: Validate multiple items in single event for efficiency
5. **Validation Retry**: Retry validation for backorder items after time delay

---

## Summary of Key Changes from v1

1. **Outbox Pattern**: Changed from direct event bus publish to transactional outbox
2. **Product Service**: Added event consumption capability (EventServiceBase)
3. **Event Structure**: Created proper CartValidationEvent with factories
4. **Entity Methods**: Added domain methods for status transitions
5. **Repository**: Added GetItemByProductID for duplicate detection
6. **Transactions**: AddItem is now transactional (item + outbox event)
7. **SSE Events**: Separated validated vs backorder events
8. **Performance Optimization**: Added 200ms outbox interval config + optimistic UI pattern
9. **Edge Cases**: Added comprehensive handling for duplicates, removals, timeouts
10. **Testing**: Added specific test scenarios

This plan ensures strict adherence to the existing architecture while implementing the decoupling goal with sub-second validation latency.
