## **Revised Phased Implementation Plan: Minio-Only Image Serving**

### **Overview**
This plan outlines the migration from external URL-based image storage to Minio-only object storage. Breaking changes are expected and all data will be reloaded from CSV after these changes.

**Key Assumptions:**
- No existing product/data in database (can safely rebuild)
- Breaking changes allowed
- Existing CSV format used by product-loader (backward compatible to CSV, but new schema in DB)
- Product-loader: bulk import of products and images (only mechanism for image ingestion)
- Product-admin: CRUD operations for individual products (no image management endpoints)
- Image serving: return presigned URL strings via API endpoints
- All data will be reloaded after schema changes (no existing data migration needed)

### **Admin Service Role**
- **Product CRUD Operations**: Create, Read, Update, Delete individual products
- **Single Image Management**: Can add, update, and delete individual images for products
- **Image Ingestion**: Images can be added/updated/deleted via admin API endpoints
- **Note**: Bulk import capabilities are only in product-loader (not admin)
- **Image Access**: Admin service uses same image serving endpoints as catalog service for returning images
- **Schema Usage**: Admin service reads and writes to products table and product_images table (both using Minio-only storage)

### **Why Each Service Needs Its Own Minio Client**

**Product-Loader** (Phase 2):
- **Purpose**: Process bulk CSV data
- **Operations**: Download images from CSV URLs → Upload to Minio
- **Use**: For large-scale image ingestion from existing CSV files

**AdminService** (Phase 6):
- **Purpose**: Manually manage individual products and their images
- **Operations**: Download image from URL -> Upload to Minio OR receive file upload -> Upload to Minio
- **Use**: When adding or updating a single image from admin interface

**CatalogService** (Phase 3):
- **Purpose**: Serve products to clients
- **Operations**: Generate presigned URLs from Minio for reading images
- **Use**: Returns URLs to frontend when fetching products (does NOT upload/download)

**Key Distinction**: The AdminService needs direct Minio access to **handle file uploads** and **store** new images. It cannot rely on the catalog service's presigned URL generation because that only addresses the *reading* of images, not the *writing/uploading* of new images.

### **Image Access Patterns**
- **Bulk Data Loading**: Product-loader downloads images from CSV URLs and uploads to Minio
- **Catalog API**: Returns products with images array containing presigned URLs (1-hour expiry)
- **Admin API**: Returns products with images array containing presigned URLs; can also add/update/delete images via admin endpoints
- **Direct Access**: Use GET /api/v1/images/{objectName} for direct browser access with proper Content-Type headers
- **Lazy Loading**: Can use direct access endpoints progressively for better caching

---

## **Phase 1: Database Schema Changes**

**Goal**: Update database schema to support Minio-only storage

**Step 1.1: Update Migration File**
- File: `internal/service/product/migrations/001-init.sql`
- Remove `main_image` column from `products` table
- Remove `image_url` column from `product_images` table
- Keep `minio_object_name`, `is_main`, `image_order`, `file_size`, `content_type`
- Add any new constraints or indexes for image_count field

**Step 1.2: Verify Schema**
- Review updated migration to ensure correct schema
- Document the new table structure
- Ensure all constraints are properly defined

---

## **Phase 2: Product-Loader Updates**

**Goal**: Update product-loader to work with new schema using existing CSV format

**Step 2.1: Update CSV Parser Configuration**
- File: `cmd/product-loader/main.go`
- Update CSV field mapping to handle 'main_image' column for determining main image position
- Update 'image_urls' column parsing (JSON array to list)
- Remove validation that requires `image_url` field in CSV
- Field validation: skip any row with missing required fields, log warning

**Step 2.2: Update Image Processing Logic**
- File: `internal/service/product/service_admin.go`
- Update `processProductImages()` method:
  - Download images using HTTP downloader (keep existing caching logic)
  - Upload to Minio instead of storing URLs
  - Generate unique Minio object names (product_id_image_0, product_id_image_1, etc.)
  - Ignore `image_url` field from CSV - not storing it
- Update `processSingleImage()` method:
  - Store Minio object name instead of URL
  - Set `is_main` based on position in image list or `main_image` column match (first image or exact match)
  - Remove any URL validation logic

