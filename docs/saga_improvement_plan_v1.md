# SAGA Pattern Improvement Plan v1

## Executive Summary

This plan addresses critical architectural violations in the current cart-product validation implementation. The current design violates the SAGA pattern by using a shared `CartValidationEvent` contract that forces services to write to topics they don't own. This plan provides a clean, idiomatic implementation using proper domain events with each service owning its own event types and topics exclusively.

**Key Issues Addressed:**
1. Product service writing to `CartEvents` topic (violates topic ownership)
2. Cart service reading validation responses from `CartEvents` (tight coupling)
3. Shared `cart_validation.go` contract between services (breaks bounded contexts)
4. Hacky outbox timing configurations (200ms polling workaround)
5. Unnecessary configuration complexity

**Solution:** True SAGA choreography pattern with domain events:
- Cart service emits `CartItemAdded` to `CartEvents` (cart domain)
- Product service emits `ProductValidated` to `ProductEvents` (product domain)
- Immediate outbox processing eliminates polling delays
- Services react to each other's events without tight coupling

---

## Architecture Decision Records (ADRs)

### ADR 1: Domain Events vs Shared Validation Events

**Context:** Need to decouple cart and product services while maintaining validation flow.

**Options Considered:**

1. **Keep CartValidationEvent with fixed topics** (Minimal Change)
   - Keep shared contract, just change topic assignments
   - Product writes to `ProductEvents` instead of `CartEvents`
   - **Rejected:** Still maintains shared contract coupling

2. **RPC-over-Events with Request/Reply Pattern** (Current v2 Plan)
   - Use `CartItemValidationRequested` and `CartItemValidationCompleted`
   - **Rejected:** Violates SAGA - services shouldn't request/reply, they should emit events

3. **Domain Events with Event Sourcing** (Overkill)
   - Full event sourcing with aggregate snapshots
   - **Rejected:** Too complex for current requirements

4. **Domain Events with Choreography** (Selected)
   - Cart emits `CartItemAdded` (fact: item was added)
   - Product emits `ProductValidated` (fact: product was validated)
   - Services react to facts, not requests
   - **Selected:** Clean SAGA pattern, proper decoupling

**Decision:** Use domain events (Option 4)
**Consequences:** 
- Positive: True decoupling, each service owns its events
- Positive: Easy to add new consumers (analytics, audit, etc.)
- Negative: More event types to manage
- Negative: Eventual consistency requires UI handling

### ADR 2: Immediate Outbox Processing vs Polling

**Context:** Need to eliminate 200ms polling delay for validation events.

**Options Considered:**

1. **Decrease polling interval** (Current approach)
   - Set `OutboxProcessInterval` to 200ms
   - **Rejected:** Wastes DB resources, still has latency

2. **Trigger-based processing** (Selected)
   - Trigger immediate outbox processing after transaction commit
   - Background polling as fallback
   - **Selected:** Near-zero latency without waste

3. **Synchronous event publishing** (Direct Kafka write)
   - Skip outbox, publish directly to Kafka
   - **Rejected:** Loses transactional guarantees

4. **Change Data Capture (CDC)**
   - Use Debezium to capture DB changes
   - **Rejected:** Adds operational complexity

**Decision:** Trigger-based with background polling (Option 2)
**Consequences:**
- Positive: Near real-time event publishing
- Positive: Maintains transactional consistency
- Negative: Requires careful goroutine management
- Negative: Need buffering to handle bursts

---

## Implementation Phases

### Phase 1: Event Contract Refactoring (Cleanup & Foundation)
**Objective:** Remove broken contracts and establish proper domain events
**Estimated Time:** 2-3 days
**Checkpoint:** Tests pass with new event structure

#### 1.1 Remove cart_validation.go Contract

**Files to Delete:**
- `internal/contracts/events/cart_validation.go`

**Files to Modify:**
- `internal/service/cart/eventhandlers/on_cart_item_validation_completed.go` - Remove entire file
- `internal/service/product/eventhandlers/on_cart_item_validation_requested.go` - Remove entire file
- `cmd/cart/main.go` - Remove handler registration
- `cmd/product/main.go` - Remove handler registration

**Cleanup Actions:**
```bash
# Remove the shared validation contract
rm internal/contracts/events/cart_validation.go

# Remove broken event handlers
rm internal/service/cart/eventhandlers/on_cart_item_validation_completed.go
rm internal/service/product/eventhandlers/on_cart_item_validation_requested.go
```

#### 1.2 Add CartItemAdded Event to Cart Domain

**File:** `internal/contracts/events/cart.go`

