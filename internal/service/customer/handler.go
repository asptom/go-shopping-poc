package customer

import (
	"context"
	"encoding/json"
	entity "go-shopping-poc/internal/entity/customer"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

// decodeMaybe unescapes only when needed to avoid double-decoding.
func decodeMaybe(raw string) (string, error) {
	if strings.Contains(raw, "%") {
		return url.QueryUnescape(raw)
	}
	return raw, nil
}

type CustomerHandler struct {
	service *CustomerService
}

func NewCustomerHandler(service *CustomerService) *CustomerHandler {
	return &CustomerHandler{service: service}
}

func (h *CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var customer entity.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.service.CreateCustomer(context.Background(), &customer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(customer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Example: email passed as query parameter ?email=...
func (h *CustomerHandler) GetCustomerByEmail(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("email")
	if raw == "" {
		http.Error(w, "missing email", http.StatusBadRequest)
		return
	}

	email, err := decodeMaybe(raw)
	if err != nil {
		http.Error(w, "invalid email encoding", http.StatusBadRequest)
		return
	}
	// validate
	if _, err := mail.ParseAddress(email); err != nil {
		http.Error(w, "invalid email address", http.StatusBadRequest)
		return
	}

	cust, err := h.service.GetCustomerByEmail(context.Background(), email)
	if err != nil {
		http.Error(w, "customer lookup failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if cust == nil {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}
	if err := json.NewEncoder(w).Encode(cust); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Example: email passed as path segment /customers/{email}
func (h *CustomerHandler) GetCustomerByEmailPath(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "email")
	// path segments should use PathUnescape
	email, err := url.PathUnescape(raw)
	if err != nil {
		http.Error(w, "invalid path encoding", http.StatusBadRequest)
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		http.Error(w, "invalid email address", http.StatusBadRequest)
		return
	}
	cust, err := h.service.GetCustomerByEmail(context.Background(), email)
	if err != nil {
		http.Error(w, "customer lookup failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if cust == nil {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}
	if err := json.NewEncoder(w).Encode(cust); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