**Step 2.3: Update Bulk Insert Logic**
- File: `internal/service/product/repository_bulk.go`
- Update `AddProductImage()` to insert only required fields:
  - `product_id`, `minio_object_name`, `is_main`, `image_order`, `file_size`, `content_type`, `created_at`
- Remove any insert logic for `image_url` field
- Update bulk insert query to match new schema

---

## **Phase 3: Catalog Service Updates**

**Goal**: Update catalog service to serve Minio images with presigned URLs

**Step 3.1: Update Service Infrastructure**
- File: `internal/service/product/service.go`
- Update `CatalogService` struct to include:
  - `minioObjectStorage minio.ObjectStorage` field
  - `minioBucket string` field (or load from config)
- Update `NewCatalogService()` to initialize Minio storage client

**Step 3.2: Add Image URL Generation Method**
- Add to `CatalogService`:
  ```go
  func (s *CatalogService) GeneratePresignedURL(ctx context.Context, objectName string) (string, error) {
      if objectName == "" {
          return "", errors.New("object name is empty")
      }
      return s.minioObjectStorage.PresignedGetObject(ctx, s.minioBucket, objectName, 3600)
  }
  ```

**Step 3.3: Update GetProduct Images with URLs**
- Update or add method:
  ```go
  func (s *CatalogService) GetProductImagesWithUrls(ctx context.Context, productID int64) ([]ProductImageDTO, error) {
      images, err := s.repo.GetProductImages(ctx, productID)
      if err != nil {
          return nil, err
      }

      urls := make([]ProductImageDTO, len(images))
      for i, img := range images {
          url, err := s.GeneratePresignedURL(ctx, img.MinioObjectName)
          if err != nil {
              log.Printf("[WARN] Failed to generate URL for image %d: %v", img.ID, err)
              continue
          }
          urls[i] = ProductImageDTO{
              ID:             img.ID,
              ProductID:      img.ProductID,
              MinioObjectName: img.MinioObjectName,
              IsMain:         img.IsMain,
              ImageOrder:     img.ImageOrder,
              FileSize:       img.FileSize,
              ContentType:    img.ContentType,
              ImageURL:       url, // Presigned URL with 1-hour expiry
          }
      }
      return urls, nil
  }
  ```

**Step 3.4: Update Product Query Methods**
- Update methods that return products with images:
  ```go
  func (s *CatalogService) GetAllProducts(ctx context.Context, limit, offset int) ([]*Product, error) {
      products, err := s.repo.GetAllProducts(ctx, limit, offset)
      if err != nil {
          return nil, err
      }

      for _, product := range products {
          images, err := s.GetProductImagesWithUrls(ctx, product.ID)
          if err != nil {
              log.Printf("[WARN] Failed to load images for product %d: %v", product.ID, err)
              continue
          }
          product.Images = images
          product.ImageCount = len(images)
      }

      return products, nil
  }

  // Similar updates for other query methods: GetProductByID, SearchProducts, etc.
  ```

**Step 3.5: Update Product Entity**
- File: `internal/service/product/entity.go`
- Update `Product` struct:
  - Remove `MainImage string` field
  - Keep `Images []ProductImage` field (will be loaded with URLs)
  - Update `GetMainImage()` to iterate over Images and return first IsMain image URL
- Update `ProductImage` struct:
  - Remove `ImageURL string` field (no longer storage)
  - Keep `MinioObjectName string` field
  - Add or update `ImageURL string` as a computed field (populated after fetching)

---

## **Phase 4: Image Serving API Endpoints**

**Goal**: Add endpoints for accessing Minio images

**Step 4.1: Add Presigned URL Endpoints**
- File: `internal/service/product/handler.go`
- Add method `GetProductImages`:
  - Endpoint: GET `/api/v1/products/{id}/images`
  - Handler loads all images for product
  - Generates presigned URLs for each
  - Returns JSON with arrays of image URLs
  - Error handling: 400 for invalid ID, 404 if no images, 500 for server error

- Add method `GetProductMainImage`:
  - Endpoint: GET `/api/v1/products/{id}/main-image`
  - Handler returns only main image URL
  - Returns URL string with 1-hour expiry metadata
  - Error handling: 404 if no main image found

