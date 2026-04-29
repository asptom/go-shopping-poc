package customer

import (
	"net/http"
	"net/mail"
	"net/url"

	"go-shopping-poc/internal/platform/auth"
	"go-shopping-poc/internal/platform/httperr"
	"go-shopping-poc/internal/platform/httpx"
)

type CustomerHandler struct {
	service *CustomerService
}

func NewCustomerHandler(service *CustomerService) *CustomerHandler {
	return &CustomerHandler{service: service}
}

func (h *CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var customer Customer
	if err := httpx.DecodeJSON(r, &customer); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON in request body")
		return
	}

	// Extract keycloak_sub from JWT if present (optional auth)
	if claims, ok := auth.GetClaims(r.Context()); ok {
		customer.KeycloakSub = claims.Subject
		if claims.PreferredUsername != "" {
			customer.Username = claims.PreferredUsername
		}
	}

	// Validate required fields
	if customer.Username == "" || customer.Email == "" {
		httperr.Validation(w, "username and email are required")
		return
	}

	if err := h.service.CreateCustomer(r.Context(), &customer); err != nil {
		httperr.Internal(w, "Failed to create customer")
		return
	}
	if err := httpx.WriteJSON(w, http.StatusCreated, customer); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

func (h *CustomerHandler) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	var customer Customer
	if err := httpx.DecodeJSON(r, &customer); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON in request body")
		return
	}

	// PUT requires complete customer record - validate required fields
	if customer.CustomerID == "" || customer.Username == "" || customer.Email == "" {
		httperr.Validation(w, "PUT requires complete customer record with customer_id, username, and email")
		return
	}

	if err := h.service.UpdateCustomer(r.Context(), &customer); err != nil {
		httperr.Internal(w, "Failed to update customer")
		return
	}
	if err := httpx.WriteJSON(w, http.StatusOK, customer); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

func (h *CustomerHandler) PatchCustomer(w http.ResponseWriter, r *http.Request) {
	var patchData PatchCustomerRequest
	if err := httpx.DecodeJSON(r, &patchData); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON in request body")
		return
	}

	customerID, ok := requiredPathParam(w, r, "id", "Missing customer_id in path")
	if !ok {
		return
	}

	if err := h.service.PatchCustomer(r.Context(), customerID, &patchData); err != nil {
		httperr.Internal(w, "Failed to patch customer")
		return
	}

	// Return updated customer
	updated, err := h.service.GetCustomerByID(r.Context(), customerID)
	if err != nil {
		httperr.Internal(w, "Failed to retrieve updated customer")
		return
	}
	if updated == nil {
		httperr.NotFound(w, "Customer not found after update")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, updated); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

