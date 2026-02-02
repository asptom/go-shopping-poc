package product

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"go-shopping-poc/internal/platform/errors"
)

// AdminHandler handles HTTP requests for product admin (write) operations.
//
// AdminHandler provides RESTful API endpoints for product CRUD operations,
// including create, update, delete, and image management.
// Authentication will be added in Phase 6.
type AdminHandler struct {
	service      *AdminService
	urlGenerator *ImageURLGenerator
}

// NewAdminHandler creates a new admin handler instance.
func NewAdminHandler(service *AdminService, urlGenerator *ImageURLGenerator) *AdminHandler {
	return &AdminHandler{
		service:      service,
		urlGenerator: urlGenerator,
	}
}

// CreateProduct handles POST /api/v1/admin/products - creates a new product
func (h *AdminHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var product Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}

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

	// ENRICH: Generate presigned URLs (product may have no images yet)
	h.urlGenerator.EnrichProductWithImageURLs(r.Context(), &product)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(product); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// UpdateProduct handles PUT /api/v1/admin/products/{id} - updates an existing product
func (h *AdminHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
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

	product.ID = productID

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

	// ENRICH: Generate presigned URLs for response
	h.urlGenerator.EnrichProductWithImageURLs(r.Context(), &product)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(product); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// DeleteProduct handles DELETE /api/v1/admin/products/{id} - deletes a product
func (h *AdminHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
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

// AddProductImage handles POST /api/v1/admin/products/{id}/images - adds an image to a product
func (h *AdminHandler) AddProductImage(w http.ResponseWriter, r *http.Request) {
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

	// UPDATED: Use AddSingleImageRequest structure
	var req struct {
		ImageURL string `json:"image_url"`
		IsMain   bool   `json:"is_main"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}

	if req.ImageURL == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "image_url is required")
		return
	}

	// CHANGED: Use AddSingleImage which handles download + Minio upload
	image, err := h.service.AddSingleImage(r.Context(), productID, req.ImageURL, req.IsMain)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to add product image")
		return
	}

	// ENRICH: Generate presigned URL for response
	url, _ := h.urlGenerator.GenerateImageURL(r.Context(), image.MinioObjectName)
	image.ImageURL = url

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(image); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// UpdateProductImage handles PUT /api/v1/admin/products/images/{id} - updates an existing product image
func (h *AdminHandler) UpdateProductImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing image ID in path")
		return
	}

	imageID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid image ID format")
		return
	}

	// CHANGED: Only allow updating metadata (not image_url or minio_object_name)
	var req struct {
		IsMain      bool   `json:"is_main"`
		ImageOrder  int    `json:"image_order"`
		ContentType string `json:"content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}

	// Get existing image to preserve MinioObjectName
	existingImage, err := h.service.GetProductImageByID(r.Context(), imageID)
	if err != nil {
		errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Image not found")
		return
	}

	image := &ProductImage{
		ID:          imageID,
		ProductID:   existingImage.ProductID,
		IsMain:      req.IsMain,
		ImageOrder:  req.ImageOrder,
		ContentType: req.ContentType,
		// Preserve existing MinioObjectName - cannot be changed
		MinioObjectName: existingImage.MinioObjectName,
	}

	if err := h.service.UpdateProductImage(r.Context(), image); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to update product image")
		return
	}

	// ENRICH: Generate presigned URL for response
	url, _ := h.urlGenerator.GenerateImageURL(r.Context(), image.MinioObjectName)
	image.ImageURL = url

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(image); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// DeleteProductImage handles DELETE /api/v1/admin/products/images/{id} - deletes a product image
func (h *AdminHandler) DeleteProductImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing image ID in path")
		return
	}

	imageID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid image ID format")
		return
	}

	if err := h.service.DeleteProductImage(r.Context(), imageID); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to delete product image")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetMainImage handles PUT /api/v1/admin/products/{id}/main-image/{imgId} - sets the main image for a product
func (h *AdminHandler) SetMainImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing product ID in path")
		return
	}

	imgIdStr := chi.URLParam(r, "imgId")
	if imgIdStr == "" {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing image ID in path")
		return
	}

	productID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid product ID format")
		return
	}

	imageID, err := strconv.ParseInt(imgIdStr, 10, 64)
	if err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid image ID format")
		return
	}

	if err := h.service.SetMainImage(r.Context(), productID, imageID); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to set main image")
		return
	}

	// Get the updated image to return with presigned URL
	image, err := h.service.GetProductImageByID(r.Context(), imageID)
	if err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve updated image")
		return
	}

	// ENRICH: Generate presigned URL for response
	url, _ := h.urlGenerator.GenerateImageURL(r.Context(), image.MinioObjectName)
	image.ImageURL = url

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(image); err != nil {
		errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
		return
	}
}

// IngestProducts handles POST /api/v1/admin/products/ingest - ingests products from CSV
func (h *AdminHandler) IngestProducts(w http.ResponseWriter, r *http.Request) {
	var req ProductIngestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
		return
	}

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
