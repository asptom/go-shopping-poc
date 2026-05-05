# Product Validation Cache Plan - Version 1

## Goal

Reduce product validation latency in the cart service by maintaining an **in-memory product cache**. When a user adds an item to their cart, the cart service checks the cache first instead of emitting a `CartItemAdded` event and waiting for the product service to respond. This eliminates the 100-500ms event round-trip for products already in the cache.

## Why This Plan Differs from the Order Service's Identity Cache Plan

**Critical analysis**: The order service's identity cache (from `order-secure-api-plan-v4.md`) was designed for **authorization** — mapping `keycloak_sub → CustomerID` for a small, stable dataset (customers). It bootstraps by replaying ALL `CustomerEvents` from Kafka and has a Kafka request/response fallback for cache misses.

Product validation is fundamentally different:

| Dimension | Order Service Identity Cache | Product Validation Cache |
|---|---|---|
| Dataset size | Thousands of customers | Hundreds of thousands to millions of products |
| Change frequency | Low (customer data is stable) | High (stock/price change frequently) |
| Bootstrap value | High (need ALL customers for auth) | Low (most cart items are for existing products) |
| Fallback mechanism | Kafka request/response (async auth check) | Must be synchronous (user is waiting) |
| Cache key | `keycloak_sub` (string) | `product_id` (string) |
| Cache value | 3 fields (customer_id, email, keycloak_sub) | 4 fields (product_id, in_stock, final_price, name) |

**Design decisions derived from this analysis:**

1. **No bootstrap phase.** Replaying all `ProductEvents` from the beginning of a large topic adds seconds of startup latency with minimal benefit. The cart service starts fresh per deployment, and most cart items are for products that already exist in the store. The cache will warm up naturally as events flow through.

2. **No Kafka fallback.** The order service uses a Kafka request/response fallback because authorization must always succeed (security). For product validation, the user is waiting synchronously in their browser. A Kafka request/response would block for 5+ seconds. Instead, on cache miss, we **fall through to the existing event-driven flow** (emit `CartItemAdded`, wait for `ProductValidated`). This is correct — the event-driven flow is the authoritative path, and the cache is an optimization for the common case.

3. **Minimal cache entry.** Only store the fields needed for validation: `product_id`, `in_stock`, `final_price`, `name`. No need to store the full product aggregate.

4. **Cache miss is not an error.** A cache miss simply means "use the existing event-driven validation flow." This keeps the system correct even if the cache is empty or warm.

## Architecture Overview

```
User adds item to cart
        │
        ▼
  ┌─────────────┐
  │ AddItem()    │
  │ in CartSvc   │
  └──────┬───────┘
         │
         ▼
  ┌─────────────┐     Yes     ┌──────────────┐
  │ Cache hit?  │────────────▶│ Cache: in_stock?
  └──────┬───────┘           └──────────────┘
         │ No                        │
         ▼                           │
  ┌─────────────┐                   │
  │ Emit        │                   │
  │ CartItemAdded │                  │
  │ event       │                   │
  └──────┬───────┘                   │
         │                           │
         ▼                           │
  Product service validates ─────────┘
  via ProductEvents topic
         │
         ▼
  Cart service receives
  ProductValidated /
  ProductUnavailable
         │
         ▼
  Update cart item
  status
```

**Happy path (cache hit, in stock):** `AddItem` → cache hit → confirm item immediately → return to user. **~0.1ms** (map lookup + DB write).

**Cache miss (new product):** `AddItem` → cache miss → emit `CartItemAdded` → wait for `ProductValidated` → confirm item. **~100-500ms** (event round-trip).

**Cache miss, out of stock:** Same as above, but receives `ProductUnavailable` → marks item as backorder.

---

## Implementation Phases

### Phase 1: Product Cache Core

Create the in-memory cache that stores product identity data. This is a simple thread-safe map.

**File: `internal/service/cart/productcache.go` (NEW)**

