## **Revised Phased Implementation Plan: Minio-Only Image Serving v3**

### **Overview**

This plan outlines the migration from external URL-based image storage to Minio-only object storage with handler-layer presigned URL generation. Breaking changes are expected; all data will be reloaded from CSV after implementation.

**Key Assumptions:**
- No existing product/data in database (can safely rebuild)
- Breaking changes allowed
- Existing CSV format remains (backward compatible to CSV, new schema in DB)
- Product-loader: bulk import mechanism for products and images
- Product-admin: CRUD operations for products AND individual image management
- Image serving: presigned URL strings returned via API endpoints
- Handler layer responsible for URL enrichment (clean separation of concerns)

**Critical Design Decisions:**
1. **Admin service retains individual image management endpoints** (POST/PUT/DELETE images)
2. **Handler layer generates presigned URLs** (services return raw entities, handlers enrich with URLs)
3. **MinioObjectName is required, ImageURL is optional** (stored as empty string)

---

## **Phase 1: Database Schema Migration**

**Goal:** Update database schema to support Minio-only storage

**Prerequisites:** None

**Deliverables:** Updated migration file ready for execution

### **Step 1.1: Update Migration File**
**File:** `internal/service/product/migrations/001-init.sql`

**Actions:**
1. Remove `main_image` column from `products` table (lines 25)
2. Modify `product_images` table:
   - Remove `NOT NULL` constraint from `image_url` column (line 42)
   - Remove `UNIQUE(product_id, image_url)` constraint (line 49)
   - Make `minio_object_name` `NOT NULL` (line 43)
   - Add `UNIQUE(product_id, minio_object_name)` constraint

**SQL Changes:**
```sql
-- products table - REMOVE main_image column
DROP TABLE IF EXISTS products.Products;
CREATE TABLE products.Products (
    id BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    initial_price DECIMAL(10,2),
    final_price DECIMAL(10,2),
    currency VARCHAR(3) DEFAULT 'USD',
    in_stock BOOLEAN DEFAULT true,
    color VARCHAR(100),
    size VARCHAR(100),
    -- REMOVED: main_image TEXT,
    country_code VARCHAR(2),
    image_count INTEGER,
    model_number VARCHAR(100),
    root_category VARCHAR(100),
    category VARCHAR(100),
    brand VARCHAR(100),
    other_attributes TEXT,
    all_available_sizes JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Product images table - UPDATE constraints
CREATE TABLE products.Product_images (
    id SERIAL PRIMARY KEY,
    product_id BIGINT REFERENCES products.Products(id) ON DELETE CASCADE,
    image_url TEXT,  -- REMOVED: NOT NULL constraint
    minio_object_name VARCHAR(500) NOT NULL,  -- ADDED: NOT NULL
    is_main BOOLEAN DEFAULT false,
    image_order INTEGER DEFAULT 0,
    file_size BIGINT,
    content_type VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, minio_object_name)  -- CHANGED: from image_url to minio_object_name
);
```

### **Step 1.2: Add Image Count Trigger (Optional Enhancement)**
**File:** `internal/service/product/migrations/001-init.sql`

Add trigger to auto-maintain `products.image_count`:
```sql
-- Trigger to update image_count on products
CREATE OR REPLACE FUNCTION update_image_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE products.Products SET image_count = image_count + 1 WHERE id = NEW.product_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE products.Products SET image_count = image_count - 1 WHERE id = OLD.product_id;
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' AND NEW.product_id != OLD.product_id THEN
        UPDATE products.Products SET image_count = image_count - 1 WHERE id = OLD.product_id;
        UPDATE products.Products SET image_count = image_count + 1 WHERE id = NEW.product_id;
        RETURN NEW;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_product_image_count
    AFTER INSERT OR DELETE OR UPDATE ON products.Product_images
    FOR EACH ROW EXECUTE FUNCTION update_image_count();
```

**Rationale:** Keeps `image_count` synchronized automatically; eliminates manual updates in repository layer.

---

## **Phase 2: Entity and Validation Updates**

**Goal:** Update domain entities to reflect Minio-only storage

**Prerequisites:** Phase 1 complete

**Deliverables:** Updated entity.go with new validation rules

### **Step 2.1: Update Product Entity**
**File:** `internal/service/product/entity.go`

**Actions:**
1. Remove `MainImage` field from `Product` struct (line 29)
2. Update `GetMainImage()` method (lines 133-147) to:
   - Remove check for `p.MainImage` field
   - Iterate over `p.Images` slice and return first image where `IsMain=true`
   - Return empty string if no main image found (caller must generate presigned URL)

