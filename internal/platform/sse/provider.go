package sse

// Provider provides SSE hub and handler instances
type Provider struct {
	hub     *Hub
	handler *Handler
}

// NewProvider creates a new SSE provider
func NewProvider() *Provider {
	hub := NewHub()
	handler := NewHandler(hub)

	return &Provider{
		hub:     hub,
		handler: handler,
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