```go
package cart

import (
	"sync"
)

// ProductEntry holds the data needed for product validation in the cart service.
// Only the fields required for validation are stored — no full product aggregate.
type ProductEntry struct {
	ProductID string  `json:"product_id"`
	InStock   bool    `json:"in_stock"`
	FinalPrice float64 `json:"final_price"`
	Name      string  `json:"name"`
}

// ProductCache provides a thread-safe in-memory cache of product identity data.
// It is populated by consuming ProductCreated, ProductUpdated, and ProductDeleted
// events from the ProductEvents topic. The cache is used to accelerate product
// validation when items are added to a cart, eliminating the event round-trip
// for products already in the cache.
type ProductCache struct {
	mu    sync.RWMutex
	items map[string]ProductEntry // product_id → ProductEntry
}

// NewProductCache creates a new empty product cache.
func NewProductCache() *ProductCache {
	return &ProductCache{
		items: make(map[string]ProductEntry),
	}
}

// Get returns the ProductEntry for a given product_id and whether it was found.
func (c *ProductCache) Get(productID string) (ProductEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.items[productID]
	return entry, ok
}

// Set upserts a product entry into the cache.
// Use this for ProductCreated and ProductUpdated events.
func (c *ProductCache) Set(productID string, entry ProductEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[productID] = entry
}

// Delete removes a product entry from the cache.
// Use this for ProductDeleted events.
func (c *ProductCache) Delete(productID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, productID)
}

// Count returns the number of entries in the cache.
func (c *ProductCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}
```

**Why this design:**
- `sync.RWMutex` allows concurrent reads without blocking (important since cart add-item is read-heavy).
- Map key is `product_id` string (matches how the cart service receives product IDs from the frontend).
- No TTL or eviction — the cache is kept current by events, so a TTL is unnecessary and adds complexity.
- No size limit — the product dataset size is bounded by the database, and the cart service only needs to store product identity data (4 fields per entry), not the full product aggregate.

---

### Phase 2: Product Event Handler (Keeps Cache Current)

