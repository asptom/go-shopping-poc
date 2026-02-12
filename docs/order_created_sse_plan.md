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

Create a new package for SSE connection management.

#### 1.1 SSE Hub (hub.go)

```go
// internal/platform/sse/hub.go
package sse

import (
    "log"
    "sync"
)

type Hub struct {
    // Map of cartID -> set of clients subscribed to that cart
    subscribers map[string]map[*Client]bool
    
    // Register channel for new clients
    register chan *Client
    
    // Unregister channel for disconnected clients
    unregister chan *Client
    
    // Mutex for thread-safe operations
    mu sync.RWMutex
}

func NewHub() *Hub {
    return &Hub{
        subscribers: make(map[string]map[*Client]bool),
        register:    make(chan *Client),
        unregister:  make(chan *Client),
    }
}

func (h *Hub) Subscribe(cartID string, client *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    if h.subscribers[cartID] == nil {
        h.subscribers[cartID] = make(map[*Client]bool)
    }
    h.subscribers[cartID][client] = true
    log.Printf("[INFO] SSE: Client subscribed to cart %s", cartID)
}

func (h *Hub) Unsubscribe(cartID string, client *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    if clients, ok := h.subscribers[cartID]; ok {
        if _, exists := clients[client]; exists {
            delete(clients, client)
            log.Printf("[INFO] SSE: Client unsubscribed from cart %s", cartID)
        }
        if len(clients) == 0 {
            delete(h.subscribers, cartID)
        }
    }
}

func (h *Hub) Publish(cartID string, event string, data interface{}) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    clients, ok := h.subscribers[cartID]
    if !ok {
        log.Printf("[DEBUG] SSE: No subscribers for cart %s", cartID)
        return
    }
    
    for client := range clients {
        select {
        case client.send <- Message{Event: event, Data: data}:
            log.Printf("[DEBUG] SSE: Published event %s to cart %s", event, cartID)
        default:
            log.Printf("[WARN] SSE: Failed to send to client, removing")
            delete(clients, client)
        }
    }
}
```

#### 1.2 SSE Client (client.go)

```go
// internal/platform/sse/client.go
package sse

import (
    "encoding/json"
    "time"
    
    "github.com/gorilla/websocket"
)

type Client struct {
    hub     *Hub
    cartID  string
    send    chan Message
    done    chan struct{}
}

type Message struct {
    Event string      `json:"event"`
    Data  interface{} `json:"data"`
}

func NewClient(hub *Hub, cartID string) *Client {
    return &Client{
        hub:    hub,
        cartID: cartID,
        send:   make(chan Message, 256),
        done:   make(chan struct{}),
    }
}

// SendEvent formats and sends an SSE event
func (c *Client) SendEvent(event string, data interface{}) error {
    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }
    
    // SSE format: event: <event>\ndata: <json>\n\n
    select {
    case c.send <- Message{Event: event, Data: jsonData}:
        return nil
    case <-c.done:
        return nil // Client disconnected
    case <-time.After(5 * time.Second):
        return nil // Timeout
    }
}

// Close signals the client to stop
func (c *Client) Close() {
    close(c.done)
}
```

#### 1.3 SSE Handler (handler.go)

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
)

type SSEHandler struct {
    hub *Hub
}

func NewSSEHandler() *SSEHandler {
    return &SSEHandler{
        hub: NewHub(),
    }
}

// GetHub returns the SSE hub for external use (e.g., from event handlers)
func (h *SSEHandler) GetHub() *Hub {
    return h.hub
}