**Additions:**
```go
// CartItemEventType defines cart item-specific event types
type CartItemEventType string

const (
    CartItemAdded     CartItemEventType = "cart.item.added"
    CartItemConfirmed CartItemEventType = "cart.item.confirmed"
    CartItemRejected  CartItemEventType = "cart.item.rejected"
)

// CartItemPayload contains cart item event data
type CartItemPayload struct {
    CartID       string `json:"cart_id"`
    LineNumber   string `json:"line_number"`
    ProductID    string `json:"product_id"`
    Quantity     int    `json:"quantity"`
    ProductName  string `json:"product_name,omitempty"`
    UnitPrice    float64 `json:"unit_price,omitempty"`
}

// CartItemEvent represents cart item lifecycle events
type CartItemEvent struct {
    ID        string          `json:"id"`
    EventType CartItemEventType `json:"type"`
    Timestamp time.Time       `json:"timestamp"`
    Payload   CartItemPayload `json:"payload"`
}

// CartItemEventFactory implements EventFactory
type CartItemEventFactory struct{}

func (f CartItemEventFactory) FromJSON(data []byte) (CartItemEvent, error) {
    var event CartItemEvent
    err := json.Unmarshal(data, &event)
    return event, err
}

// Event interface implementations
func (e CartItemEvent) Type() string            { return string(e.EventType) }
func (e CartItemEvent) Topic() string           { return "CartEvents" }
func (e CartItemEvent) Payload() any            { return e.Payload }
func (e CartItemEvent) ToJSON() ([]byte, error) { return json.Marshal(e) }
func (e CartItemEvent) GetEntityID() string     { return e.Payload.CartID }
func (e CartItemEvent) GetResourceID() string   { return e.ID }

// Constructor for CartItemAdded
func NewCartItemAddedEvent(cartID, lineNumber, productID string, quantity int) *CartItemEvent {
    return &CartItemEvent{
        ID:        uuid.New().String(),
        EventType: CartItemAdded,
        Timestamp: time.Now(),
        Payload: CartItemPayload{
            CartID:     cartID,
            LineNumber: lineNumber,
            ProductID:  productID,
            Quantity:   quantity,
        },
    }
}

// Constructor for CartItemConfirmed
func NewCartItemConfirmedEvent(cartID, lineNumber, productID, productName string, unitPrice float64, quantity int) *CartItemEvent {
    return &CartItemEvent{
        ID:        uuid.New().String(),
        EventType: CartItemConfirmed,
        Timestamp: time.Now(),
        Payload: CartItemPayload{
            CartID:      cartID,
            LineNumber:  lineNumber,
            ProductID:   productID,
            ProductName: productName,
            UnitPrice:   unitPrice,
            Quantity:    quantity,
        },
    }
}

// Constructor for CartItemRejected
func NewCartItemRejectedEvent(cartID, lineNumber, productID, reason string) *CartItemEvent {
    return &CartItemEvent{
        ID:        uuid.New().String(),
        EventType: CartItemRejected,
        Timestamp: time.Now(),
        Payload: CartItemPayload{
            CartID:     cartID,
            LineNumber: lineNumber,
            ProductID:  productID,
        },
    }
}
```

#### 1.3 Add ProductValidated Event to Product Domain

**File:** `internal/contracts/events/product.go`

**Additions:**
```go
// Add to existing ProductEventType constants
const (
    // ... existing constants ...
    ProductValidated   ProductEventType = "product.validated"
    ProductUnavailable ProductEventType = "product.unavailable"
)

// ProductValidationPayload contains product validation results
type ProductValidationPayload struct {
    ProductID     string  `json:"product_id"`
    ProductName   string  `json:"product_name,omitempty"`
    UnitPrice     float64 `json:"unit_price,omitempty"`
    IsAvailable   bool    `json:"is_available"`
    Reason        string  `json:"reason,omitempty"` // e.g., "out_of_stock", "not_found"
    
    // Context information (optional, for correlation)
    CartID        string  `json:"cart_id,omitempty"`
    LineNumber    string  `json:"line_number,omitempty"`
}

// NewProductValidatedEvent creates a product validation success event
func NewProductValidatedEvent(productID, productName string, unitPrice float64, cartID, lineNumber string) *ProductEvent {
    details := map[string]string{
        "cart_id":     cartID,
        "line_number": lineNumber,
        "unit_price":  fmt.Sprintf("%.2f", unitPrice),
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
func NewProductUnavailableEvent(productID, reason string, cartID, lineNumber string) *ProductEvent {
    details := map[string]string{
        "cart_id":     cartID,
        "line_number": lineNumber,
        "reason":      reason,
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
```

**Checkpoint 1.1:** Run `go build ./...` - should compile without cart_validation.go

---

### Phase 2: Product Service Event Handler Implementation
**Objective:** Create handler that listens for cart events and emits product events
**Estimated Time:** 1-2 days
**Checkpoint:** Product service can receive cart events and emit validation events

#### 2.1 Create CartItemAdded Handler in Product Service

**New File:** `internal/service/product/eventhandlers/on_cart_item_added.go`