**Code Changes:**
```go
// REMOVE this field from Product struct:
// MainImage         string        `json:"main_image" db:"main_image"`

// UPDATE GetMainImage method:
func (p *Product) GetMainImage() string {
    // Iterate over Images slice for main image
    for _, img := range p.Images {
        if img.IsMain {
            return img.MinioObjectName  // Return object name, not URL
        }
    }
    return ""
}
```

### **Step 2.2: Update ProductImage Entity**
**File:** `internal/service/product/entity.go`

**Actions:**
1. Keep `ImageURL` field but make it transient (not persisted)
2. Update `Validate()` method (lines 207-237) to:
   - Remove validation requiring `ImageURL` (lines 212-218)
   - Add validation requiring `MinioObjectName` (must be non-empty and ≤500 chars)
   - Keep existing validation for other fields

**Code Changes:**
```go
// UPDATE Validate method for ProductImage:
func (pi *ProductImage) Validate() error {
    if pi.ProductID <= 0 {
        return errors.New("product ID is required and must be positive")
    }

    // CHANGED: Require MinioObjectName instead of ImageURL
    if strings.TrimSpace(pi.MinioObjectName) == "" {
        return errors.New("minio object name is required")
    }

    if len(pi.MinioObjectName) > 500 {
        return errors.New("MinIO object name must be 500 characters or less")
    }

    // ImageURL is now optional (transient field)
    if pi.ImageURL != "" && len(pi.ImageURL) > 2000 {
        return errors.New("image URL must be 2000 characters or less")
    }

    if pi.ImageOrder < 0 {
        return errors.New("image order cannot be negative")
    }

    if pi.FileSize < 0 {
        return errors.New("file size cannot be negative")
    }

    if pi.ContentType != "" && len(pi.ContentType) > 100 {
        return errors.New("content type must be 100 characters or less")
    }

    return nil
}
```

**Rationale:** `ImageURL` becomes a computed/transient field populated at handler layer with presigned URLs.

---

## **Phase 3: Repository Layer Updates**

**Goal:** Update repository methods to work with new schema

**Prerequisites:** Phase 2 complete

**Deliverables:** Updated repository files

### **Step 3.1: Update Image Repository Methods**
**File:** `internal/service/product/repository_image.go`

**Actions:**
1. Update `AddProductImage()` (lines 14-46):
   - Remove `image_url` from INSERT query (line 28)
   - Remove `ImageURL` from ExecContext parameters (line 35)
   - Update query to only insert: `product_id, minio_object_name, is_main, image_order, file_size, content_type, created_at`

2. Update `UpdateProductImage()` (lines 49-83):
   - Remove `image_url` from UPDATE query (line 62)
   - Remove `ImageURL` from ExecContext parameters (line 67)

3. Update `GetProductImages()` (lines 111-149):
   - Remove `image_url` from SELECT query (line 119)
   - Remove `ImageURL` from Scan operation (line 135)

4. **REMOVE** `SetMainImage()` method entirely (lines 152-205):
   - This method updates the now-removed `products.main_image` column
   - Main image logic moves to `is_main` flag in `product_images` table only

**Code Changes for AddProductImage:**
```go
func (r *productRepository) AddProductImage(ctx context.Context, image *ProductImage) error {
    log.Printf("[DEBUG] Repository: Adding image for product %d", image.ProductID)

    if err := image.Validate(); err != nil {
        return fmt.Errorf("image validation failed: %w", err)
    }

    _, err := r.GetProductByID(ctx, image.ProductID)
    if err != nil {
        return fmt.Errorf("cannot add image to non-existent product: %w", err)
    }

    // UPDATED: Removed image_url from query
    query := `
        INSERT INTO products.product_images (
            product_id, minio_object_name, is_main, image_order,
            file_size, content_type, created_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7
        )`

    // UPDATED: Removed image.ImageURL from parameters
    _, err = r.db.ExecContext(ctx, query,
        image.ProductID, image.MinioObjectName, image.IsMain,
        image.ImageOrder, image.FileSize, image.ContentType, image.CreatedAt,
    )
    if err != nil {
        if isDuplicateError(err) {
            return fmt.Errorf("%w: minio object name already exists for product", ErrDuplicateImage)
        }
        return fmt.Errorf("%w: failed to add product image: %v", ErrDatabaseOperation, err)
    }

    return nil
}
```

### **Step 3.2: Add Repository Method for Setting Main Image Flag**
**File:** `internal/service/product/repository_image.go`

