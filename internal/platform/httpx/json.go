package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// DecodeJSON decodes a single JSON object.
func DecodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(target); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain a single JSON object")
		}
		return err
	}

	return nil
}

// WriteJSON writes a JSON payload with the provided status code.
func WriteJSON(w http.ResponseWriter, status int, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(payload)
	return err
}

// WriteNoContent writes an HTTP 204 response.
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
