# Product Catalog API - Frontend Development Guide

This document describes the API endpoints exposed by the Product Catalog service (`./cmd/product`) for use in building the products portion of the front-end shopping application.

## API Overview

**Base URL:** `http://localhost:{port}` (port configured via service config)

**Version:** `/api/v1`

**Content-Type:** All responses are JSON (`application/json`)

**CORS:** CORS is enabled for frontend access

**Authentication:** No authentication required for catalog endpoints (read-only access)

---

## Product Entity Reference

All endpoints return product objects with the following structure:

```json
{
  "id": 1,
  "name": "Junsun Android 13 Car Radio Stereo",
  "description": "Free Returns ✓ Free Shipping ✓. Junsun Android 13 Car Radio Stereo...",
  "initial_price": 148.60,
  "final_price": 105.50,
  "currency": "USD",
  "in_stock": true,
  "color": "WiFi 2+32GB",
  "size": "9\"",
  "country_code": "US",
  "image_count": 22,
  "model_number": "",
  "other_attributes": "{}",
  "root_category": "Automotive",
  "category": "Car Electronics",
  "brand": "Junsun",
  "all_available_sizes": "[\"S\",\"M\",\"L\"]",
  "created_at": "2026-01-15T10:30:00Z",
  "updated_at": "2026-01-15T10:30:00Z",
  "images": [
    {
          "id": 1,
          "product_id": 1,
          "minio_object_name": "products/1/image_1.jpg",
      "is_main": false,
      "image_order": 1,
      "file_size": 98765,
      "content_type": "image/jpeg",
      "created_at": "2026-01-15T10:30:00Z"
    }
  ],
  "count": 2
}
```

**Error Response:** `404 Not Found`

```json
{
  "error_type": "not_found",
  "message": "No images found for product"
}
```

**Use Case:** Product image gallery on product detail page

---

### 8. Get Product Main Image

**Endpoint:** `GET /api/v1/products/{id}/main-image`

**Description:** Retrieve only the main image (the one with `is_main: true`) for a specific product. Use the direct image endpoint to fetch image data. Use this for product thumbnails, cards, or when you only need the primary image metadata.

**Path Parameters:**

| Parameter | Type | Validation | Description |
|-----------|------|------------|-------------|
| `id` | integer | Required, positive | Unique product identifier |

**Example Request:**
```
GET /api/v1/products/1/main-image
```

**Success Response:** `200 OK`

```json
{
  "id": 1,
  "product_id": 1,
  "minio_object_name": "products/1/image_0.jpg",
  "is_main": true,
  "image_order": 0,
  "file_size": 152344,
  "content_type": "image/jpeg",
  "created_at": "2026-01-15T10:30:00Z"
}
```

**Error Response:** `404 Not Found`

```json
{
  "error_type": "not_found",
  "message": "No main image found for product"
}
```

**Use Case:** Product thumbnails in catalog lists, product cards, search results

---

### 9. Get Direct Image

**Endpoint:** `GET /api/v1/products/{id}/images/{imageName}`

**Description:** Stream an image directly from Minio storage. This endpoint returns the raw image data with appropriate cache headers. This is the recommended way to access product images as it doesn't use presigned URLs and avoids signature mismatch issues.

**Path Parameters:**

| Parameter | Type | Validation | Description |
|-----------|------|------------|-------------|
| `id` | integer | Required, positive | Product ID |
| `imageName` | string | Required | Image filename (e.g., "image_0.jpg") |

**Example Request:**
```
GET /api/v1/products/40121298/images/image_0.jpg
```

**Success Response:** `200 OK`

Returns raw image bytes with headers:
- `Content-Type`: image/jpeg (or appropriate MIME type)
- `Cache-Control: public, max-age=3600` (1 hour caching)
- `ETag`: Object ETag for cache validation

**Error Response:** `404 Not Found`

```json
{
  "error_type": "not_found",
  "message": "Image not found"
}
```

**Security Note:** Object names are validated to prevent path traversal attacks (e.g., "../" is rejected).

**Use Case:** Direct image embedding with CDN-style caching, image hotlinking

---

## Pagination Guide

All endpoints except `GET /products/{id}` support pagination. Use the `limit` and `offset` parameters to implement pagination controls:

- `limit`: Number of items per page (default: 50, max: 1000)
- `offset`: Number of items to skip (calculated as: `(page_number - 1) * limit`)

**Example Pagination Logic:**

```
Page 1: offset=0, limit=20
Page 2: offset=20, limit=20
Page 3: offset=40, limit=20
```

**Implementation Tips:**

