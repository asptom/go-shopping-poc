# Order Created SSE Implementation Plan

## Overview

This document provides a detailed implementation plan for adding Server-Sent Events (SSE) support to the Cart Service. The SSE endpoint will allow the frontend to receive real-time notifications when an order is created from a cart checkout, completing the saga pattern implementation.

**Use Case:** Frontend calls `POST /api/v1/carts/{id}/checkout`, then connects to `GET /api/v1/carts/{id}/stream` to receive the order number when the `order.created` event completes the checkout saga.

**Assumptions:**
- Single-instance cart service deployment (no need for distributed pub/sub)
- Frontend is in a separate repository
- Use in-memory subscription map for simplicity (POC-appropriate)

---

## Architecture

### Event Flow

```
┌──────────────┐     POST /checkout      ┌──────────────┐
│   Frontend   │ ──────────────────────▶│ Cart Service │
└──────────────┘                         └──────┬───────┘
                                                │
                    ┌───────────────────────────┘
                    ▼
          ┌─────────────────────┐
          │  Cart.Checkout()     │
          │  - Validates cart    │
          │  - Emits event       │
          └─────────────────────┘
                    │
                    ▼
          ┌─────────────────────┐
          │  Outbox → Kafka     │
          │  cart.checked_out   │
          └─────────────────────┘
                    │                    ┌─────────────────────┐
                    │                    │  Order Service      │
                    │                    │  (future)           │
                    │                    └──────────┬──────────┘
                    │                               │
                    │                    ┌──────────▼──────────┐
                    │                    │  order.created      │
                    │                    │  (contains          │
                    │                    │   order_number)     │
                    │                    └──────────┬──────────┘
                    │                               │
                    │         ┌─────────────────────┴─────────────┐
                    │         ▼                                   ▼
                    │  ┌─────────────────────┐    ┌─────────────────────────┐
                    │  │ on_order_created    │    │ SSE Subscriber         │
                    │  │ Event Handler       │───▶│ (in-memory map)        │
                    │  └─────────────────────┘    │ - Gets cartID from     │
                    │         │                   │   order event          │
                    │         │                   │ - Pushes to connected  │
                    │         ▼                   │   clients              │
                    │  ┌─────────────────────┐    └─────────────────────────┘
                    │  │ Update cart to      │              │
                    │  │ "completed" status  │              │
                    │  └─────────────────────┘              │
                    │                                       ▼
                    │                         ┌─────────────────────────┐
                    │                         │  GET /carts/{id}/stream │
                    │                         │  (SSE connection)        │
                    │                         └─────────────────────────┘
                    │                                       │
┌──────────────┐    │                         ┌─────────────────────────┐
│   Frontend   │◀──┘                         │  event: order.created  │
│  (receives)  │                             │  data: {orderNumber,    │
└──────────────┘                             │         orderId,        │
                                              │         cartId}         │
                                              └─────────────────────────┘
```

---

## Implementation Approach

### Approaches Considered

1. **WebSockets** - More complex, requires bidirectional communication, harder to scale
2. **Polling** - Simple but inefficient, creates unnecessary HTTP requests
3. **SSE (Chosen)** - Lightweight, server-to-client only, built-in reconnection, works over HTTP/1.1

### Why SSE is the Best Fit

| Factor | SSE | WebSockets | Polling |
|--------|-----|------------|---------|
| Complexity | Low | High | Low |
| Server-to-client | Native | Bidirectional | No |
| Reconnection | Automatic | Manual | N/A |
| Works over HTTP/1.1 | Yes | No (requires upgrade) | Yes |
| Firewall friendly | Yes | Sometimes | Yes |
| Fits saga pattern | ✓ (server pushes) | ✓ | ✗ (client polls) |

---

## Implementation Steps

### Step 1: Create SSE Package

**File:** `internal/platform/sse/`

Create a new package for SSE connection management following the provider pattern used throughout the codebase.

#### 1.1 SSE Hub (hub.go)

