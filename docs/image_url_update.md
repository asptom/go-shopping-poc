# Image URL Propagation Plan

## Problem

The current implementation attempts to JOIN across service boundaries (cart service querying `products.product_images`), which violates the microservice/SAGA pattern. Services should not directly access each other's databases.

## Solution

Instead of fetching images from the products database, store the image URL as a snapshot in the cart when an item is added, then propagate it to the order when checkout occurs.

---

## Changes Required

### 1. Database Schema Changes

**Add `image_url` column to `carts.CartItem` table:**
```sql
ALTER TABLE carts.CartItem ADD COLUMN image_url VARCHAR(500);
```

**Add `image_url` column to `orders.OrderItem` table:**
```sql
ALTER TABLE orders.OrderItem ADD COLUMN image_url VARCHAR(500);
```

### 2. Cart Service Changes

**Entity (`internal/service/cart/entity.go`):**
- Add `ImageURL` field to `CartItem` struct

**Repository - Add Item (`internal/service/cart/repository_items.go`):**
- Update `AddItem` and `AddItemTx` to accept and store `image_url`

**Handler (`internal/service/cart/handlers.go`):**
- Update `AddItemRequest` to accept `image_url` field
- Pass `image_url` when creating cart item

**GET /carts/{id} Handler:**
- Remove the code that constructs URLs from `products.product_images`
- The stored `ImageURL` is already in the entity and will be returned

### 3. Order Service Changes

**Entity (`internal/service/order/entity.go`):**
- Add `ImageURL` field to `OrderItem` struct

**Repository (`internal/service/order/repository_crud.go`):**
- Update `CreateOrderFromSnapshot` or similar to copy `image_url` from cart items to order items

**Handler/Service:**
- Ensure image_url flows through when order is created from cart

**GET /orders/customer/{id} Handler:**
- Already returns items - no changes needed as ImageURL will be in entity

### 4. Frontend Changes

**Add to Cart:**
- When calling `POST /carts/{id}/items`, include the product's `main_image_url` in the request body
- Frontend already has this from the product list response

---

## Implementation Order

1. **Database migrations** - Add columns to CartItem and OrderItem tables
2. **Cart entity** - Add ImageURL field
3. **Cart repository** - Update AddItem to store image_url
4. **Cart handler** - Update AddItemRequest and handler
5. **Order entity** - Add ImageURL field
6. **Order repository** - Propagate image_url from cart snapshot to order items
7. **Test** - Verify GET endpoints return image_url

---

## Data Flow

```
Product List Response:
  main_image_url: "/api/v1/products/123/images/image_0.jpg"
         │
         ▼
POST /carts/{id}/items (with image_url in body)
         │
         ▼
carts.CartItem (image_url stored)
         │
         ▼
Checkout -> Create Order
         │
         ▼
orders.OrderItem (image_url copied from cart)
         │
         ▼
GET /carts/{id}    → returns image_url
GET /orders/customer/{id} → returns image_url
```

---

## Questions

1. Should the image_url column allow NULL? Yes - existing cart items won't have it
2. What happens if the frontend doesn't provide image_url? Store as empty string
3. Should we validate the URL format? Not necessary - store whatever frontend sends
4. Should this be part of the cart migration that creates items from the cart snapshot event? Need to check how orders are created from carts