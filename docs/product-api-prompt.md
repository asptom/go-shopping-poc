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
      "image_url": "https://minio.example.com/productimages/products/1/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
      "minio_object_name": "products/1/image_0.jpg",
      "is_main": true,
      "image_order": 0,
      "file_size": 152344,
      "content_type": "image/jpeg",
      "created_at": "2026-01-15T10:30:00Z"
    }
  ]
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
| `images` | array | Product images array with presigned URLs (1-hour expiry) |

### Frontend Integration Notes

- **Price Display**: Show `final_price` as current price. If `final_price < initial_price`, display both prices to show discount
- **Discount Calculation**: Use formula `((initial_price - final_price) / initial_price) * 100` to calculate discount percentage
- **Availability**: Use `in_stock` flag to show/hide "Add to Cart" or "Out of Stock" status
- **Images**: Use the `images` array. Find the image with `is_main: true` for the primary display. All image URLs are presigned Minio URLs with 1-hour expiry
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
      "images": [
        {
          "id": 1,
          "product_id": 1,
          "image_url": "https://minio.example.com/productimages/products/1/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
          "minio_object_name": "products/1/image_0.jpg",
          "is_main": true,
          "image_order": 0,
          "file_size": 152344,
          "content_type": "image/jpeg",
          "created_at": "2026-01-15T10:30:00Z"
        }
      ]
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
      "images": [
        {
          "id": 2,
          "product_id": 2,
          "image_url": "https://minio.example.com/productimages/products/2/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
          "minio_object_name": "products/2/image_0.jpg",
          "is_main": true,
          "image_order": 0,
          "file_size": 98765,
          "content_type": "image/jpeg",
          "created_at": "2026-01-15T10:30:00Z"
        }
      ]
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
  "images": [
    {
      "id": 1,
      "product_id": 1,
      "image_url": "https://minio.example.com/productimages/products/1/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
      "minio_object_name": "products/1/image_0.jpg",
      "is_main": true,
      "image_order": 0,
      "file_size": 152344,
      "content_type": "image/jpeg",
      "created_at": "2026-01-15T10:30:00Z"
    }
  ]
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
      "images": [
        {
          "id": 2,
          "product_id": 2,
          "image_url": "https://minio.example.com/productimages/products/2/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
          "minio_object_name": "products/2/image_0.jpg",
          "is_main": true,
          "image_order": 0,
          "file_size": 98765,
          "content_type": "image/jpeg",
          "created_at": "2026-01-15T10:30:00Z"
        }
      ]
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
      "images": [
        {
          "id": 3,
          "product_id": 3,
          "image_url": "https://minio.example.com/productimages/products/3/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
          "minio_object_name": "products/3/image_0.jpg",
          "is_main": true,
          "image_order": 0,
          "file_size": 234567,
          "content_type": "image/jpeg",
          "created_at": "2026-01-15T10:30:00Z"
        }
      ]
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
      "images": [
        {
          "id": 1,
          "product_id": 1,
          "image_url": "https://minio.example.com/productimages/products/1/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
          "minio_object_name": "products/1/image_0.jpg",
          "is_main": true,
          "image_order": 0,
          "file_size": 152344,
          "content_type": "image/jpeg",
          "created_at": "2026-01-15T10:30:00Z"
        }
      ]
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
      "images": [
        {
          "id": 4,
          "product_id": 4,
          "image_url": "https://minio.example.com/productimages/products/4/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
          "minio_object_name": "products/4/image_0.jpg",
          "is_main": true,
          "image_order": 0,
          "file_size": 87654,
          "content_type": "image/jpeg",
          "created_at": "2026-01-15T10:30:00Z"
        }
      ]
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
      "images": [
        {
          "id": 1,
          "product_id": 1,
          "image_url": "https://minio.example.com/productimages/products/1/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
          "minio_object_name": "products/1/image_0.jpg",
          "is_main": true,
          "image_order": 0,
          "file_size": 152344,
          "content_type": "image/jpeg",
          "created_at": "2026-01-15T10:30:00Z"
        }
      ]
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
      "images": [
        {
          "id": 5,
          "product_id": 5,
          "image_url": "https://minio.example.com/productimages/products/5/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
          "minio_object_name": "products/5/image_0.jpg",
          "is_main": true,
          "image_order": 0,
          "file_size": 145678,
          "content_type": "image/jpeg",
          "created_at": "2026-01-15T10:30:00Z"
        }
      ]
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
      "images": [
        {
          "id": 1,
          "product_id": 1,
          "image_url": "https://minio.example.com/productimages/products/1/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
          "minio_object_name": "products/1/image_0.jpg",
          "is_main": true,
          "image_order": 0,
          "file_size": 152344,
          "content_type": "image/jpeg",
          "created_at": "2026-01-15T10:30:00Z"
        }
      ]
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
      "images": [
        {
          "id": 4,
          "product_id": 4,
          "image_url": "https://minio.example.com/productimages/products/4/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
          "minio_object_name": "products/4/image_0.jpg",
          "is_main": true,
          "image_order": 0,
          "file_size": 87654,
          "content_type": "image/jpeg",
          "created_at": "2026-01-15T10:30:00Z"
        }
      ]
    }
  ],
  "limit": 15,
  "offset": 0,
  "count": 3
}
```

**Use Case:** "Shop Now" homepage section, availability filter, or showing only purchasable products

---

### 7. Get Product Images

**Endpoint:** `GET /api/v1/products/{id}/images`

**Description:** Retrieve all images for a specific product with presigned URLs. Use this for product detail galleries or when you need to display all product images.

**Path Parameters:**

| Parameter | Type | Validation | Description |
|-----------|------|------------|-------------|
| `id` | integer | Required, positive | Unique product identifier |

**Example Request:**
```
GET /api/v1/products/1/images
```

**Success Response:** `200 OK`

```json
{
  "product_id": 1,
  "images": [
    {
      "id": 1,
      "product_id": 1,
      "image_url": "https://minio.example.com/productimages/products/1/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
      "minio_object_name": "products/1/image_0.jpg",
      "is_main": true,
      "image_order": 0,
      "file_size": 152344,
      "content_type": "image/jpeg",
      "created_at": "2026-01-15T10:30:00Z"
    },
    {
      "id": 2,
      "product_id": 1,
      "image_url": "https://minio.example.com/productimages/products/1/image_1.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
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

**Description:** Retrieve only the main image (the one with `is_main: true`) for a specific product with a presigned URL. Use this for product thumbnails, cards, or when you only need the primary image.

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
  "image_url": "https://minio.example.com/productimages/products/1/image_0.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
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

Product images are now stored in Minio object storage and served via **presigned URLs** with a **1-hour expiry**. This provides secure, temporary access to images without exposing the storage backend.

### Key Changes from Previous Implementation

1. **Removed `main_image` field**: The `main_image` field has been removed from Product entities
2. **New `images` array**: All products now include an `images` array containing `ProductImage` objects
3. **Presigned URLs**: Image URLs in `image_url` field are presigned Minio URLs (1-hour expiry)
4. **Main image flag**: Use `is_main: true` to identify the primary product image

### Image URL Types

There are two ways to access product images:

#### 1. Presigned URLs (Recommended for most use cases)
- **Source**: Returned in `image_url` field from all product endpoints
- **Expiry**: 1 hour from generation
- **Best for**: Product listings, detail pages, immediate display
- **Pros**: Direct access, no additional requests needed
- **Cons**: URLs expire after 1 hour (refresh by re-fetching product)

#### 2. Direct Image Endpoint (Best for caching/CDN)
- **Endpoint**: `GET /api/v1/images/{objectName}`
- **Best for**: Heavy traffic pages, CDN integration, long-term caching
- **Pros**: Better caching (1-hour browser cache), no URL expiry issues
- **Cons**: Requires separate request per image

### Finding the Main Image

```typescript
// Option 1: From product.images array
const mainImage = product.images.find(img => img.is_main);

// Option 2: Using the dedicated endpoint
const mainImage = await productService.getProductMainImage(productId);
```

### Image URL Refresh Strategy

Since presigned URLs expire after 1 hour, consider these strategies:

1. **Short-lived pages**: For product detail pages where users spend < 1 hour, use presigned URLs directly
2. **Long-lived pages**: For catalog pages displayed for extended periods:
   - Use the direct image endpoint: `/api/v1/images/{objectName}`
   - Or implement periodic refresh by re-fetching products
3. **Image galleries**: Use presigned URLs and refresh when user navigates

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
    return mainImage?.image_url || '/assets/placeholder.jpg';
  }
}
```

### Example: Product Gallery Component

```typescript
@Component({
  selector: 'app-product-gallery',
  template: `
    <div class="gallery">
      <img [src]="selectedImage?.image_url" [alt]="product.name">
      <div class="thumbnails">
        <img *ngFor="let img of product.images"
             [src]="img.image_url"
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
}
```

---

## Frontend Development Notes

1. **Image Loading**: Product images are now served via Minio with presigned URLs (1-hour expiry). The `images` array contains all product images with `image_url` as presigned URLs. Use `is_main` flag to identify the primary image
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

  getDirectImage(productId: number, imageName: string): Observable<Blob> {
    return this.http.get(`${this.apiUrl}/products/${productId}/images/${imageName}`, {
      responseType: 'blob'
    });
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
- All 9 endpoints with various parameter combinations
- Image URL expiry handling (test that presigned URLs expire after 1 hour)
- Image loading performance with presigned URLs vs direct endpoint
- Pagination edge cases (empty results, single page, multiple pages)
- Error conditions (invalid product ID, empty search query, missing images)
- Performance with large result sets
- URL encoding for category/brand names with special characters
- Main image detection using `is_main` flag
- Products with no images (empty `images` array)
