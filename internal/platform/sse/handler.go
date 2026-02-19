package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go-shopping-poc/internal/platform/errors"
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

	log.Printf("[DEBUG] SSE: ========== NEW CONNECTION ==========")
	log.Printf("[DEBUG] SSE: New connection request for cart %s from %s", cartID, r.RemoteAddr)
	log.Printf("[DEBUG] SSE: Request headers: %v", r.Header)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
	w.WriteHeader(http.StatusOK)

	// Create client and subscribe
	client := NewClient(h.hub, cartID)
	h.hub.Subscribe(cartID, client)
	log.Printf("[DEBUG] SSE: Client subscribed to cart %s - total subscribers: %d", cartID, h.hub.GetSubscriberCount(cartID))

	// Ensure cleanup on disconnect
	defer func() {
		log.Printf("[DEBUG] SSE: ========== CONNECTION CLOSING ==========")
		log.Printf("[DEBUG] SSE: Cleaning up connection for cart %s", cartID)
		h.hub.Unsubscribe(cartID, client)
		client.Close()
		log.Printf("[DEBUG] SSE: Connection cleanup complete for cart %s", cartID)
	}()

	// Handle client close
	notify := r.Context().Done()

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("[ERROR] SSE: Streaming not supported by ResponseWriter")
		return
	}

	// Send initial connection message
	log.Printf("[DEBUG] SSE: Sending initial 'connected' event to cart %s", cartID)
	fmt.Fprintf(w, "event: connected\ndata: {\"cartId\":\"%s\",\"status\":\"connected\"}\n\n", cartID)
	flusher.Flush()
	log.Printf("[DEBUG] SSE: Initial event sent, entering event loop for cart %s", cartID)

	// Keep connection alive and send events
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	loopCount := 0
	for {
		loopCount++
		log.Printf("[DEBUG] SSE: Event loop iteration #%d for cart %s", loopCount, cartID)

		select {
		case <-notify:
			// Client disconnected
			log.Printf("[INFO] SSE: Client disconnected via context.Done() for cart %s", cartID)
			return

		case <-ticker.C:
			// Send heartbeat to keep connection alive
			log.Printf("[DEBUG] SSE: Sending heartbeat to cart %s (iteration #%d)", cartID, loopCount)
			fmt.Fprintf(w, ": ping %d\n\n", loopCount)
			flusher.Flush()
			log.Printf("[DEBUG] SSE: Heartbeat sent to cart %s", cartID)

		case msg, ok := <-client.send:
			log.Printf("[DEBUG] SSE: Received message from client.send channel for cart %s (ok=%v)", cartID, ok)
			if !ok {
				// Channel closed
				log.Printf("[WARN] SSE: client.send channel closed for cart %s", cartID)
				return
			}

			dataBytes, err := json.Marshal(msg.Data)
			if err != nil {
				log.Printf("[ERROR] SSE: Failed to marshal message data: %v", err)
				continue
			}

			if msg.Event != "" {
				fmt.Fprintf(w, "event: %s\n", msg.Event)
			}
			fmt.Fprintf(w, "data: %s\n\n", dataBytes)
			flusher.Flush()
			log.Printf("[DEBUG] SSE: Sent event '%s' with data to cart %s", msg.Event, cartID)
		}
	}
}
