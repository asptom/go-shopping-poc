package httperr

import (
	"net/http"

	platformerrors "go-shopping-poc/internal/platform/errors"
)

// Send writes a structured JSON error envelope.
func Send(w http.ResponseWriter, statusCode int, errorType, message string) {
	platformerrors.SendError(w, statusCode, errorType, message)
}

// SendWithCode writes a structured JSON error envelope including an error code.
func SendWithCode(w http.ResponseWriter, statusCode int, errorType, message, code string) {
	platformerrors.SendErrorWithCode(w, statusCode, errorType, message, code)
}

// InvalidRequest writes a 400 invalid_request error response.
func InvalidRequest(w http.ResponseWriter, message string) {
	Send(w, http.StatusBadRequest, platformerrors.ErrorTypeInvalidRequest, message)
}

// Validation writes a 400 validation_error response.
func Validation(w http.ResponseWriter, message string) {
	Send(w, http.StatusBadRequest, platformerrors.ErrorTypeValidation, message)
}

// NotFound writes a 404 not_found error response.
func NotFound(w http.ResponseWriter, message string) {
	Send(w, http.StatusNotFound, platformerrors.ErrorTypeNotFound, message)
}

// Internal writes a 500 internal_error response.
func Internal(w http.ResponseWriter, message string) {
	Send(w, http.StatusInternalServerError, platformerrors.ErrorTypeInternal, message)
}

// Forbidden writes a 403 forbidden error response.
func Forbidden(w http.ResponseWriter, message string) {
	Send(w, http.StatusForbidden, platformerrors.ErrorTypeForbidden, message)
}

// GatewayTimeout writes a 504 gateway_timeout error response.
func GatewayTimeout(w http.ResponseWriter, message string) {
	Send(w, http.StatusGatewayTimeout, platformerrors.ErrorTypeGatewayTimeout, message)
}
