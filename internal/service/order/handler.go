package order

import (
	"errors"
	"net/http"

	"go-shopping-poc/internal/platform/auth"
	"go-shopping-poc/internal/platform/httperr"
	"go-shopping-poc/internal/platform/httpx"
)

type OrderHandler struct {
	service *OrderService
}

func NewOrderHandler(service *OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := requiredPathParam(w, r, "id", "Missing order ID")
	if !ok {
		return
	}

	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	identity, err := h.service.VerifyCustomerIdentity(r.Context(), claims)
	if err != nil {
		httperr.Forbidden(w, "authentication failed")
		return
	}

	order, err := h.service.GetOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			httperr.NotFound(w, "Order not found")
			return
		}
		httperr.Internal(w, "Failed to get order")
		return
	}

	// Defense in depth: verify order belongs to authenticated customer
	if order.CustomerID != nil && order.CustomerID.String() != identity.CustomerID {
		httperr.Forbidden(w, "you can only access your own orders")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, order); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

func (h *OrderHandler) GetOrdersByCustomer(w http.ResponseWriter, r *http.Request) {
	customerId, ok := requiredPathParam(w, r, "customerId", "Missing customer_id parameter")
	if !ok {
		return
	}

	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	identity, err := h.service.VerifyCustomerIdentity(r.Context(), claims)
	if err != nil {
		if err.Error() == "identity verification timed out after 5s" {
			httperr.GatewayTimeout(w, "identity verification failed: timeout")
		} else {
			httperr.Forbidden(w, "cannot verify customer identity")
		}
		return
	}

	if identity.CustomerID != customerId {
		httperr.Forbidden(w, "you can only access your own orders")
		return
	}

	orders, err := h.service.GetOrdersByCustomer(r.Context(), identity.CustomerID)
	if err != nil {
		httperr.Internal(w, "Failed to get orders")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, orders); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID, ok := requiredPathParam(w, r, "id", "Missing order ID")
	if !ok {
		return
	}

	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	identity, err := h.service.VerifyCustomerIdentity(r.Context(), claims)
	if err != nil {
		httperr.Forbidden(w, "authentication failed")
		return
	}

	// Verify ownership before cancellation
	order, err := h.service.GetOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			httperr.NotFound(w, "Order not found")
			return
		}
		httperr.Internal(w, "Failed to get order")
		return
	}
	if order.CustomerID != nil && order.CustomerID.String() != identity.CustomerID {
		httperr.Forbidden(w, "you can only cancel your own orders")
		return
	}

	err = h.service.CancelOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, ErrOrderCannotBeCancelled) {
			httperr.Validation(w, err.Error())
			return
		}
		httperr.Internal(w, "Failed to cancel order")
		return
	}

	httpx.WriteNoContent(w)
}

func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID, ok := requiredPathParam(w, r, "id", "Missing order ID")
	if !ok {
		return
	}

	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	identity, err := h.service.VerifyCustomerIdentity(r.Context(), claims)
	if err != nil {
		httperr.Forbidden(w, "authentication failed")
		return
	}

	var req UpdateStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON in request body")
		return
	}

	if req.Status == "" {
		httperr.Validation(w, "status is required")
		return
	}

	order, err := h.service.GetOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			httperr.NotFound(w, "Order not found")
			return
		}
		httperr.Internal(w, "Failed to get order")
		return
	}

	// Verify ownership - customer can only update their own orders
	if order.CustomerID != nil && order.CustomerID.String() != identity.CustomerID {
		httperr.Forbidden(w, "you can only update your own orders")
		return
	}

	err = h.service.UpdateOrderStatus(r.Context(), orderID, req.Status)
	if err != nil {
		if errors.Is(err, ErrInvalidStatusTransition) {
			httperr.Validation(w, err.Error())
			return
		}
		httperr.Internal(w, "Failed to update order status")
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

type UpdateStatusRequest struct {
	Status string `json:"status"`
}