- Display "Showing X-Y of Z results" using `count`, `limit`, and `offset`
- Disable "Previous" button when `offset = 0`
- Disable "Next" button when `count < limit`
- Consider implementing infinite scroll for better UX

---

## Common Response Patterns

### List Response Structure
All list endpoints return objects with this structure:
```json
{
  "products": [...],      // Array of product objects
  "limit": 20,           // Pagination limit requested
  "offset": 0,           // Pagination offset requested
  "count": 2,            // Number of products returned
  // Additional fields:
  "category": "...",     // For category endpoint
  "brand": "...",        // For brand endpoint
  "query": "..."         // For search endpoint
}
```

### Error Response Structure
All errors follow this pattern:
```json
{
  "error_type": "error_code",  // Error type identifier
  "message": "Human readable error message"
}
```

---

## Service Configuration

The Product Catalog service is configured via the following environment variables (set in service config):

| Variable | Description | Example |
|----------|-------------|---------|
| `db_url` | PostgreSQL connection string | `postgres://user:pass@localhost/products` |
| `product_service_port` | HTTP server port | `:8081` |
| `minio_bucket` | MinIO bucket for product images | `productimages` |

---

## Image Handling Guide

### Overview

Product images are stored in Minio object storage and accessed via a **direct streaming endpoint**. You must construct image URLs using the direct endpoint pattern.

### Key Points

1. **Use direct endpoint**: Access images via `GET /api/v1/products/{id}/images/{imageName}`
3. **`minio_object_name` provides the path**: Use this to construct the image name for the URL
4. **Main image flag**: Use `is_main: true` to identify the primary product image

### Constructing Image URLs

To display an image, construct the URL using the product ID and image name extracted from `minio_object_name`:

```typescript
// Extract image name from minio_object_name
// Example: "products/40121298/image_0.jpg" -> "image_0.jpg"
function getImageName(minioObjectName: string): string {
  const parts = minioObjectName.split('/');
  return parts[parts.length - 1];
}

// Construct the direct image URL
function getImageUrl(productId: number, minioObjectName: string): string {
  const imageName = getImageName(minioObjectName);
  return `/api/v1/products/${productId}/images/${imageName}`;
}

// Example usage
const imageUrl = getImageUrl(40121298, 'products/40121298/image_0.jpg');
// Result: /api/v1/products/40121298/images/image_0.jpg
```

### Finding the Main Image

```typescript
// Find main image from product.images array
const mainImage = product.images.find(img => img.is_main);

// Get the URL for the main image
const mainImageUrl = mainImage 
  ? getImageUrl(product.id, mainImage.minio_object_name)
  : '/assets/placeholder.jpg';
```

### Example: Product Card Component

```typescript
@Component({
  selector: 'app-product-card',
  template: `
    <div class="product-card">
      <img [src]="mainImageUrl" [alt]="product.name" loading="lazy">
      <h3>{{ product.name }}</h3>
      <p class="price">{{ product.final_price | currency:product.currency }}</p>
    </div>
  `
})
export class ProductCardComponent {
  @Input() product!: Product;

  get mainImageUrl(): string {
    const mainImage = this.product.images.find(img => img.is_main);
    if (!mainImage) {
      return '/assets/placeholder.jpg';
    }
    const imageName = mainImage.minio_object_name.split('/').pop();
    return `/api/v1/products/${this.product.id}/images/${imageName}`;
  }
}
```

### Example: Product Gallery Component

```typescript
@Component({
  selector: 'app-product-gallery',
  template: `
    <div class="gallery">
      <img [src]="selectedImageUrl" [alt]="product.name">
      <div class="thumbnails">
        <img *ngFor="let img of product.images"
             [src]="getImageUrl(img)"
             [class.active]="img.id === selectedImage?.id"
             (click)="selectedImage = img"
             loading="lazy">
      </div>
    </div>
  `
})
export class ProductGalleryComponent {
  @Input() product!: Product;
  selectedImage: ProductImage | undefined;

  ngOnInit() {
    this.selectedImage = this.product.images.find(img => img.is_main);
  }

  getImageUrl(image: ProductImage): string {
    const imageName = image.minio_object_name.split('/').pop();
    return `/api/v1/products/${image.product_id}/images/${imageName}`;
  }

  get selectedImageUrl(): string {
    if (!this.selectedImage) {
      return '/assets/placeholder.jpg';
    }
    return this.getImageUrl(this.selectedImage);
  }
}
```

### Direct Image Endpoint

For direct streaming of image bytes (useful for downloads or when you need the raw image data):

```typescript
getDirectImage(productId: number, imageName: string): Observable<Blob> {
  return this.http.get(`${this.apiUrl}/products/${productId}/images/${imageName}`, {
    responseType: 'blob'
  });
}
```