```go
package eventhandlers

import (
    "context"
    "fmt"
    "log"
    "strconv"

    events "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/platform/event/handler"
    "go-shopping-poc/internal/service/product"
)

// OnCartItemAdded handles cart item addition events
// It validates the product and emits product validation events
type OnCartItemAdded struct {
    service *product.CatalogService
}

// NewOnCartItemAdded creates a new cart item added handler
func NewOnCartItemAdded(service *product.CatalogService) *OnCartItemAdded {
    return &OnCartItemAdded{
        service: service,
    }
}

// Handle processes cart item addition events
func (h *OnCartItemAdded) Handle(ctx context.Context, event events.Event) error {
    cartItemEvent, ok := event.(events.CartItemEvent)
    if !ok {
        log.Printf("[ERROR] Product: Expected CartItemEvent, got %T", event)
        return nil
    }

    // Only handle CartItemAdded events
    if cartItemEvent.EventType != events.CartItemAdded {
        log.Printf("[DEBUG] Product: Ignoring event type: %s", cartItemEvent.EventType)
        return nil
    }

    payload := cartItemEvent.Payload
    
    utils := handler.NewEventUtils()
    utils.LogEventProcessing(ctx, string(cartItemEvent.EventType),
        payload.ProductID,
        payload.CartID)

    log.Printf("[DEBUG] Product: Processing cart item added - CartID: %s, ProductID: %s, Quantity: %d",
        payload.CartID, payload.ProductID, payload.Quantity)

    // Parse product ID
    productID, err := strconv.ParseInt(payload.ProductID, 10, 64)
    if err != nil {
        log.Printf("[ERROR] Product: Invalid product ID format: %s", payload.ProductID)
        return h.publishValidationResult(ctx, payload, false, 0, "invalid_product_id")
    }

    // Get product details
    product, err := h.service.GetProductByID(ctx, productID)
    if err != nil {
        log.Printf("[DEBUG] Product: Product %s not found: %v", payload.ProductID, err)
        return h.publishValidationResult(ctx, payload, false, 0, "product_not_found")
    }

    if !product.InStock {
        log.Printf("[DEBUG] Product: Product %s is out of stock", payload.ProductID)
        return h.publishValidationResult(ctx, payload, false, 0, "out_of_stock")
    }

    log.Printf("[DEBUG] Product: Product %s validated successfully", payload.ProductID)
    return h.publishValidationResult(ctx, payload, true, product.FinalPrice, "")
}

func (h *OnCartItemAdded) publishValidationResult(ctx context.Context, payload events.CartItemPayload, isAvailable bool, unitPrice float64, reason string) error {
    infra := h.service.GetInfrastructure()

    tx, err := infra.Database.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }

    committed := false
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()

    var event *events.ProductEvent
    if isAvailable {
        // Get product name from service (we need to fetch it again or pass it)
        productID, _ := strconv.ParseInt(payload.ProductID, 10, 64)
        product, _ := h.service.GetProductByID(ctx, productID)
        productName := ""
        if product != nil {
            productName = product.Name
        }
        event = events.NewProductValidatedEvent(payload.ProductID, productName, unitPrice, payload.CartID, payload.LineNumber)
    } else {
        event = events.NewProductUnavailableEvent(payload.ProductID, reason, payload.CartID, payload.LineNumber)
    }

    if err := infra.OutboxWriter.WriteEvent(ctx, tx, event); err != nil {
        return fmt.Errorf("failed to write validation event to outbox: %w", err)
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit validation event: %w", err)
    }
    committed = true

    log.Printf("[DEBUG] Product: Published validation result for product %s (available: %v)",
        payload.ProductID, isAvailable)
    return nil
}

// CreateHandler returns a bus.HandlerFunc that wraps the Handle method
func (h *OnCartItemAdded) CreateHandler() bus.HandlerFunc[events.CartItemEvent] {
    return func(ctx context.Context, event events.CartItemEvent) error {
        return h.Handle(ctx, event)
    }
}

// CreateFactory returns an EventFactory for CartItemEvent
func (h *OnCartItemAdded) CreateFactory() events.EventFactory[events.CartItemEvent] {
    return events.CartItemEventFactory{}
}

// Ensure OnCartItemAdded implements HandlerFactory
var _ handler.HandlerFactory[events.CartItemEvent] = (*OnCartItemAdded)(nil)
```

#### 2.2 Update Product Service Main

**File:** `cmd/product/main.go`

**Changes:**
```go
// Replace old handler registration:
// OLD:
// validationHandler := eventhandlers.NewOnCartItemValidationRequested(catalogService)
// if err := product.RegisterHandler(...)

// NEW:
cartItemAddedHandler := eventhandlers.NewOnCartItemAdded(catalogService)
if err := product.RegisterHandler(
    catalogService,
    cartItemAddedHandler.CreateFactory(),
    cartItemAddedHandler.CreateHandler(),
); err != nil {
    log.Fatalf("Product: Failed to register CartItemAdded handler: %v", err)
}
log.Printf("[INFO] Product: Registered CartItemAdded handler")
```

