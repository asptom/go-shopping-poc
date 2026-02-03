package product

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"go-shopping-poc/internal/platform/errors"
	"go-shopping-poc/internal/platform/storage/minio"
)

// CatalogHandler handles HTTP requests for product catalog (read-only) operations.
//
// CatalogHandler provides RESTful API endpoints for product browsing,
// including list, search, and filter operations.
type CatalogHandler struct {
	service       *CatalogService
	objectStorage minio.ObjectStorage
	bucket        string
}

// NewCatalogHandler creates a new catalog handler instance.
func NewCatalogHandler(service *CatalogService, objectStorage minio.ObjectStorage, bucket string) *CatalogHandler {
	return &CatalogHandler{
		service:       service,
		objectStorage: objectStorage,
		bucket:        bucket,
	}
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

// GetProductImages handles GET /api/v1/products/{id}/images - lists all images
func (h *CatalogHandler) GetProductImages(w http.ResponseWriter, r *http.Request) {
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

	// Get product with images loaded
	product, err := h.service.GetProductByID(r.Context(), productID)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve product")
		return
	}

	if product == nil {
		errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Product not found")
		return
	}

	if len(product.Images) == 0 {
		errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "No images found for product")
		return
	}

	response := map[string]interface{}{
		"product_id": productID,
		"images":     product.Images,
		"count":      len(product.Images),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// GetProductMainImage handles GET /api/v1/products/{id}/main-image - returns main image
func (h *CatalogHandler) GetProductMainImage(w http.ResponseWriter, r *http.Request) {
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

	// Get product to access images via the loaded relationship
	product, err := h.service.GetProductByID(r.Context(), productID)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve product")
		return
	}

	if product == nil {
		errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Product not found")
		return
	}

	// Find main image
	var mainImage *ProductImage
	for i := range product.Images {
		if product.Images[i].IsMain {
			mainImage = &product.Images[i]
			break
		}
	}

	if mainImage == nil {
		errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "No main image found for product")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mainImage); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// GetDirectImage handles GET /api/v1/products/{id}/images/{imageName:.+} - streams image directly from Minio
// Example: /api/v1/products/40121298/images/image_0.jpg -> object name: products/40121298/image_0.jpg
func (h *CatalogHandler) GetDirectImage(w http.ResponseWriter, r *http.Request) {
	// Log the full request info for debugging
	log.Printf("[DEBUG] GetDirectImage: Handler invoked!")
	log.Printf("[DEBUG] GetDirectImage: Method=%s, URL=%s, Path=%s", r.Method, r.URL.String(), r.URL.Path)

	// Extract product ID and image name from URL parameters
	productIDStr := chi.URLParam(r, "id")
	imageName := chi.URLParam(r, "imageName")

	log.Printf("[DEBUG] GetDirectImage: URL params - productID='%s', imageName='%s'", productIDStr, imageName)

	if productIDStr == "" || imageName == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing product ID or image name in path")
		return
	}

	// Validate product ID is numeric
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil || productID <= 0 {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid product ID format")
		return
	}

	// Security: Validate image name format (prevent path traversal)
	if strings.Contains(imageName, "..") || strings.HasPrefix(imageName, "/") {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid image name")
		return
	}

	// Construct Minio object name: products/{productID}/{imageName}
	objectName := fmt.Sprintf("products/%d/%s", productID, imageName)
	log.Printf("[DEBUG] GetDirectImage: Constructed objectName: '%s'", objectName)

	// Log bucket and object details
	log.Printf("[DEBUG] GetDirectImage: Attempting to get object from bucket '%s', objectName: '%s'", h.bucket, objectName)

	// Get object from Minio
	object, err := h.objectStorage.GetObject(r.Context(), h.bucket, objectName)
	if err != nil {
		log.Printf("[ERROR] GetDirectImage: Failed to get object from Minio: %v", err)
		errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Image not found")
		return
	}
	defer object.Close()

	// Get object info for Content-Type
	info, err := h.objectStorage.StatObject(r.Context(), h.bucket, objectName)
	if err == nil && info.ContentType != "" {
		w.Header().Set("Content-Type", info.ContentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Set cache headers
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if err == nil {
		w.Header().Set("ETag", info.ETag)
	}

	// Stream object to response
	if _, err := io.Copy(w, object); err != nil {
		log.Printf("[ERROR] Failed to stream image: %v", err)
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
