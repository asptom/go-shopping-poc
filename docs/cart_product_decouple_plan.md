# Cart-Product Decoupling Plan: Optimistic Add with Backorder

## Overview

Replace synchronous HTTP calls between cart and product services with an event-driven, eventually consistent pattern using the SAGA pattern.

## Current Problem

The `CartService.AddItem()` method makes a synchronous HTTP call to the product service:
- Validates product exists
- Checks if product is in stock
- Retrieves product name and price

This creates tight coupling and availability issues if product service is down.

## Proposed Solution: Optimistic Add with Backorder

### Event Flow

```
CartService                          ProductService
    |                                      |
    |------ AddItem() -------------------->|
    |    (publishes to CartEvents)         |
    |                                      |
    |<----- validates product -------------|
    |    (subscribes to CartEvents)        |
    |                                      |
    |------ validation result ------------>|
    |    (publishes to ProductEvents)      |
    |                                      |
    |<---- receives result ----------------|
    |    (subscribes to ProductEvents)     |
    |                                      |
    |------ SSE push to frontend --------->|
```

**Each service:**
- **Writes** only to its own topic (CartEvents for cart, ProductEvents for product)
- **Reads** from the other service's topic to receive responses

## Answers to Design Questions

### 1. Include backorder items in cart totals?
**Yes** - Include backorder items in cart totals for now. This provides visibility to the user that they've attempted to add an item and allows them to see the full cost including backorder items.

### 2. How to handle validation timeouts?

Instead of complex timer/timeout logic, use **SSE push notifications**:

- When cart item status changes (pending → confirmed or backorder), push an SSE event to the frontend
- Frontend listens for cart updates and refreshes the cart automatically
- No server-side timers needed
- If product service is down, item remains in "pending_validation" indefinitely until it recovers and processes the event
- Frontend can show a "validating..." spinner for pending items

### 3. Allow checkout with backorder items?
**Yes** - Allow checkout with backorder items. Frontend developers will need to:

- Display backorder items differently (e.g., different color, badge, or separate section)
- Show the backorder reason to the user (e.g., "Out of stock - will ship when available")
- Allow user to remove backorder items before checkout if desired

## Implementation Plan

### Phase 0: Existing SSE Implementation (Already Done)

The cart service already has SSE support implemented. This plan leverages that existing infrastructure.

**Existing components:**
- `internal/platform/sse/hub.go` - SSE hub with `Publish(cartID, event, data)` method
- `internal/platform/sse/provider.go` - SSE provider
- `internal/platform/sse/handler.go` - SSE HTTP handler
- `internal/service/cart/eventhandlers/on_order_created.go` - Example of SSE integration

**SSE route already exists:** `GET /api/v1/carts/{id}/stream`

The SSE hub is accessible via `infrastructure.SSEProvider.GetHub()`.

### Phase 1: Define Events

**New file: `internal/contracts/events/product_validation.go`**

```go
// Event types
const (
    CartItemValidationRequested ProductEventType = "cart.item.validation.requested"
    ProductValidationResult     ProductEventType = "product.validation.result"
)

// CartItemValidationRequested payload
type CartItemValidationPayload struct {
    CorrelationID string `json:"correlation_id"`
    CartID        string `json:"cart_id"`
    ProductID     string `json:"product_id"`
    Quantity      int    `json:"quantity"`
}

// ProductValidationResult payload
type ProductValidationResultPayload struct {
    CorrelationID string  `json:"correlation_id"`
    IsValid       bool    `json:"is_valid"`
    InStock       bool    `json:"in_stock"`
    ProductName   string  `json:"product_name,omitempty"`
    UnitPrice     float64 `json:"unit_price,omitempty"`
    Reason        string  `json:"reason,omitempty"` // "out_of_stock", "product_not_found"
}
```

### Phase 2: Update CartItem Entity

**File: `internal/service/cart/entity.go`**

Add to CartItem struct:
```go
type CartItem struct {
    // ... existing fields ...
    Status          string  `json:"status" db:"status"`           // "pending_validation", "confirmed", "backorder"
    BackorderReason string  `json:"backorder_reason,omitempty" db:"backorder_reason"`
    ValidationID    string  `json:"validation_id,omitempty" db:"validation_id"` // correlationID
}
```

### Phase 3: Repository Updates

**File: `internal/service/cart/repository_items.go`**

- Add `UpdateItemStatus(ctx, cartID, lineNumber, status, reason string)` method
- Update DB schema: add `status`, `backorder_reason`, `validation_id` columns

### Phase 4: Modify AddItem Service Method