```go
// internal/platform/sse/hub.go
package sse

import (
	"log"
	"sync"
)

// Hub manages SSE client subscriptions for a given cart ID
type Hub struct {
	// Map of cartID -> set of clients subscribed to that cart
	subscribers map[string]map[*Client]bool
	mu          sync.RWMutex
}

// NewHub creates a new SSE hub
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]map[*Client]bool),
	}
}

// Subscribe adds a client to the subscription list for a cart
func (h *Hub) Subscribe(cartID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.subscribers[cartID] == nil {
		h.subscribers[cartID] = make(map[*Client]bool)
	}
	h.subscribers[cartID][client] = true
	log.Printf("[INFO] Cart: SSE client subscribed to cart %s", cartID)
}

// Unsubscribe removes a client from the subscription list
func (h *Hub) Unsubscribe(cartID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if clients, ok := h.subscribers[cartID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			log.Printf("[INFO] Cart: SSE client unsubscribed from cart %s", cartID)
		}
		if len(clients) == 0 {
			delete(h.subscribers, cartID)
		}
	}
}

// Publish sends an event to all subscribers of a cart
func (h *Hub) Publish(cartID string, event string, data interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	clients, ok := h.subscribers[cartID]
	if !ok {
		log.Printf("[DEBUG] Cart: No SSE subscribers for cart %s", cartID)
		return
	}
	
	for client := range clients {
		select {
		case client.send <- Message{Event: event, Data: data}:
			log.Printf("[DEBUG] Cart: Published SSE event %s to cart %s", event, cartID)
		default:
			log.Printf("[WARN] Cart: SSE client buffer full, removing")
			delete(clients, client)
		}
	}
}
```

#### 1.2 SSE Client (client.go)

```go
// internal/platform/sse/client.go
package sse

// Client represents a single SSE client connection
type Client struct {
	hub    *Hub
	cartID string
	send   chan Message
	done   chan struct{}
}

// Message represents an SSE message to be sent
type Message struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// NewClient creates a new SSE client
func NewClient(hub *Hub, cartID string) *Client {
	return &Client{
		hub:    hub,
		cartID: cartID,
		send:   make(chan Message, 256),
		done:   make(chan struct{}),
	}
}

// Close signals the client to stop
func (c *Client) Close() {
	close(c.done)
}
```

#### 1.3 SSE Provider (provider.go)

**Aligned with codebase provider pattern**

```go
// internal/platform/sse/provider.go
package sse

// Provider provides SSE hub and handler instances
type Provider struct {
	hub     *Hub
	handler *Handler
}

// NewProvider creates a new SSE provider
func NewProvider() *Provider {
	hub := NewHub()
	handler := NewHandler(hub)
	
	return &Provider{
		hub:     hub,
		handler: handler,
	}
}

// GetHub returns the SSE hub for event handlers
func (p *Provider) GetHub() *Hub {
	return p.hub
}

// GetHandler returns the SSE HTTP handler
func (p *Provider) GetHandler() *Handler {
	return p.handler
}
```

#### 1.4 SSE Handler (handler.go)

**Aligned with codebase patterns**

```go
// internal/platform/sse/handler.go
package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	
	"github.com/go-chi/chi/v5"
	"github.com/go-shopping-poc/internal/platform/errors"
)

// Handler handles SSE HTTP connections
type Handler struct {
	hub *Hub
}

// Verify interface compliance
var _ http.Handler = (*Handler)(nil)

// NewHandler creates a new SSE handler
func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub: hub,
	}
}

// ServeHTTP implements http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract cart ID from URL using chi router
	cartID := chi.URLParam(r, "id")
	if cartID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
		return
	}
	
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
	
	// Create client and subscribe
	client := NewClient(h.hub, cartID)
	h.hub.Subscribe(cartID, client)
	
	// Ensure cleanup on disconnect
	defer func() {
		h.hub.Unsubscribe(cartID, client)
		client.Close()
	}()
	
	// Handle client close
	notify := r.Context().Done()
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("[ERROR] Cart: Streaming not supported")
		return
	}
	
	// Send initial connection message
	fmt.Fprintf(w, "event: connected\ndata: {\"cartId\":\"%s\"}\n\n", cartID)
	flusher.Flush()
	
	// Keep connection alive and send events
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-notify:
			// Client disconnected
			log.Printf("[INFO] Cart: SSE client disconnected for cart %s", cartID)
			return
			
		case <-ticker.C:
			// Send heartbeat to keep connection alive
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
			
		case msg, ok := <-client.send:
			if !ok {
				// Channel closed
				return
			}
			
			dataBytes, _ := json.Marshal(msg.Data)
			
			if msg.Event != "" {
				fmt.Fprintf(w, "event: %s\n", msg.Event)
			}
			fmt.Fprintf(w, "data: %s\n\n", dataBytes)
			flusher.Flush()
		}
	}
}
```

