package sse

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go-shopping-poc/internal/platform/errors"

	"github.com/go-chi/chi/v5"
)

// HandlerOption configures handler behavior.
type HandlerOption func(*HandlerConfig)

// HandlerConfig controls stream extraction and response semantics.
type HandlerConfig struct {
	PathParam        string
	MissingIDMsg     string
	ConnectedIDField string
	LogIDKey         string
}

func defaultHandlerConfig() HandlerConfig {
	return HandlerConfig{
		PathParam:        "id",
		MissingIDMsg:     "Missing stream ID",
		ConnectedIDField: "streamId",
		LogIDKey:         "stream_id",
	}
}

// WithPathParam sets the URL path parameter name used as stream ID.
func WithPathParam(name string) HandlerOption {
	return func(cfg *HandlerConfig) {
		if name != "" {
			cfg.PathParam = name
		}
	}
}

// WithMissingIDMessage sets the error message returned when stream ID is absent.
func WithMissingIDMessage(message string) HandlerOption {
	return func(cfg *HandlerConfig) {
		if message != "" {
			cfg.MissingIDMsg = message
		}
	}
}

// WithConnectedIDField sets the JSON key for the initial connected event payload.
func WithConnectedIDField(field string) HandlerOption {
	return func(cfg *HandlerConfig) {
		if field != "" {
			cfg.ConnectedIDField = field
		}
	}
}

// WithLogIDKey sets the structured logging key for the stream identifier.
func WithLogIDKey(key string) HandlerOption {
	return func(cfg *HandlerConfig) {
		if key != "" {
			cfg.LogIDKey = key
		}
	}
}

// Handler handles SSE HTTP connections
type Handler struct {
	hub    *Hub
	logger *slog.Logger
	cfg    HandlerConfig
}

// Verify interface compliance
var _ http.Handler = (*Handler)(nil)

// NewHandler creates a new SSE handler.
func NewHandler(hub *Hub, opts ...HandlerOption) *Handler {
	cfg := defaultHandlerConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Handler{
		hub:    hub,
		logger: Logger().With("component", "sse_handler"),
		cfg:    cfg,
	}
}

// ServeHTTP implements http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, h.cfg.PathParam)
	if streamID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, h.cfg.MissingIDMsg)
		return
	}
	startedAt := time.Now()
	requestID := requestIDFromRequest(r)
	log := h.logger.With("operation", "stream_events", "request_id", requestID, h.cfg.LogIDKey, streamID)

	log.Debug("SSE connection opened", "remote_addr", r.RemoteAddr)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
	w.WriteHeader(http.StatusOK)

	// Create client and subscribe
	client := NewClient(h.hub, streamID)
	h.hub.Subscribe(streamID, client)
	log.Debug("SSE client subscribed", "subscriber_count", h.hub.GetSubscriberCount(streamID))

	// Ensure cleanup on disconnect
	defer func() {
		log.Debug("SSE connection closing")
		h.hub.Unsubscribe(streamID, client)
		client.Close()
		log.Info("SSE connection closed", "duration_ms", time.Since(startedAt).Milliseconds())
	}()

	// Handle client close
	notify := r.Context().Done()

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Error("SSE streaming unsupported")
		return
	}

	// Send initial connection message
	log.Debug("Send initial SSE event")
	initialData, _ := json.Marshal(map[string]string{
		h.cfg.ConnectedIDField: streamID,
		"status":               "connected",
	})
	fmt.Fprintf(w, "event: connected\ndata: %s\n\n", initialData)
	flusher.Flush()
	log.Debug("Initial SSE event sent")

	// Keep connection alive and send events
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	loopCount := 0
	for {
		loopCount++

		select {
		case <-notify:
			// Client disconnected
			log.Info("SSE client disconnected", "duration_ms", time.Since(startedAt).Milliseconds())
			return

		case <-ticker.C:
			// Send heartbeat to keep connection alive
			log.Debug("Send SSE heartbeat", "iteration", loopCount)
			fmt.Fprintf(w, ": ping %d\n\n", loopCount)
			flusher.Flush()

		case msg, ok := <-client.send:
			if !ok {
				// Channel closed
				log.Warn("SSE client channel closed")
				return
			}

			dataBytes, err := json.Marshal(msg.Data)
			if err != nil {
				log.Error("Marshal SSE message failed", "error", err.Error())
				continue
			}

			if msg.Event != "" {
				fmt.Fprintf(w, "event: %s\n", msg.Event)
			}
			fmt.Fprintf(w, "data: %s\n\n", dataBytes)
			flusher.Flush()
			log.Debug("SSE event sent", "event_type", msg.Event)
		}
	}
}

func requestIDFromRequest(r *http.Request) string {
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		return requestID
	}
	if requestID := r.Header.Get("X-Request-Id"); requestID != "" {
		return requestID
	}

	return ""
}