**Step 4.2: Add Direct Image Serving Endpoint**
- File: `internal/service/product/handler.go`
- Add method `GetDirectImage`:
  - Endpoint: GET `/api/v1/images/{objectName}`
  - Handler fetches object from Minio and streams to HTTP response
  - Must verify object name (prevent path traversal attacks)
  - Set appropriate Content-Type header
  - Handle image retrieval errors (404 for missing, 500 for errors)
  - Cache headers for performance (Cache-Control, ETag)

**Step 4.3: Update Router**
- File: `cmd/product/main.go`
- Add routes:
  ```go
  productRouter.Get("/images/{objectName}", catalogHandler.GetDirectImage)
  ```
- Update other routes to use image URLs from responses

---

## **Phase 5: Catalog Service Cleanups**

**Goal**: Remove all external URL dependencies and legacy code from catalog service

**Step 5.1: Remove External URL References**
- File: `internal/service/product/service_admin.go`
- Remove any methods that reference or process external image URLs after Minio upload
- Remove validation that checks for valid URLs in image processing

**Step 5.2: Remove Product Count Logic**
- File: `internal/service/product/repository_image.go`
- Keep `image_count` field for querying but simplify update logic
- Remove or update `SetImageCount()` methods if needed
- Update any queries that joined image_count with product data

**Step 5.3: Update Product Image Loading**
- File: `internal/service/product/service.go`
- Remove any method that returns MainImage string directly
- Ensure images are always loaded via GetProductImagesWithUrls method
- Update all product query methods to include image loading

---

## **Phase 6: Admin Service and Handler Updates**

**Goal**: Update product-admin to work with Minio-only storage while supporting single image CRUD operations

**Step 6.1: Update Admin Infrastructure**
- File: `internal/service/product/service_admin.go`
- Update AdminService struct to include:
  - `minioObjectStorage minio.ObjectStorage` field (needed for uploading new images)
  - `minioBucket string` field (or load from config)
- Update `NewAdminService()` to initialize Minio storage client
- **Why Minio Client?**: The AdminService needs direct Minio access to **upload/download images** when handling individual image CRUD operations. The catalog service only uses Minio to generate presigned URLs for reading images.

**Step 6.2: Update Admin Image Processing**
- File: `internal/service/product/service_admin.go`
- Ensure AdminService has image processing methods for single operations:
  - `AddSingleImage()` - Download and upload to Minio, save metadata
  - `UpdateImage()` - Update image metadata (IsMain, ImageOrder, etc.)
  - `DeleteImage()` - Remove image metadata and delete from Minio
- Remove any methods that reference or process `main_image` field for products
- Update image metadata operations to use new schema structure

**Step 6.3: Update Admin CRUD Handlers**
- File: `internal/service/product/handler_admin.go`
- Update CreateProduct handler:
  - Accept product payload without main_image field
  - Product is created, images must be added via AddSingleImage endpoints
- Update UpdateProduct handler:
  - Remove main_image field from validation
  - Allow updating product fields only (not images, use image endpoints)
- Update GetProduct handler response:
  - Return Product with Images array populated with URLs via GetProductImagesWithUrls
  - Remove main_image from response object
- Ensure no references to main_image string field remain

**Step 6.4: Add Single Image Endpoints**
- File: `internal/service/product/handler_admin.go`
- Add endpoint for adding single image:
  - POST `/api/v1/products/{id}/images`:
    - Accept JSON with MinioObjectName or URL/File upload
    - Download and upload to Minio if needed
    - Save metadata to database
    - Respond with image object and presigned URL
- Add endpoint for updating single image:
  - PUT `/api/v1/products/{id}/images/{imageId}`:
    - Update image metadata (IsMain, ImageOrder, FileSize, ContentType)
    - Generate new presigned URL in response
- Add endpoint for deleting single image:
  - DELETE `/api/v1/products/{id}/images/{imageId}`:
    - Remove image metadata from database
    - Delete from Minio storage
    - Ensure exactly one main image remains per product

**Step 6.5: Update Database Operations for Single Image**
- File: `internal/service/product/repository_image.go`
- Update to support image CRUD operations:
  - `AddProductImage()` - Insert new image metadata (updated for new schema without image_url)
  - `UpdateProductImage()` - Update existing image metadata
  - `DeleteProductImage()` - Remove image and handle main image reassignment
  - `GetProductImages()` - Fetch images from new schema structure