**Checkpoint 2.1:** Product service compiles and starts successfully

---

### Phase 3: Cart Service Event Handler Implementation
**Objective:** Create handler that listens for product validation events
**Estimated Time:** 1-2 days
**Checkpoint:** Cart service receives validation events and updates items

#### 3.1 Create ProductValidated Handler in Cart Service

**New File:** `internal/service/cart/eventhandlers/on_product_validated.go`

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

// OnProductValidated handles product validation events
// It updates cart items based on product validation results
type OnProductValidated struct {
    repo   cart.CartRepository
    sseHub *sse.Hub
}

// NewOnProductValidated creates a new product validation handler
func NewOnProductValidated(repo cart.CartRepository, sseHub *sse.Hub) *OnProductValidated {
    return &OnProductValidated{
        repo:   repo,
        sseHub: sseHub,
    }
}

// Handle processes product validation events
func (h *OnProductValidated) Handle(ctx context.Context, event events.Event) error {
    productEvent, ok := event.(events.ProductEvent)
    if !ok {
        log.Printf("[ERROR] Cart: Expected ProductEvent, got %T", event)
        return nil
    }

    // Handle both validated and unavailable events
    switch productEvent.EventType {
    case events.ProductValidated:
        return h.handleProductValidated(ctx, productEvent)
    case events.ProductUnavailable:
        return h.handleProductUnavailable(ctx, productEvent)
    default:
        log.Printf("[DEBUG] Cart: Ignoring product event type: %s", productEvent.EventType)
        return nil
    }
}

func (h *OnProductValidated) handleProductValidated(ctx context.Context, event events.ProductEvent) error {
    details := event.EventPayload.Details
    cartID := details["cart_id"]
    lineNumber := details["line_number"]
    productID := event.EventPayload.ProductID

    if cartID == "" || lineNumber == "" {
        log.Printf("[ERROR] Cart: Missing cart_id or line_number in ProductValidated event")
        return nil
    }

    utils := handler.NewEventUtils()
    utils.LogEventProcessing(ctx, string(event.EventType), productID, cartID)

    // Get cart and item
    cartObj, err := h.repo.GetCartByID(ctx, cartID)
    if err != nil {
        log.Printf("[ERROR] Cart: Failed to get cart %s: %v", cartID, err)
        return err
    }

    // Find the item
    var targetItem *cart.CartItem
    for i := range cartObj.Items {
        if cartObj.Items[i].LineNumber == lineNumber {
            targetItem = &cartObj.Items[i]
            break
        }
    }

    if targetItem == nil {
        log.Printf("[DEBUG] Cart: Item %s not found in cart %s - may have been removed", lineNumber, cartID)
        return nil
    }

    // Parse unit price from details
    unitPrice := 0.0
    if priceStr := details["unit_price"]; priceStr != "" {
        if parsed, err := strconv.ParseFloat(priceStr, 64); err == nil {
            unitPrice = parsed
        }
    }

    // Confirm the item
    productName := event.EventPayload.Details["product_name"]
    if productName == "" {
        productName = targetItem.ProductName // Keep existing if not provided
    }

    if err := targetItem.ConfirmItem(productName, unitPrice); err != nil {
        log.Printf("[ERROR] Cart: Failed to confirm item %s: %v", lineNumber, err)
        return err
    }

    // Update item in database
    if err := h.repo.UpdateItemStatus(ctx, targetItem); err != nil {
        log.Printf("[ERROR] Cart: Failed to update item %s status: %v", lineNumber, err)
        return err
    }

    // Recalculate cart totals
    cartObj.CalculateTotals()
    if err := h.repo.UpdateCart(ctx, cartObj); err != nil {
        log.Printf("[ERROR] Cart: Failed to update cart %s totals: %v", cartID, err)
        return err
    }

    // Emit CartItemConfirmed event for audit/completeness
    confirmedEvent := events.NewCartItemConfirmedEvent(
        cartID, lineNumber, productID, productName, unitPrice, targetItem.Quantity,
    )
    // Note: This would require outbox writer access - may skip for now

    // Send SSE notification
    if h.sseHub != nil {
        h.sseHub.Publish(cartID, "cart.item.confirmed", map[string]interface{}{
            "lineNumber":  lineNumber,
            "productId":   productID,
            "productName": productName,
            "unitPrice":   unitPrice,
            "quantity":    targetItem.Quantity,
            "totalPrice":  targetItem.TotalPrice,
            "status":      "confirmed",
        })
    }

    log.Printf("[INFO] Cart: Item %s confirmed for cart %s", lineNumber, cartID)
    return nil
}

