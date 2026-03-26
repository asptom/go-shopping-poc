package cart

import (
	"errors"
	"log/slog"
	"net/http"

	"go-shopping-poc/internal/platform/httperr"
	"go-shopping-poc/internal/platform/httpx"
)

type CreateCartRequest struct {
	CustomerID *string `json:"customer_id,omitempty"`
}

type AddItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	ImageURL  string `json:"image_url"`
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
	return &CartHandler{logger: logger.With("component", "cart_handler"), service: service}
}

func (h *CartHandler) CreateCart(w http.ResponseWriter, r *http.Request) {
	var req CreateCartRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON")
		return
	}

	cart, err := h.service.CreateCart(r.Context(), req.CustomerID)
	if err != nil {
		httperr.Internal(w, "Failed to create cart")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusCreated, cart); err != nil {
		h.logger.Error("Failed to write create cart response", "error", err.Error())
	}
}

func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	cartID, ok := requiredPathParam(w, r, "id", "Missing cart ID")
	if !ok {
		return
	}

	cart, err := h.service.GetCart(r.Context(), cartID)
	if err != nil {
		if errors.Is(err, ErrCartNotFound) {
			httperr.NotFound(w, "Cart not found")
			return
		}
		httperr.Internal(w, "Failed to get cart")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, cart); err != nil {
		h.logger.Error("Failed to write get cart response", "error", err.Error())
	}
}

func (h *CartHandler) DeleteCart(w http.ResponseWriter, r *http.Request) {
	cartID, ok := requiredPathParam(w, r, "id", "Missing cart ID")
	if !ok {
		return
	}

	if err := h.service.DeleteCart(r.Context(), cartID); err != nil {
		if errors.Is(err, ErrCartNotFound) {
			httperr.NotFound(w, "Cart not found")
			return
		}
		httperr.Internal(w, "Failed to delete cart")
		return
	}

	httpx.WriteNoContent(w)
}

func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	cartID, ok := requiredPathParam(w, r, "id", "Missing cart ID")
	if !ok {
		return
	}

	var req AddItemRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON")
		return
	}

	if req.ProductID == "" || req.Quantity <= 0 {
		httperr.Validation(w, "Invalid product_id or quantity")
		return
	}

	h.logger.Debug("Adding item to cart",
		"cart_id", cartID,
		"product_id", req.ProductID,
		"quantity", req.Quantity,
		"image_url", req.ImageURL,
	)
	item, err := h.service.AddItem(r.Context(), cartID, req.ProductID, req.Quantity, req.ImageURL)
	if err != nil {
		httperr.Internal(w, "Failed to add item")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusCreated, item); err != nil {
		h.logger.Error("Failed to write add item response", "error", err.Error())
	}
}

func (h *CartHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	cartID, ok := requiredPathParam(w, r, "id", "Missing cart ID or line number")
	if !ok {
		return
	}
	lineNumber, ok := requiredPathParam(w, r, "line", "Missing cart ID or line number")
	if !ok {
		return
	}

	var req UpdateItemRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON")
		return
	}

	if req.Quantity <= 0 {
		httperr.Validation(w, "Quantity must be positive")
		return
	}

	if err := h.service.UpdateItemQuantity(r.Context(), cartID, lineNumber, req.Quantity); err != nil {
		httperr.Internal(w, "Failed to update item")
		return
	}

	httpx.WriteNoContent(w)
}

func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	cartID, ok := requiredPathParam(w, r, "id", "Missing cart ID or line number")
	if !ok {
		return
	}
	lineNumber, ok := requiredPathParam(w, r, "line", "Missing cart ID or line number")
	if !ok {
		return
	}

	if err := h.service.RemoveItem(r.Context(), cartID, lineNumber); err != nil {
		httperr.Internal(w, "Failed to remove item")
		return
	}

	httpx.WriteNoContent(w)
}

func (h *CartHandler) SetContact(w http.ResponseWriter, r *http.Request) {
	cartID, ok := requiredPathParam(w, r, "id", "Missing cart ID")
	if !ok {
		return
	}

	var req SetContactRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON")
		return
	}

	contact := &Contact{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
	}

	if err := h.service.SetContact(r.Context(), cartID, contact); err != nil {
		httperr.Internal(w, "Failed to set contact")
		return
	}

	httpx.WriteNoContent(w)
}

func (h *CartHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	cartID, ok := requiredPathParam(w, r, "id", "Missing cart ID")
	if !ok {
		return
	}

	var req AddAddressRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON")
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
		httperr.Internal(w, "Failed to add address")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusCreated, address); err != nil {
		h.logger.Error("Failed to write add address response", "error", err.Error())
	}
}

func (h *CartHandler) SetPayment(w http.ResponseWriter, r *http.Request) {
	cartID, ok := requiredPathParam(w, r, "id", "Missing cart ID")
	if !ok {
		return
	}

	var req SetPaymentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		h.logger.Debug("Failed to decode SetPayment request", "cart_id", cartID, "error", err)
		httperr.InvalidRequest(w, "Invalid JSON")
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
		httperr.Internal(w, "Failed to set payment")
		return
	}

	httpx.WriteNoContent(w)
}

func (h *CartHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	cartID, ok := requiredPathParam(w, r, "id", "Missing cart ID")
	if !ok {
		return
	}

	cart, err := h.service.Checkout(r.Context(), cartID)
	if err != nil {
		if h.handleCheckoutValidationError(w, err) {
			return
		}
		httperr.Internal(w, "Checkout failed")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, cart); err != nil {
		h.logger.Error("Failed to write checkout response", "error", err.Error())
	}
}

func (h *CartHandler) handleCheckoutValidationError(w http.ResponseWriter, err error) bool {
	if errors.Is(err, ErrCartMustBeActiveForCheckout) {
		httperr.Validation(w, "cart not ready for checkout: cart must be active to checkout")
		return true
	}
	if errors.Is(err, ErrCartMustHaveItemsForCheckout) {
		httperr.Validation(w, "Cart is empty")
		return true
	}
	if errors.Is(err, ErrCartContactRequiredForCheckout) {
		httperr.Validation(w, "Contact information required")
		return true
	}
	if errors.Is(err, ErrCartPaymentRequiredForCheckout) {
		httperr.Validation(w, "Payment method required")
		return true
	}

	return false
}

func requiredPathParam(w http.ResponseWriter, r *http.Request, key, missingMessage string) (string, bool) {
	value, err := httpx.RequirePathParam(r, key)
	if err != nil {
		if errors.Is(err, httpx.ErrMissingPathParam) {
			httperr.InvalidRequest(w, missingMessage)
			return "", false
		}
		httperr.InvalidRequest(w, missingMessage)
		return "", false
	}

	return value, true
}