**Add New Method:**
```go
// SetMainImageFlag sets the is_main flag for an image without updating product.main_image
func (r *productRepository) SetMainImageFlag(ctx context.Context, productID int64, imageID int64) error {
    log.Printf("[DEBUG] Repository: Setting main image flag %d for product %d", imageID, productID)

    if productID <= 0 || imageID <= 0 {
        return fmt.Errorf("%w: product ID and image ID must be positive", ErrInvalidProductID)
    }

    tx, err := r.db.BeginTxx(ctx, nil)
    if err != nil {
        return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
    }
    committed := false
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()

    // Unset all main images for this product
    unsetQuery := `UPDATE products.product_images SET is_main = false WHERE product_id = $1`
    _, err = tx.ExecContext(ctx, unsetQuery, productID)
    if err != nil {
        return fmt.Errorf("%w: failed to unset main images: %v", ErrDatabaseOperation, err)
    }

    // Set the specified image as main
    setQuery := `UPDATE products.product_images SET is_main = true WHERE id = $1 AND product_id = $2`
    result, err := tx.ExecContext(ctx, setQuery, imageID, productID)
    if err != nil {
        return fmt.Errorf("%w: failed to set main image flag: %v", ErrDatabaseOperation, err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
    }
    if rowsAffected == 0 {
        return fmt.Errorf("%w: image %d not found for product %d", ErrProductImageNotFound, imageID, productID)
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
    }
    committed = true

    return nil
}
```

### **Step 3.3: Update Repository Interface**
**File:** `internal/service/product/repository.go`

**Actions:**
1. Update `ProductRepository` interface:
   - Replace `SetMainImage(ctx context.Context, productID int64, imageID int64) error` 
   - With `SetMainImageFlag(ctx context.Context, productID int64, imageID int64) error`

**Code:**
```go
type ProductRepository interface {
    // ... existing methods ...
    
    // UPDATED: Changed from SetMainImage to SetMainImageFlag
    SetMainImageFlag(ctx context.Context, productID int64, imageID int64) error
}
```

---

## **Phase 4: Admin Service Updates**

**Goal:** Update admin service to work with new schema and handle Minio uploads

**Prerequisites:** Phase 3 complete

**Deliverables:** Updated service_admin.go

### **Step 4.1: Update Image Processing Logic**
**File:** `internal/service/product/service_admin.go`

**Actions:**
1. Update `processSingleImage()` method (lines 374-425):
   - Change `ImageURL: imageURL` to `ImageURL: ""` (empty string, not stored)
   - Ensure `IsMain` is set correctly based on `main_image` column match from CSV
   - Keep Minio upload logic (already implemented)

**Code Changes:**
```go
func (s *AdminService) processSingleImage(ctx context.Context, product *Product, imageURL string, imageIndex int, useCache bool, result *ProductIngestionResult) error {
    // ... existing download/upload logic ...

    productImage := &ProductImage{
        ProductID:       product.ID,
        ImageURL:        "",  // CHANGED: Empty string - not stored in DB
        MinioObjectName: minioObjectName,
        IsMain:          imageURL == product.MainImage,  // Match against CSV main_image column
        ImageOrder:      imageIndex,
        FileSize:        fileSize,
        ContentType:     contentType,
    }

    if s.repo != nil {
        if err := s.repo.AddProductImage(ctx, productImage); err != nil {
            return fmt.Errorf("failed to insert product image: %w", err)
        }
    }

    // Note: Event still publishes original URL for audit/traceability
    if err := s.publishProductImageAddedEvent(ctx, product.ID, productImage.ID, imageURL); err != nil {
        log.Printf("[WARN] AdminService: Failed to publish image added event: %v", err)
    }

    log.Printf("[DEBUG] AdminService: Successfully processed image %d for product %d", imageIndex, product.ID)
    return nil
}
```

### **Step 4.2: Update SetMainImage Service Method**
**File:** `internal/service/product/service_admin.go`

**Actions:**
1. Update `SetMainImage()` method (lines 212-222):
   - Change call from `s.repo.SetMainImage()` to `s.repo.SetMainImageFlag()`

**Code:**
```go
func (s *AdminService) SetMainImage(ctx context.Context, productID int64, imageID int64) error {
    log.Printf("[INFO] AdminService: Setting main image %d for product %d", imageID, productID)

    // CHANGED: Use SetMainImageFlag instead of SetMainImage
    if err := s.repo.SetMainImageFlag(ctx, productID, imageID); err != nil {
        log.Printf("[ERROR] AdminService: Failed to set main image: %v", err)
        return fmt.Errorf("failed to set main image: %w", err)
    }

    return nil
}
```