func (h *OnProductValidated) handleProductUnavailable(ctx context.Context, event events.ProductEvent) error {
    details := event.EventPayload.Details
    cartID := details["cart_id"]
    lineNumber := details["line_number"]
    productID := event.EventPayload.ProductID
    reason := details["reason"]

    if cartID == "" || lineNumber == "" {
        log.Printf("[ERROR] Cart: Missing cart_id or line_number in ProductUnavailable event")
        return nil
    }

    utils := handler.NewEventUtils()
    utils.LogEventProcessing(ctx, string(event.EventType), productID, cartID)

    // Get cart and item
    cartObj, err := h.repo.GetCartByID(ctx, cartID)
    if err != nil {
        log.Printf("[ERROR] Cart: Failed to get cart %s: %v", cartID, err)
        return err
    }

    // Find the item
    var targetItem *cart.CartItem
    for i := range cartObj.Items {
        if cartObj.Items[i].LineNumber == lineNumber {
            targetItem = &cartObj.Items[i]
            break
        }
    }

    if targetItem == nil {
        log.Printf("[DEBUG] Cart: Item %s not found in cart %s - may have been removed", lineNumber, cartID)
        return nil
    }

    // Mark as backorder
    if err := targetItem.MarkAsBackorder(reason); err != nil {
        log.Printf("[ERROR] Cart: Failed to mark item %s as backorder: %v", lineNumber, err)
        return err
    }

    // Update item in database
    if err := h.repo.UpdateItemStatus(ctx, targetItem); err != nil {
        log.Printf("[ERROR] Cart: Failed to update item %s status: %v", lineNumber, err)
        return err
    }

    // Recalculate cart totals
    cartObj.CalculateTotals()
    if err := h.repo.UpdateCart(ctx, cartObj); err != nil {
        log.Printf("[ERROR] Cart: Failed to update cart %s totals: %v", cartID, err)
        return err
    }

    // Send SSE notification
    if h.sseHub != nil {
        h.sseHub.Publish(cartID, "cart.item.backorder", map[string]interface{}{
            "lineNumber":      lineNumber,
            "productId":       productID,
            "status":          "backorder",
            "backorderReason": reason,
        })
    }

    log.Printf("[INFO] Cart: Item %s marked as backorder for cart %s: %s", lineNumber, cartID, reason)
    return nil
}

// CreateHandler returns a bus.HandlerFunc that wraps the Handle method
func (h *OnProductValidated) CreateHandler() bus.HandlerFunc[events.ProductEvent] {
    return func(ctx context.Context, event events.ProductEvent) error {
        return h.Handle(ctx, event)
    }
}

// CreateFactory returns an EventFactory for ProductEvent
func (h *OnProductValidated) CreateFactory() events.EventFactory[events.ProductEvent] {
    return events.ProductEventFactory{}
}

// Ensure OnProductValidated implements HandlerFactory
var _ handler.HandlerFactory[events.ProductEvent] = (*OnProductValidated)(nil)
```

#### 3.2 Update Cart Service Main

**File:** `cmd/cart/main.go`

**Changes:**
```go
// Replace old handler registration:
// OLD:
// validationHandler := eventhandlers.NewOnCartItemValidationCompleted(...)
// if err := cart.RegisterHandler(...)

// NEW:
productValidatedHandler := eventhandlers.NewOnProductValidated(service.GetRepository(), sseProvider.GetHub())
if err := cart.RegisterHandler(
    service,
    productValidatedHandler.CreateFactory(),
    productValidatedHandler.CreateHandler(),
); err != nil {
    log.Fatalf("Cart: Failed to register ProductValidated handler: %v", err)
}
log.Printf("[INFO] Cart: Registered ProductValidated handler")
```

**Checkpoint 3.1:** Cart service compiles and starts successfully

---

### Phase 4: Update Cart Service AddItem Implementation
**Objective:** Modify AddItem to emit CartItemAdded event instead of validation request
**Estimated Time:** 1 day
**Checkpoint:** Adding items to cart emits proper events

#### 4.1 Modify CartService.AddItem

**File:** `internal/service/cart/service.go`

**Current AddItem Implementation:**
```go
// Look for the section that writes validation event to outbox
// Around line 188 in current implementation:
validationEvent := events.NewCartItemValidationRequestedEvent(cartID, productID, quantity, correlationID)
if err := s.infrastructure.OutboxWriter.WriteEvent(ctx, tx, validationEvent); err != nil {
    return nil, fmt.Errorf("failed to write validation event: %w", err)
}
```

**New Implementation:**
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

    // Create item with pending status
    item := &CartItem{
        ProductID: productID,
        Quantity:  quantity,
        Status:    "pending_validation",
        // LineNumber will be assigned by repository
    }

    // Begin transaction to add item and write event
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

    // Add item to cart within transaction
    if err := s.repo.AddItemTx(ctx, tx, cartID, item); err != nil {
        return nil, fmt.Errorf("failed to add item: %w", err)
    }

    // Emit CartItemAdded event to outbox (transactional)
    // This notifies other services (like product) that an item was added
    cartItemEvent := events.NewCartItemAddedEvent(cartID, item.LineNumber, productID, quantity)
    if err := s.infrastructure.OutboxWriter.WriteEvent(ctx, tx, cartItemEvent); err != nil {
        return nil, fmt.Errorf("failed to write cart item event: %w", err)
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit transaction: %w", err)
    }
    committed = true

    // Trigger immediate outbox processing for low latency
    // This is non-blocking and ensures the event is published quickly
    if s.infrastructure.OutboxPublisher != nil {
        go func() {
            if err := s.infrastructure.OutboxPublisher.ProcessNow(); err != nil {
                log.Printf("[WARN] Cart: Failed to trigger immediate outbox processing: %v", err)
            }
        }()
    }

    // Update cart totals with pending item (best effort, not transactional)
    cart.Items = append(cart.Items, *item)
    cart.CalculateTotals()

    log.Printf("[DEBUG] CartService: Updating cart totals for cart %s after adding pending item", cartID)
    if err := s.repo.UpdateCart(ctx, cart); err != nil {
        log.Printf("[WARN] CartService: failed to update cart totals for cart %s: %v", cartID, err)
        // Don't fail the request, cart totals will be updated on validation
    }

    log.Printf("[INFO] CartService: Added pending item %s to cart %s", item.LineNumber, cartID)
    return item, nil
}
```

