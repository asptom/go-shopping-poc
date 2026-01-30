## **Revised Phased Implementation Plan: Minio-Only Image Serving**

### **Phase 1: Catalog Service Image Serving Foundation (2 days)**
**Goal**: Enable catalog service to serve Minio-stored images

**Step 1.1: Update Catalog Infrastructure**
```go
// File: cmd/product/main.go
catalogInfra := &product.CatalogInfrastructure{
    Database:     platformDB,
    OutboxWriter: writerProvider.GetWriter(),
    ObjectStorage: minioStorage,  // ADD Minio client
}
```

**Step 1.2: Extend Catalog Infrastructure Interface**
```go
// File: internal/service/product/infrastructure.go
type CatalogInfrastructure struct {
    Database     database.Database
    OutboxWriter *outbox.Writer
    ObjectStorage minio.ObjectStorage  // ADD THIS
}
```

**Step 1.3: Add Image URL Generation Service**
```go
// File: internal/service/product/service.go
// ADD these methods to CatalogService:

func (s *CatalogService) GetProductImagesWithUrls(ctx context.Context, productID int64) ([]ProductImage, error) {
    images, err := s.repo.GetProductImages(ctx, productID)
    if err != nil {
        return nil, err
    }
    
    for i := range images {
        if images[i].MinioObjectName != "" {
            url, err := s.infrastructure.ObjectStorage.PresignedGetObject(
                ctx, s.config.MinIOBucket, images[i].MinioObjectName, 3600)
            if err != nil {
                return nil, fmt.Errorf("failed to generate image URL: %w", err)
            }
            images[i].ImageURL = url  // Replace external URL with Minio URL
        }
    }
    return images, nil
}

func (s *CatalogService) GetMainImageUrl(ctx context.Context, productID int64) (string, error) {
    images, err := s.GetProductImagesWithUrls(ctx, productID)
    if err != nil {
        return "", err
    }
    
    for _, img := range images {
        if img.IsMain {
            return img.ImageURL, nil
        }
    }
    return "", fmt.Errorf("no main image found for product %d", productID)
}
```

**Step 1.4: Update All Product Query Methods**
```go
// File: internal/service/product/service.go
// UPDATE these methods to load images and generate URLs:

func (s *CatalogService) GetAllProducts(ctx context.Context, limit, offset int) ([]*Product, error) {
    products, err := s.repo.GetAllProducts(ctx, limit, offset)
    if err != nil {
        return nil, err
    }
    
    for _, product := range products {
        if err := s.loadProductImagesAndUrls(ctx, product); err != nil {
            log.Printf("[WARN] Failed to load images for product %d: %v", product.ID, err)
        }
    }
    
    return products, nil
}

func (s *CatalogService) loadProductImagesAndUrls(ctx context.Context, product *Product) error {
    images, err := s.GetProductImagesWithUrls(ctx, product.ID)
    if err != nil {
        return err
    }
    
    product.Images = images
    product.ImageCount = len(images)
    
    // Update MainImage to be Minio URL instead of external URL
    for _, img := range images {
        if img.IsMain {
            product.MainImage = img.ImageURL
            break
        }
    }
    
    return nil
}
```

**Step 1.5: Update Product Entity (Remove External URL Dependency)**
```go
// File: internal/service/product/entity.go
// UPDATE these methods:

func (p *Product) GetMainImage() string {
    // REMOVED: Return external URL first
    // NEW: Use only the loaded Images array
    for _, img := range p.Images {
        if img.IsMain {
            return img.ImageURL
        }
    }
    return ""
}
```

---

### **Phase 2: New Image Serving Endpoints (1 day)**
**Goal**: Provide dedicated endpoints for image serving

**Step 2.1: Add Image Endpoints to Catalog Handler**
```go
// File: internal/service/product/handler.go
// ADD these new handler methods:

// GetProductImages handles GET /api/v1/products/{id}/images
func (h *CatalogHandler) GetProductImages(w http.ResponseWriter, r *http.Request) {
    productID, err := parseProductID(r)
    if err != nil {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid product ID")
        return
    }

    images, err := h.service.GetProductImagesWithUrls(r.Context(), productID)
    if err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to get product images")
        return
    }

    response := map[string]interface{}{
        "product_id": productID,
        "images":     images,
        "count":      len(images),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// GetProductMainImage handles GET /api/v1/products/{id}/main-image
func (h *CatalogHandler) GetProductMainImage(w http.ResponseWriter, r *http.Request) {
    productID, err := parseProductID(r)
    if err != nil {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid product ID")
        return
    }

    imageUrls, err := h.service.GetMainImageUrl(r.Context(), productID)
    if err != nil {
        if err.Error() == "no main image found" {
            errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Main image not found")
        } else {
            errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to get main image")
        }
        return
    }

    response := map[string]interface{}{
        "product_id":  productID,
        "image_url":  imageUrls,
        "expires_in": 3600,  // 1 hour
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

**Step 2.2: Update Router**
```go
// File: cmd/product/main.go
// ADD these routes to productRouter:

