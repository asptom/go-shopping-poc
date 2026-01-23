package product

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"go-shopping-poc/internal/platform/errors"
)

// CatalogHandler handles HTTP requests for product catalog (read-only) operations.
//
// CatalogHandler provides RESTful API endpoints for product browsing,
// including list, search, and filter operations.
type CatalogHandler struct {
	service *CatalogService
}

// NewCatalogHandler creates a new catalog handler instance.
func NewCatalogHandler(service *CatalogService) *CatalogHandler {
	return &CatalogHandler{service: service}
}

// GetAllProducts handles GET /api/v1/products - lists all products with pagination
func (h *CatalogHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	products, err := h.service.GetAllProducts(r.Context(), limit, offset)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve products")
		return
	}

	response := map[string]interface{}{
		"products": products,
		"limit":    limit,
		"offset":   offset,
		"count":    len(products),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// GetProduct handles GET /api/v1/products/{id} - retrieves a product by ID
func (h *CatalogHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing product ID in path")
		return
	}

	productID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid product ID format")
		return
	}

	if productID <= 0 {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Product ID must be positive")
		return
	}

	product, err := h.service.GetProductByID(r.Context(), productID)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve product")
		return
	}

	if product == nil {
		errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Product not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(product); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// GetProductsByCategory handles GET /api/v1/products/category/{category} - retrieves products by category
func (h *CatalogHandler) GetProductsByCategory(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	if category == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing category in path")
		return
	}

	limit, offset := parsePagination(r)

	products, err := h.service.GetProductsByCategory(r.Context(), category, limit, offset)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve products by category")
		return
	}

	response := map[string]interface{}{
		"products": products,
		"category": category,
		"limit":    limit,
		"offset":   offset,
		"count":    len(products),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// GetProductsByBrand handles GET /api/v1/products/brand/{brand} - retrieves products by brand
func (h *CatalogHandler) GetProductsByBrand(w http.ResponseWriter, r *http.Request) {
	brand := chi.URLParam(r, "brand")
	if brand == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing brand in path")
		return
	}

	limit, offset := parsePagination(r)

	products, err := h.service.GetProductsByBrand(r.Context(), brand, limit, offset)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve products by brand")
		return
	}

	response := map[string]interface{}{
		"products": products,
		"brand":    brand,
		"limit":    limit,
		"offset":   offset,
		"count":    len(products),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// SearchProducts handles GET /api/v1/products/search - searches products by query
func (h *CatalogHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing search query parameter 'q'")
		return
	}

	limit, offset := parsePagination(r)

	products, err := h.service.SearchProducts(r.Context(), query, limit, offset)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to search products")
		return
	}

	response := map[string]interface{}{
		"products": products,
		"query":    query,
		"limit":    limit,
		"offset":   offset,
		"count":    len(products),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// GetProductsInStock handles GET /api/v1/products/in-stock - retrieves in-stock products
func (h *CatalogHandler) GetProductsInStock(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	products, err := h.service.GetProductsInStock(r.Context(), limit, offset)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve in-stock products")
		return
	}

	response := map[string]interface{}{
		"products": products,
		"limit":    limit,
		"offset":   offset,
		"count":    len(products),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// parsePagination extracts and validates pagination parameters from request
func parsePagination(r *http.Request) (limit, offset int) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit = 50
	offset = 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	return limit, offset
}
