# Frontend Update: Add Image URL to Cart Item API

## Overview

The cart API now accepts an `image_url` field when adding items. This allows the product image to be stored with the cart item and displayed without making additional API calls.

## Change Required

### Adding Items to Cart

When calling `POST /api/v1/carts/{cartId}/items`, include the product's image URL in the request body.

**Request Body:**

```json
{
  "product_id": "12345",
  "quantity": 2,
  "image_url": "/api/v1/products/12345/images/image_0.jpg"
}
```

**Where to get image_url:**
- From the product list response (`GET /api/v1/products`) - the `main_image_url` field
- From the product search response (`GET /api/v1/products/search?q=...`) - the `main_image_url` field

## Benefits

1. **Single Request**: Cart page loads without additional image requests
2. **Consistency**: Shows the exact image the user saw when adding to cart
3. **Performance**: Reduces API calls from N+1 to 1

## Example Usage

```javascript
// When adding item to cart
const product = products.find(p => p.id === productId);

await fetch(`/api/v1/carts/${cartId}/items`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    product_id: product.id,
    quantity: quantity,
    image_url: product.main_image_url  // Include this!
  })
});
```

## Migration Steps

1. **Update Add to Cart**: Include `main_image_url` from product when adding to cart
2. **Cart Display**: Use `image_url` from cart item response directly (no transformation needed)
3. **Order Display**: Use `image_url` from order item response directly

## Response Format

### GET /api/v1/carts/{cartId}

Cart items now include `image_url` directly:

```json
{
  "cart_id": "...",
  "items": [
    {
      "product_id": "12345",
      "product_name": "Product Name",
      "quantity": 2,
      "image_url": "/api/v1/products/12345/images/image_0.jpg"
    }
  ]
}
```

### GET /api/v1/orders/customer/{customerId}

Order items include `image_url` directly:

```json
{
  "orders": [
    {
      "order_id": "...",
      "items": [
        {
          "product_id": "12345",
          "product_name": "Product Name",
          "quantity": 2,
          "image_url": "/api/v1/products/12345/images/image_0.jpg"
        }
      ]
    }
  ]
}
```

## Notes

- `image_url` is optional - existing items will have empty string
- The URL is stored as-is; no transformation needed in frontend
- This follows the snapshot pattern - the image shown at time of add is preserved