productRouter.Get("/products/{id}/images", catalogHandler.GetProductImages)
productRouter.Get("/products/{id}/main-image", catalogHandler.GetProductMainImage)
```

---

### **Phase 3: Database Schema Cleanup (1 day)**
**Goal**: Eliminate dual storage and external URL references

**Step 3.1: Create Migration Script**
```sql
-- File: migrations/cleanup_image_storage.sql

-- Step 1: Remove main_image column from products table (external URL reference)
ALTER TABLE products.products DROP COLUMN IF EXISTS main_image;

-- Step 2: Remove image_url column from product_images table (external URL reference)
ALTER TABLE products.product_images DROP COLUMN IF EXISTS image_url;

-- Step 3: Add unique constraint on main image per product
ALTER TABLE products.product_images ADD CONSTRAINT unique_main_image 
    UNIQUE (product_id) 
    WHERE is_main = true;

-- Step 4: Update products table to track image count
ALTER TABLE products.products ADD COLUMN IF NOT EXISTS image_count INTEGER DEFAULT 0;

-- Step 5: Update image counts
UPDATE products.products p 
SET image_count = (
    SELECT COUNT(*) 
    FROM products.product_images pi 
    WHERE pi.product_id = p.id
);
```

**Step 3.2: Update Product Entity**
```go
// File: internal/service/product/entity.go
// REMOVE these fields:
type Product struct {
    // REMOVE: MainImage string `json:"main_image" db:"main_image"`
    // KEEP: Images []ProductImage `json:"images,omitempty"`
}
```

**Step 3.3: Update ProductImage Entity**
```go
// File: internal/service/product/entity.go
// REMOVE this field:
type ProductImage struct {
    // REMOVE: ImageURL string `json:"image_url" db:"image_url"`
    // KEEP: MinioObjectName string `json:"minio_object_name" db:"minio_object_name"`
}
```

**Step 3.4: Add Helper Method for Image URL Generation**
```go
// File: internal/service/product/entity.go
// ADD this method to ProductImage:

func (pi *ProductImage) GetImageURL(objectStorage minio.ObjectStorage, bucket string, ctx context.Context) (string, error) {
    if pi.MinioObjectName == "" {
        return "", fmt.Errorf("no Minio object name for image %d", pi.ID)
    }
    
    return objectStorage.PresignedGetObject(ctx, bucket, pi.MinioObjectName, 3600)
}
```

---

### **Phase 4: Admin Service Updates (1 day)**
**Goal**: Update product-admin to work with Minio-only storage

**Step 4.1: Update Admin Infrastructure**
```go
// File: internal/service/product/service_admin.go
// UPDATE image processing in convertCSVRecordToProduct():

func (s *AdminService) convertCSVRecordToProduct(record ProductCSVRecord) (*Product, error) {
    product := &Product{
        Name:              record.Name,
        Description:       record.Description,
        InitialPrice:      record.InitialPrice,
        FinalPrice:        record.FinalPrice,
        Currency:          record.Currency,
        Color:             record.Color,
        Size:              record.Size,
        // REMOVE: MainImage field assignment
        CountryCode:       record.CountryCode,
        ModelNumber:       record.ModelNumber,
        RootCategory:      record.RootCategory,
        Category:          record.Category,
        Brand:             record.Brand,
        AllAvailableSizes: record.AllAvailableSizes,
        ImageURLs:         record.ImageURLs,
    }
    // ... rest of method unchanged
}
```

**Step 4.2: Update Image Processing Logic**
```go
// File: internal/service/product/service_admin.go
// UPDATE processSingleImage():

func (s *AdminService) processSingleImage(ctx context.Context, product *Product, imageURL string, imageIndex int, useCache bool, result *ProductIngestionResult) error {
    // Download and upload logic remains the same
    
    // UPDATE: Don't store external URL
    productImage := &ProductImage{
        ProductID:       product.ID,
        // REMOVE: ImageURL field
        MinioObjectName: minioObjectName,
        IsMain:          s.isMainImage(product, imageURL, imageIndex),  // New helper method
        ImageOrder:      imageIndex,
        FileSize:        fileSize,
        ContentType:     contentType,
    }
    
    // Rest of method unchanged
}

// ADD this helper method:
func (s *AdminService) isMainImage(product *Product, imageURL string, imageIndex int) bool {
    // Check if this URL matches what would have been the main image
    // or use first image as main if no main image specified
    return imageIndex == 0 || imageURL == product.MainImage
}
```

**Step 4.3: Update Repository Image Methods**
```go
// File: internal/service/product/repository_image.go
// UPDATE queries to remove image_url references:

