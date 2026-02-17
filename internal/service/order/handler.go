package order

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"go-shopping-poc/internal/platform/errors"
)

type OrderHandler struct {
	service *OrderService
}

func NewOrderHandler(service *OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing order ID")
		return
	}

	order, err := h.service.GetOrder(r.Context(), orderID)
	if err != nil {
		if err == ErrOrderNotFound {
			errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Order not found")
			return
		}
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to get order")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

func (h *OrderHandler) GetOrdersByCustomer(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	if customerID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing customer_id parameter")
		return
	}

	orders, err := h.service.GetOrdersByCustomer(r.Context(), customerID)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to get orders")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing order ID")
		return
	}

	err := h.service.CancelOrder(r.Context(), orderID)
	if err != nil {
		if err == ErrOrderNotFound {
			errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Order not found")
			return
		}
		if err == ErrOrderCannotBeCancelled {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, err.Error())
			return
		}
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to cancel order")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing order ID")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}

	if req.Status == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "status is required")
		return
	}

	err := h.service.UpdateOrderStatus(r.Context(), orderID, req.Status)
	if err != nil {
		if err == ErrOrderNotFound {
			errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Order not found")
			return
		}
		if err == ErrInvalidStatusTransition {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, err.Error())
			return
		}
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to update order status")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}
