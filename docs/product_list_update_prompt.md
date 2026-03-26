# API Updates - Frontend Integration Guide

## Overview

This document details the API changes made to reduce the number of HTTP requests needed for common frontend operations. All changes include image URLs or status information directly in the response, eliminating the need for additional API calls.

---

## Change 1: Product List - Add main_image_url

### Endpoints Affected

- `GET /api/v1/products`
- `GET /api/v1/products/search?q={query}`

### Description

Each product in the list now includes a `main_image_url` field pointing to the primary product image.

### Response Format

```json
{
  "products": [
    {
      "id": 123,
      "name": "Product Name",
      "description": "...",
      "initial_price": 29.99,
      "final_price": 19.99,
      "currency": "USD",
      "in_stock": true,
      "main_image_url": "/api/v1/products/123/images/image_0.jpg",
      ...
    }
  ],
  "limit": 50,
  "offset": 0,
  "count": 20
}
```

### About main_image_url

- **Type**: String (relative URL path)
- **Format**: `/api/v1/products/{product_id}/images/{filename}`
- **Empty Value**: Products without images will have `main_image_url` as empty string (`""`)

### Before vs After

| Before (21 requests) | After (1 request) |
|---------------------|-------------------|
| GET /products | GET /products |
| GET /products/1/images | (images included) |
| GET /products/2/images | |
| ... (x20) | |

---

## Change 2: Cart Items - Add product_image_url

### Endpoint Affected

- `GET /api/v1/carts/{cartId}`

### Description

Each cart item now includes a `product_image_url` field pointing to the product image.

### Response Format

```json
{
  "cart_id": "...",
  "current_status": "active",
  "currency": "USD",
  "total_price": 59.99,
  "items": [
    {
      "id": 1,
      "product_id": "123",
      "product_name": "Product Name",
      "unit_price": 29.99,
      "quantity": 2,
      "total_price": 59.98,
      "product_image_url": "/api/v1/products/123/images/image_0.jpg"
    }
  ]
}
```

### About product_image_url

- **Type**: String (relative URL path)
- **Format**: `/api/v1/products/{product_id}/images/{filename}`
- **Empty Value**: Items without images will have `product_image_url` as empty string (`""`)

### Before vs After

| Before (N+1 requests) | After (1 request) |
|-----------------------|-------------------|
| GET /carts/{id} | GET /carts/{id} |
| GET /products/1/images | (images included) |
| GET /products/2/images | |
| ... | |

---

## Change 3: Order History - Add status_history

### Endpoint Affected

- `GET /api/v1/orders/customer/{customerId}`

### Description

Each order now includes a `status_history` array showing the timeline of status changes.

### Response Format

```json
{
  "orders": [
    {
      "order_id": "...",
      "order_number": "ORD-001",
      "current_status": "shipped",
      "created_at": "2025-01-01T10:00:00Z",
      "total_price": 59.99,
      "status_history": [
        { "status": "created", "timestamp": "2025-01-01T10:00:00Z" },
        { "status": "confirmed", "timestamp": "2025-01-01T10:05:00Z" },
        { "status": "processing", "timestamp": "2025-01-01T14:00:00Z" },
        { "status": "shipped", "timestamp": "2025-01-02T08:30:00Z" }
      ]
    }
  ]
}
```

### About status_history

- **Type**: Array of objects
- **Fields**:
  - `status`: string - The order status at this point in time
  - `timestamp`: string (ISO 8601) - When the status changed
- **Order**: Chronological order (oldest first)
- **Empty**: Orders may have empty status_history if no transitions have been recorded

### Status Values

Valid order statuses: `created`, `confirmed`, `processing`, `shipped`, `delivered`, `cancelled`, `refunded`

### Timeline Visualization

The `status_history` array enables a timeline component:

```
created (Jan 1, 10:00)
    ↓
confirmed (Jan 1, 10:05)
    ↓
processing (Jan 1, 14:00)
    ↓
shipped (Jan 2, 08:30) ← current status
```

---

## Summary of URL Format

All image URLs use the same pattern:

```
/api/v1/products/{product_id}/images/{filename}
```

Examples:
- `/api/v1/products/123/images/image_0.jpg`
- `/api/v1/products/456/images/photo_main.png`

The URLs point to the existing direct image endpoint that streams images from MinIO storage with caching (Cache-Control: public, max-age=3600).

---

## Frontend Migration Checklist

- [ ] **Product list**: Remove loop making `GET /products/{id}/images` calls - use `main_image_url` directly
- [ ] **Product search**: Remove loop making `GET /products/{id}/images` calls - use `main_image_url` directly
- [ ] **Cart view**: Remove loop making `GET /products/{id}/images` calls - use `product_image_url` directly
- [ ] **Order history**: Implement status timeline component using `status_history` array
- [ ] Handle empty image URLs gracefully (show placeholder or hide image element)