### **Step 4.3: Update Single Image CRUD Methods**
**File:** `internal/service/product/service_admin.go`

**Actions:**
1. **Add method** `AddSingleImage()` for admin API to add individual images:
```go
// AddSingleImage adds a single image to a product by downloading from URL and uploading to Minio
func (s *AdminService) AddSingleImage(ctx context.Context, productID int64, imageURL string, isMain bool) (*ProductImage, error) {
    log.Printf("[INFO] AdminService: Adding single image for product %d from URL", productID)

    if productID <= 0 {
        return nil, fmt.Errorf("product ID must be positive")
    }

    if imageURL == "" {
        return nil, fmt.Errorf("image URL is required")
    }

    // Download image
    localPath, err := s.infrastructure.HTTPDownloader.Download(ctx, imageURL)
    if err != nil {
        return nil, fmt.Errorf("failed to download image: %w", err)
    }

    // Get existing image count for ordering
    existingImages, err := s.repo.GetProductImages(ctx, productID)
    if err != nil {
        return nil, fmt.Errorf("failed to get existing images: %w", err)
    }
    imageOrder := len(existingImages)

    // Upload to Minio
    minioObjectName, err := s.uploadImageToMinIO(ctx, localPath, productID, imageOrder)
    if err != nil {
        return nil, fmt.Errorf("failed to upload image to MinIO: %w", err)
    }

    // Get file info
    fileSize, contentType, err := s.getImageInfo(localPath)
    if err != nil {
        return nil, fmt.Errorf("failed to get image info: %w", err)
    }

    // Handle is_main logic - if this is main, unset others
    if isMain {
        if err := s.unsetAllMainImages(ctx, productID); err != nil {
            log.Printf("[WARN] AdminService: Failed to unset existing main images: %v", err)
        }
    }

    // Create image record
    productImage := &ProductImage{
        ProductID:       productID,
        ImageURL:        "",  // Not stored
        MinioObjectName: minioObjectName,
        IsMain:          isMain,
        ImageOrder:      imageOrder,
        FileSize:        fileSize,
        ContentType:     contentType,
    }

    if err := s.repo.AddProductImage(ctx, productImage); err != nil {
        return nil, fmt.Errorf("failed to add product image: %w", err)
    }

    if err := s.publishProductImageAddedEvent(ctx, productID, productImage.ID, imageURL); err != nil {
        log.Printf("[WARN] AdminService: Failed to publish image added event: %v", err)
    }

    log.Printf("[INFO] AdminService: Successfully added image %d to product %d", productImage.ID, productID)
    return productImage, nil
}

// unsetAllMainImages unsets the is_main flag for all images of a product
func (s *AdminService) unsetAllMainImages(ctx context.Context, productID int64) error {
    query := `UPDATE products.product_images SET is_main = false WHERE product_id = $1`
    _, err := s.infrastructure.Database.ExecContext(ctx, query, productID)
    return err
}
```

2. **Update** `DeleteProductImage()` method (lines 197-210) to delete from Minio:
```go
func (s *AdminService) DeleteProductImage(ctx context.Context, imageID int64) error {
    log.Printf("[INFO] AdminService: Deleting image %d", imageID)

    // Get image details first (for Minio deletion)
    images, err := s.repo.GetProductImages(ctx, 0) // Need to get by image ID
    // TODO: Need to add GetProductImageByID repository method
    
    var imageToDelete *ProductImage
    for _, img := range images {
        if img.ID == imageID {
            imageToDelete = &img
            break
        }
    }
    
    if imageToDelete == nil {
        return fmt.Errorf("image %d not found", imageID)
    }

    // Delete from database first
    if err := s.repo.DeleteProductImage(ctx, imageID); err != nil {
        log.Printf("[ERROR] AdminService: Failed to delete image: %v", err)
        return fmt.Errorf("failed to delete image: %w", err)
    }

    // Delete from Minio
    if s.infrastructure.ObjectStorage != nil && imageToDelete.MinioObjectName != "" {
        if err := s.infrastructure.ObjectStorage.RemoveObject(ctx, s.config.MinIOBucket, imageToDelete.MinioObjectName); err != nil {
            log.Printf("[WARN] AdminService: Failed to delete image from Minio: %v", err)
            // Don't fail the operation if Minio deletion fails
        }
    }

    if err := s.publishProductImageDeletedEvent(ctx, imageID); err != nil {
        log.Printf("[WARN] AdminService: Failed to publish image deleted event: %v", err)
    }

    return nil
}
```

---

## **Phase 5: Handler Layer URL Enrichment Infrastructure**

