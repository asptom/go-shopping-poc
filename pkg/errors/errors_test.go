package errors

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendError(t *testing.T) {
	w := httptest.NewRecorder()

	SendError(w, http.StatusBadRequest, ErrorTypeInvalidRequest, "Test error message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"error":"invalid_request"`) {
		t.Errorf("Expected error type in response, got: %s", body)
	}

	if !strings.Contains(body, `"message":"Test error message"`) {
		t.Errorf("Expected error message in response, got: %s", body)
	}
}

func TestSendErrorWithCode(t *testing.T) {
	w := httptest.NewRecorder()

	SendErrorWithCode(w, http.StatusNotFound, ErrorTypeNotFound, "Resource not found", "ERR_404")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"error":"not_found"`) {
		t.Errorf("Expected error type in response, got: %s", body)
	}

	if !strings.Contains(body, `"message":"Resource not found"`) {
		t.Errorf("Expected error message in response, got: %s", body)
	}

	if !strings.Contains(body, `"code":"ERR_404"`) {
		t.Errorf("Expected error code in response, got: %s", body)
	}
}

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"ErrorTypeInvalidRequest", ErrorTypeInvalidRequest, "invalid_request"},
		{"ErrorTypeValidation", ErrorTypeValidation, "validation_error"},
		{"ErrorTypeInternal", ErrorTypeInternal, "internal_error"},
		{"ErrorTypeNotFound", ErrorTypeNotFound, "not_found"},
		{"ErrorTypeUnauthorized", ErrorTypeUnauthorized, "unauthorized"},
		{"ErrorTypeForbidden", ErrorTypeForbidden, "forbidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s to be '%s', got '%s'", tt.name, tt.expected, tt.constant)
			}
		})
	}
}