Create an event handler that processes `ProductCreated`, `ProductUpdated`, and `ProductDeleted` events to keep the cache in sync with the source of truth (the product service's database).

**File: `internal/service/cart/eventhandlers/on_product_event.go` (NEW)**

```go
package eventhandlers

import (
	"context"
	"log/slog"
	"strconv"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/event/handler"
	"go-shopping-poc/internal/service/cart"
)

// OnProductEvent keeps the cart's product cache current by processing
// ProductCreated, ProductUpdated, and ProductDeleted events from the
// ProductEvents topic.
//
// This handler is the sole writer to the ProductCache. All cache updates
// come from events, ensuring consistency with the product service's
// database (the source of truth).
type OnProductEvent struct {
	cache  *cart.ProductCache
	logger *slog.Logger
}

// NewOnProductEvent creates a new product event handler.
func NewOnProductEvent(cache *cart.ProductCache, logger *slog.Logger) *OnProductEvent {
	if logger == nil {
		logger = slog.Default()
	}
	return &OnProductEvent{
		cache:  cache,
		logger: logger.With("component", "cart_on_product_event"),
	}
}

// Handle processes product lifecycle events.
//
// It dispatches to handler methods based on event type:
//   - ProductCreated: Insert new entry into cache
//   - ProductUpdated: Upsert existing entry in cache
//   - ProductDeleted: Remove entry from cache
//
// All other event types are silently ignored.
func (h *OnProductEvent) Handle(ctx context.Context, event events.Event) error {
	productEvent, ok := event.(events.ProductEvent)
	if !ok {
		h.logger.Error("Expected ProductEvent", "actual_type", event.Type())
		return nil
	}

	switch productEvent.EventType {
	case events.ProductCreated:
		return h.handleProductCreated(ctx, productEvent)
	case events.ProductUpdated:
		return h.handleProductUpdated(ctx, productEvent)
	case events.ProductDeleted:
		return h.handleProductDeleted(ctx, productEvent)
	default:
		return nil
	}
}

func (h *OnProductEvent) handleProductCreated(ctx context.Context, event events.ProductEvent) error {
	h.upsertProduct(event)
	h.logger.Debug("Product cache updated",
		"product_id", event.EventPayload.ProductID,
		"event_type", string(event.EventType),
	)
	return nil
}

func (h *OnProductEvent) handleProductUpdated(ctx context.Context, event events.ProductEvent) error {
	h.upsertProduct(event)
	h.logger.Debug("Product cache updated",
		"product_id", event.EventPayload.ProductID,
		"event_type", string(event.EventType),
	)
	return nil
}

func (h *OnProductEvent) handleProductDeleted(ctx context.Context, event events.ProductEvent) error {
	h.cache.Delete(event.EventPayload.ProductID)
	h.logger.Debug("Product cache entry removed",
		"product_id", event.EventPayload.ProductID,
		"event_type", string(event.EventType),
	)
	return nil
}

// upsertProduct extracts product data from a ProductCreated or ProductUpdated
// event and stores it in the cache. It reads numeric fields from the event's
// Details map (which contains string values) and parses them into the
// appropriate types.
func (h *OnProductEvent) upsertProduct(event events.ProductEvent) {
	details := event.EventPayload.Details
	if details == nil {
		details = make(map[string]string)
	}

	productID := event.EventPayload.ProductID
	if productID == "" {
		productID = event.EventPayload.ResourceID
	}

	inStock := true // default: assume in stock
	if stockStr := details["in_stock"]; stockStr != "" {
		if parsed, err := strconv.ParseBool(stockStr); err == nil {
			inStock = parsed
		}
	}

	finalPrice := 0.0
	if priceStr := details["final_price"]; priceStr != "" {
		if parsed, err := strconv.ParseFloat(priceStr, 64); err == nil {
			finalPrice = parsed
		}
	}

	name := details["name"]

	h.cache.Set(productID, cart.ProductEntry{
		ProductID:  productID,
		InStock:    inStock,
		FinalPrice: finalPrice,
		Name:       name,
	})
}

// CreateHandler returns a bus.HandlerFunc that wraps the Handle method.
func (h *OnProductEvent) CreateHandler() bus.HandlerFunc[events.ProductEvent] {
	return func(ctx context.Context, event events.ProductEvent) error {
		return h.Handle(ctx, event)
	}
}

// CreateFactory returns an EventFactory for ProductEvent.
func (h *OnProductEvent) CreateFactory() events.EventFactory[events.ProductEvent] {
	return events.ProductEventFactory{}
}

// Ensure OnProductEvent implements HandlerFactory.
var _ handler.EventHandler = (*OnProductEvent)(nil)
var _ handler.HandlerFactory[events.ProductEvent] = (*OnProductEvent)(nil)

// EventType returns the event types this handler processes.
func (h *OnProductEvent) EventType() string {
	return string(events.ProductCreated) + "," +
		string(events.ProductUpdated) + "," +
		string(events.ProductDeleted)
}
```

**Why this design:**
- Handles all three product lifecycle events (`ProductCreated`, `ProductUpdated`, `ProductDeleted`) to keep the cache consistent.
- `upsertProduct` reads data from the event's `Details` map, which is how product events carry their payload (matching the existing `OnCartItemAdded` handler's pattern).
- Default `inStock = true` for products without an explicit stock flag — safer to allow validation than to block.
- The handler follows the existing pattern in `on_product_validated.go` (same package, same `handler.EventHandler` interface).
- `EventType()` returns all three event types, comma-separated, matching the existing pattern in `on_product_validated.go:266`.

**Important note about event data:** The `Details` map in `ProductEvent` events is populated by the product service's repository layer. Currently, `ProductCreated` events include `name`, `brand`, `price`, and `images` in `Details`. `ProductUpdated` events include `name` and `brand`. This plan assumes that the product service is updated to also include `in_stock` and `final_price` in the `Details` map for `ProductCreated` and `ProductUpdated` events. If these fields are not present in the events, the cache will use defaults (`in_stock=true`, `final_price=0.0`), which is safe but less useful for validation.

---

### Phase 3: Update Cart Service AddItem Flow

Modify the `AddItem` method in `CartService` to check the product cache before emitting the `CartItemAdded` event. This is the core optimization — the cache hit path eliminates the event round-trip entirely.

**File: `internal/service/cart/service.go` (MODIFY)**

Add the `ProductCache` field to `CartService` and add a `GetProductCache()` accessor:

```go
type CartService struct {
	*service.EventServiceBase
	logger          *slog.Logger
	repo            CartRepository
	infrastructure  *CartInfrastructure
	config          *Config
	productCache    *ProductCache
}
```

