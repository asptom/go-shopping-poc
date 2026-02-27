package sse

import (
	"log/slog"
	"sync"
)

// Hub manages SSE client subscriptions for a given cart ID
type Hub struct {
	// Map of cartID -> set of clients subscribed to that cart
	subscribers map[string]map[*Client]bool
	mu          sync.RWMutex
	logger      *slog.Logger
}

// NewHub creates a new SSE hub
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]map[*Client]bool),
		logger:      Logger(),
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
	h.logger.Info("SSE: Client subscribed to cart", "cartID", cartID)
}

// Unsubscribe removes a client from the subscription list
func (h *Hub) Unsubscribe(cartID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.subscribers[cartID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			h.logger.Info("SSE: Client unsubscribed from cart", "cartID", cartID)
		}
		if len(clients) == 0 {
			delete(h.subscribers, cartID)
		}
	}
}

// Publish sends an event to all subscribers of a cart
func (h *Hub) Publish(cartID string, event string, data interface{}) {
	h.logger.Debug("SSE: ========== PUBLISH REQUEST ==========", "cartID", cartID, "event", event)
	h.logger.Debug("SSE: Publish called for cart", "cartID", cartID, "event", event)

	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.subscribers[cartID]
	if !ok {
		h.logger.Debug("SSE: No subscribers found for cart", "cartID", cartID, "event", event)
		h.logger.Debug("SSE: Available carts with subscribers", "carts", h.getSubscriberCartIDs())
		return
	}

	h.logger.Debug("SSE: Found subscribers for cart", "cartID", cartID, "count", len(clients))

	sentCount := 0
	for client := range clients {
		h.logger.Debug("SSE: Attempting to send to client for cart", "cartID", cartID)
		select {
		case client.send <- Message{Event: event, Data: data}:
			h.logger.Debug("SSE: Event successfully queued for cart", "event", event, "cartID", cartID)
			sentCount++
		default:
			h.logger.Warn("SSE: Client buffer full for cart, removing client", "cartID", cartID)
			delete(clients, client)
		}
	}
	h.logger.Debug("SSE: Publish complete - sent to %d/%d clients for cart %s", sentCount, len(clients), cartID)
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