// Stream handles SSE connections
func (h *SSEHandler) Stream(w http.ResponseWriter, r *http.Request) {
    cartID := chi.URLParam(r, "id")
    if cartID == "" {
        http.Error(w, "Missing cart ID", http.StatusBadRequest)
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
        log.Printf("[ERROR] SSE: Streaming not supported")
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
            log.Printf("[INFO] SSE: Client disconnected for cart %s", cartID)
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

### Step 2: Integrate SSE Handler into Cart Service

#### 2.1 Add SSE Handler to Cart Service

**File:** `internal/service/cart/handlers.go`

Add the SSE handler as a dependency:

```go
type CartHandler struct {
    service     *CartService
    sseHandler  *sse.SSEHandler  // Add this
}
```

#### 2.2 Update Handler Constructor

**File:** `internal/service/cart/handlers.go`

```go
func NewCartHandler(service *CartService, sseHandler *sse.SSEHandler) *CartHandler {
    return &CartHandler{
        service:    service,
        sseHandler: sseHandler,
    }
}
```

#### 2.3 Register SSE Route

**File:** `cmd/cart/main.go`

```go
cartRouter.Get("/carts/{id}/stream", handler.StreamCartEvents)
```

---

### Step 3: Modify Order Created Event Handler

#### 3.1 Update on_order_created.go to Push SSE Event

**File:** `internal/service/cart/eventhandlers/on_order_created.go`

Add the SSE hub as a dependency and publish the order created event:

```go
type OnOrderCreated struct {
    sseHub *sse.Hub  // Add this dependency
}

func NewOnOrderCreated(sseHub *sse.Hub) *OnOrderCreated {
    return &OnOrderCreated{
        sseHub: sseHub,
    }
}

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
                "orderId":    orderEvent.Data.OrderID,
                "orderNumber": orderEvent.Data.OrderNumber,  // Add to event payload
                "cartId":     orderEvent.Data.CartID,
                "total":      orderEvent.Data.Total,
            },
        )
    }
    
    return h.updateCartStatus(ctx, orderEvent.Data.CartID)
}
```

#### 3.2 Add OrderNumber to Event Payload

**File:** `internal/contracts/events/order.go`

Update `OrderEventPayload` to include `OrderNumber`:

```go
type OrderEventPayload struct {
    OrderID      string  `json:"order_id"`
    OrderNumber  string  `json:"order_number"`  // Add this field
    CartID       string  `json:"cart_id"`
    CustomerID   *string `json:"customer_id,omitempty"`
    Total        float64 `json:"total"`
}
```

---

### Step 4: Wire Everything Together

#### 4.1 Update Main.go

**File:** `cmd/cart/main.go`

```go
// Initialize SSE handler
sseHandler := sse.NewSSEHandler()

// Create event handler with SSE hub
orderCreatedHandler := eventhandlers.NewOnOrderCreated(sseHandler.GetHub())

// Register handlers
if err := registerEventHandlers(service, orderCreatedHandler); err != nil {
    log.Fatalf("Cart: Failed to register event handlers: %v", err)
}

// Create cart handler with SSE
cartHandler := cart.NewCartHandler(service, sseHandler)

// Register routes (add SSE route)
cartRouter.Get("/carts/{id}/stream", cartHandler.StreamCartEvents)
```

#### 4.2 Update registerEventHandlers Function

**File:** `cmd/cart/main.go`

```go
func registerEventHandlers(service *cart.CartService, orderHandler *eventhandlers.OnOrderCreated) error {
    // Subscribe to OrderEvents topic and register the handler
    // ... existing code ...
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
| `error` | Connection error |

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

## Testing Checklist

### Unit Tests
- [ ] Test SSE Hub subscribe/unsubscribe
- [ ] Test SSE Hub publish to multiple clients
- [ ] Test SSE Handler missing cart ID
- [ ] Test event handler integration

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
- SSE connections should have a 30-second heartbeat to prevent timeout
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
| `internal/platform/sse/handler.go` | HTTP handler for SSE streams |

### Modified Files

| File | Changes |
|------|---------|
| `internal/contracts/events/order.go` | Add OrderNumber to payload |
| `internal/service/cart/eventhandlers/on_order_created.go` | Add SSE hub dependency, publish events |
| `internal/service/cart/handlers.go` | Add SSE handler, new StreamCartEvents method |
| `cmd/cart/main.go` | Initialize SSE, register route, wire dependencies |

---

## Implementation Order

1. Create `internal/platform/sse/` package (hub, client, handler)
2. Update `internal/contracts/events/order.go` to add OrderNumber
3. Update `internal/service/cart/eventhandlers/on_order_created.go` to use SSE hub
4. Update `internal/service/cart/handlers.go` to add SSE handler and StreamCartEvents
5. Update `cmd/cart/main.go` to wire everything together
6. Write tests
7. Manual testing