---

### Step 2: Modify Order Created Event Handler

#### 2.1 Update on_order_created.go

**File:** `internal/service/cart/eventhandlers/on_order_created.go`

Update the event handler to implement the existing factory pattern and include SSE hub dependency:

```go
// internal/service/cart/eventhandlers/on_order_created.go
package eventhandlers

import (
	"context"
	"log"
	
	"github.com/go-shopping-poc/internal/contracts/events"
	"github.com/go-shopping-poc/internal/platform/event/handler"
	"github.com/go-shopping-poc/internal/platform/sse"
)

// OnOrderCreated handles order.created events and publishes SSE notifications
type OnOrderCreated struct {
	sseHub *sse.Hub
}

// Verify interface compliance
var _ handler.EventHandler = (*OnOrderCreated)(nil)
var _ handler.HandlerFactory[events.OrderEvent] = (*OnOrderCreated)(nil)

// NewOnOrderCreated creates a new order created handler
func NewOnOrderCreated(sseHub *sse.Hub) *OnOrderCreated {
	return &OnOrderCreated{
		sseHub: sseHub,
	}
}

// Handle processes the order.created event
func (h *OnOrderCreated) Handle(ctx context.Context, event events.Event) error {
	orderEvent, ok := event.(events.OrderEvent)
	if !ok {
		log.Printf("[ERROR] Cart: Expected OrderEvent, got %T", event)
		return nil
	}
	
	if orderEvent.EventType != events.OrderCreated {
		log.Printf("[DEBUG] Cart: Ignoring event type: %s", orderEvent.EventType)
		return nil
	}
	
	utils := handler.NewEventUtils()
	utils.LogEventProcessing(ctx, string(orderEvent.EventType),
		orderEvent.Data.OrderID,
		orderEvent.Data.CartID)
	
	// Push SSE event to subscribers
	if h.sseHub != nil {
		h.sseHub.Publish(
			orderEvent.Data.CartID,
			"order.created",
			map[string]interface{}{
				"orderId":     orderEvent.Data.OrderID,
				"orderNumber": orderEvent.Data.OrderNumber,
				"cartId":      orderEvent.Data.CartID,
				"total":       orderEvent.Data.Total,
			},
		)
	}
	
	return h.updateCartStatus(ctx, orderEvent.Data.CartID)
}

// EventType returns the event type this handler processes
func (h *OnOrderCreated) EventType() string {
	return string(events.OrderCreated)
}

// CreateFactory returns the event factory for this handler
func (h *OnOrderCreated) CreateFactory() events.EventFactory[events.OrderEvent] {
	return events.OrderEventFactory{}
}

// CreateHandler returns the handler function for this handler
func (h *OnOrderCreated) CreateHandler() handler.EventHandler {
	return h
}

func (h *OnOrderCreated) updateCartStatus(ctx context.Context, cartID string) error {
	// Existing implementation
	return nil
}
```

#### 2.2 Add OrderNumber to Event Payload

**File:** `internal/contracts/events/order.go`

Update `OrderEventPayload` to include `OrderNumber`:

```go
// internal/contracts/events/order.go
type OrderEventPayload struct {
	OrderID     string  `json:"order_id"`
	OrderNumber string  `json:"order_number"` // Add this field
	CartID      string  `json:"cart_id"`
	CustomerID  *string `json:"customer_id,omitempty"`
	Total       float64 `json:"total"`
}
```

---

### Step 3: Wire Everything Together

#### 3.1 Update Cart Infrastructure

**File:** `internal/service/cart/config.go`

Add SSE provider to the cart infrastructure:

