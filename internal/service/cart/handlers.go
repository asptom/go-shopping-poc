package cart

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"go-shopping-poc/internal/platform/errors"

	"github.com/go-chi/chi/v5"
)

type CreateCartRequest struct {
	CustomerID *string `json:"customer_id,omitempty"`
}

type AddItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type UpdateItemRequest struct {
	Quantity int `json:"quantity"`
}

type SetContactRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
}

type AddAddressRequest struct {
	AddressType string `json:"address_type"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Address1    string `json:"address_1"`
	Address2    string `json:"address_2,omitempty"`
	City        string `json:"city"`
	State       string `json:"state"`
	Zip         string `json:"zip"`
}

type SetPaymentRequest struct {
	CardType       string `json:"card_type"`
	CardNumber     string `json:"card_number"`
	CardHolderName string `json:"card_holder_name"`
	CardExpires    string `json:"card_expires"`
	CardCVV        string `json:"card_cvv"`
}

type CartHandler struct {
	service *CartService
	logger  *slog.Logger
}

func NewCartHandler(logger *slog.Logger, service *CartService) *CartHandler {
	return &CartHandler{logger: logger.With("component", "CartHandler"), service: service}
}

func (h *CartHandler) CreateCart(w http.ResponseWriter, r *http.Request) {
	var req CreateCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON")
		return
	}

	cart, err := h.service.CreateCart(r.Context(), req.CustomerID)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to create cart")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cart)
}

func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	cartID := chi.URLParam(r, "id")
	if cartID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
		return
	}

	cart, err := h.service.GetCart(r.Context(), cartID)
	if err != nil {
		if err == ErrCartNotFound {
			errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Cart not found")
			return
		}
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to get cart")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

func (h *CartHandler) DeleteCart(w http.ResponseWriter, r *http.Request) {
	cartID := chi.URLParam(r, "id")
	if cartID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
		return
	}

	if err := h.service.DeleteCart(r.Context(), cartID); err != nil {
		if err == ErrCartNotFound {
			errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Cart not found")
			return
		}
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to delete cart")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Starting handler AddItem")

	cartID := chi.URLParam(r, "id")
	if cartID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
		return
	}

	var req AddItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON")
		return
	}

	if req.ProductID == "" || req.Quantity <= 0 {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid product_id or quantity")
		return
	}

	h.logger.Debug("Adding item to cart",
		"cart_id", cartID,
		"product_id", req.ProductID,
		"quantity", req.Quantity,
	)
	h.logger.Debug("Calling service.AddItem for cart", "cart_id", cartID)
	item, err := h.service.AddItem(r.Context(), cartID, req.ProductID, req.Quantity)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to add item")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (h *CartHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	cartID := chi.URLParam(r, "id")
	lineNumber := chi.URLParam(r, "line")

	if cartID == "" || lineNumber == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID or line number")
		return
	}

	var req UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON")
		return
	}

	if req.Quantity <= 0 {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Quantity must be positive")
		return
	}

	if err := h.service.UpdateItemQuantity(r.Context(), cartID, lineNumber, req.Quantity); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to update item")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	cartID := chi.URLParam(r, "id")
	lineNumber := chi.URLParam(r, "line")

	if cartID == "" || lineNumber == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID or line number")
		return
	}

	if err := h.service.RemoveItem(r.Context(), cartID, lineNumber); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to remove item")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) SetContact(w http.ResponseWriter, r *http.Request) {
	cartID := chi.URLParam(r, "id")
	if cartID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
		return
	}

	var req SetContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON")
		return
	}

	// contactCartId, err := uuid.Parse(cartID)
	// if err != nil {
	// 	errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid cart ID format")
	// 	return
	// }

	contact := &Contact{
		// CartID:    contactCartId,
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
	}

	if err := h.service.SetContact(r.Context(), cartID, contact); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to set contact")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	cartID := chi.URLParam(r, "id")
	if cartID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
		return
	}

	var req AddAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON")
		return
	}

	address := &Address{
		AddressType: req.AddressType,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Address1:    req.Address1,
		Address2:    req.Address2,
		City:        req.City,
		State:       req.State,
		Zip:         req.Zip,
	}

	if err := h.service.AddAddress(r.Context(), cartID, address); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to add address")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(address)
}

func (h *CartHandler) SetPayment(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Starting SetPayment handler", "cart_id", chi.URLParam(r, "id"))
	cartID := chi.URLParam(r, "id")
	if cartID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
		return
	}

	var req SetPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("Failed to decode SetPayment request", "cart_id", cartID, "error", err)
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON")
		return
	}

	card := &CreditCard{
		CardType:       req.CardType,
		CardNumber:     req.CardNumber,
		CardHolderName: req.CardHolderName,
		CardExpires:    req.CardExpires,
		CardCVV:        req.CardCVV,
	}

	if err := h.service.SetCreditCard(r.Context(), cartID, card); err != nil {
		h.logger.Debug("Failed to set payment for cart", "cart_id", cartID, "error", err)
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to set payment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	cartID := chi.URLParam(r, "id")
	if cartID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
		return
	}

	cart, err := h.service.Checkout(r.Context(), cartID)
	if err != nil {
		if err.Error() == "cart not ready for checkout: cart must be active to checkout" {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, err.Error())
			return
		}
		if err.Error() == "cart not ready for checkout: cart must have at least one item" {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Cart is empty")
			return
		}
		if err.Error() == "cart not ready for checkout: contact information required" {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Contact information required")
			return
		}
		if err.Error() == "cart not ready for checkout: payment method required" {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Payment method required")
			return
		}
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Checkout failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(cart)
}
