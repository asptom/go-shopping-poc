package product

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"go-shopping-poc/internal/platform/errors"
)

// ProductHandler handles HTTP requests for product operations.
//
// ProductHandler provides RESTful API endpoints for product management,
// including CRUD operations, validation, and proper error handling.
type ProductHandler struct {
	service *ProductService
}

// NewProductHandler creates a new product handler instance.
func NewProductHandler(service *ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

// CreateProduct handles POST /products - creates a new product
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var product Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}

	// Validate required fields
	if product.Name == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "product name is required")
		return
	}

	if product.InitialPrice <= 0 {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "initial price must be greater than 0")
		return
	}

	if err := h.service.CreateProduct(r.Context(), &product); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to create product")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(product); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// GetProduct handles GET /products/{id} - retrieves a product by ID
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
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

// UpdateProduct handles PUT /products/{id} - updates an existing product
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
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

	var product Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}

	// Ensure the ID matches the path parameter
	product.ID = productID

	// Validate required fields for PUT (complete update)
	if product.Name == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "PUT requires complete product record with name")
		return
	}

	if product.InitialPrice <= 0 {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "PUT requires complete product record with valid initial price")
		return
	}

	if err := h.service.UpdateProduct(r.Context(), &product); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to update product")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(product); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// DeleteProduct handles DELETE /products/{id} - deletes a product
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.DeleteProduct(r.Context(), productID); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to delete product")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// IngestProducts handles POST /products/ingest - ingests products from CSV
func (h *ProductHandler) IngestProducts(w http.ResponseWriter, r *http.Request) {
	var req ProductIngestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}

	// Validate required fields
	if req.CSVPath == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "csv_path is required")
		return
	}

	result, err := h.service.IngestProductsFromCSV(r.Context(), &req)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to ingest products from CSV")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// GetProductsByCategory handles GET /products/category/{category} - retrieves products by category
func (h *ProductHandler) GetProductsByCategory(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	if category == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing category in path")
		return
	}

	// Parse query parameters for pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default limit
	offset := 0 // default offset

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		} else {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid limit parameter (must be 1-1000)")
			return
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		} else {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid offset parameter (must be non-negative)")
			return
		}
	}

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

// GetProductsByBrand handles GET /products/brand/{brand} - retrieves products by brand
func (h *ProductHandler) GetProductsByBrand(w http.ResponseWriter, r *http.Request) {
	brand := chi.URLParam(r, "brand")
	if brand == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing brand in path")
		return
	}

	// Parse query parameters for pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default limit
	offset := 0 // default offset

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		} else {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid limit parameter (must be 1-1000)")
			return
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		} else {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid offset parameter (must be non-negative)")
			return
		}
	}

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

// SearchProducts handles GET /products/search - searches products by query
func (h *ProductHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing search query parameter 'q'")
		return
	}

	// Parse query parameters for pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default limit
	offset := 0 // default offset

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		} else {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid limit parameter (must be 1-1000)")
			return
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		} else {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid offset parameter (must be non-negative)")
			return
		}
	}

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

// GetProductsInStock handles GET /products/in-stock - retrieves products that are in stock
func (h *ProductHandler) GetProductsInStock(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default limit
	offset := 0 // default offset

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		} else {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid limit parameter (must be 1-1000)")
			return
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		} else {
			errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid offset parameter (must be non-negative)")
			return
		}
	}

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
