package httpx

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

var (
	// ErrMissingPathParam indicates a required path parameter is absent.
	ErrMissingPathParam = errors.New("missing path parameter")
	// ErrMissingQueryParam indicates a required query parameter is absent.
	ErrMissingQueryParam = errors.New("missing query parameter")
)

// ParamError provides context for missing parameter errors.
type ParamError struct {
	Kind string
	Key  string
}

func (e *ParamError) Error() string {
	return fmt.Sprintf("%s parameter %q is required", e.Kind, e.Key)
}

func (e *ParamError) Unwrap() error {
	if e.Kind == "path" {
		return ErrMissingPathParam
	}
	if e.Kind == "query" {
		return ErrMissingQueryParam
	}
	return nil
}

// RequirePathParam returns a required path parameter or a typed error.
func RequirePathParam(r *http.Request, key string) (string, error) {
	value := chi.URLParam(r, key)
	if value == "" {
		return "", &ParamError{Kind: "path", Key: key}
	}
	return value, nil
}

// RequireQueryParam returns a required query parameter or a typed error.
func RequireQueryParam(r *http.Request, key string) (string, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return "", &ParamError{Kind: "query", Key: key}
	}
	return value, nil
}
