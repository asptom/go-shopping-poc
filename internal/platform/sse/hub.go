package sse

import (
	"log/slog"
	"sync"
)

// Hub manages SSE client subscriptions for a stream identifier.
type Hub struct {
	// Map of streamID -> set of clients subscribed to that stream.
	subscribers map[string]map[*Client]bool
	mu          sync.RWMutex
	logger      *slog.Logger
}

// NewHub creates a new SSE hub
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]map[*Client]bool),
		logger:      Logger().With("component", "sse_hub"),
	}
}

// Subscribe adds a client to the subscription list for a stream.
func (h *Hub) Subscribe(streamID string, client *Client) {
	log := h.logger.With("operation", "subscribe", "stream_id", streamID)
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.subscribers[streamID] == nil {
		h.subscribers[streamID] = make(map[*Client]bool)
	}
	h.subscribers[streamID][client] = true
	log.Info("SSE client subscribed")
}

// Unsubscribe removes a client from the subscription list
func (h *Hub) Unsubscribe(streamID string, client *Client) {
	log := h.logger.With("operation", "unsubscribe", "stream_id", streamID)
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.subscribers[streamID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			log.Info("SSE client unsubscribed")
		}
		if len(clients) == 0 {
			delete(h.subscribers, streamID)
		}
	}
}

// Publish sends an event to all subscribers of a stream.
func (h *Hub) Publish(streamID string, event string, data interface{}) {
	log := h.logger.With("operation", "publish", "stream_id", streamID, "event_type", event)
	log.Debug("SSE publish requested")

	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.subscribers[streamID]
	if !ok {
		log.Debug("No stream subscribers")
		return
	}

	log.Debug("Stream subscribers found", "subscriber_count", len(clients))

	sentCount := 0
	for client := range clients {
		select {
		case client.send <- Message{Event: event, Data: data}:
			sentCount++
		default:
			log.Warn("SSE client buffer full", "status", "removed_client")
			delete(clients, client)
		}
	}
	log.Debug("SSE publish complete", "sent_count", sentCount, "subscriber_count", len(clients))
}

// getSubscriberStreamIDs returns a list of stream IDs that have subscribers.
func (h *Hub) getSubscriberStreamIDs() []string {
	var ids []string
	for id := range h.subscribers {
		ids = append(ids, id)
	}
	return ids
}

// GetSubscriberCount returns the number of subscribers for a stream.
func (h *Hub) GetSubscriberCount(streamID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.subscribers[streamID]; ok {
		return len(clients)
	}
	return 0
}