---

## Frontend Development Notes

1. **Image Loading**: Product images are accessed via direct endpoint. The `images` array contains all product images. Construct URLs using the pattern `/api/v1/products/{productId}/images/{imageName}`. Use `is_main` flag to identify the primary image
2. **Currency Formatting**: Products use `currency` field (default USD). Format prices appropriately
3. **Discount Display**: Show discount badge/indicator when `final_price < initial_price`
4. **Stock Indicators**: Use `in_stock` field to control "Add to Cart" button state
5. **Search UX**: Implement debouncing for search input to avoid excessive API calls
6. **Category Navigation**: Consider fetching all unique categories to build navigation menu
7. **Brand Filtering**: Consider fetching all unique brands to build brand filter
8. **URL Encoding**: Category and brand names in URL paths must be URL-encoded (e.g., spaces to `%20`)

---

## Example Frontend Integration

### Angular Service Example

```typescript
@Injectable({ providedIn: 'root' })
export class ProductService {
  private apiUrl = 'http://localhost:8081/api/v1';

  constructor(private http: HttpClient) {}

  getAllProducts(limit: number = 50, offset: number = 0): Observable<any> {
    return this.http.get(`${this.apiUrl}/products`, {
      params: { limit: limit.toString(), offset: offset.toString() }
    });
  }

  getProductById(id: number): Observable<Product> {
    return this.http.get<Product>(`${this.apiUrl}/products/${id}`);
  }

  searchProducts(query: string, limit: number = 50, offset: number = 0): Observable<any> {
    return this.http.get(`${this.apiUrl}/products/search`, {
      params: { q: query, limit: limit.toString(), offset: offset.toString() }
    });
  }

  getProductsByCategory(category: string, limit: number = 50, offset: number = 0): Observable<any> {
    return this.http.get(`${this.apiUrl}/products/category/${encodeURIComponent(category)}`, {
      params: { limit: limit.toString(), offset: offset.toString() }
    });
  }

  getProductsByBrand(brand: string, limit: number = 50, offset: number = 0): Observable<any> {
    return this.http.get(`${this.apiUrl}/products/brand/${encodeURIComponent(brand)}`, {
      params: { limit: limit.toString(), offset: offset.toString() }
    });
  }

  getProductsInStock(limit: number = 50, offset: number = 0): Observable<any> {
    return this.http.get(`${this.apiUrl}/products/in-stock`, {
      params: { limit: limit.toString(), offset: offset.toString() }
    });
  }

  getProductImages(productId: number): Observable<any> {
    return this.http.get(`${this.apiUrl}/products/${productId}/images`);
  }

  getProductMainImage(productId: number): Observable<ProductImage> {
    return this.http.get<ProductImage>(`${this.apiUrl}/products/${productId}/main-image`);
  }

  // Helper method to find main image from product
  getMainImage(product: Product): ProductImage | undefined {
    return product.images.find(img => img.is_main);
  }

  // Helper method to extract image name from minio_object_name
  // Example: "products/40121298/image_0.jpg" -> "image_0.jpg"
  getImageName(minioObjectName: string): string {
    const parts = minioObjectName.split('/');
    return parts[parts.length - 1];
  }

  // Helper method to construct direct image URL
  getImageUrl(productId: number, minioObjectName: string): string {
    const imageName = this.getImageName(minioObjectName);
    return `${this.apiUrl}/products/${productId}/images/${imageName}`;
  }
}
```

### Product Interface

```typescript
export interface Product {
  id: number;
  name: string;
  description: string;
  initial_price: number;
  final_price: number;
  currency: string;
  in_stock: boolean;
  color: string;
  size: string;
  country_code: string;
  image_count: number;
  model_number: string;
  other_attributes: string;
  root_category: string;
  category: string;
  brand: string;
  all_available_sizes: string;
  created_at: string;
  updated_at: string;
  images: ProductImage[];
}

export interface ProductImage {
  id: number;
  product_id: number;
  minio_object_name: string;
  is_main: boolean;
  image_order: number;
  file_size: number;
  content_type: string;
  created_at: string;
}
```

---

## Testing

When developing the frontend, ensure you test:
- All 9 endpoints with various parameter combinations
- Image URL construction from `minio_object_name`
- Direct image endpoint functionality
- Pagination edge cases (empty results, single page, multiple pages)
- Error conditions (invalid product ID, empty search query, missing images)
- Performance with large result sets
- URL encoding for category/brand names with special characters
- Main image detection using `is_main` flag
- Products with no images (empty `images` array)