**Checkpoint 4.1:** AddItem operation works and emits CartItemAdded events

---

### Phase 5: Immediate Outbox Processing Implementation
**Objective:** Implement trigger-based outbox processing to eliminate polling delays
**Estimated Time:** 1-2 days
**Checkpoint:** Events published immediately after transaction commit

#### 5.1 Add ProcessNow Method to Outbox Publisher

**File:** `internal/platform/outbox/publisher.go`

**Additions:**
```go
// ProcessNow immediately processes pending outbox events
// This is non-blocking and can be called after writing events
func (p *Publisher) ProcessNow() error {
    select {
    case <-p.shutdownCtx.Done():
        return nil // Publisher is shutting down
    default:
        // Process immediately in a goroutine to avoid blocking
        go p.processOutbox()
        return nil
    }
}

// ProcessNowBlocking immediately processes pending outbox events and waits for completion
// Use sparingly - only when you need to ensure events are published before continuing
func (p *Publisher) ProcessNowBlocking(ctx context.Context) error {
    return p.processOutbox()
}
```

#### 5.2 Add Buffered Channel for Rate Limiting

**File:** `internal/platform/outbox/publisher.go`

**Modify Publisher struct:**
```go
type Publisher struct {
    db              database.Database
    publisher       bus.Bus
    batchSize       int           // Number of events to process in a single batch
    processInterval time.Duration // Time between outbox scans
    shutdownCtx     context.Context
    shutdownCancel  context.CancelFunc
    immediateQueue  chan struct{} // Buffered channel for immediate processing requests
    maxConcurrent   int           // Maximum concurrent immediate processing goroutines
    currentProcessing int32       // Atomic counter for current processing goroutines
}
```

**Update NewPublisher:**
```go
func NewPublisher(db database.Database, publisher bus.Bus, cfg Config) *Publisher {
    ctx, cancel := context.WithCancel(context.Background())
    return &Publisher{
        db:              db,
        publisher:       publisher,
        batchSize:       cfg.BatchSize,
        processInterval: cfg.ProcessInterval,
        shutdownCtx:     ctx,
        shutdownCancel:  cancel,
        immediateQueue:  make(chan struct{}, 100), // Buffer up to 100 immediate requests
        maxConcurrent:   10,                       // Max 10 concurrent processing goroutines
    }
}
```

**Update Start method to process immediate queue:**
```go
// Start begins the publishing process.
func (p *Publisher) Start() {
    // Start background ticker for polling
    go func() {
        ticker := time.NewTicker(p.processInterval)
        defer ticker.Stop()

        for {
            select {
            case <-p.shutdownCtx.Done():
                return
            case <-ticker.C:
                p.processOutbox()
            }
        }
    }()

    // Start immediate processing worker
    go p.processImmediateQueue()
}

func (p *Publisher) processImmediateQueue() {
    for {
        select {
        case <-p.shutdownCtx.Done():
            return
        case <-p.immediateQueue:
            // Check if we're at max concurrent
            current := atomic.LoadInt32(&p.currentProcessing)
            if current >= int32(p.maxConcurrent) {
                // Too many concurrent, skip this one (will be picked up by polling)
                log.Printf("[DEBUG] Outbox Publisher: Max concurrent processing reached, skipping immediate")
                continue
            }

            // Increment counter and process
            atomic.AddInt32(&p.currentProcessing, 1)
            if err := p.processOutbox(); err != nil {
                log.Printf("[ERROR] Outbox Publisher: Immediate processing error: %v", err)
            }
            atomic.AddInt32(&p.currentProcessing, -1)
        }
    }
}

// ProcessNow triggers immediate outbox processing
func (p *Publisher) ProcessNow() error {
    select {
    case <-p.shutdownCtx.Done():
        return nil
    case p.immediateQueue <- struct{}{}:
        // Queued for immediate processing
        return nil
    default:
        // Queue full, will be picked up by polling
        log.Printf("[DEBUG] Outbox Publisher: Immediate queue full, event will be processed by polling")
        return nil
    }
}
```

