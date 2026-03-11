package sse

import "testing"

func TestNewHandler_DefaultConfig(t *testing.T) {
	h := NewHandler(NewHub())

	if h.cfg.PathParam != "id" {
		t.Fatalf("expected default path param id, got %q", h.cfg.PathParam)
	}
	if h.cfg.MissingIDMsg != "Missing stream ID" {
		t.Fatalf("unexpected default missing message: %q", h.cfg.MissingIDMsg)
	}
	if h.cfg.ConnectedIDField != "streamId" {
		t.Fatalf("unexpected default connected field: %q", h.cfg.ConnectedIDField)
	}
	if h.cfg.LogIDKey != "stream_id" {
		t.Fatalf("unexpected default log key: %q", h.cfg.LogIDKey)
	}
}

func TestNewHandler_CustomConfig(t *testing.T) {
	h := NewHandler(NewHub(),
		WithPathParam("entity"),
		WithMissingIDMessage("Missing entity ID"),
		WithConnectedIDField("entityId"),
		WithLogIDKey("entity_id"),
	)

	if h.cfg.PathParam != "entity" {
		t.Fatalf("unexpected path param: %q", h.cfg.PathParam)
	}
	if h.cfg.MissingIDMsg != "Missing entity ID" {
		t.Fatalf("unexpected missing message: %q", h.cfg.MissingIDMsg)
	}
	if h.cfg.ConnectedIDField != "entityId" {
		t.Fatalf("unexpected connected id field: %q", h.cfg.ConnectedIDField)
	}
	if h.cfg.LogIDKey != "entity_id" {
		t.Fatalf("unexpected log key: %q", h.cfg.LogIDKey)
	}
}