**Goal:** Create infrastructure for handler-layer presigned URL generation

**Prerequisites:** Phase 4 complete

**Deliverables:** URL enrichment utilities and handler updates

### **Step 5.1: Create Image URL Enrichment Helper**
**File:** `internal/service/product/handler_util.go` (new file)

**Purpose:** Shared utilities for enriching product responses with presigned URLs

**Code:**
```go
package product

import (
    "context"
    "log"
    "net/http"

    "go-shopping-poc/internal/platform/storage/minio"
)

// ImageURLGenerator handles presigned URL generation for handlers
type ImageURLGenerator struct {
    objectStorage minio.ObjectStorage
    bucket        string
}

// NewImageURLGenerator creates a new URL generator
func NewImageURLGenerator(objectStorage minio.ObjectStorage, bucket string) *ImageURLGenerator {
    return &ImageURLGenerator{
        objectStorage: objectStorage,
        bucket:        bucket,
    }
}

// EnrichProductWithImageURLs generates presigned URLs for all product images
func (g *ImageURLGenerator) EnrichProductWithImageURLs(ctx context.Context, product *Product) {
    if product == nil || g.objectStorage == nil {
        return
    }

    for i := range product.Images {
        if product.Images[i].MinioObjectName != "" {
            url, err := g.objectStorage.PresignedGetObject(ctx, g.bucket, product.Images[i].MinioObjectName, 3600)
            if err != nil {
                log.Printf("[WARN] Failed to generate presigned URL for image %d: %v", product.Images[i].ID, err)
                continue
            }
            product.Images[i].ImageURL = url
        }
    }
}

// EnrichProductsWithImageURLs generates presigned URLs for a slice of products
func (g *ImageURLGenerator) EnrichProductsWithImageURLs(ctx context.Context, products []*Product) {
    for _, product := range products {
        g.EnrichProductWithImageURLs(ctx, product)
    }
}

// GenerateImageURL generates a single presigned URL for an image
func (g *ImageURLGenerator) GenerateImageURL(ctx context.Context, minioObjectName string) (string, error) {
    if minioObjectName == "" {
        return "", nil
    }
    if g.objectStorage == nil {
        return "", nil
    }
    return g.objectStorage.PresignedGetObject(ctx, g.bucket, minioObjectName, 3600)
}
```

### **Step 5.2: Update Catalog Handler Infrastructure**
**File:** `internal/service/product/handler.go`

**Actions:**
1. Update `CatalogHandler` struct to include URL generator:
```go
type CatalogHandler struct {
    service       *CatalogService
    urlGenerator  *ImageURLGenerator
}
```

2. Update constructor:
```go
func NewCatalogHandler(service *CatalogService, urlGenerator *ImageURLGenerator) *CatalogHandler {
    return &CatalogHandler{
        service:      service,
        urlGenerator: urlGenerator,
    }
}
```

### **Step 5.3: Update Admin Handler Infrastructure**
**File:** `internal/service/product/handler_admin.go`

**Actions:**
1. Update `AdminHandler` struct:
```go
type AdminHandler struct {
    service       *AdminService
    urlGenerator  *ImageURLGenerator
}
```

2. Update constructor:
```go
func NewAdminHandler(service *AdminService, urlGenerator *ImageURLGenerator) *AdminHandler {
    return &AdminHandler{
        service:      service,
        urlGenerator: urlGenerator,
    }
}
```

---

## **Phase 6: Catalog Handler Updates**

**Goal:** Update catalog handlers to enrich responses with presigned URLs

**Prerequisites:** Phase 5 complete

**Deliverables:** Updated handler.go with URL enrichment

### **Step 6.1: Update GetAllProducts Handler**
**File:** `internal/service/product/handler.go`

**Actions:**
1. Add URL enrichment after fetching products (line 30-48):
```go
func (h *CatalogHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
    limit, offset := parsePagination(r)

    products, err := h.service.GetAllProducts(r.Context(), limit, offset)
    if err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve products")
        return
    }

    // ENRICH: Generate presigned URLs for all product images
    h.urlGenerator.EnrichProductsWithImageURLs(r.Context(), products)

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
```

### **Step 6.2: Update GetProduct Handler**
**File:** `internal/service/product/handler.go`

**Actions:**
1. Add URL enrichment after fetching product (line 69-84):
```go
func (h *CatalogHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
    // ... existing ID parsing logic ...

    product, err := h.service.GetProductByID(r.Context(), productID)
    if err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve product")
        return
    }

    if product == nil {
        errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Product not found")
        return
    }

    // ENRICH: Generate presigned URLs for product images
    h.urlGenerator.EnrichProductWithImageURLs(r.Context(), product)

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(product); err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
        return
    }
}
```