**Add required import:**
```go
import (
    // ... existing imports ...
    "sync/atomic"
)
```

**Checkpoint 5.1:** Outbox publisher can process events immediately

---

### Phase 6: Configuration Cleanup
**Objective:** Remove unnecessary configuration fields and update deployment files
**Estimated Time:** 0.5 days
**Checkpoint:** Clean configuration with no unused fields

#### 6.1 Remove Outbox Configuration from Cart Service

**File:** `internal/service/cart/config.go`

**Remove:**
```go
// REMOVE THESE FIELDS:
// OutboxBatchSize       int           `mapstructure:"cart_outbox_batch_size"`
// OutboxProcessInterval time.Duration `mapstructure:"cart_outbox_process_interval"`
```

**Update Validate method:**
```go
func (c *Config) Validate() error {
    if c.DatabaseURL == "" {
        return errors.New("database URL is required")
    }
    if c.ServicePort == "" {
        return errors.New("service port is required")
    }
    if c.WriteTopic == "" {
        return errors.New("write topic is required")
    }
    // Removed outbox configuration validation
    return nil
}
```

**Update cmd/cart/main.go:**
```go
// OLD:
outboxConfig := outbox.Config{
    BatchSize:       cfg.OutboxBatchSize,
    ProcessInterval: cfg.OutboxProcessInterval,
}

// NEW: Use platform defaults
outboxConfig := outbox.Config{
    BatchSize:       10,    // Use platform default
    ProcessInterval: 5 * time.Second, // Use platform default (polling fallback only)
}
```

#### 6.2 Remove Outbox Configuration from Product Service

**File:** `internal/service/product/config.go`

**Remove:**
```go
// REMOVE THESE FIELDS:
// OutboxBatchSize       int           `mapstructure:"product_outbox_batch_size"`
// OutboxProcessInterval time.Duration `mapstructure:"product_outbox_process_interval"`
```

**Update Validate method:**
```go
func (c *Config) Validate() error {
    if c.DatabaseURL == "" {
        return errors.New("database URL is required")
    }
    if c.ServicePort == "" {
        return errors.New("service port is required")
    }
    if c.MinIOBucket == "" {
        return errors.New("MinIO bucket is required")
    }
    // Removed outbox configuration validation
    return nil
}
```

**Update cmd/product/main.go:**
```go
// OLD: Create outbox publisher with custom config
// outboxConfig := outbox.Config{...}
// outboxPublisher := outbox.NewPublisher(...)

// NEW: Product service doesn't need outbox publisher (it only consumes events)
// Only needs outbox writer for publishing events
```

#### 6.3 Update Deployment ConfigMaps

**File:** `deploy/k8s/service/cart/cart-configmap.yaml`

**Remove:**
```yaml
# REMOVE THESE LINES:
# CART_OUTBOX_BATCH_SIZE: "10"
# CART_OUTBOX_PROCESS_INTERVAL: "200ms"
```

**Update ReadTopics:**
```yaml
# OLD:
CART_READ_TOPICS: "OrderEvents"

# NEW (add ProductEvents to read topics):
CART_READ_TOPICS: "OrderEvents,ProductEvents"
```

**File:** `deploy/k8s/service/product/product-configmap.yaml`

**Add ReadTopics:**
```yaml
# Add to existing config:
PRODUCT_READ_TOPICS: "CartEvents"
```

**Checkpoint 6.1:** All services use platform default outbox configuration

---

### Phase 7: Integration and End-to-End Testing
**Objective:** Verify complete flow works end-to-end
**Estimated Time:** 2 days
**Checkpoint:** Full validation flow works with <100ms latency

#### 7.1 Test Scenarios

**Scenario 1: Happy Path**
1. Create cart
2. Add item to cart
3. Verify `CartItemAdded` event emitted
4. Verify product service receives event
5. Verify `ProductValidated` event emitted
6. Verify cart service receives validation
7. Verify item status updated to "confirmed"
8. Verify SSE event sent to frontend

**Scenario 2: Out of Stock**
1. Add out-of-stock product to cart
2. Verify `ProductUnavailable` event emitted
3. Verify item status updated to "backorder"
4. Verify SSE event sent with reason

**Scenario 3: Duplicate Prevention**
1. Add item (pending validation)
2. Try to add same product again
3. Verify error returned immediately

**Scenario 4: Item Removed During Validation**
1. Add item
2. Remove item before validation completes
3. Verify validation result handled gracefully (no error)

#### 7.2 Performance Testing