- Ensure all operations use new schema (main_image column removed, image_url column removed)

**Step 6.6: Verify Product CRUD Operations**
- Ensure product CRUD operations (Create, Read, Update, Delete) work with updated schema
- Verify product listing and detail views include images array with URLs
- Verify single image CRUD operations work correctly
- Ensure no references to main_image string field remain in any admin operations

---

## **Phase 7: API Documentation Update**

**Goal**: Update API documentation to reflect new Minio image serving endpoints

**Step 7.1: Review and Update API Documentation**
- File: `docs/product-api-prompt.md`
- Document all product-related API endpoints:
  - Product CRUD endpoints (Create, Read, Update, Delete)
  - Product query endpoints (List, Search, GetByID)
- Document product image endpoints:
  - GET /api/v1/products/{id}/images - Lists all images with presigned URLs
  - GET /api/v1/products/{id}/main-image - Returns main image presigned URL
  - GET /api/v1/images/{objectName} - Direct Minio image access
- Detail image endpoint responses with example payloads
- Document presigned URL expiry (1 hour default)
- Document content-type handling for direct image access
- Document error responses for all endpoints

**Step 7.2: Document Schema Changes**
- Document Products table schema changes:
  - Removed: main_image column
  - Field removed: product_images.image_url
  - Additional field: image_count (if added)
- Document product_images table schema:
  - Fields: id, product_id, minio_object_name, is_main, image_order, file_size, content_type, created_at

**Step 7.3: Document Product-Loader Changes**
- Document CSV format usage with existing columns (main_image, image_urls)
- Explain how product-loader processes images and stores in Minio
- Document the migration from external URLs to Minio storage approach

**Step 7.4: Document Integration Flow**
- Explain how frontend can use presigned URLs vs direct image access
- Document recommended patterns for image loading in client applications
- Explain caching strategies for direct image access

---

## **Phase 8: Implementation Testing Note**

**NOTE**: The following testing phase is informational only and not part of the implementation plan. You will handle actual testing and validation after implementation.

**Goal**: Validate complete workflow from database rebuild to application testing

**Step 8.1: Database Rebuild**
- Drop existing `products` database if exists
- Run `001-init.sql` migration to create new schema
- Verify tables created correctly with expected fields
- Verify constraints and indexes are in place

**Step 8.2: Product-Loader Execution**
- Update config to point to new CSV file if different
- Run loader with proper configuration
- Monitor logs for any errors during image processing
- Verify all images uploaded to Minio correctly
- Verify database entries match Minio objects

**Step 8.3: Application Testing**
- Start catalog service and verify it initializes Minio client
- Test GET /api/v1/products endpoints - verify images loaded with URLs
- Test GET /api/v1/products/{id}/images - verify all images returned with correct URLs
- Test GET /api/v1/products/{id}/main-image - verify main image URL with expiry
- Test GET /api/v1/images/{objectName} - verify direct image access works
- Verify presigned URLs are valid and accessible (1 hour expiry)
- Test error cases: missing product, no images, invalid object names
- Test admin CRUD operations for products

**Step 8.4: Integration Validation**
- Verify no external image URLs remain in any response
- Check that main_image field is completely removed from data flow
- Ensure product-admin works correctly with new schema
- Validate that image loading patterns work in your application
- Verify performance of direct image access vs presigned URLs

**Goal**: Rebuild database and test complete workflow

**Step 8.1: Rebuild Database**
- Drop existing `products` database if exists
- Run `001-init.sql` migration to create new schema
- Verify tables created correctly

**Step 8.2: Run Product-Loader**
- Update config to point to new CSV file if different
- Run loader with proper configuration
- Monitor logs for any errors during image processing
- Verify all images uploaded to Minio correctly
- Verify database entries match Minio objects

**Step 8.3: Test Catalog Service**
- Test GET /api/v1/products - verify images loaded with URLs
- Test GET /api/v1/products/{id}/images - verify all images returned with URLs
- Test GET /api/v1/products/{id}/main-image - verify main image URL
- Test URL generation and expiry (1 hour)
- Verify image URLs are accessible

**Step 8.4: Test Direct Image Access**
- Test GET /api/v1/images/{objectName}
- Verify content is delivered correctly
- Verify Content-Type headers are set
- Test error cases (missing image, invalid object name)

