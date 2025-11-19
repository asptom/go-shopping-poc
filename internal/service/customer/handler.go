package customer

import (
	"context"
	"encoding/json"
	entity "go-shopping-poc/internal/entity/customer"
	"net/http"
	"net/mail"
	"net/url"

	"github.com/go-chi/chi/v5"
)

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

func (h *CustomerHandler) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	var customer entity.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// PUT requires complete customer record - validate required fields
	if customer.CustomerID == "" || customer.Username == "" || customer.Email == "" {
		http.Error(w, "PUT requires complete customer record with customer_id, username, and email", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateCustomer(context.Background(), &customer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(customer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *CustomerHandler) PatchCustomer(w http.ResponseWriter, r *http.Request) {
	var patchData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patchData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Extract customer_id from path
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		http.Error(w, "missing customer_id in path", http.StatusBadRequest)
		return
	}

	if err := h.service.PatchCustomer(context.Background(), customerID, patchData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated customer
	updated, err := h.service.GetCustomerByID(context.Background(), customerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if updated == nil {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updated); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Example: email passed as query parameter ?email=...
func (h *CustomerHandler) GetCustomerByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "missing email", http.StatusBadRequest)
		return
	}
	// validate
	if _, err := mail.ParseAddress(email); err != nil {
		http.Error(w, "invalid email address", http.StatusBadRequest)
		return
	}

	cust, err := h.service.GetCustomerByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "customer lookup failed", http.StatusInternalServerError)
		return
	}
	if cust == nil {
		// No customer found -> return 204 No Content
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
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

// AddAddress - POST /customers/{id}/addresses
func (h *CustomerHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing customer id", http.StatusBadRequest)
		return
	}
	var addr entity.Address
	if err := json.NewDecoder(r.Body).Decode(&addr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := h.service.AddAddress(r.Context(), id, &addr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// UpdateAddress - PUT /customers/addresses/{addressId}
func (h *CustomerHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	addressID := chi.URLParam(r, "addressId")
	if addressID == "" {
		http.Error(w, "missing address id", http.StatusBadRequest)
		return
	}

	var addr entity.Address
	if err := json.NewDecoder(r.Body).Decode(&addr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateAddress(r.Context(), addressID, &addr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteAddress - DELETE /customers/addresses/{addressId}
func (h *CustomerHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	addressID := chi.URLParam(r, "addressId")
	if addressID == "" {
		http.Error(w, "missing address id", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteAddress(r.Context(), addressID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AddCreditCard - POST /customers/{id}/credit-cards
func (h *CustomerHandler) AddCreditCard(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing customer id", http.StatusBadRequest)
		return
	}
	var card entity.CreditCard
	if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := h.service.AddCreditCard(r.Context(), id, &card); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// UpdateCreditCard - PUT /customers/credit-cards/{cardId}
func (h *CustomerHandler) UpdateCreditCard(w http.ResponseWriter, r *http.Request) {
	cardId := chi.URLParam(r, "cardId")
	if cardId == "" {
		http.Error(w, "missing card id", http.StatusBadRequest)
		return
	}
	var card entity.CreditCard
	if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.service.UpdateCreditCard(r.Context(), cardId, &card); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteCreditCard - DELETE /customers/credit-cards/{cardId}
func (h *CustomerHandler) DeleteCreditCard(w http.ResponseWriter, r *http.Request) {
	cardId := chi.URLParam(r, "cardId")
	if cardId == "" {
		http.Error(w, "missing card id", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteCreditCard(r.Context(), cardId); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetDefaultShippingAddress - PUT /customers/{id}/default-shipping-address/{addressId}
func (h *CustomerHandler) SetDefaultShippingAddress(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		http.Error(w, "missing customer id", http.StatusBadRequest)
		return
	}
	addressID := chi.URLParam(r, "addressId")
	if addressID == "" {
		http.Error(w, "missing address id", http.StatusBadRequest)
		return
	}
	if err := h.service.SetDefaultShippingAddress(r.Context(), customerID, addressID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetDefaultBillingAddress - PUT /customers/{id}/default-billing-address/{addressId}
func (h *CustomerHandler) SetDefaultBillingAddress(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		http.Error(w, "missing customer id", http.StatusBadRequest)
		return
	}
	addressID := chi.URLParam(r, "addressId")
	if addressID == "" {
		http.Error(w, "missing address id", http.StatusBadRequest)
		return
	}
	if err := h.service.SetDefaultBillingAddress(r.Context(), customerID, addressID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetDefaultCreditCard - PUT /customers/{id}/default-credit-card/{cardId}
func (h *CustomerHandler) SetDefaultCreditCard(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		http.Error(w, "missing customer id", http.StatusBadRequest)
		return
	}
	cardID := chi.URLParam(r, "cardId")
	if cardID == "" {
		http.Error(w, "missing card id", http.StatusBadRequest)
		return
	}
	if err := h.service.SetDefaultCreditCard(r.Context(), customerID, cardID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClearDefaultShippingAddress - DELETE /customers/{id}/default-shipping-address
func (h *CustomerHandler) ClearDefaultShippingAddress(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		http.Error(w, "missing customer id", http.StatusBadRequest)
		return
	}
	if err := h.service.ClearDefaultShippingAddress(r.Context(), customerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClearDefaultBillingAddress - DELETE /customers/{id}/default-billing-address
func (h *CustomerHandler) ClearDefaultBillingAddress(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		http.Error(w, "missing customer id", http.StatusBadRequest)
		return
	}
	if err := h.service.ClearDefaultBillingAddress(r.Context(), customerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClearDefaultCreditCard - DELETE /customers/{id}/default-credit-card
func (h *CustomerHandler) ClearDefaultCreditCard(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		http.Error(w, "missing customer id", http.StatusBadRequest)
		return
	}
	if err := h.service.ClearDefaultCreditCard(r.Context(), customerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
