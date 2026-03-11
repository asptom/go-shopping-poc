package httperr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type errorEnvelope struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func TestSend(t *testing.T) {
	rr := httptest.NewRecorder()
	Send(rr, http.StatusBadRequest, "invalid_request", "bad input")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var payload errorEnvelope
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if payload.Error != "invalid_request" || payload.Message != "bad input" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestSendWithCodeAndHelpers(t *testing.T) {
	rr := httptest.NewRecorder()
	SendWithCode(rr, http.StatusUnauthorized, "unauthorized", "token expired", "AUTH-001")

	var payload errorEnvelope
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if payload.Code != "AUTH-001" {
		t.Fatalf("unexpected code: %s", payload.Code)
	}

	rr = httptest.NewRecorder()
	Validation(rr, "invalid state")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	rr = httptest.NewRecorder()
	NotFound(rr, "missing")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	rr = httptest.NewRecorder()
	Internal(rr, "boom")
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}
