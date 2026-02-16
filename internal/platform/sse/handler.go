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
