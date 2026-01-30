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
  "main_image": "https://img.ltwebstatic.com/images3_spmp/2024/08/22/8d/1724320460505d5f92185ac1622479673d08b3e9b3.jpg",
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
  "images": []
}
```

### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | Unique product identifier |
| `name` | string | Product name (max 500 chars) |
| `description` | string | Full product description |
| `initial_price` | float | Original/listing price |
| `final_price` | float | Current selling price |
| `currency` | string | 3-letter ISO currency code |
| `in_stock` | boolean | Whether product is currently available |
| `color` | string | Product color variant |
| `size` | string | Product size |
| `main_image` | string | URL to primary product image |
| `country_code` | string | 2-letter ISO country code |
| `image_count` | integer | Number of product images |
| `model_number` | string | Product model/SKU |
| `other_attributes` | string | JSON string with additional attributes |
| `root_category` | string | Top-level category |
| `category` | string | Specific product category |
| `brand` | string | Product brand |
| `all_available_sizes` | string | JSON string array of available sizes |
| `created_at` | datetime | Product creation timestamp |
| `updated_at` | datetime | Last update timestamp |
| `images` | array | Additional product images (currently empty) |

### Frontend Integration Notes

- **Price Display**: Show `final_price` as current price. If `final_price < initial_price`, display both prices to show discount
- **Discount Calculation**: Use formula `((initial_price - final_price) / initial_price) * 100` to calculate discount percentage
- **Availability**: Use `in_stock` flag to show/hide "Add to Cart" or "Out of Stock" status
- **Images**: Use `main_image` for product thumbnails and main display
- **Search**: Product search is case-insensitive full-text search across name, description, and other fields

---

## Endpoints

### 1. List All Products

**Endpoint:** `GET /api/v1/products`

**Description:** Retrieve a paginated list of all products in the catalog. Use this for the main product catalog page, with pagination controls to navigate through the full inventory.

**Query Parameters:**

| Parameter | Type | Default | Range | Description |
|-----------|------|---------|-------|-------------|
| `limit` | integer | 50 | 1-1000 | Number of products to return |
| `offset` | integer | 0 | ≥0 | Number of products to skip (for pagination) |

**Example Request:**
```
GET /api/v1/products?limit=20&offset=0
```

**Success Response:** `200 OK`

```json
{
  "products": [
    {
      "id": 1,
      "name": "Junsun Android 13 Car Radio Stereo For Hundai Santa Fe 2 2006 2007 2008 2009 2010 2011 2012 Car Auto Radio Built-In Wireless Carplay For Apple & Android Auto 9 Inch Automotive Multimedia Touch Screen 2GB RAM 32GB ROM Car Intelligent Systems Head Unit With GPS Navigation DSP Bluetooth",
      "description": "Free Returns ✓ Free Shipping ✓. Junsun Android 13 Car Radio Stereo For Hundai Santa Fe 2 2006 2007 2008 2009 2010 2011 2012 Car Auto Radio Built-In Wireless Carplay For Apple & Android Auto 9 Inch Automotive Multimedia Touch Screen 2GB RAM 32GB ROM Car Intelligent Systems Head Unit With GPS Navigation DSP Bluetooth- Car Intelligence System at SHEIN.",
      "initial_price": 148.60,
      "final_price": 105.50,
      "currency": "USD",
      "in_stock": true,
      "color": "WiFi 2+32GB",
      "size": "9\"",
      "main_image": "https://img.ltwebstatic.com/images3_spmp/2024/08/22/8d/1724320460505d5f92185ac1622479673d08b3e9b3.jpg",
      "country_code": "US",
      "image_count": 22,
      "model_number": "",
      "other_attributes": "{}",
      "root_category": "Automotive",
      "category": "Car Electronics",
      "brand": "Junsun",
      "all_available_sizes": "[]",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-15T10:30:00Z",
      "images": []
    },
    {
      "id": 2,
      "name": "Couples TPU Black & Silver Car Anti-Drop Key Case + For Chevrolet Malibu XI/Corvette/Cruze/Trax/Sail Exclusive Use",
      "description": "Free Returns ✓ Free Shipping ✓. Couples TPU Black & Silver Car Anti-Drop Key Case + For Chevrolet Malibu XI/Corvette/Cruze/Trax/Sail Exclusive Use- Car Key Case at SHEIN.",
      "initial_price": 2.30,
      "final_price": 2.30,
      "currency": "USD",
      "in_stock": true,
      "color": "Black",
      "size": "one-size",
      "main_image": "https://img.ltwebstatic.com/images3_spmp/2024/08/02/4a/1722580878d4864b8ed432eca43af9e8a9c2226916.jpg",
      "country_code": "US",
      "image_count": 15,
      "model_number": "",
      "other_attributes": "{}",
      "root_category": "Automotive",
      "category": "Car Accessories",
      "brand": "SHEIN",
      "all_available_sizes": "[]",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-15T10:30:00Z",
      "images": []
    }
  ],
  "limit": 20,
  "offset": 0,
  "count": 2
}
```

**Use Case:** Display product catalog with pagination controls (Previous/Next, page numbers)

---

### 2. Get Product by ID

**Endpoint:** `GET /api/v1/products/{id}`

**Description:** Retrieve detailed information for a single product by its unique ID. Use this for the product detail page when a user clicks on a product from the catalog.

**Path Parameters:**

| Parameter | Type | Validation | Description |
|-----------|------|------------|-------------|
| `id` | integer | Required, positive | Unique product identifier |

**Example Request:**
```
GET /api/v1/products/1
```

**Success Response:** `200 OK`

```json
{
  "id": 1,
  "name": "Junsun Android 13 Car Radio Stereo For Hundai Santa Fe 2 2006 2007 2008 2009 2010 2011 2012 Car Auto Radio Built-In Wireless Carplay For Apple & Android Auto 9 Inch Automotive Multimedia Touch Screen 2GB RAM 32GB ROM Car Intelligent Systems Head Unit With GPS Navigation DSP Bluetooth",
  "description": "Free Returns ✓ Free Shipping ✓. Junsun Android 13 Car Radio Stereo For Hundai Santa Fe 2 2006 2007 2008 2009 2010 2011 2012 Car Auto Radio Built-In Wireless Carplay For Apple & Android Auto 9 Inch Automotive Multimedia Touch Screen 2GB RAM 32GB ROM Car Intelligent Systems Head Unit With GPS Navigation DSP Bluetooth- Car Intelligence System at SHEIN.",
  "initial_price": 148.60,
  "final_price": 105.50,
  "currency": "USD",
  "in_stock": true,
  "color": "WiFi 2+32GB",
  "size": "9\"",
  "main_image": "https://img.ltwebstatic.com/images3_spmp/2024/08/22/8d/1724320460505d5f92185ac1622479673d08b3e9b3.jpg",
  "country_code": "US",
  "image_count": 22,
  "model_number": "",
  "other_attributes": "{}",
  "root_category": "Automotive",
  "category": "Car Electronics",
  "brand": "Junsun",
  "all_available_sizes": "[]",
  "created_at": "2026-01-15T10:30:00Z",
  "updated_at": "2026-01-15T10:30:00Z",
  "images": []
}
```

**Error Response:** `404 Not Found`

```json
{
  "error_type": "not_found",
  "message": "Product not found"
}
```

**Use Case:** Product detail page showing full product information, pricing, and add-to-cart functionality

---

### 3. Search Products

**Endpoint:** `GET /api/v1/products/search`

**Description:** Search products by query string. Performs full-text search across product name, description, and other text fields. Use this for the search bar functionality.

**Query Parameters:**

| Parameter | Type | Required | Default | Range | Description |
|-----------|------|----------|---------|-------|-------------|
| `q` | string | Yes | - | - | Search query string |
| `limit` | integer | No | 50 | 1-1000 | Number of products to return |
| `offset` | integer | No | 0 | ≥0 | Number of products to skip |

**Example Request:**
```
GET /api/v1/products/search?q=Chevrolet&limit=10&offset=0
```

**Success Response:** `200 OK`

```json
{
  "products": [
    {
      "id": 2,
      "name": "Couples TPU Black & Silver Car Anti-Drop Key Case + For Chevrolet Malibu XI/Corvette/Cruze/Trax/Sail Exclusive Use",
      "description": "Free Returns ✓ Free Shipping ✓. Couples TPU Black & Silver Car Anti-Drop Key Case + For Chevrolet Malibu XI/Corvette/Cruze/Trax/Sail Exclusive Use- Car Key Case at SHEIN.",
      "initial_price": 2.30,
      "final_price": 2.30,
      "currency": "USD",
      "in_stock": true,
      "color": "Black",
      "size": "one-size",
      "main_image": "https://img.ltwebstatic.com/images3_spmp/2024/08/02/4a/1722580878d4864b8ed432eca43af9e8a9c2226916.jpg",
      "country_code": "US",
      "image_count": 15,
      "model_number": "",
      "other_attributes": "{}",
      "root_category": "Automotive",
      "category": "Car Accessories",
      "brand": "SHEIN",
      "all_available_sizes": "[]",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-15T10:30:00Z",
      "images": []
    },
    {
      "id": 3,
      "name": "Chevrolet Camaro 2016-2023 Front Bumper Splitter Carbon Fiber",
      "description": "High-quality carbon fiber front bumper splitter compatible with Chevrolet Camaro 2016-2023 models",
      "initial_price": 299.99,
      "final_price": 249.99,
      "currency": "USD",
      "in_stock": true,
      "color": "Carbon Fiber",
      "size": "Standard",
      "main_image": "https://example.com/images/camaro-splitter.jpg",
      "country_code": "US",
      "image_count": 8,
      "model_number": "CAM-FBS-2023",
      "other_attributes": "{}",
      "root_category": "Automotive",
      "category": "Car Accessories",
      "brand": "AutoParts Pro",
      "all_available_sizes": "[\"Standard\",\"Custom\"]",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-15T10:30:00Z",
      "images": []
    }
  ],
  "query": "Chevrolet",
  "limit": 10,
  "offset": 0,
  "count": 2
}
```

**Error Response:** `400 Bad Request`

```json
{
  "error_type": "invalid_request",
  "message": "Missing search query parameter 'q'"
}
```

**Use Case:** Search results page after user enters a search query

---

### 4. Get Products by Category

**Endpoint:** `GET /api/v1/products/category/{category}`

**Description:** Retrieve products filtered by category. Use this for category browsing pages, category navigation, and filtering results.

**Path Parameters:**

| Parameter | Type | Validation | Description |
|-----------|------|------------|-------------|
| `category` | string | Required | Product category name (e.g., "Car Electronics", "Car Accessories") |

**Query Parameters:**

| Parameter | Type | Default | Range | Description |
|-----------|------|---------|-------|-------------|
| `limit` | integer | 50 | 1-1000 | Number of products to return |
| `offset` | integer | 0 | ≥0 | Number of products to skip |

**Example Request:**
```
GET /api/v1/products/category/Car%20Electronics?limit=10&offset=0
```

**Success Response:** `200 OK`

```json
{
  "products": [
    {
      "id": 1,
      "name": "Junsun Android 13 Car Radio Stereo For Hundai Santa Fe 2 2006 2007 2008 2009 2010 2011 2012 Car Auto Radio Built-In Wireless Carplay For Apple & Android Auto 9 Inch Automotive Multimedia Touch Screen 2GB RAM 32GB ROM Car Intelligent Systems Head Unit With GPS Navigation DSP Bluetooth",
      "description": "Free Returns ✓ Free Shipping ✓. Junsun Android 13 Car Radio Stereo For Hundai Santa Fe 2 2006 2007 2008 2009 2010 2011 2012 Car Auto Radio Built-In Wireless Carplay For Apple & Android Auto 9 Inch Automotive Multimedia Touch Screen 2GB RAM 32GB ROM Car Intelligent Systems Head Unit With GPS Navigation DSP Bluetooth- Car Intelligence System at SHEIN.",
      "initial_price": 148.60,
      "final_price": 105.50,
      "currency": "USD",
      "in_stock": true,
      "color": "WiFi 2+32GB",
      "size": "9\"",
      "main_image": "https://img.ltwebstatic.com/images3_spmp/2024/08/22/8d/1724320460505d5f92185ac1622479673d08b3e9b3.jpg",
      "country_code": "US",
      "image_count": 22,
      "model_number": "",
      "other_attributes": "{}",
      "root_category": "Automotive",
      "category": "Car Electronics",
      "brand": "Junsun",
      "all_available_sizes": "[]",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-15T10:30:00Z",
      "images": []
    },
    {
      "id": 4,
      "name": "Wireless Carplay Adapter 5G WiFi Auto Connect",
      "description": "Convert wired CarPlay to wireless with 5G WiFi. Automatic connection, easy installation. Compatible with most car models.",
      "initial_price": 89.99,
      "final_price": 69.99,
      "currency": "USD",
      "in_stock": true,
      "color": "Black",
      "size": "Compact",
      "main_image": "https://example.com/images/carplay-adapter.jpg",
      "country_code": "US",
      "image_count": 5,
      "model_number": "CP-5G-WF",
      "other_attributes": "{}",
      "root_category": "Automotive",
      "category": "Car Electronics",
      "brand": "TechDrive",
      "all_available_sizes": "[]",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-15T10:30:00Z",
      "images": []
    }
  ],
  "category": "Car Electronics",
  "limit": 10,
  "offset": 0,
  "count": 2
}
```

**Error Response:** `400 Bad Request`

```json
{
  "error_type": "invalid_request",
  "message": "Missing category in path"
}
```

**Use Case:** Category page when user clicks on a category from navigation menu

---

### 5. Get Products by Brand

**Endpoint:** `GET /api/v1/products/brand/{brand}`

**Description:** Retrieve products filtered by brand. Use this for brand-specific browsing and brand pages.

**Path Parameters:**

| Parameter | Type | Validation | Description |
|-----------|------|------------|-------------|
| `brand` | string | Required | Brand name (e.g., "Junsun", "SHEIN", "AutoParts Pro") |

**Query Parameters:**

| Parameter | Type | Default | Range | Description |
|-----------|------|---------|-------|-------------|
| `limit` | integer | 50 | 1-1000 | Number of products to return |
| `offset` | integer | 0 | ≥0 | Number of products to skip |

**Example Request:**
```
GET /api/v1/products/brand/Junsun?limit=20&offset=0
```

**Success Response:** `200 OK`

```json
{
  "products": [
    {
      "id": 1,
      "name": "Junsun Android 13 Car Radio Stereo For Hundai Santa Fe 2 2006 2007 2008 2009 2010 2011 2012 Car Auto Radio Built-In Wireless Carplay For Apple & Android Auto 9 Inch Automotive Multimedia Touch Screen 2GB RAM 32GB ROM Car Intelligent Systems Head Unit With GPS Navigation DSP Bluetooth",
      "description": "Free Returns ✓ Free Shipping ✓. Junsun Android 13 Car Radio Stereo For Hundai Santa Fe 2 2006 2007 2008 2009 2010 2011 2012 Car Auto Radio Built-In Wireless Carplay For Apple & Android Auto 9 Inch Automotive Multimedia Touch Screen 2GB RAM 32GB ROM Car Intelligent Systems Head Unit With GPS Navigation DSP Bluetooth- Car Intelligence System at SHEIN.",
      "initial_price": 148.60,
      "final_price": 105.50,
      "currency": "USD",
      "in_stock": true,
      "color": "WiFi 2+32GB",
      "size": "9\"",
      "main_image": "https://img.ltwebstatic.com/images3_spmp/2024/08/22/8d/1724320460505d5f92185ac1622479673d08b3e9b3.jpg",
      "country_code": "US",
      "image_count": 22,
      "model_number": "",
      "other_attributes": "{}",
      "root_category": "Automotive",
      "category": "Car Electronics",
      "brand": "Junsun",
      "all_available_sizes": "[]",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-15T10:30:00Z",
      "images": []
    },
    {
      "id": 5,
      "name": "Junsun 7\" Head Unit Android 12 Car Stereo",
      "description": "7-inch Android 12 head unit with GPS, Bluetooth, and WiFi. Easy installation with plug-and-play design.",
      "initial_price": 119.99,
      "final_price": 99.99,
      "currency": "USD",
      "in_stock": true,
      "color": "Black",
      "size": "7\"",
      "main_image": "https://example.com/images/junsun-7inch.jpg",
      "country_code": "US",
      "image_count": 12,
      "model_number": "JS-AND12-7",
      "other_attributes": "{}",
      "root_category": "Automotive",
      "category": "Car Electronics",
      "brand": "Junsun",
      "all_available_sizes": "[]",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-15T10:30:00Z",
      "images": []
    }
  ],
  "brand": "Junsun",
  "limit": 20,
  "offset": 0,
  "count": 2
}
```

**Error Response:** `400 Bad Request`

```json
{
  "error_type": "invalid_request",
  "message": "Missing brand in path"
}
```

**Use Case:** Brand page when user clicks on a brand from product details or navigation

---

### 6. Get In-Stock Products

**Endpoint:** `GET /api/v1/products/in-stock`

**Description:** Retrieve only products that are currently in stock (`in_stock: true`). Use this for "Available Products" section, "Shop Now" page, or when user filters by availability.

**Query Parameters:**

| Parameter | Type | Default | Range | Description |
|-----------|------|---------|-------|-------------|
| `limit` | integer | 50 | 1-1000 | Number of products to return |
| `offset` | integer | 0 | ≥0 | Number of products to skip |

**Example Request:**
```
GET /api/v1/products/in-stock?limit=15&offset=0
```

**Success Response:** `200 OK`

```json
{
  "products": [
    {
      "id": 1,
      "name": "Junsun Android 13 Car Radio Stereo For Hundai Santa Fe 2 2006 2007 2008 2009 2010 2011 2012 Car Auto Radio Built-In Wireless Carplay For Apple & Android Auto 9 Inch Automotive Multimedia Touch Screen 2GB RAM 32GB ROM Car Intelligent Systems Head Unit With GPS Navigation DSP Bluetooth",
      "description": "Free Returns ✓ Free Shipping ✓. Junsun Android 13 Car Radio Stereo For Hundai Santa Fe 2 2006 2007 2008 2009 2010 2011 2012 Car Auto Radio Built-In Wireless Carplay For Apple & Android Auto 9 Inch Automotive Multimedia Touch Screen 2GB RAM 32GB ROM Car Intelligent Systems Head Unit With GPS Navigation DSP Bluetooth- Car Intelligence System at SHEIN.",
      "initial_price": 148.60,
      "final_price": 105.50,
      "currency": "USD",
      "in_stock": true,
      "color": "WiFi 2+32GB",
      "size": "9\"",
      "main_image": "https://img.ltwebstatic.com/images3_spmp/2024/08/22/8d/1724320460505d5f92185ac1622479673d08b3e9b3.jpg",
      "country_code": "US",
      "image_count": 22,
      "model_number": "",
      "other_attributes": "{}",
      "root_category": "Automotive",
      "category": "Car Electronics",
      "brand": "Junsun",
      "all_available_sizes": "[]",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-15T10:30:00Z",
      "images": []
    },
    {
      "id": 2,
      "name": "Couples TPU Black & Silver Car Anti-Drop Key Case + For Chevrolet Malibu XI/Corvette/Cruze/Trax/Sail Exclusive Use",
      "description": "Free Returns ✓ Free Shipping ✓. Couples TPU Black & Silver Car Anti-Drop Key Case + For Chevrolet Malibu XI/Corvette/Cruze/Trax/Sail Exclusive Use- Car Key Case at SHEIN.",
      "initial_price": 2.30,
      "final_price": 2.30,
      "currency": "USD",
      "in_stock": true,
      "color": "Black",
      "size": "one-size",
      "main_image": "https://img.ltwebstatic.com/images3_spmp/2024/08/02/4a/1722580878d4864b8ed432eca43af9e8a9c2226916.jpg",
      "country_code": "US",
      "image_count": 15,
      "model_number": "",
      "other_attributes": "{}",
      "root_category": "Automotive",
      "category": "Car Accessories",
      "brand": "SHEIN",
      "all_available_sizes": "[]",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-15T10:30:00Z",
      "images": []
    },
    {
      "id": 4,
      "name": "Wireless Carplay Adapter 5G WiFi Auto Connect",
      "description": "Convert wired CarPlay to wireless with 5G WiFi. Automatic connection, easy installation. Compatible with most car models.",
      "initial_price": 89.99,
      "final_price": 69.99,
      "currency": "USD",
      "in_stock": true,
      "color": "Black",
      "size": "Compact",
      "main_image": "https://example.com/images/carplay-adapter.jpg",
      "country_code": "US",
      "image_count": 5,
      "model_number": "CP-5G-WF",
      "other_attributes": "{}",
      "root_category": "Automotive",
      "category": "Car Electronics",
      "brand": "TechDrive",
      "all_available_sizes": "[]",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-15T10:30:00Z",
      "images": []
    }
  ],
  "limit": 15,
  "offset": 0,
  "count": 3
}
```

**Use Case:** "Shop Now" homepage section, availability filter, or showing only purchasable products

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

## Frontend Development Notes

1. **Image Loading**: Product images are hosted externally. Implement lazy loading for performance
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
  main_image: string;
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
  image_url: string;
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
- All 6 endpoints with various parameter combinations
- Pagination edge cases (empty results, single page, multiple pages)
- Error conditions (invalid product ID, empty search query)
- Performance with large result sets
- URL encoding for category/brand names with special characters