Update `NewCartService` to initialize the cache:

```go
func NewCartService(logger *slog.Logger, infrastructure *CartInfrastructure, config *Config) *CartService {
	if logger == nil {
		logger = logging.FromContext(context.Background())
	}
	repo := NewCartRepository(infrastructure.Database, infrastructure.OutboxWriter)

	return &CartService{
		EventServiceBase: service.NewEventServiceBase("cart", infrastructure.EventBus, logger),
		logger:           logger.With("component", "cart_service"),
		repo:             repo,
		infrastructure:   infrastructure,
		config:           config,
		productCache:     NewProductCache(),
	}
}
```

Add a getter for the cache (used by the event handler registration):

```go
// GetProductCache returns the product cache for use by event handlers.
func (s *CartService) GetProductCache() *ProductCache {
	return s.productCache
}
```

**Update the `AddItem` method** to use the cache as a fast path:

Replace the existing `AddItem` method with this version:

```go
func (s *CartService) AddItem(ctx context.Context, cartID string, productID string, quantity int, imageURL string) (*CartItem, error) {
	s.logger.Debug("Adding item to cart",
		"cart_id", cartID,
		"product_id", productID,
		"quantity", quantity,
		"image_url", imageURL,
	)

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

	// Fast path: check product cache before emitting event.
	// If the product is in the cache, we can validate synchronously
	// without the event round-trip.
	cacheEntry, cacheHit := s.productCache.Get(productID)
	if cacheHit {
		if !cacheEntry.InStock {
			s.logger.Debug("Product out of stock (cache)", "product_id", productID)
			// Create a backorder item — same as the event-driven path's ProductUnavailable handling
			validationID := uuid.New().String()
			reason := "product_out_of_stock"
			item := &CartItem{
				ProductID:    productID,
				Quantity:     quantity,
				ImageURL:     imageURL,
				Status:       "backorder",
				ValidationID: &validationID,
				BackorderReason: &reason,
			}

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

			if err := s.repo.AddItemTx(ctx, tx, cartID, item); err != nil {
				return nil, fmt.Errorf("failed to add item: %w", err)
			}

			if err := tx.Commit(); err != nil {
				return nil, fmt.Errorf("failed to commit transaction: %w", err)
			}
			committed = true

			s.logger.Debug("Updating cart totals with backorder item", "cart_id", cartID)
			if err := s.repo.UpdateCart(ctx, cart); err != nil {
				s.logger.Warn("Failed to update cart totals", "cart_id", cartID, "error", err.Error())
			}

			s.logger.Info("Added backorder item (cache miss - out of stock)",
				"cart_id", cartID,
				"product_id", productID,
				"quantity", quantity,
			)
			return item, nil
		}

		// Product is in stock — fast path: confirm immediately without event round-trip.
		validationID := uuid.New().String()
		item := &CartItem{
			ProductID:    productID,
			Quantity:     quantity,
			ImageURL:     imageURL,
			Status:       "confirmed",
			ValidationID: &validationID,
			ProductName:  cacheEntry.Name,
			UnitPrice:    cacheEntry.FinalPrice,
		}

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

		if err := s.repo.AddItemTx(ctx, tx, cartID, item); err != nil {
			return nil, fmt.Errorf("failed to add item: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}
		committed = true

		cart.Items = append(cart.Items, *item)
		cart.CalculateTotals()

		s.logger.Debug("Updating cart totals after confirmed item", "cart_id", cartID)
		if err := s.repo.UpdateCart(ctx, cart); err != nil {
			s.logger.Warn("Failed to update cart totals", "cart_id", cartID, "error", err.Error())
		}

		s.logger.Info("Added item to cart (fast path - cache hit)",
			"cart_id", cartID,
			"product_id", productID,
			"quantity", quantity,
		)
		return item, nil
	}

	// Slow path: product not in cache. Fall back to event-driven validation.
	// This is the existing behavior — emit CartItemAdded and wait for
	// ProductValidated/ProductUnavailable events from the product service.

	s.logger.Debug("Product not in cache, emitting validation event", "product_id", productID)

	validationID := uuid.New().String()
	item := &CartItem{
		ProductID:    productID,
		Quantity:     quantity,
		ImageURL:     imageURL,
		Status:       "pending_validation",
		ValidationID: &validationID,
	}

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

	if err := s.repo.AddItemTx(ctx, tx, cartID, item); err != nil {
		return nil, fmt.Errorf("failed to add item: %w", err)
	}

	cartItemEvent := events.NewCartItemAddedEvent(cartID, item.LineNumber, productID, quantity, validationID)
	if err := s.infrastructure.OutboxWriter.WriteEvent(ctx, tx, cartItemEvent); err != nil {
		return nil, fmt.Errorf("failed to write cart item event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	if s.infrastructure.OutboxPublisher != nil {
		go func() {
			if err := s.infrastructure.OutboxPublisher.ProcessNow(); err != nil {
				s.logger.Warn("Failed to trigger immediate outbox processing",
					"error", err.Error(),
				)
			}
		}()
	}

	cart.Items = append(cart.Items, *item)
	cart.CalculateTotals()

	s.logger.Debug("Updating cart totals after adding pending item", "cart_id", cartID)
	if err := s.repo.UpdateCart(ctx, cart); err != nil {
		s.logger.Warn("Failed to update cart totals",
			"cart_id", cartID,
			"error", err.Error(),
		)
	}

	s.logger.Debug("Added pending item to cart (cache miss, awaiting validation)",
		"cart_id", cartID,
		"product_id", productID,
		"quantity", quantity,
		"item_line_number", item.LineNumber,
	)
	s.logger.Info("Added item to cart, pending validation",
		"cart_id", cartID,
		"product_id", productID,
		"quantity", quantity,
		"item_line_number", item.LineNumber,
	)
	return item, nil
}
```