```go
// internal/service/cart/config.go

// CartInfrastructure holds all infrastructure dependencies
type CartInfrastructure struct {
	db              database.Database
	eventBus        bus.Bus
	outboxWriter    *outbox.OutboxWriter
	outboxPublisher *outbox.OutboxPublisher
	productClient   *ProductClient
	corsHandler     http.Handler
	sseProvider     *sse.Provider // Add SSE provider
}

// NewCartInfrastructure creates infrastructure dependencies
func NewCartInfrastructure(
	db database.Database,
	eventBus bus.Bus,
	outboxWriter *outbox.OutboxWriter,
	outboxPublisher *outbox.OutboxPublisher,
	productClient *ProductClient,
	corsHandler http.Handler,
	sseProvider *sse.Provider, // Add parameter
) *CartInfrastructure {
	return &CartInfrastructure{
		db:              db,
		eventBus:        eventBus,
		outboxWriter:    outboxWriter,
		outboxPublisher: outboxPublisher,
		productClient:   productClient,
		corsHandler:     corsHandler,
		sseProvider:     sseProvider,
	}
}

// GetSSEHub returns the SSE hub for event handlers
func (i *CartInfrastructure) GetSSEHub() *sse.Hub {
	return i.sseProvider.GetHub()
}
```

#### 3.2 Update Main.go

**File:** `cmd/cart/main.go`

```go
// cmd/cart/main.go

func main() {
	// ... existing initialization code ...
	
	// Initialize SSE provider
	sseProvider := sse.NewProvider()
	
	// Create cart infrastructure with SSE provider
	infrastructure := cart.NewCartInfrastructure(
		db,
		eventBus,
		outboxWriter,
		outboxPublisher,
		productClient,
		corsHandler,
		sseProvider,
	)
	
	// Create cart service
	service := cart.NewCartService(infrastructure, cfg)
	
	// Register event handlers with SSE hub
	if err := registerEventHandlers(service, infrastructure.GetSSEHub()); err != nil {
		log.Fatalf("Cart: Failed to register event handlers: %v", err)
	}
	
	// Create cart HTTP handler
	cartHandler := cart.NewCartHandler(service)
	
	// Setup router
	cartRouter := chi.NewRouter()
	cartRouter.Use(corsHandler.Handler)
	
	// Register existing cart routes
	cartRouter.Get("/carts/{id}", cartHandler.GetCart)
	cartRouter.Post("/carts", cartHandler.CreateCart)
	cartRouter.Post("/carts/{id}/items", cartHandler.AddItem)
	cartRouter.Post("/carts/{id}/checkout", cartHandler.Checkout)
	// ... other routes ...
	
	// Register SSE route separately (doesn't go through CartHandler)
	cartRouter.Get("/carts/{id}/stream", sseProvider.GetHandler().ServeHTTP)
	
	// ... rest of main function ...
}

// registerEventHandlers registers all event handlers for the cart service
func registerEventHandlers(service *cart.CartService, sseHub *sse.Hub) error {
	// Create event handler with SSE hub dependency
	orderCreatedHandler := eventhandlers.NewOnOrderCreated(sseHub)
	
	// Register using the generic handler registration pattern
	return cart.RegisterHandler(
		service,
		orderCreatedHandler.CreateFactory(),
		orderCreatedHandler.CreateHandler(),
	)
}
```

---

## API Specification

### SSE Endpoint

**GET** `/api/v1/carts/{id}/stream`

**Headers:**
```
Accept: text/event-stream
Cache-Control: no-cache
```

**Response (SSE stream):**

```
event: connected
data: {"cartId":"uuid-here"}

event: order.created
data: {"orderId":"uuid","orderNumber":"ORD-123","cartId":"uuid","total":99.99}
```

**Possible Events:**

| Event | Description |
|-------|-------------|
| `connected` | Sent on initial connection |
| `order.created` | Order has been created from cart |
| `order.failed` | Order creation failed (optional) |

### Frontend Usage

