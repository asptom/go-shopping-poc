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
