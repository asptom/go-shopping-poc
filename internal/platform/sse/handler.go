package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go-shopping-poc/internal/platform/errors"

	"github.com/go-chi/chi/v5"
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

	logger.Debug("SSE: ========== NEW CONNECTION ==========")
	logger.Info("SSE: New connection request for cart", "cartID", cartID, "remoteAddr", r.RemoteAddr)
	logger.Debug("SSE: Request headers", "headers", r.Header)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
	w.WriteHeader(http.StatusOK)

	// Create client and subscribe
	client := NewClient(h.hub, cartID)
	h.hub.Subscribe(cartID, client)
	logger.Debug("SSE: Client subscribed to cart", "cartID", cartID, "totalSubscribers", h.hub.GetSubscriberCount(cartID))

	// Ensure cleanup on disconnect
	defer func() {
		logger.Debug("SSE: ========== CONNECTION CLOSING ==========")
		logger.Debug("SSE: Cleaning up connection for cart", "cartID", cartID)
		h.hub.Unsubscribe(cartID, client)
		client.Close()
		logger.Debug("SSE: Connection cleanup complete for cart", "cartID", cartID)
	}()

	// Handle client close
	notify := r.Context().Done()

	flusher, ok := w.(http.Flusher)
	if !ok {
		logger.Error("SSE: Streaming not supported by ResponseWriter")
		return
	}

	// Send initial connection message
	logger.Debug("SSE: Sending initial 'connected' event to cart", "cartID", cartID)
	fmt.Fprintf(w, "event: connected\ndata: {\"cartId\":\"%s\",\"status\":\"connected\"}\n\n", cartID)
	flusher.Flush()
	logger.Debug("SSE: Initial event sent, entering event loop for cart", "cartID", cartID)

	// Keep connection alive and send events
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	loopCount := 0
	for {
		loopCount++
		logger.Debug("SSE: Event loop iteration", "iteration", loopCount, "cartID", cartID)

		select {
		case <-notify:
			// Client disconnected
			logger.Info("SSE: Client disconnected via context.Done()", "cartID", cartID)
			return

		case <-ticker.C:
			// Send heartbeat to keep connection alive
			logger.Debug("SSE: Sending heartbeat to cart", "cartID", cartID, "iteration", loopCount)
			fmt.Fprintf(w, ": ping %d\n\n", loopCount)
			flusher.Flush()
			logger.Debug("SSE: Heartbeat sent to cart", "cartID", cartID)

		case msg, ok := <-client.send:
			logger.Debug("SSE: Received message from client.send channel", "cartID", cartID, "ok", ok)
			if !ok {
				// Channel closed
				logger.Warn("SSE: client.send channel closed for cart", "cartID", cartID)
				return
			}

			dataBytes, err := json.Marshal(msg.Data)
			if err != nil {
				logger.Error("SSE: Failed to marshal message data", "cartID", cartID, "error", err.Error())
				continue
			}

			if msg.Event != "" {
				fmt.Fprintf(w, "event: %s\n", msg.Event)
			}
			fmt.Fprintf(w, "data: %s\n\n", dataBytes)
			flusher.Flush()
			logger.Debug("SSE: Sent event", "event", msg.Event, "cartID", cartID)
		}
	}
}