```javascript
// Open SSE connection
const cartId = 'cart-uuid-from-checkout-response';
const eventSource = new EventSource(`/api/v1/carts/${cartId}/stream`);

eventSource.addEventListener('connected', (e) => {
  console.log('Connected to cart stream:', JSON.parse(e.data));
});

eventSource.addEventListener('order.created', (e) => {
  const order = JSON.parse(e.data);
  console.log('Order created:', order);
  
  // Redirect to order confirmation page
  window.location.href = `/order-confirmation/${order.orderNumber}`;
  
  // Close connection
  eventSource.close();
});

eventSource.addEventListener('error', (e) => {
  console.error('SSE error:', e);
  // Fall back to polling or show error
});
```

---

## Key Alignment Improvements Made

### 1. Package Naming
- Changed `NewSSEHandler()` to `NewHandler()` following codebase convention

### 2. Error Handling
- Uses `platform/errors` package with `errors.SendError()` instead of `http.Error()`

### 3. Handler Design
- CartHandler remains clean, only holding service reference
- SSE route registered directly with SSE handler (not through CartHandler)

### 4. Event Handler Pattern
- Implements `handler.EventHandler` and `handler.HandlerFactory[events.OrderEvent]` interfaces
- Includes explicit interface verification: `var _ handler.EventHandler = (*OnOrderCreated)(nil)`

### 5. Provider Pattern
- Created `sse.Provider` matching other platform providers (database, event, etc.)
- Added SSE provider to `CartInfrastructure` struct

### 6. Logging Format
- Aligned with codebase: `[INFO] Cart: message` instead of `[INFO] SSE: message`

### 7. Route Registration
- SSE endpoint registered directly: `cartRouter.Get("/carts/{id}/stream", sseProvider.GetHandler().ServeHTTP)`

---

## Testing Checklist

### Unit Tests
- [ ] Test SSE Hub subscribe/unsubscribe
- [ ] Test SSE Hub publish to multiple clients
- [ ] Test SSE Handler missing cart ID (uses errors package)
- [ ] Test event handler factory pattern implementation
- [ ] Test event handler interface compliance

### Integration Tests
- [ ] Test checkout triggers SSE event
- [ ] Test SSE reconnection with Last-Event-ID
- [ ] Test SSE connection timeout handling
- [ ] Test multiple clients for same cart

### Manual Testing
- [ ] Start cart service
- [ ] Call checkout endpoint
- [ ] Connect to SSE stream
- [ ] Verify order.created event received with order number

---

## Error Handling

### Connection Errors
- If SSE connection fails, frontend should fall back to polling `GET /api/v1/carts/{id}` until successful connection

### Timeout Handling
- SSE connections have a 30-second heartbeat to prevent timeout
- Frontend should implement reconnection logic

### Race Conditions
- If order.created event fires before frontend connects, the event is lost
- Frontend should poll immediately after checkout to verify if order already exists

---

## Future Enhancements (Out of Scope)

1. **Multi-instance deployment** - Replace in-memory map with Redis pub/sub
2. **Reconnection support** - Implement Last-Event-ID header
3. **Order failure events** - Push error events when order creation fails
4. **Authentication** - Add JWT validation to SSE endpoint

---

## File Summary

### New Files

| File | Description |
|------|-------------|
| `internal/platform/sse/hub.go` | Manages SSE client subscriptions |
| `internal/platform/sse/client.go` | Represents a single SSE client connection |
| `internal/platform/sse/handler.go` | HTTP handler for SSE streams (implements http.Handler) |
| `internal/platform/sse/provider.go` | Provider for SSE hub and handler instances |

### Modified Files

| File | Changes |
|------|---------|
| `internal/contracts/events/order.go` | Add OrderNumber to payload |
| `internal/service/cart/eventhandlers/on_order_created.go` | Add SSE hub dependency, implement factory pattern |
| `internal/service/cart/config.go` | Add sseProvider to CartInfrastructure |
| `cmd/cart/main.go` | Initialize SSE provider, register route, wire dependencies |

---

## Implementation Order

1. Create `internal/platform/sse/` package (hub, client, handler, provider)
2. Update `internal/contracts/events/order.go` to add OrderNumber
3. Update `internal/service/cart/eventhandlers/on_order_created.go` to use SSE hub
4. Update `internal/service/cart/config.go` to include SSE provider
5. Update `cmd/cart/main.go` to wire everything together
6. Write tests
7. Manual testing