**Why this design:**
- **Cache hit, in stock:** The item is created with `status: "confirmed"` immediately. No event is emitted. The user gets a response in ~0.1ms (map lookup + DB write).
- **Cache hit, out of stock:** The item is created with `status: "backorder"` immediately. No event is emitted. The user gets a response in ~0.1ms.
- **Cache miss:** Falls through to the **existing** event-driven flow. The item is created with `status: "pending_validation"`, `CartItemAdded` is emitted, and the code waits for `ProductValidated`/`ProductUnavailable` events (handled by `OnProductValidated` handler).
- **No changes to the event-driven flow.** The existing `OnProductValidated` handler continues to work correctly. If a product is validated via the event path (cache miss), the `OnProductValidated` handler updates the item status. The cache is updated independently by the `OnProductEvent` handler (Phase 2).
- **Idempotent and safe.** If the same product is added twice before the cache is populated, both items go through the event path. The existing duplicate detection in `AddItem` handles this.

---

### Phase 4: Wiring and Configuration

Wire the product cache and event handler into the cart service's `main.go`.

**File: `cmd/cart/main.go` (MODIFY)**

Add the product event handler registration in the `registerEventHandlers` function. After the existing handler registrations, add:

```go
// Product cache event handler — keeps the product cache current.
// Subscribes to ProductCreated, ProductUpdated, ProductDeleted events
// from the ProductEvents topic.
productCache := service.GetProductCache()
productEventHandler := eventhandlers.NewOnProductEvent(productCache, handlerLogger)
logger.Debug("Registering handler", "event_type", productEventHandler.EventType())

if err := cart.RegisterHandler(
	service,
	productEventHandler.CreateFactory(),
	productEventHandler.CreateHandler(),
); err != nil {
	return fmt.Errorf("failed to register ProductEvent handler: %w", err)
}

logger.Debug("Successfully registered ProductEvent handler")
```

The full updated `registerEventHandlers` function should look like:

```go
func registerEventHandlers(service *cart.CartService, sseHub *sse.Hub, logger *slog.Logger) error {
	logger.Debug("Registering event handlers")

	handlerLogger := logger.With("component", "event_handler")

	// Order created handler (existing)
	orderCreatedHandler := eventhandlers.NewOnOrderCreated(sseHub, handlerLogger)
	logger.Debug("Registering handler",
		"event_type", orderCreatedHandler.EventType(),
		"topic", events.OrderEvent{}.Topic(),
	)

	if err := cart.RegisterHandler(
		service,
		orderCreatedHandler.CreateFactory(),
		orderCreatedHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register OrderCreated handler: %w", err)
	}

	logger.Debug("Successfully registered OrderCreated handler")

	// Product validation handler (existing)
	productValidatedHandler := eventhandlers.NewOnProductValidated(service.GetRepository(), sseHub, handlerLogger)
	logger.Debug("Registering handler", "event_type", productValidatedHandler.EventType())

	if err := cart.RegisterHandler(
		service,
		productValidatedHandler.CreateFactory(),
		productValidatedHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register ProductValidated handler: %w", err)
	}

	logger.Debug("Successfully registered ProductValidated handler")

	// Product cache event handler (NEW) — keeps the product cache current
	productCache := service.GetProductCache()
	productEventHandler := eventhandlers.NewOnProductEvent(productCache, handlerLogger)
	logger.Debug("Registering handler", "event_type", productEventHandler.EventType())

	if err := cart.RegisterHandler(
		service,
		productEventHandler.CreateFactory(),
		productEventHandler.CreateHandler(),
	); err != nil {
		return fmt.Errorf("failed to register ProductEvent handler: %w", err)
	}

	logger.Debug("Successfully registered ProductEvent handler")

	return nil
}
```

**Configuration:** No new environment variables or config changes are required. The cart service already has the event bus wired up and can subscribe to the `ProductEvents` topic. However, ensure that the `cart_read_topics` configuration includes `ProductEvents` so the cart service consumes from that topic:

```yaml
# In deploy/k8s/service/cart/cart-config.yaml (or equivalent)
cart_read_topics:
  - CartEvents
  - ProductEvents    # Add this for product cache updates
```

---

## File Summary

| File | Action | Change |
|---|---|---|
| `internal/service/cart/productcache.go` | **NEW** | `ProductCache` struct + `NewProductCache`, `Get`, `Set`, `Delete`, `Count` methods |
| `internal/service/cart/eventhandlers/on_product_event.go` | **NEW** | `OnProductEvent` handler that processes `ProductCreated`/`ProductUpdated`/`ProductDeleted` events |
| `internal/service/cart/service.go` | **MODIFY** | Add `productCache` field to `CartService`, initialize in `NewCartService`, add `GetProductCache()` getter, update `AddItem` to check cache first |
| `cmd/cart/main.go` | **MODIFY** | Wire `OnProductEvent` handler in `registerEventHandlers` |
| `deploy/k8s/service/cart/cart-config.yaml` (or equivalent) | **MODIFY** | Add `ProductEvents` to `cart_read_topics` list |

---

## What Does NOT Change