func (r *productRepository) AddProductImage(ctx context.Context, image *ProductImage) error {
    query := `
        INSERT INTO products.product_images (
            product_id, minio_object_name, is_main, image_order,
            file_size, content_type, created_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7
        )`
    
    _, err = r.db.ExecContext(ctx, query,
        image.ProductID, image.MinioObjectName, image.IsMain,
        image.ImageOrder, image.FileSize, image.ContentType, image.CreatedAt,
    )
    // ... rest unchanged
}

func (r *productRepository) SetMainImage(ctx context.Context, productID int64, imageID int64) error {
    // REMOVE the part that updates products.main_image
    // Just update the product_images table
    
    tx, err := r.db.BeginTxx(ctx, nil)
    // ... transaction setup
    
    unsetQuery := `UPDATE products.product_images SET is_main = false WHERE product_id = $1`
    _, err = tx.ExecContext(ctx, unsetQuery, productID)
    
    setQuery := `UPDATE products.product_images SET is_main = true WHERE id = $1 AND product_id = $2`
    result, err := tx.ExecContext(ctx, setQuery, imageID, productID)
    
    // REMOVE: Update products.main_image query
    
    // ... transaction commit unchanged
}
```

---

### **Phase 5: Admin Handler Updates (1 day)**
**Goal**: Update admin endpoints to work with new schema

**Step 5.1: Update Admin Handler Validation**
```go
// File: internal/service/product/handler_admin.go
// UPDATE CreateProduct and UpdateProduct handlers:

// REMOVE main_image field validation
// Products no longer have a main_image field in JSON

// UPDATE response handling to include populated Images array
```

**Step 5.2: Update Image Management Endpoints**
```go
// File: internal/service/product/handler_admin.go
// UPDATE AddProductImage:

func (h *AdminHandler) AddProductImage(w http.ResponseWriter, r *http.Request) {
    var image ProductImage
    if err := json.NewDecoder(r.Body).Decode(&image); err != nil {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON in request body")
        return
    }

    // REMOVE validation of ImageURL field
    image.ProductID = productID
    
    // ADD validation for MinioObjectName or file upload
    if image.MinioObjectName == "" && r.MultipartForm == nil {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Either MinioObjectName or file upload required")
        return
    }

    // Rest of method unchanged
}
```

---

### **Phase 6: Frontend Integration Updates (1 day)**
**Goal**: Update frontend to use new image endpoints

**Step 6.1: Update Product API Documentation**
```markdown
// File: docs/product-api-prompt.md
// UPDATE image handling section:

**Image Loading**: Product images are served from Minio storage via dedicated endpoints:
- GET /api/v1/products/{id}/images - Lists all images with presigned URLs
- GET /api/v1/products/{id}/main-image - Gets main image with presigned URL
- Images URLs expire after 1 hour for security

**Product Images Array**: Each product includes an `images` array with objects containing:
- `id`, `minio_object_name`, `is_main`, `image_order`, `file_size`, `content_type`
- `image_url` field contains presigned Minio URL (1-hour expiry)
```

**Step 6.2: Angular Service Updates**
```typescript
// ADD these methods to ProductService:

getProductImages(productId: number): Observable<ImageResponse> {
    return this.http.get<ImageResponse>(`${this.apiUrl}/products/${productId}/images`);
}

getProductMainImage(productId: number): Observable<MainImageResponse> {
    return this.http.get<MainImageResponse>(`${this.apiUrl}/products/${productId}/main-image`);
}
```

---

## **Implementation Priority and Dependencies**

### **Critical Path** (Must be done in order):
1. **Phase 1** (Foundation) → Enables Minio serving in catalog
2. **Phase 2** (Endpoints) → Provides image access endpoints  
3. **Phase 3** (Cleanup) → Removes dual storage completely
4. **Phase 4** (Admin Updates) → Aligns admin with new schema
5. **Phase 5** (Handler Updates) → Updates admin endpoints
6. **Phase 6** (Frontend) → Integrates with new endpoints

### **Parallel Development Opportunities**:
- **Phase 1 & 2**: Can be developed in parallel by different developers
- **Phase 3 & 4**: Can be done concurrently after Phase 1-2 complete
- **Phase 5 & 6**: Can start once Phase 3-4 are done

### **Risk Mitigation**:
- **Database Migration**: Backup before running migration script
- **Image Validation**: Add image accessibility tests after migration
- **Rollback Plan**: Keep migration scripts reversible
- **Testing**: Add integration tests for all image serving paths

---

## **Key Benefits of This Approach**

1. **Clean Architecture**: Eliminates dual storage complexity
2. **Performance**: Direct Minio serving with presigned URLs
3. **Security**: Time-limited URLs prevent unauthorized access
4. **Scalability**: Minio handles object storage efficiently
5. **Maintainability**: Single source of truth for image storage
6. **Future-Ready**: Foundation for image processing pipeline
