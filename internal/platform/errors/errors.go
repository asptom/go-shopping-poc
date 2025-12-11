// Package errors provides centralized error handling utilities.
//
// This package defines structured error responses and helper functions
// for consistent error handling across all services in the application.
package errors

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// ErrorType constants for consistent error categorization
const (
	ErrorTypeInvalidRequest = "invalid_request"
	ErrorTypeValidation     = "validation_error"
	ErrorTypeInternal       = "internal_error"
	ErrorTypeNotFound       = "not_found"
	ErrorTypeUnauthorized   = "unauthorized"
	ErrorTypeForbidden      = "forbidden"
)

// SendError sends a structured JSON error response
func SendError(w http.ResponseWriter, statusCode int, errorType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := ErrorResponse{
		Error:   errorType,
		Message: message,
	}
	json.NewEncoder(w).Encode(response)
}

// SendErrorWithCode sends a structured JSON error response with error code
func SendErrorWithCode(w http.ResponseWriter, statusCode int, errorType, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := ErrorResponse{
		Error:   errorType,
		Message: message,
		Code:    code,
	}
	json.NewEncoder(w).Encode(response)
}
