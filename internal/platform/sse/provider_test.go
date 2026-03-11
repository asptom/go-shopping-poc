package sse

import "testing"

func TestProvider_AppliesHandlerOptions(t *testing.T) {
	p := NewProvider(
		WithHandlerOptions(
			WithMissingIDMessage("Missing cart ID"),
			WithConnectedIDField("cartId"),
			WithLogIDKey("cart_id"),
		),
	)

	h := p.GetHandler()
	if h.cfg.MissingIDMsg != "Missing cart ID" {
		t.Fatalf("unexpected missing id message: %q", h.cfg.MissingIDMsg)
	}
	if h.cfg.ConnectedIDField != "cartId" {
		t.Fatalf("unexpected connected id field: %q", h.cfg.ConnectedIDField)
	}
	if h.cfg.LogIDKey != "cart_id" {
		t.Fatalf("unexpected log key: %q", h.cfg.LogIDKey)
	}
}