**File: `internal/service/cart/service.go` - `AddItem`**

Before (sync HTTP):
```go
product, err := s.infrastructure.ProductClient.GetProduct(ctx, productID)
if err != nil { return nil, err }
if !product.InStock { return nil, errors.New("out of stock") }
item := &CartItem{
    ProductID:   productID,
    ProductName: product.Name,
    UnitPrice:   product.FinalPrice,
    Quantity:    quantity,
}
```

After (optimistic + async):
```go
correlationID := uuid.New().String()

item := &CartItem{
    ProductID:    productID,
    Quantity:     quantity,
    Status:       "pending_validation",
    ValidationID: correlationID,
    // Use placeholder values or null - update after validation
}

s.repo.AddItem(ctx, cartID, item)

// Emit validation event to ProductEvents topic (product service reads from here)
validationEvent := events.NewCartItemValidationRequestedEvent(cartID, productID, quantity, correlationID)
s.infrastructure.EventBus.Publish(ctx, "ProductEvents", validationEvent)

return item, nil
```

**Event Flow:**
1. **Cart** publishes to **CartEvents** (cart's write topic)
2. **Product** subscribes to **CartEvents**, processes validation request
3. **Product** publishes response to **ProductEvents** (product's write topic)
4. **Cart** subscribes to **ProductEvents**, receives validation result
5. **Cart** pushes SSE to frontend

### Phase 5: Add Product Service Handler

**New file: `internal/service/product/eventhandlers/on_cart_item_validation_requested.go`**

- Subscribe to: `CartEvents` topic (cart's write topic)
- Validate product (exists, in stock)
- Publish: `product.validation.result` to `ProductEvents` topic (product's write topic)

**Note:** The product service currently does not have event bus configured. Following the pattern in `cmd/eventreader/main.go`, the product service needs to be updated to listen for CartEvents.

### Phase 5a: Update Product Service to Listen for CartEvents

The product service needs to be enhanced to consume events from the event bus, similar to `cmd/eventreader`.

#### 5a.1 Update Product Config

**File: `internal/service/product/config.go`**

Add event bus configuration:

```go
type Config struct {
    // ... existing fields ...
    
    // Event bus configuration (for consuming CartEvents)
    ReadTopics []string `mapstructure:"product_read_topics"`
    WriteTopic string   `mapstructure:"product_write_topic" validate:"required"`
    Group      string   `mapstructure:"product_group"`
}
```

#### 5a.2 Update Product Infrastructure

**File: `internal/service/product/service.go`** (or create new infrastructure)

Add event bus to infrastructure:

```go
type CatalogInfrastructure struct {
    Database     database.Database
    OutboxWriter *outbox.Writer
    EventBus     bus.Bus  // Add event bus for consuming CartEvents
}
```

#### 5a.3 Update Product Main.go

**File: `cmd/product/main.go`**

Add event bus setup (following `cmd/eventreader/main.go` pattern):

```go
import (
    // ... existing imports ...
    "go-shopping-poc/internal/platform/event"
    "go-shopping-poc/internal/service/product/eventhandlers"
)

func main() {
    // ... existing setup code ...
    
    // Add event bus configuration
    eventBusConfig := event.EventBusConfig{
        WriteTopic: cfg.WriteTopic,
        GroupID:    cfg.Group,
    }
    eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
    if err != nil {
        log.Fatalf("Product: Failed to create event bus provider: %v", err)
    }
    eventBus := eventBusProvider.GetEventBus()
    
    // Update infrastructure to include event bus
    catalogInfra := &product.CatalogInfrastructure{
        Database:     platformDB,
        OutboxWriter: writerProvider.GetWriter(),
        EventBus:     eventBus,
    }
    
    // Create service
    catalogService := product.NewCatalogService(catalogInfra, cfg)
    
    // Register event handlers
    if err := registerEventHandlers(catalogService, eventBus); err != nil {
        log.Fatalf("Product: Failed to register event handlers: %v", err)
    }
    
    // Start event consumer
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go func() {
        if err := catalogService.Start(ctx); err != nil {
            log.Printf("[ERROR] Product: Event consumer stopped: %v", err)
        }
    }()
    
    // ... rest of HTTP server setup ...
}

func registerEventHandlers(service *product.CatalogService, eventBus bus.Bus) error {
    log.Printf("[INFO] Product: Registering event handlers...")
    
    // Register CartItemValidationRequested handler
    validationHandler := eventhandlers.NewOnCartItemValidationRequestedHandler(service)
    
    log.Printf("[INFO] Product: Registering handler for event type: %s", validationHandler.EventType())
    
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

#### 5a.4 Create Product Event Handler

**New file: `internal/service/product/eventhandlers/on_cart_item_validation_requested.go`**

```go
package eventhandlers

import (
    "context"
    "log"
    
    events "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/platform/event/handler"
)

type OnCartItemValidationRequested struct {
    catalogService *product.CatalogService
    eventBus       bus.Bus
}

func NewOnCartItemValidationRequestedHandler(catalogService *product.CatalogService, eventBus bus.Bus) *OnCartItemValidationRequested {
    return &OnCartItemValidationRequested{
        catalogService: catalogService,
        eventBus:       eventBus,
    }
}

func (h *OnCartItemValidationRequested) Handle(ctx context.Context, event events.Event) error {
    validationEvent, ok := event.(events.CartItemValidationRequestedEvent)
    if !ok {
        log.Printf("[ERROR] Product: Expected CartItemValidationRequestedEvent, got %T", event)
        return nil
    }
    
    if validationEvent.EventType != events.CartItemValidationRequested {
        log.Printf("[DEBUG] Product: Ignoring event type: %s", validationEvent.EventType)
        return nil
    }
    
    payload := validationEvent.EventPayload
    
    log.Printf("[DEBUG] Product: Validating product %s for cart %s", payload.ProductID, payload.CartID)
    
    // Validate product exists and is in stock
    product, err := h.catalogService.GetProduct(ctx, payload.ProductID)
    
    result := events.ProductValidationResultPayload{
        CorrelationID: payload.CorrelationID,
    }
    
    if err != nil {
        result.IsValid = false
        result.Reason = "product_not_found"
        log.Printf("[DEBUG] Product: Product %s not found", payload.ProductID)
    } else if !product.InStock {
        result.IsValid = false
        result.Reason = "out_of_stock"
        result.InStock = false
        log.Printf("[DEBUG] Product: Product %s is out of stock", payload.ProductID)
    } else {
        result.IsValid = true
        result.InStock = true
        result.ProductName = product.Name
        result.UnitPrice = product.FinalPrice
        log.Printf("[DEBUG] Product: Product %s validated successfully", payload.ProductID)
    }
    
    // Emit response event to ProductEvents topic (cart subscribes to ProductEvents)
    responseEvent := events.NewProductValidationResultEvent(result)
    if err := h.eventBus.Publish(ctx, "ProductEvents", responseEvent); err != nil {
        log.Printf("[ERROR] Product: Failed to publish validation result: %v", err)
        return err
    }
    
    return nil
}

func (h *OnCartItemValidationRequested) EventType() string {
    return string(events.CartItemValidationRequested)
}

func (h *OnCartItemValidationRequested) CreateHandler() bus.HandlerFunc[events.CartItemValidationRequestedEvent] {
    return func(ctx context.Context, event events.CartItemValidationRequestedEvent) error {
        return h.Handle(ctx, event)
    }
}

func (h *OnCartItemValidationRequested) CreateFactory() events.EventFactory[events.CartItemValidationRequestedEvent] {
    return events.CartItemValidationRequestedEventFactory{}
}
```

#### 5a.5 Add RegisterHandler to Product Service

**File: `internal/service/product/service.go`**

Add the generic RegisterHandler method (similar to cart service):

```go
// RegisterHandler adds a new event handler for any event type to the service
func RegisterHandler[T events.Event](s Service, factory events.EventFactory[T], handler bus.HandlerFunc[T]) error {
    return service.RegisterHandler(s, factory, handler)
}
```

#### 5a.6 Update Product Service to Implement Service Interface

The product service needs to implement the `service.Service` interface to support event handler registration. This may require adding:
- `Name()` method
- `Start(ctx context.Context)` method (for starting event consumer)
- `Stop(ctx context.Context)` method
- Event bus accessor

```go
func (h *OnCartItemValidationRequested) Handle(ctx context.Context, event events.Event) error {
    // Extract payload
    payload := event.(events.CartItemValidationRequestedEvent).EventPayload
    
    // Validate product
    product, err := h.productRepo.GetProduct(ctx, payload.ProductID)
    
    result := events.ProductValidationResultPayload{
        CorrelationID: payload.CorrelationID,
        IsValid:       err == nil && product.InStock,
        InStock:       product.InStock,
        ProductName:   product.Name,
        UnitPrice:     product.FinalPrice,
    }
    
    if err != nil {
        result.Reason = "product_not_found"
    } else if !product.InStock {
        result.Reason = "out_of_stock"
    }
    
    // Emit response event
    responseEvent := events.NewProductValidationResultEvent(result)
    h.eventBus.Publish(ctx, "CartEvents", responseEvent)
    
    return nil
}
```

### Phase 6: Add Cart Service Response Handler

**New file: `internal/service/cart/eventhandlers/on_product_validation_result.go`**

Following the same pattern as `OnOrderCreated` (see existing implementation at `internal/service/cart/eventhandlers/on_order_created.go`):

```go
package eventhandlers

import (
    "context"
    "log"
    
    events "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/platform/event/handler"
    "go-shopping-poc/internal/platform/sse"
)

type OnProductValidationResult struct {
    repo    cart.CartRepository
    sseHub *sse.Hub
}

func NewOnProductValidationResult(repo cart.CartRepository, sseHub *sse.Hub) *OnProductValidationResult {
    return &OnProductValidationResult{
        repo:    repo,
        sseHub: sseHub,
    }
}

func (h *OnProductValidationResult) Handle(ctx context.Context, event events.Event) error {
    validationEvent, ok := event.(events.ProductValidationResultEvent)
    if !ok {
        log.Printf("[ERROR] Cart: Expected ProductValidationResultEvent, got %T", event)
        return nil
    }
    
    if validationEvent.EventType != events.ProductValidationResult {
        log.Printf("[DEBUG] Cart: Ignoring event type: %s", validationEvent.EventType)
        return nil
    }
    
    payload := validationEvent.EventPayload
    
    // Find cart item by ValidationID (correlationID)
    item, err := h.repo.GetItemByValidationID(ctx, payload.CorrelationID)
    if err != nil {
        log.Printf("[DEBUG] Cart: Item not found for validation ID %s - may have been removed", payload.CorrelationID)
        return nil // Item not found - may have been removed by user
    }
    
    if payload.IsValid {
        // Update item to confirmed
        item.Status = "confirmed"
        item.ProductName = payload.ProductName
        item.UnitPrice = payload.UnitPrice
        log.Printf("[DEBUG] Cart: Item %s validated and confirmed for cart %s", item.LineNumber, item.CartID)
    } else {
        // Mark as backorder
        item.Status = "backorder"
        item.BackorderReason = payload.Reason
        log.Printf("[DEBUG] Cart: Item %s marked as backorder for cart %s: %s", item.LineNumber, item.CartID, payload.Reason)
    }
    
    if err := h.repo.UpdateItem(ctx, item); err != nil {
        return err
    }
    
    // Recalculate cart totals
    cart, err := h.repo.GetCartByID(ctx, item.CartID)
    if err != nil {
        return err
    }
    cart.CalculateTotals()
    if err := h.repo.UpdateCart(ctx, cart); err != nil {
        return err
    }
    
    // Push SSE to frontend - using existing SSE Hub
    if h.sseHub != nil {
        h.sseHub.Publish(
            item.CartID,
            "cart.item.validated",
            map[string]interface{}{
                "lineNumber":      item.LineNumber,
                "productId":       item.ProductID,
                "status":          item.Status,
                "productName":     item.ProductName,
                "unitPrice":       item.UnitPrice,
                "backorderReason": item.BackorderReason,
            },
        )
    }
    
    return nil
}

func (h *OnProductValidationResult) EventType() string {
    return string(events.ProductValidationResult)
}

func (h *OnProductValidationResult) CreateHandler() bus.HandlerFunc[events.ProductValidationResultEvent] {
    return func(ctx context.Context, event events.ProductValidationResultEvent) error {
        return h.Handle(ctx, event)
    }
}

func (h *OnProductValidationResult) CreateFactory() events.EventFactory[events.ProductValidationResultEvent] {
    return events.ProductValidationResultEventFactory{}
}
```

### Phase 7: Register New Event Handler

**File: `cmd/cart/main.go`** - Update `registerEventHandlers` function

Add registration for the new product validation result handler:

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
    
    // NEW: ProductValidationResult handler
    // Pass the repository to find items by correlation ID
    validationHandler := eventhandlers.NewOnProductValidationResult(service.GetRepository(), sseHub)
    if err := cart.RegisterHandler(
        service,
        validationHandler.CreateFactory(),
        validationHandler.CreateHandler(),
    ); err != nil {
        return fmt.Errorf("failed to register ProductValidationResult handler: %w", err)
    }
    
    log.Printf("[INFO] Cart: Successfully registered all event handlers")
    return nil
}
```

**Note:** The cart service needs to expose its repository (or add a method to find item by validation ID) for the handler to use.

### Phase 8: Handle Checkout with Backorders

**File: `internal/service/cart/service.go` - `Checkout`**

- Allow checkout with backorder items (don't block)
- Cart totals already include backorder items
- Frontend handles display of backorder status

### Phase 9: Remove HTTP ProductClient

**Files to update:**
- `internal/service/cart/service.go` - Remove `ProductClient` from `CartInfrastructure`
- `internal/service/cart/product_client.go` - Delete file
- `cmd/cart/main.go` - Remove ProductClient initialization

### Implementation Notes

#### Repository Method Required

The `OnProductValidationResult` handler needs to find cart items by validation ID (correlation ID). Add this method to the repository:

**File: `internal/service/cart/repository_items.go`**

```go
// GetItemByValidationID finds a cart item by its validation correlation ID
func (r *CartRepository) GetItemByValidationID(ctx context.Context, validationID string) (*CartItem, error) {
    // Query database to find item where validation_id = validationID
    // Return the item or ErrItemNotFound
}
```

#### CartService Repository Access

The event handler needs access to the repository. Options:
1. Expose repository via `CartService` method: `service.GetRepository()`
2. Add a new method to `CartService`: `service.GetItemByValidationID(ctx, validationID)`

#### Event Factory Registration

The new `ProductValidationResultEvent` needs an event factory for deserialization. This is defined in Phase 1 and follows the same pattern as `OrderEventFactory`.

## Frontend Integration Guide

### Cart Display Changes

1. **Pending Items**: Show item with "Validating..." status or spinner
2. **Confirmed Items**: Display normally (green checkmark)
3. **Backorder Items**: Display with warning badge and reason (e.g., "Out of stock")

### Example UI States

```
Item: Widget A          [✓ In Stock]     $10.00 x 2 = $20.00
Item: Widget B          [⟳ Validating...] $15.00 x 1 = $15.00
Item: Widget C          [! Backorder]     $5.00 x 3 = $15.00 (Out of stock)
```

### SSE Events

The SSE endpoint is already implemented at `GET /api/v1/carts/{id}/stream`.

**Existing events:**
- `connected` - Sent on initial connection
- `order.created` - Order has been created from cart (existing)

**New events for cart-item validation:**
- `cart.item.validated` - Item validation completed (confirmed or backorder)

**Frontend usage:**

```javascript
// Open SSE connection
const cartId = 'cart-uuid';
const eventSource = new EventSource(`/api/v1/carts/${cartId}/stream`);

eventSource.addEventListener('connected', (e) => {
  console.log('Connected to cart stream:', JSON.parse(e.data));
});

// Listen for item validation events
eventSource.addEventListener('cart.item.validated', (e) => {
  const data = JSON.parse(e.data);
  console.log('Item validated:', data);
  
  // Update the specific item in the cart UI
  // data.lineNumber - the item line number
  // data.status - "confirmed" or "backorder"
  // data.productName - validated product name
  // data.unitPrice - validated unit price
  // data.backorderReason - if backorder, the reason (e.g., "out_of_stock", "product_not_found")
  
  updateCartItemUI(data.lineNumber, data);
});

// Listen for order created events (existing)
eventSource.addEventListener('order.created', (e) => {
  const order = JSON.parse(e.data);
  console.log('Order created:', order);
  window.location.href = `/order-confirmation/${order.orderNumber}`;
  eventSource.close();
});

eventSource.addEventListener('error', (e) => {
  console.error('SSE error:', e);
});
```

## Trade-offs

| Aspect | Before | After |
|--------|--------|-------|
| Coupling | Tight (HTTP) | Loose (events) |
| Availability | Fails if product service down | Cart works if product service down (pending state) |
| Latency | Sync wait for product service | Immediate response, async validation |
| Consistency | Strong | Eventual |
| Complexity | Simple | Higher (more events, handlers) |

## Testing Considerations

### Unit Tests
- Test event creation and serialization
- Test CartItem status transitions
- Test repository GetItemByValidationID method

### Integration Tests
1. **Happy path**: Add item → validation event → confirmation event → SSE event → status updated
2. **Product not found**: Add item → validation event → result event (invalid) → backorder
3. **Product out of stock**: Add item → validation event → result event (out of stock) → backorder
4. **Product service down**: Add item → validation event (sent) → item stays pending → when service recovers, processes event
5. **Item removed before validation**: Validation result arrives → item not found → ignore (log and return nil)
6. **SSE connection**: Verify `cart.item.validated` event is pushed to connected clients
