package order

import (
	"errors"
	"net/http"

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

	order, err := h.service.GetOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			httperr.NotFound(w, "Order not found")
			return
		}
		httperr.Internal(w, "Failed to get order")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, order); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

func (h *OrderHandler) GetOrdersByCustomer(w http.ResponseWriter, r *http.Request) {
	customerID, ok := requiredPathParam(w, r, "customerId", "Missing customer_id parameter")
	if !ok {
		return
	}

	orders, err := h.service.GetOrdersByCustomer(r.Context(), customerID)
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

	err := h.service.CancelOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			httperr.NotFound(w, "Order not found")
			return
		}
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

	var req UpdateStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httperr.InvalidRequest(w, "Invalid JSON in request body")
		return
	}

	if req.Status == "" {
		httperr.Validation(w, "status is required")
		return
	}

	err := h.service.UpdateOrderStatus(r.Context(), orderID, req.Status)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			httperr.NotFound(w, "Order not found")
			return
		}
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
