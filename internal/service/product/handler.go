package product

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-shopping-poc/internal/platform/httperr"
	"go-shopping-poc/internal/platform/httpx"
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
	logger        *slog.Logger
}

// NewCatalogHandler creates a new catalog handler instance.
func NewCatalogHandler(logger *slog.Logger, service *CatalogService, objectStorage minio.ObjectStorage, bucket string) *CatalogHandler {
	return &CatalogHandler{
		service:       service,
		objectStorage: objectStorage,
		bucket:        bucket,
		logger:        logger.With("component", "catalog_handler"),
	}
}

// GetAllProducts handles GET /api/v1/products - lists all products with pagination
func (h *CatalogHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	products, err := h.service.GetAllProducts(r.Context(), limit, offset)
	if err != nil {
		httperr.Internal(w, "Failed to retrieve products")
		return
	}

	response := map[string]interface{}{
		"products": products,
		"limit":    limit,
		"offset":   offset,
		"count":    len(products),
	}

	if err := httpx.WriteJSON(w, http.StatusOK, response); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

// GetProduct handles GET /api/v1/products/{id} - retrieves a product by ID
func (h *CatalogHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	idStr, ok := requiredPathParam(w, r, "id", "Missing product ID in path")
	if !ok {
		return
	}

	productID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httperr.Validation(w, "Invalid product ID format")
		return
	}

	if productID <= 0 {
		httperr.Validation(w, "Product ID must be positive")
		return
	}

	product, err := h.service.GetProductByID(r.Context(), productID)
	if err != nil {
		httperr.Internal(w, "Failed to retrieve product")
		return
	}

	if product == nil {
		httperr.NotFound(w, "Product not found")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, product); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

// GetProductsByCategory handles GET /api/v1/products/category/{category} - retrieves products by category
func (h *CatalogHandler) GetProductsByCategory(w http.ResponseWriter, r *http.Request) {
	category, ok := requiredPathParam(w, r, "category", "Missing category in path")
	if !ok {
		return
	}

	limit, offset := parsePagination(r)

	products, err := h.service.GetProductsByCategory(r.Context(), category, limit, offset)
	if err != nil {
		httperr.Internal(w, "Failed to retrieve products by category")
		return
	}

	response := map[string]interface{}{
		"products": products,
		"category": category,
		"limit":    limit,
		"offset":   offset,
		"count":    len(products),
	}

	if err := httpx.WriteJSON(w, http.StatusOK, response); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

// GetProductsByBrand handles GET /api/v1/products/brand/{brand} - retrieves products by brand
func (h *CatalogHandler) GetProductsByBrand(w http.ResponseWriter, r *http.Request) {
	brand, ok := requiredPathParam(w, r, "brand", "Missing brand in path")
	if !ok {
		return
	}

	limit, offset := parsePagination(r)

	products, err := h.service.GetProductsByBrand(r.Context(), brand, limit, offset)
	if err != nil {
		httperr.Internal(w, "Failed to retrieve products by brand")
		return
	}

	response := map[string]interface{}{
		"products": products,
		"brand":    brand,
		"limit":    limit,
		"offset":   offset,
		"count":    len(products),
	}

	if err := httpx.WriteJSON(w, http.StatusOK, response); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

// SearchProducts handles GET /api/v1/products/search - searches products by query
func (h *CatalogHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	query, ok := requiredQueryParam(w, r, "q", "Missing search query parameter 'q'")
	if !ok {
		return
	}

	limit, offset := parsePagination(r)

	products, err := h.service.SearchProducts(r.Context(), query, limit, offset)
	if err != nil {
		httperr.Internal(w, "Failed to search products")
		return
	}

	response := map[string]interface{}{
		"products": products,
		"query":    query,
		"limit":    limit,
		"offset":   offset,
		"count":    len(products),
	}

	if err := httpx.WriteJSON(w, http.StatusOK, response); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

// GetProductsInStock handles GET /api/v1/products/in-stock - retrieves in-stock products
func (h *CatalogHandler) GetProductsInStock(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	products, err := h.service.GetProductsInStock(r.Context(), limit, offset)
	if err != nil {
		httperr.Internal(w, "Failed to retrieve in-stock products")
		return
	}

	response := map[string]interface{}{
		"products": products,
		"limit":    limit,
		"offset":   offset,
		"count":    len(products),
	}

	if err := httpx.WriteJSON(w, http.StatusOK, response); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

// GetProductImages handles GET /api/v1/products/{id}/images - lists all images
func (h *CatalogHandler) GetProductImages(w http.ResponseWriter, r *http.Request) {
	idStr, ok := requiredPathParam(w, r, "id", "Missing product ID in path")
	if !ok {
		return
	}

	productID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httperr.Validation(w, "Invalid product ID format")
		return
	}

	if productID <= 0 {
		httperr.Validation(w, "Product ID must be positive")
		return
	}

	// Get product with images loaded
	product, err := h.service.GetProductByID(r.Context(), productID)
	if err != nil {
		httperr.Internal(w, "Failed to retrieve product")
		return
	}

	if product == nil {
		httperr.NotFound(w, "Product not found")
		return
	}

	if len(product.Images) == 0 {
		httperr.NotFound(w, "No images found for product")
		return
	}

	response := map[string]interface{}{
		"product_id": productID,
		"images":     product.Images,
		"count":      len(product.Images),
	}

	if err := httpx.WriteJSON(w, http.StatusOK, response); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

// GetProductMainImage handles GET /api/v1/products/{id}/main-image - returns main image
func (h *CatalogHandler) GetProductMainImage(w http.ResponseWriter, r *http.Request) {
	idStr, ok := requiredPathParam(w, r, "id", "Missing product ID in path")
	if !ok {
		return
	}

	productID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httperr.Validation(w, "Invalid product ID format")
		return
	}

	if productID <= 0 {
		httperr.Validation(w, "Product ID must be positive")
		return
	}

	// Get product to access images via the loaded relationship
	product, err := h.service.GetProductByID(r.Context(), productID)
	if err != nil {
		httperr.Internal(w, "Failed to retrieve product")
		return
	}

	if product == nil {
		httperr.NotFound(w, "Product not found")
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
		httperr.NotFound(w, "No main image found for product")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, mainImage); err != nil {
		httperr.Internal(w, "Failed to encode response")
		return
	}
}

// GetDirectImage handles GET /api/v1/products/{id}/images/{imageName:.+} - streams image directly from Minio
// Example: /api/v1/products/40121298/images/image_0.jpg -> object name: products/40121298/image_0.jpg
func (h *CatalogHandler) GetDirectImage(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	requestID := requestIDFromRequest(r)
	log := h.logger.With("operation", "get_direct_image", "request_id", requestID)
	log.Debug("Direct image request received", "method", r.Method, "path", r.URL.Path)

	// Extract product ID and image name from URL parameters
	productIDStr, ok := requiredPathParam(w, r, "id", "Missing product ID or image name in path")
	if !ok {
		return
	}
	imageName, ok := requiredPathParam(w, r, "imageName", "Missing product ID or image name in path")
	if !ok {
		return
	}

	log.Debug("Direct image params parsed", "product_id", productIDStr, "image_name", imageName)

	// Validate product ID is numeric
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil || productID <= 0 {
		httperr.Validation(w, "Invalid product ID format")
		return
	}

	// Security: Validate image name format (prevent path traversal)
	if strings.Contains(imageName, "..") || strings.HasPrefix(imageName, "/") {
		httperr.Validation(w, "Invalid image name")
		return
	}

	// Construct Minio object name: products/{productID}/{imageName}
	objectName := fmt.Sprintf("products/%d/%s", productID, imageName)
	log.Debug("Direct image object resolved", "object_name", objectName)

	// Log bucket and object details
	log.Debug("Direct image fetch started", "bucket", h.bucket, "object_name", objectName)

	// Get object from Minio
	object, err := h.objectStorage.GetObject(r.Context(), h.bucket, objectName)
	if err != nil {
		log.Error("Direct image fetch failed", "error", err.Error())
		httperr.NotFound(w, "Image not found")
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
		log.Error("Direct image stream failed", "error", err.Error())
		return
	}

	log.Info("Direct image streamed", "product_id", productID, "image_name", imageName, "duration_ms", time.Since(startedAt).Milliseconds())
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

func requestIDFromRequest(r *http.Request) string {
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		return requestID
	}
	if requestID := r.Header.Get("X-Request-Id"); requestID != "" {
		return requestID
	}

	return ""
}
