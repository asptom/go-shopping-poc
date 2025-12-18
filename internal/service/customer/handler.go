package customer

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"net/url"

	"github.com/go-chi/chi/v5"

	"go-shopping-poc/internal/platform/errors"
)

type CustomerHandler struct {
	service *CustomerService
}

func NewCustomerHandler(service *CustomerService) *CustomerHandler {
	return &CustomerHandler{service: service}
}

func (h *CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var customer Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}

	// Validate required fields
	if customer.Username == "" || customer.Email == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "username and email are required")
		return
	}

	if err := h.service.CreateCustomer(r.Context(), &customer); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to create customer")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(customer); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

func (h *CustomerHandler) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	var customer Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}

	// PUT requires complete customer record - validate required fields
	if customer.CustomerID == "" || customer.Username == "" || customer.Email == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "PUT requires complete customer record with customer_id, username, and email")
		return
	}

	if err := h.service.UpdateCustomer(r.Context(), &customer); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to update customer")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(customer); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

func (h *CustomerHandler) PatchCustomer(w http.ResponseWriter, r *http.Request) {
	var patchData PatchCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&patchData); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}

	// Extract customer_id from path
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing customer_id in path")
		return
	}

	if err := h.service.PatchCustomer(r.Context(), customerID, &patchData); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to patch customer")
		return
	}

	// Return updated customer
	updated, err := h.service.GetCustomerByID(r.Context(), customerID)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve updated customer")
		return
	}
	if updated == nil {
		errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Customer not found after update")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updated); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// Example: email passed as query parameter ?email=...
func (h *CustomerHandler) GetCustomerByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing email parameter")
		return
	}
	// validate
	if _, err := mail.ParseAddress(email); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid email address")
		return
	}

	cust, err := h.service.GetCustomerByEmail(r.Context(), email)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Customer lookup failed")
		return
	}
	if cust == nil {
		// No customer found -> return 204 No Content
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cust); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// Example: email passed as path segment /customers/{email}
func (h *CustomerHandler) GetCustomerByEmailPath(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "email")
	// path segments should use PathUnescape
	email, err := url.PathUnescape(raw)
	if err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid path encoding")
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid email address")
		return
	}
	cust, err := h.service.GetCustomerByEmail(r.Context(), email)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Customer lookup failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if cust == nil {
		errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Customer not found")
		return
	}
	if err := json.NewEncoder(w).Encode(cust); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// AddAddress - POST /customers/{id}/addresses
func (h *CustomerHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing customer id")
		return
	}
	var addr Address
	if err := json.NewDecoder(r.Body).Decode(&addr); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}
	if _, err := h.service.AddAddress(r.Context(), id, &addr); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to add address")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// UpdateAddress - PUT /customers/addresses/{addressId}
func (h *CustomerHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	addressID := chi.URLParam(r, "addressId")
	if addressID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing address id")
		return
	}

	var addr Address
	if err := json.NewDecoder(r.Body).Decode(&addr); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}

	if err := h.service.UpdateAddress(r.Context(), addressID, &addr); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to update address")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteAddress - DELETE /customers/addresses/{addressId}
func (h *CustomerHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	addressID := chi.URLParam(r, "addressId")
	if addressID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing address id")
		return
	}
	if err := h.service.DeleteAddress(r.Context(), addressID); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to delete address")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AddCreditCard - POST /customers/{id}/credit-cards
func (h *CustomerHandler) AddCreditCard(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing customer id")
		return
	}
	var card CreditCard
	if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}
	if _, err := h.service.AddCreditCard(r.Context(), id, &card); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to add credit card")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// UpdateCreditCard - PUT /customers/credit-cards/{cardId}
func (h *CustomerHandler) UpdateCreditCard(w http.ResponseWriter, r *http.Request) {
	cardId := chi.URLParam(r, "cardId")
	if cardId == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing card id")
		return
	}
	var card CreditCard
	if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}
	if err := h.service.UpdateCreditCard(r.Context(), cardId, &card); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to update credit card")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteCreditCard - DELETE /customers/credit-cards/{cardId}
func (h *CustomerHandler) DeleteCreditCard(w http.ResponseWriter, r *http.Request) {
	cardId := chi.URLParam(r, "cardId")
	if cardId == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing card id")
		return
	}
	if err := h.service.DeleteCreditCard(r.Context(), cardId); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to delete credit card")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetDefaultShippingAddress - PUT /customers/{id}/default-shipping-address/{addressId}
func (h *CustomerHandler) SetDefaultShippingAddress(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing customer id")
		return
	}
	addressID := chi.URLParam(r, "addressId")
	if addressID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing address id")
		return
	}
	if err := h.service.SetDefaultShippingAddress(r.Context(), customerID, addressID); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to set default shipping address")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetDefaultBillingAddress - PUT /customers/{id}/default-billing-address/{addressId}
func (h *CustomerHandler) SetDefaultBillingAddress(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing customer id")
		return
	}
	addressID := chi.URLParam(r, "addressId")
	if addressID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing address id")
		return
	}
	if err := h.service.SetDefaultBillingAddress(r.Context(), customerID, addressID); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to set default billing address")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetDefaultCreditCard - PUT /customers/{id}/default-credit-card/{cardId}
func (h *CustomerHandler) SetDefaultCreditCard(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing customer id")
		return
	}
	cardID := chi.URLParam(r, "cardId")
	if cardID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing card id")
		return
	}
	if err := h.service.SetDefaultCreditCard(r.Context(), customerID, cardID); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to set default credit card")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClearDefaultShippingAddress - DELETE /customers/{id}/default-shipping-address
func (h *CustomerHandler) ClearDefaultShippingAddress(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing customer id")
		return
	}
	if err := h.service.ClearDefaultShippingAddress(r.Context(), customerID); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to clear default shipping address")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClearDefaultBillingAddress - DELETE /customers/{id}/default-billing-address
func (h *CustomerHandler) ClearDefaultBillingAddress(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing customer id")
		return
	}
	if err := h.service.ClearDefaultBillingAddress(r.Context(), customerID); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to clear default billing address")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClearDefaultCreditCard - DELETE /customers/{id}/default-credit-card
func (h *CustomerHandler) ClearDefaultCreditCard(w http.ResponseWriter, r *http.Request) {
	customerID := chi.URLParam(r, "id")
	if customerID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing customer id")
		return
	}
	if err := h.service.ClearDefaultCreditCard(r.Context(), customerID); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to clear default credit card")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