### **Step 6.3: Update All Other Catalog Handlers**
**File:** `internal/service/product/handler.go`

**Actions:**
Apply the same URL enrichment pattern to:
- `GetProductsByCategory()` (line 88-116)
- `GetProductsByBrand()` (line 118-147)
- `SearchProducts()` (line 149-178)
- `GetProductsInStock()` (line 180-202)

**Pattern for each:**
```go
// After fetching products:
h.urlGenerator.EnrichProductsWithImageURLs(r.Context(), products)
```

### **Step 6.4: Add New Image Endpoints**
**File:** `internal/service/product/handler.go`

**Actions:**
Add the following new endpoints:

**1. GetProductImages:**
```go
// GetProductImages handles GET /api/v1/products/{id}/images - lists all images with presigned URLs
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

    images, err := h.service.repo.GetProductImages(r.Context(), productID)
    if err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve product images")
        return
    }

    if len(images) == 0 {
        errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "No images found for product")
        return
    }

    // ENRICH: Generate presigned URLs
    for i := range images {
        url, err := h.urlGenerator.GenerateImageURL(r.Context(), images[i].MinioObjectName)
        if err != nil {
            log.Printf("[WARN] Failed to generate URL for image %d: %v", images[i].ID, err)
            continue
        }
        images[i].ImageURL = url
    }

    response := map[string]interface{}{
        "product_id": productID,
        "images":     images,
        "count":      len(images),
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(response); err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
        return
    }
}
```

**2. GetProductMainImage:**
```go
// GetProductMainImage handles GET /api/v1/products/{id}/main-image - returns main image with presigned URL
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

    images, err := h.service.repo.GetProductImages(r.Context(), productID)
    if err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to retrieve product images")
        return
    }

    // Find main image
    var mainImage *ProductImage
    for i := range images {
        if images[i].IsMain {
            mainImage = &images[i]
            break
        }
    }

    if mainImage == nil {
        errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "No main image found for product")
        return
    }

    // ENRICH: Generate presigned URL
    url, err := h.urlGenerator.GenerateImageURL(r.Context(), mainImage.MinioObjectName)
    if err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to generate image URL")
        return
    }
    mainImage.ImageURL = url

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(mainImage); err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to encode response")
        return
    }
}
```

**3. GetDirectImage:**
```go
// GetDirectImage handles GET /api/v1/images/{objectName} - streams image directly from Minio
func (h *CatalogHandler) GetDirectImage(w http.ResponseWriter, r *http.Request) {
    objectName := chi.URLParam(r, "objectName")
    if objectName == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing object name in path")
        return
    }

    // Security: Validate object name format (prevent path traversal)
    if strings.Contains(objectName, "..") || strings.HasPrefix(objectName, "/") {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid object name")
        return
    }

    // Get object from Minio
    object, err := h.urlGenerator.objectStorage.GetObject(r.Context(), h.urlGenerator.bucket, objectName)
    if err != nil {
        errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Image not found")
        return
    }
    defer object.Close()

    // Get object info for Content-Type
    info, err := h.urlGenerator.objectStorage.StatObject(r.Context(), h.urlGenerator.bucket, objectName)
    if err == nil && info.ContentType != "" {
        w.Header().Set("Content-Type", info.ContentType)
    } else {
        w.Header().Set("Content-Type", "application/octet-stream")
    }

    // Set cache headers
    w.Header().Set("Cache-Control", "public, max-age=3600")
    w.Header().Set("ETag", info.ETag)

    // Stream object to response
    if _, err := io.Copy(w, object); err != nil {
        log.Printf("[ERROR] Failed to stream image: %v", err)
        return
    }
}
```

**Note:** Add required imports: `"io"`, `"strings"`, `"log"`

---

## **Phase 7: Admin Handler Updates**

**Goal:** Update admin handlers to work with new schema and support single image operations

**Prerequisites:** Phase 6 complete

**Deliverables:** Updated handler_admin.go with URL enrichment

### **Step 7.1: Update Get Operations with URL Enrichment**
**File:** `internal/service/product/handler_admin.go`

**Actions:**
1. **Update CreateProduct handler** (lines 27-56):
   - Remove `main_image` from validation (no longer a field)
   - Enrich response with URLs (if images were somehow included)

2. **Update UpdateProduct handler** (lines 58-106):
   - Remove `main_image` field validation
   - Enrich response with URLs