// Example: email passed as query parameter ?email=...
func (h *CustomerHandler) GetCustomerByEmail(w http.ResponseWriter, r *http.Request) {
	email, ok := requiredQueryParam(w, r, "email", "Missing email parameter")
	if !ok {
		return
	}
	// validate
	if _, err := mail.ParseAddress(email); err != nil {
		httperr.Validation(w, "Invalid email address")
		return
	}

	cust, err := h.service.GetCustomerByEmail(r.Context(), email)
	if err != nil {
		httperr.Internal(w, "Customer lookup failed")
		return
	}
	if cust == nil {
		// No customer found -> return 204 NoContent
		httpx.WriteNoContent(w)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, cust); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

// Example: email passed as path segment /customers/{email}
func (h *CustomerHandler) GetCustomerByEmailPath(w http.ResponseWriter, r *http.Request) {
	raw, ok := requiredPathParam(w, r, "email", "Missing email in path")
	if !ok {
		return
	}
	// path segments should use PathUnescape
	email, err := url.PathUnescape(raw)
	if err != nil {
		httperr.InvalidRequest(w, "Invalid path encoding")
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		httperr.Validation(w, "Invalid email address")
		return
	}
	cust, err := h.service.GetCustomerByEmail(r.Context(), email)
	if err != nil {
		httperr.Internal(w, "Customer lookup failed")
		return
	}
	if cust == nil {
		httperr.NotFound(w, "Customer not found")
		return
	}
	if err := httpx.WriteJSON(w, http.StatusOK, cust); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

// AddAddress - POST /customers/{id}/addresses
func (h *CustomerHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredPathParam(w, r, "id", "Missing customer id")
	if !ok {
		return
	}
	var addr Address
	if err := httpx.DecodeJSON(r, &addr); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON in request body")
		return
	}
	if _, err := h.service.AddAddress(r.Context(), id, &addr); err != nil {
		httperr.Internal(w, "Failed to add address")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// UpdateAddress - PUT /customers/addresses/{addressId}
func (h *CustomerHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	addressID, ok := requiredPathParam(w, r, "addressId", "Missing address id")
	if !ok {
		return
	}

	var addr Address
	if err := httpx.DecodeJSON(r, &addr); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON in request body")
		return
	}

	if err := h.service.UpdateAddress(r.Context(), addressID, &addr); err != nil {
		httperr.Internal(w, "Failed to update address")
		return
	}
	httpx.WriteNoContent(w)
}

// DeleteAddress - DELETE /customers/addresses/{addressId}
func (h *CustomerHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	addressID, ok := requiredPathParam(w, r, "addressId", "Missing address id")
	if !ok {
		return
	}
	if err := h.service.DeleteAddress(r.Context(), addressID); err != nil {
		httperr.Internal(w, "Failed to delete address")
		return
	}
	httpx.WriteNoContent(w)
}

// AddCreditCard - POST /customers/{id}/credit-cards
func (h *CustomerHandler) AddCreditCard(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredPathParam(w, r, "id", "Missing customer id")
	if !ok {
		return
	}
	var card CreditCard
	if err := httpx.DecodeJSON(r, &card); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON in request body")
		return
	}
	if _, err := h.service.AddCreditCard(r.Context(), id, &card); err != nil {
		httperr.Internal(w, "Failed to add credit card")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// UpdateCreditCard - PUT /customers/credit-cards/{cardId}
func (h *CustomerHandler) UpdateCreditCard(w http.ResponseWriter, r *http.Request) {
	cardID, ok := requiredPathParam(w, r, "cardId", "Missing card id")
	if !ok {
		return
	}
	var card CreditCard
	if err := httpx.DecodeJSON(r, &card); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON in request body")
		return
	}
	if err := h.service.UpdateCreditCard(r.Context(), cardID, &card); err != nil {
		httperr.Internal(w, "Failed to update credit card")
		return
	}
	httpx.WriteNoContent(w)
}

// DeleteCreditCard - DELETE /customers/credit-cards/{cardId}
func (h *CustomerHandler) DeleteCreditCard(w http.ResponseWriter, r *http.Request) {
	cardID, ok := requiredPathParam(w, r, "cardId", "Missing card id")
	if !ok {
		return
	}
	if err := h.service.DeleteCreditCard(r.Context(), cardID); err != nil {
		httperr.Internal(w, "Failed to delete credit card")
		return
	}
	httpx.WriteNoContent(w)
}

// SetDefaultShippingAddress - PUT /customers/{id}/default-shipping-address/{addressId}
func (h *CustomerHandler) SetDefaultShippingAddress(w http.ResponseWriter, r *http.Request) {
	customerID, ok := requiredPathParam(w, r, "id", "Missing customer id")
	if !ok {
		return
	}
	addressID, ok := requiredPathParam(w, r, "addressId", "Missing address id")
	if !ok {
		return
	}
	if err := h.service.SetDefaultShippingAddress(r.Context(), customerID, addressID); err != nil {
		httperr.Internal(w, "Failed to set default shipping address")
		return
	}
	httpx.WriteNoContent(w)
}

// SetDefaultBillingAddress - PUT /customers/{id}/default-billing-address/{addressId}
func (h *CustomerHandler) SetDefaultBillingAddress(w http.ResponseWriter, r *http.Request) {
	customerID, ok := requiredPathParam(w, r, "id", "Missing customer id")
	if !ok {
		return
	}
	addressID, ok := requiredPathParam(w, r, "addressId", "Missing address id")
	if !ok {
		return
	}
	if err := h.service.SetDefaultBillingAddress(r.Context(), customerID, addressID); err != nil {
		httperr.Internal(w, "Failed to set default billing address")
		return
	}
	httpx.WriteNoContent(w)
}

// SetDefaultCreditCard - PUT /customers/{id}/default-credit-card/{cardId}
func (h *CustomerHandler) SetDefaultCreditCard(w http.ResponseWriter, r *http.Request) {
	customerID, ok := requiredPathParam(w, r, "id", "Missing customer id")
	if !ok {
		return
	}
	cardID, ok := requiredPathParam(w, r, "cardId", "Missing card id")
	if !ok {
		return
	}
	if err := h.service.SetDefaultCreditCard(r.Context(), customerID, cardID); err != nil {
		httperr.Internal(w, "Failed to set default credit card")
		return
	}
	httpx.WriteNoContent(w)
}

// ClearDefaultShippingAddress - DELETE /customers/{id}/default-shipping-address
func (h *CustomerHandler) ClearDefaultShippingAddress(w http.ResponseWriter, r *http.Request) {
	customerID, ok := requiredPathParam(w, r, "id", "Missing customer id")
	if !ok {
		return
	}
	if err := h.service.ClearDefaultShippingAddress(r.Context(), customerID); err != nil {
		httperr.Internal(w, "Failed to clear default shipping address")
		return
	}
	httpx.WriteNoContent(w)
}

// ClearDefaultBillingAddress - DELETE /customers/{id}/default-billing-address
func (h *CustomerHandler) ClearDefaultBillingAddress(w http.ResponseWriter, r *http.Request) {
	customerID, ok := requiredPathParam(w, r, "id", "Missing customer id")
	if !ok {
		return
	}
	if err := h.service.ClearDefaultBillingAddress(r.Context(), customerID); err != nil {
		httperr.Internal(w, "Failed to clear default billing address")
		return
	}
	httpx.WriteNoContent(w)
}

// ClearDefaultCreditCard - DELETE /customers/{id}/default-credit-card
func (h *CustomerHandler) ClearDefaultCreditCard(w http.ResponseWriter, r *http.Request) {
	customerID, ok := requiredPathParam(w, r, "id", "Missing customer id")
	if !ok {
		return
	}
	if err := h.service.ClearDefaultCreditCard(r.Context(), customerID); err != nil {
		httperr.Internal(w, "Failed to clear default credit card")
		return
	}
	httpx.WriteNoContent(w)
}

func requiredPathParam(w http.ResponseWriter, r *http.Request, key, missingMessage string) (string, bool) {
	value, err := httpx.RequirePathParam(r, key)
	if err != nil {
		httperr.InvalidRequest(w, missingMessage)
		return "", false
	}

	return value, true
}

func requiredQueryParam(w http.ResponseWriter, r *http.Request, key, missingMessage string) (string, bool) {
	value, err := httpx.RequireQueryParam(r, key)
	if err != nil {
		httperr.InvalidRequest(w, missingMessage)
		return "", false
	}

	return value, true
}