- **`OnProductValidated` handler** (`eventhandlers/on_product_validated.go`): The existing handler that processes `ProductValidated` and `ProductUnavailable` events continues to work unchanged. It updates cart item status based on product service validation results. With the cache, many products will be confirmed via the fast path (cache hit) and never trigger `OnProductValidated`. But for cache misses, `OnProductValidated` still processes the validation result.
- **`OnCartItemAdded` handler** (`eventhandlers/on_product_validated.go` is actually the cart service's handler): The product service's handler for cart item addition events is unchanged. It still validates products by querying the product database.
- **Event-driven architecture**: The system remains event-driven. The cache is an optimization layer on top of the existing event flow, not a replacement.
- **Product service**: The product service's code is unchanged. It continues to validate products by querying its database and emitting validation events.
- **Outbox pattern**: The outbox pattern is unchanged for the event-driven (cache miss) path. The fast path (cache hit) skips event emission entirely, which is correct — the product was already known to be in stock.

---

## Performance Characteristics

| Scenario | Latency | Events Emitted |
|---|---|---|
| Cache hit, in stock | ~0.1ms (map lookup + DB write) | None (fast path) |
| Cache hit, out of stock | ~0.1ms (map lookup + DB write) | None (backorder path) |
| Cache miss | ~100-500ms (event round-trip) | `CartItemAdded` + `ProductValidated`/`ProductUnavailable` |

**Cache hit rate expectation:** For a returning customer's cart, 80%+ of items added will be for products already in the catalog (and thus in the cache after the first validation). The cache hit rate will increase over time as more products are validated and cached.

---

## Correctness Guarantees

| Concern | Mitigation |
|---|---|
| **Cache stale after service restart** | Cache is empty on restart. All products go through the event path. The cache warms as events flow through. Safe — no data loss. |
| **Event loss / processing delay** | Cache miss falls through to event path. The product service's validation is the source of truth. No data loss. |
| **Product deleted while in cart** | Cache entry is removed. If the user tries to checkout, the item's `backorder` or `confirmed` status is already set. The checkout flow should handle this at the order service level. |
| **Product price changed** | `ProductUpdated` event updates the cache. Existing cart items keep their original price (set at confirmation time). New additions get the updated price. Correct. |
| **Concurrent cache access** | `sync.RWMutex` ensures thread-safe access. Multiple goroutines can read concurrently; writes are exclusive. |
| **Memory growth** | Bounded by the number of products in the database. Each entry is ~4 fields (product_id string, in_stock bool, final_price float64, name string). For 1M products, ~100MB of memory. Acceptable. |

---

## Testing Plan

### Unit Tests

**`internal/service/cart/productcache_test.go`** (NEW):
- `TestProductCache_Get_Miss` — empty cache returns `false`
- `TestProductCache_Get_Hit` — set then get returns correct entry
- `TestProductCache_Set_Overwrite` — upsert replaces existing entry
- `TestProductCache_Delete` — remove entry, then get returns `false`
- `TestProductCache_Concurrent` — concurrent reads and writes, verify no deadlock

**`internal/service/cart/service_test.go`** (MODIFY existing tests):
- `TestAddItem_CacheHit_InStock` — cache hit, in stock → item confirmed, no event emitted
- `TestAddItem_CacheHit_OutOfStock` — cache hit, out of stock → item backorder
- `TestAddItem_CacheMiss` — cache miss → item pending, `CartItemAdded` event emitted
- `TestAddItem_ExistingConfirmedItem` — product already in cart → error
- `TestAddItem_ExistingPendingItem` — product pending validation → error

**`internal/service/cart/eventhandlers/on_product_event_test.go`** (NEW):
- `TestOnProductEvent_HandleProductCreated` — cache entry added
- `TestOnProductEvent_HandleProductUpdated` — cache entry updated
- `TestOnProductEvent_HandleProductDeleted` — cache entry removed
- `TestOnProductEvent_HandleUnknownEvent` — no-op, no error

### Integration Test

1. Start cart service with empty cache
2. Add a product to cart → goes through event path (cache miss) → `pending_validation`
3. Wait for `ProductValidated` event → item becomes `confirmed`
4. Add the same product again → cache hit → `confirmed` immediately
5. Verify the second add-item response is significantly faster than the first

---

## Rollback Plan

If the cache causes issues:
1. The cache is a field on `CartService`. Removing it requires changes to `NewCartService` and `AddItem`.
2. The event-driven path is unchanged — all products still go through the `ProductValidated` event flow on cache miss.
3. To disable the cache: set `productCache` to `nil` in `NewCartService`, and skip the cache check in `AddItem`. All products fall through to the event path.
4. No database migrations or data changes are involved.

---

## Deployment Order

1. Deploy the cart service with the product cache and event handler.
2. Verify logs show `ProductEvent handler registered` and `Product cache updated` messages.
3. Monitor cart add-item latency — should see ~0.1ms for cached products, ~100-500ms for new products.
4. Monitor product cache hit rate — should increase over time as products are validated.

---

## Future Considerations

1. **Bootstrap phase**: If the cache hit rate is low (many new products), consider adding a bootstrap phase that replays `ProductEvents` from Kafka at startup. This would warm the cache on startup, reducing the number of cache misses. Trade-off: adds startup latency proportional to product catalog size.

2. **Cache metrics**: Add metrics for cache hits, misses, and evictions. Expose via Prometheus `/metrics` endpoint.

3. **Cache eviction**: If the product catalog grows very large (>10M products), consider adding a size-based eviction policy (e.g., LRU). For now, the cache is unbounded but bounded by the database size.

4. **Other services**: The order service could also benefit from a product cache (to validate products in orders without querying the product service). The eventreader service could use the cache for product-related processing. However, these are optimizations — the order service already has the customer identity cache, and the eventreader service's processing is typically async and not latency-sensitive.