3. **Add URL enrichment to handlers that return products**

**Example for CreateProduct:**
```go
func (h *AdminHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
    var product Product
    if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
        return
    }

    // REMOVED: No main_image validation

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
```

### **Step 7.2: Update AddProductImage Handler**
**File:** `internal/service/product/handler_admin.go`

**Actions:**
1. Update handler to use new `AddSingleImage` service method:
```go
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
```

### **Step 7.3: Update UpdateProductImage Handler**
**File:** `internal/service/product/handler_admin.go`

**Actions:**
1. Remove `image_url` from updateable fields (only metadata: is_main, image_order, etc.)
2. Enrich response with presigned URL:
```go
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
    // TODO: Need to add GetProductImageByID to repository
    
    image := &ProductImage{
        ID:          imageID,
        IsMain:      req.IsMain,
        ImageOrder:  req.ImageOrder,
        ContentType: req.ContentType,
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
```

### **Step 7.4: Add Repository Method for GetProductImageByID**
**File:** `internal/service/product/repository_image.go`

**Actions:**
1. Add method to fetch single image by ID:
```go
// GetProductImageByID retrieves a single image by its ID
func (r *productRepository) GetProductImageByID(ctx context.Context, imageID int64) (*ProductImage, error) {
    log.Printf("[DEBUG] Repository: Fetching image %d", imageID)

    if imageID <= 0 {
        return nil, fmt.Errorf("%w: image ID must be positive", ErrInvalidProductID)
    }

    query := `
        SELECT id, product_id, minio_object_name, is_main, image_order,
               file_size, content_type, created_at
        FROM products.product_images
        WHERE id = $1`

    var image ProductImage
    err := r.db.QueryRowContext(ctx, query, imageID).Scan(
        &image.ID, &image.ProductID, &image.MinioObjectName,
        &image.IsMain, &image.ImageOrder, &image.FileSize, &image.ContentType, &image.CreatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("%w: image %d not found", ErrProductImageNotFound, imageID)
        }
        return nil, fmt.Errorf("%w: failed to query product image: %v", ErrDatabaseOperation, err)
    }

    return &image, nil
}
```

**Note:** Add import: `"database/sql"`

---

## **Phase 8: Main Application Wiring**

**Goal:** Wire up all components in main.go files

**Prerequisites:** Phase 7 complete

**Deliverables:** Updated cmd/product/main.go and cmd/product-loader/main.go

### **Step 8.1: Update Catalog Service Main**
**File:** `cmd/product/main.go`

**Actions:**
1. Add Minio client initialization:
```go
// After database initialization:
minioConfig := &minio.Config{
    Endpoint:  config.MinIOEndpoint,
    AccessKey: config.MinIOAccessKey,
    SecretKey: config.MinIOSecretKey,
    Secure:    config.MinIOSecure,
}
minioClient, err := minio.NewClient(minioConfig)
if err != nil {
    log.Fatalf("[ERROR] Failed to create Minio client: %v", err)
}
```

2. Update infrastructure to include Minio:
```go
// Update CatalogInfrastructure
catalogInfra := &product.CatalogInfrastructure{
    Database:     db,
    OutboxWriter: outboxWriter,
}

// Create URL generator
urlGenerator := product.NewImageURLGenerator(minioClient, config.MinIOBucket)
```

3. Update handler initialization:
```go
// Create handlers with URL generator
catalogHandler := product.NewCatalogHandler(catalogService, urlGenerator)
```

4. Add new routes:
```go
// Add new image routes
productRouter.Get("/{id}/images", catalogHandler.GetProductImages)
productRouter.Get("/{id}/main-image", catalogHandler.GetProductMainImage)
productRouter.Get("/images/{objectName}", catalogHandler.GetDirectImage)
```

### **Step 8.2: Update Product-Loader Main**
**File:** `cmd/product-loader/main.go`

**Actions:**
1. Ensure Minio client is initialized (already present in AdminInfrastructure)
2. Verify AdminInfrastructure includes ObjectStorage

**Current code check:**
```go
adminInfra := &product.AdminInfrastructure{
    Database:       db,
    ObjectStorage:  minioClient,  // Should already exist
    OutboxWriter:   outboxWriter,
    HTTPDownloader: downloader,
}
```

---

## **Phase 9: Cleanup and Validation**

**Goal:** Remove legacy code and validate implementation

**Prerequisites:** Phase 8 complete

**Deliverables:** Clean codebase ready for testing

### **Step 9.1: Remove Legacy References**

