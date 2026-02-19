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
	log.Printf("[DEBUG] SSE: ========== PUBLISH REQUEST ==========")
	log.Printf("[DEBUG] SSE: Publish called for cart %s, event '%s'", cartID, event)
	log.Printf("[DEBUG] SSE: Data type: %T", data)

	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.subscribers[cartID]
	if !ok {
		log.Printf("[DEBUG] SSE: No subscribers found for cart %s - event '%s' not sent", cartID, event)
		log.Printf("[DEBUG] SSE: Available carts with subscribers: %v", h.getSubscriberCartIDs())
		return
	}

	log.Printf("[DEBUG] SSE: Found %d subscriber(s) for cart %s", len(clients), cartID)

	sentCount := 0
	for client := range clients {
		log.Printf("[DEBUG] SSE: Attempting to send to client for cart %s", cartID)
		select {
		case client.send <- Message{Event: event, Data: data}:
			log.Printf("[DEBUG] SSE: Event '%s' successfully queued for cart %s", event, cartID)
			sentCount++
		default:
			log.Printf("[WARN] SSE: Client buffer full for cart %s, removing client", cartID)
			delete(clients, client)
		}
	}
	log.Printf("[DEBUG] SSE: Publish complete - sent to %d/%d clients for cart %s", sentCount, len(clients), cartID)
}

// getSubscriberCartIDs returns a list of cart IDs that have subscribers (for debugging)
func (h *Hub) getSubscriberCartIDs() []string {
	var ids []string
	for id := range h.subscribers {
		ids = append(ids, id)
	}
	return ids
}

// GetSubscriberCount returns the number of subscribers for a cart
func (h *Hub) GetSubscriberCount(cartID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.subscribers[cartID]; ok {
		return len(clients)
	}
	return 0
}
