package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestDecodeJSON_SingleObject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"book"}`))
		var body struct {
			Name string `json:"name"`
		}

		if err := DecodeJSON(req, &body); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if body.Name != "book" {
			t.Fatalf("unexpected value: %s", body.Name)
		}
	})

	t.Run("allow unknown fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"book","extra":"x"}`))
		var body struct {
			Name string `json:"name"`
		}

		if err := DecodeJSON(req, &body); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if body.Name != "book" {
			t.Fatalf("unexpected value: %s", body.Name)
		}
	})

	t.Run("reject multiple objects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"book"}{"name":"other"}`))
		var body struct {
			Name string `json:"name"`
		}

		err := DecodeJSON(req, &body)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestWriteJSONAndNoContent(t *testing.T) {
	rr := httptest.NewRecorder()
	err := WriteJSON(rr, http.StatusCreated, map[string]string{"id": "123"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("unexpected content-type: %s", ct)
	}
	if body := rr.Body.String(); body != `{"id":"123"}` {
		t.Fatalf("unexpected body: %s", body)
	}

	rr = httptest.NewRecorder()
	WriteNoContent(rr)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestRequireParamHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items?q=abc", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "42")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	if id, err := RequirePathParam(req, "id"); err != nil || id != "42" {
		t.Fatalf("expected path param id=42, got id=%q err=%v", id, err)
	}

	if _, err := RequirePathParam(req, "missing"); !errors.Is(err, ErrMissingPathParam) {
		t.Fatalf("expected missing path param error, got %v", err)
	}

	if q, err := RequireQueryParam(req, "q"); err != nil || q != "abc" {
		t.Fatalf("expected query param, got q=%q err=%v", q, err)
	}

	if _, err := RequireQueryParam(req, "missing"); !errors.Is(err, ErrMissingQueryParam) {
		t.Fatalf("expected missing query param error, got %v", err)
	}
}