**Files to update:**
1. **entity.go:** Remove any remaining references to MainImage field usage
2. **service_admin.go:** Remove CSV processing for main_image field (keep for CSV parsing, but don't store)
3. **repository_image.go:** Ensure SetMainImage method is removed
4. **config files:** Ensure MinIO configuration is present

### **Step 9.2: Add Repository Interface Method**
**File:** `internal/service/product/repository.go`

**Actions:**
1. Add `GetProductImageByID` to interface
2. Remove old `SetMainImage` if still present

### **Step 9.3: Update CSV Processing in Product-Loader**
**File:** `cmd/product-loader/main.go` or `service_admin.go`

**Verification:**
- Ensure `convertCSVRecordToProduct` still reads `main_image` column from CSV (needed to determine IsMain)
- Ensure `processSingleImage` sets `ImageURL: ""` when saving to database

---

## **Phase 10: Documentation Update**

**Goal:** Update API documentation

**Prerequisites:** All implementation complete

**Deliverables:** Updated docs/product-api-prompt.md

### **Step 10.1: Document Schema Changes**
- Document removed `main_image` column
- Document `image_url` is no longer stored
- Document `minio_object_name` is required

### **Step 10.2: Document API Endpoints**
Document new endpoints:
- `GET /api/v1/products/{id}/images` - List all images with presigned URLs
- `GET /api/v1/products/{id}/main-image` - Get main image with presigned URL  
- `GET /api/v1/images/{objectName}` - Direct image access

Document changes:
- All product endpoints now return images with presigned URLs (1-hour expiry)
- Removed `main_image` from product responses

### **Step 10.3: Document Admin API Changes**
- `POST /api/v1/admin/products/{id}/images` now accepts `{image_url, is_main}` and downloads/uploads to Minio
- `PUT /api/v1/admin/products/images/{id}` now only updates metadata (is_main, image_order, content_type)
- Images can no longer be updated via URL (must delete and re-add)

---

## **Implementation Order Summary**

**Critical Path (Sequential):**
1. **Phase 1** → Database schema (foundation)
2. **Phase 2** → Entity validation (enforces new rules)
3. **Phase 3** → Repository layer (data access)
4. **Phase 4** → Admin service (business logic)
5. **Phase 5** → Handler infrastructure (URL generation)
6. **Phase 6** → Catalog handlers (read endpoints)
7. **Phase 7** → Admin handlers (write endpoints)
8. **Phase 8** → Main wiring (dependency injection)
9. **Phase 9** → Cleanup (remove legacy)
10. **Phase 10** → Documentation

**Dependencies:**
- Each phase builds on the previous (no parallel work recommended)
- Phase 5 must complete before Phases 6-7
- Phase 8 (wiring) requires all previous phases

---

## **Key Design Decisions Summary**

1. **Handler-Layer URL Generation:** Services return raw entities with empty ImageURL; handlers enrich with presigned URLs before JSON encoding. Maintains clean separation and allows caching.

2. **MinioObjectName Required:** Database schema and entity validation require MinioObjectName; ImageURL becomes transient computed field.

3. **Admin Image Operations:** Admin service retains individual image CRUD with automatic download-to-Minio workflow for new images.

4. **No main_image Column:** Products table no longer has main_image; main image determined by is_main flag in product_images table.

5. **Direct Image Access:** Added direct Minio streaming endpoint for better caching and CDN compatibility.

---

## **Testing Checklist**

**Database:**
- [ ] Migration runs successfully
- [ ] image_count trigger works
- [ ] Constraints prevent duplicate minio_object_name per product

**Product-Loader:**
- [ ] CSV ingestion works with existing format
- [ ] Images uploaded to Minio
- [ ] Database records have empty ImageURL
- [ ] IsMain flag set correctly from CSV main_image column

**Catalog API:**
- [ ] GET /api/v1/products returns products with presigned URLs
- [ ] GET /api/v1/products/{id}/images returns images with URLs
- [ ] GET /api/v1/products/{id}/main-image returns main image
- [ ] GET /api/v1/images/{objectName} streams image directly
- [ ] URLs are valid and expire after 1 hour

**Admin API:**
- [ ] POST /api/v1/admin/products/{id}/images downloads and stores in Minio
- [ ] PUT /api/v1/admin/products/images/{id} updates metadata only
- [ ] DELETE removes from both DB and Minio
- [ ] SetMainImage updates is_main flag
- [ ] All responses include presigned URLs

**Error Cases:**
- [ ] Invalid object names rejected (path traversal)
- [ ] Missing products return 404
- [ ] Missing images return 404
- [ ] Invalid IDs return 400

---

**End of Revised Implementation Plan v3**