**Latency Requirements:**
- AddItem API response: <50ms
- End-to-end validation: <100ms (CartItemAdded → ProductValidated → CartItemConfirmed)

**Load Testing:**
- 100 concurrent AddItem requests
- Verify no goroutine leaks
- Verify outbox queue doesn't overflow

**Checkpoint 7.1:** All test scenarios pass

---

### Phase 8: Documentation and Cleanup
**Objective:** Update documentation and remove all deprecated files
**Estimated Time:** 0.5 days
**Checkpoint:** Clean codebase with updated documentation

#### 8.1 Update Documentation

**File:** `docs/cart_product_decouple_plan_v2.md`

Add deprecation notice at top:
```markdown
# DEPRECATED

This plan is deprecated. See `saga_improvement_plan_v1.md` for the current implementation.

This file is kept for historical reference only.
```

#### 8.2 Remove Deprecated Documentation

**Files to Consider Removing:**
- `docs/cart_product_decouple_plan_v2.md` (after verifying new plan works)
- `docs/cart_product_decouple_plan.md` (if exists and is older)

#### 8.3 Update README

Update any README files that reference the old validation approach.

**Checkpoint 8.1:** Documentation is current and accurate

---

## File Modification Summary

### Files to Delete:
1. `internal/contracts/events/cart_validation.go`
2. `internal/service/cart/eventhandlers/on_cart_item_validation_completed.go`
3. `internal/service/product/eventhandlers/on_cart_item_validation_requested.go`
4. `docs/cart_product_decouple_plan_v2.md` (after migration)

### Files to Modify:
1. `internal/contracts/events/cart.go` - Add CartItemEvent types
2. `internal/contracts/events/product.go` - Add ProductValidated/ProductUnavailable events
3. `internal/platform/outbox/publisher.go` - Add immediate processing
4. `internal/service/cart/service.go` - Update AddItem to emit CartItemAdded
5. `internal/service/cart/config.go` - Remove outbox configuration
6. `internal/service/product/config.go` - Remove outbox configuration
7. `cmd/cart/main.go` - Update handler registration and outbox config
8. `cmd/product/main.go` - Update handler registration
9. `deploy/k8s/service/cart/cart-configmap.yaml` - Remove outbox config, update read topics
10. `deploy/k8s/service/product/product-configmap.yaml` - Add read topics

### Files to Create:
1. `internal/service/product/eventhandlers/on_cart_item_added.go`
2. `internal/service/cart/eventhandlers/on_product_validated.go`
3. `docs/saga_improvement_plan_v1.md` (this file)

---

## Risk Assessment

### High Risk:
1. **Event Ordering**: If CartItemAdded is processed before item is committed to DB
   - Mitigation: Events written in same transaction as item
   
2. **Goroutine Leaks**: Immediate processing creates many goroutines
   - Mitigation: Buffered channel with max concurrent limit

### Medium Risk:
1. **Duplicate Events**: Network issues may cause duplicate events
   - Mitigation: Idempotent handlers (already implemented)
   
2. **Configuration Conflicts**: Old config values may linger in environments
   - Mitigation: Explicit cleanup in deployment files

### Low Risk:
1. **Performance Degradation**: Immediate processing adds overhead
   - Mitigation: Benchmark testing, fallback to polling

---

## Success Criteria

1. ✅ Cart service only writes to `CartEvents` topic
2. ✅ Product service only writes to `ProductEvents` topic
3. ✅ No shared event contracts between services
4. ✅ End-to-end validation latency <100ms
5. ✅ No service-specific outbox timing configurations
6. ✅ All existing functionality preserved
7. ✅ All tests pass
8. ✅ No goroutine leaks under load

---

## Rollback Plan

If issues arise:

1. **Immediate**: Revert to previous deployment using git tags
2. **Configuration**: Can temporarily re-enable old config values
3. **Code**: Can restore cart_validation.go and old handlers
4. **Data**: No data migration needed - events are transient

**Rollback Command:**
```bash
git checkout <previous-stable-tag>
kubectl apply -f deploy/k8s/
```

---

## Post-Implementation Monitoring

### Metrics to Watch:
1. `outbox_events_processed_total` - Should see immediate processing spike
2. `outbox_processing_duration_seconds` - Should be <10ms for immediate
3. `cart_item_validation_duration_seconds` - End-to-end latency
4. Goroutine count - Should not grow unbounded

### Alerts:
1. Validation latency P95 > 200ms
2. Pending validation items > 100 for > 5 minutes
3. Goroutine count > 1000

---

## Conclusion

This plan provides a clean, idiomatic implementation of the SAGA pattern that:
- Eliminates architectural violations
- Removes tight coupling between services
- Achieves sub-100ms validation latency
- Follows Go and project conventions
- Maintains backward compatibility of data
- Is fully testable and monitorable

The implementation uses proper domain events where each service owns its own event types, creating true decoupling while maintaining the validation flow required for cart-product interaction.