**Step 8.5: Test Frontend Integration**
- Load products and verify images display
- Test image loading errors handling
- Test image access patterns
- Verify no external image references remain

---

## **Implementation Priority and Dependencies**

### **Critical Path** (Must be in order):
1. **Phase 1** → Database schema changes
2. **Phase 2** → Product-loader updates for new schema
3. **Phase 3** → Catalog service updates for Minio image serving
4. **Phase 4** → Image serving API endpoints
5. **Phase 5** → Catalog service cleanup and validation
6. **Phase 6** → Product-admin service updates
7. **Phase 7** → API documentation update
8. **Phase 8** → Validation and testing (not implemented, handled separately)

### **Phase Dependencies**:
- Phase 1 must complete before any application code changes (Phase 2-6)
- Phase 2 and Phase 3 can be developed in parallel by different developers
- Phase 6 can begin once Phase 4 endpoints are stable
- Phase 5 and Phase 6 are independent service layers

### **Parallel Development Opportunities**:
- **Phase 1 & Phase 2**: Database schema and loader updates (different repositories)
- **Phase 3 & 4**: Catalog service and endpoints development (separate functions)
- **Phase 5 & 6**: Service cleanup and admin updates (different services)
- **Phase 7**: Documentation can be maintained throughout or done at end

### **Risk Mitigation**:
- **Database Rebuild**: All testing is handled separately; database will be dropped and recreated
- **Image Loading**: Verify Minio object names are unique and accessible during loader verification
- **URL Expiry**: Document presigned URL expiry (1 hour) in API documentation
- **Service Integration**: Test catalog and admin APIs together after Phase 6
- **Error Handling**: Update error logging in service methods for image retrieval failures

### **Assumed Outcome**:
- Complete removal of external URL references from database and application code
- Product-loader successfully uploads all images to Minio
- Catalog and admin services return images with presigned URLs only
- No individual image management endpoints in product-admin
- Clean separation between product-image data storage and image serving access

---

## **Key Benefits of This Approach**

1. **Clean Architecture**: Eliminates dual storage and external URL references
2. **Performance**: Direct Minio serving with presigned URLs for temporary access
3. **Security**: Time-limited URLs prevent unauthorized access
4. **Simplicity**: Admin service only handles bulk import (no individual image management)
5. **Forward Compatible**: Foundation for future image processing (resizing, optimization)
6. **Flexible Serving**: Both presigned URLs and direct access for different use cases

---

## **Summary of Schema Changes**

```sql
-- products table
DROP COLUMN main_image;
-- Fields: id, name, description, initial_price, final_price, currency, ... image_count, ...

-- product_images table
DROP COLUMN image_url;
-- Fields: id, product_id, minio_object_name, is_main, image_order, file_size, content_type, created_at
-- Removed: image_url
```

## **CSV Processing Changes**

- **Input**: CSV with `main_image` and `image_urls` columns (backward compatible)
- **Processing**: Download to Minio, store only `minio_object_name` and processing metadata
- **Database**: Insert without `image_url`, use position and/or `main_image` column to set `is_main`
- **Products**: Loaded via product-loader only (no main_image storage)

## **API Changes Summary**

- **Product Endpoints**: Remove main_image from responses, always include images array with URLs
- **New Image Endpoints**:
  - GET /api/v1/products/{id}/images - All images with presigned URLs
  - GET /api/v1/products/{id}/main-image - Main image presigned URL
  - GET /api/v1/images/{objectName} - Direct Minio image access
- **Presigned URLs**: 1-hour expiry for temporary access
- **Content-Type**: Correct headers for direct image access

## **Service Changes Summary**

- **Product-Loader**: Updated to use Minio storage, processes existing CSV format
- **Catalog Service**: Updated to serve Minio images with URL generation
- **Product-Admin**: Updated to work with new schema, includes Minio client for image loading
- **No Image Management**: Admin only handles product CRUD, images managed via product-loader

## **Admin Service Capabilities**

- **Product CRUD**: Create, Read, Update, Delete individual products
- **No Image Operations**: No AddProductImage, UpdateProductImage, DeleteProductImage endpoints
- **Image Loading**: Admin service uses GetProductImagesWithUrls for image data
- **Minio Integration**: Admin service initialized with Minio client for image access