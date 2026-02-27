package sse

import "log/slog"

// Option is a functional option for configuring Provider.
type Option func(*Provider)

// WithLogger sets the logger for the Provider.
func WithLogger(logger *slog.Logger) Option {
	return func(p *Provider) {
		p.logger = logger
	}
}

// Provider provides SSE hub and handler instances
type Provider struct {
	hub     *Hub
	handler *Handler
	logger  *slog.Logger
}

// NewProvider creates a new SSE provider
//
// Parameters:
//   - opts: Optional functional options
//
// Usage:
//
//	provider := sse.NewProvider()
//	// or with custom logger
//	provider := sse.NewProvider(sse.WithLogger(logger))
func NewProvider(opts ...Option) *Provider {
	p := &Provider{}

	for _, opt := range opts {
		opt(p)
	}

	if p.logger == nil {
		p.logger = Logger()
	}

	p.logger = p.logger.With("platform", "sse")
	p.logger.Debug("SSE provider created")

	hub := NewHub()
	handler := NewHandler(hub)

	return &Provider{
		hub:     hub,
		handler: handler,
		logger:  p.logger,
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
