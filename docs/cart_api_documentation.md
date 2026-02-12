# Cart Service API Documentation

## Overview

The Cart Service provides a RESTful API for managing shopping carts in the go-shopping-poc application. It supports both guest and authenticated users, integrates with the Product Service for real-time product validation, and uses an event-driven architecture for checkout flow.

**Base URL:** `/api/v1`

**Key Features:**
- Guest and authenticated user support
- Real-time product validation via Product Service
- Event-driven checkout with outbox pattern
- Denormalized product data in cart items
- 3% tax calculation, $0 shipping (placeholders)
- Status workflow: active → checked_out → completed/cancelled

---

## API Endpoints

### Cart Management

#### Create a Cart
```
POST /api/v1/carts
```

Creates a new shopping cart. Can be created for a guest (no customer_id) or an authenticated user.

**Request Body:**
```json
{
  "customer_id": "optional-uuid-string-for-authenticated-users"
}
```

**Response (201 Created):**
```json
{
  "cart_id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": null,
  "current_status": "active",
  "currency": "USD",
  "net_price": 0,
  "tax": 0,
  "shipping": 0,
  "total_price": 0,
  "created_at": "2026-01-15T10:30:00Z",
  "updated_at": "2026-01-15T10:30:00Z",
  "version": 1,
  "contact": null,
  "addresses": [],
  "credit_card": null,
  "items": [],
  "status_history": []
}
```

**Important Notes:**
- Only one active cart per customer is allowed (enforced by unique constraint)
- Guest carts (no customer_id) have no limitations
- Store the returned `cart_id` - it's needed for all subsequent operations
- Cart ID is a UUID v4 string

---

#### Get Cart
```
GET /api/v1/carts/{cart_id}
```

Retrieves a complete cart with all related data (items, contact, addresses, payment, status history).

**Path Parameters:**
- `cart_id` (string, required): The UUID of the cart

**Response (200 OK):**
```json
{
  "cart_id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": null,
  "current_status": "active",
  "currency": "USD",
  "net_price": 299.97,
  "tax": 9.00,
  "shipping": 0,
  "total_price": 308.97,
  "created_at": "2026-01-15T10:30:00Z",
  "updated_at": "2026-01-15T10:45:00Z",
  "version": 1,
  "contact": {
    "id": 1,
    "cart_id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "john.doe@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "phone": "+1-555-123-4567"
  },
  "addresses": [
    {
      "id": 1,
      "cart_id": "550e8400-e29b-41d4-a716-446655440000",
      "address_type": "shipping",
      "first_name": "John",
      "last_name": "Doe",
      "address_1": "123 Main Street",
      "address_2": "Apt 4B",
      "city": "New York",
      "state": "NY",
      "zip": "10001"
    }
  ],
  "credit_card": {
    "id": 1,
    "cart_id": "550e8400-e29b-41d4-a716-446655440000",
    "card_type": "visa",
    "card_number": "4111111111111111",
    "card_holder_name": "John Doe",
    "card_expires": "12/27",
    "card_cvv": "123"
  },
  "items": [
    {
      "id": 1,
      "cart_id": "550e8400-e29b-41d4-a716-446655440000",
      "line_number": "001",
      "product_id": "prod-123",
      "product_name": "Premium Widget",
      "unit_price": 99.99,
      "quantity": 3,
      "total_price": 299.97
    }
  ],
  "status_history": [
    {
      "id": 1,
      "cart_id": "550e8400-e29b-41d4-a716-446655440000",
      "cart_status": "active",
      "status_date_time": "2026-01-15T10:30:00Z"
    }
  ]
}
```

**Error Responses:**
- `404 Not Found`: Cart does not exist
- `400 Bad Request`: Invalid cart ID format

---

#### Delete Cart
```
DELETE /api/v1/carts/{cart_id}
```

Permanently deletes a cart and all its associated data (items, addresses, contact, payment).

**Path Parameters:**
- `cart_id` (string, required): The UUID of the cart

**Response:** `204 No Content`

**Error Responses:**
- `404 Not Found`: Cart does not exist

**Important:** This publishes a `cart.deleted` event to the event bus.

---

### Cart Items

#### Add Item to Cart
```
POST /api/v1/carts/{cart_id}/items
```

Adds a product to the cart. The service validates the product with the Product Service and denormalizes product data (name, price) into the cart item.

**Path Parameters:**
- `cart_id` (string, required): The UUID of the cart

**Request Body:**
```json
{
  "product_id": "prod-123",
  "quantity": 2
}
```

**Field Requirements:**
- `product_id` (string, required): The product identifier from Product Service
- `quantity` (integer, required): Must be positive (> 0)

**Response (201 Created):**
```json
{
  "id": 1,
  "cart_id": "550e8400-e29b-41d4-a716-446655440000",
  "line_number": "001",
  "product_id": "prod-123",
  "product_name": "Premium Widget",
  "unit_price": 99.99,
  "quantity": 2,
  "total_price": 199.98
}
```

**Important Notes:**
- Line numbers are auto-generated as 3-digit strings (001, 002, 003, etc.) using a database sequence
- Product name and price are fetched from Product Service and cached in the cart item
- Cart totals are automatically recalculated after adding an item
- Can only add items to carts with `current_status: "active"`
- Product must be in stock (checked via Product Service)

**Error Responses:**
- `400 Bad Request`: Invalid JSON, missing product_id, or quantity <= 0
- `404 Not Found`: Cart does not exist
- `400 Bad Request`: Product is out of stock
- `400 Bad Request`: Cannot add items to non-active cart

---

#### Update Item Quantity
```
PUT /api/v1/carts/{cart_id}/items/{line_number}
```

Updates the quantity of an existing cart item. Recalculates cart totals automatically.

**Path Parameters:**
- `cart_id` (string, required): The UUID of the cart
- `line_number` (string, required): The 3-digit line number (e.g., "001")

**Request Body:**
```json
{
  "quantity": 5
}
```

**Field Requirements:**
- `quantity` (integer, required): Must be positive (> 0)

**Response:** `204 No Content`

**Important Notes:**
- Use the `line_number` from the cart item (not the database `id`)
- Cart totals are recalculated automatically
- Can only update items in active carts

**Error Responses:**
- `400 Bad Request`: Invalid JSON, missing cart_id/line_number, or quantity <= 0
- `404 Not Found`: Cart or item does not exist
- `400 Bad Request`: Cannot modify items in non-active cart

---

#### Remove Item from Cart
```
DELETE /api/v1/carts/{cart_id}/items/{line_number}
```

Removes an item from the cart. Recalculates cart totals automatically.

**Path Parameters:**
- `cart_id` (string, required): The UUID of the cart
- `line_number` (string, required): The 3-digit line number (e.g., "001")

**Response:** `204 No Content`

**Important Notes:**
- Use the `line_number` from the cart item
- Cart totals are recalculated automatically
- Can only remove items from active carts

**Error Responses:**
- `400 Bad Request`: Missing cart_id or line_number
- `404 Not Found`: Cart or item does not exist
- `400 Bad Request`: Cannot remove items from non-active cart

---

### Contact Information

#### Set Contact Information
```
PUT /api/v1/carts/{cart_id}/contact
```

Sets or updates the contact information for the cart. Replaces any existing contact.

**Path Parameters:**
- `cart_id` (string, required): The UUID of the cart

**Request Body:**
```json
{
  "email": "john.doe@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "phone": "+1-555-123-4567"
}
```

**Field Requirements:**
- `email` (string, required): Must contain "@" character
- `first_name` (string, required): Cannot be empty/whitespace only
- `last_name` (string, required): Cannot be empty/whitespace only
- `phone` (string, required): Cannot be empty/whitespace only

**Response:** `204 No Content`

**Important Notes:**
- This replaces any existing contact information
- Required before checkout
- Can only set contact for active carts

**Error Responses:**
- `400 Bad Request`: Invalid JSON or validation errors
- `404 Not Found`: Cart does not exist
- `400 Bad Request`: Cannot modify contact for non-active cart

---

### Addresses

#### Add Address
```
POST /api/v1/carts/{cart_id}/addresses
```

Adds a shipping or billing address to the cart.

**Path Parameters:**
- `cart_id` (string, required): The UUID of the cart

**Request Body:**
```json
{
  "address_type": "shipping",
  "first_name": "John",
  "last_name": "Doe",
  "address_1": "123 Main Street",
  "address_2": "Apt 4B",
  "city": "New York",
  "state": "NY",
  "zip": "10001"
}
```

**Field Requirements:**
- `address_type` (string, required): Must be either "shipping" or "billing"
- `first_name` (string, required): Recipient's first name
- `last_name` (string, required): Recipient's last name
- `address_1` (string, required): Street address line 1
- `address_2` (string, optional): Street address line 2 (apartment, suite, etc.)
- `city` (string, required): City name
- `state` (string, required): State or province
- `zip` (string, required): Postal/ZIP code

**Response (201 Created):**
```json
{
  "id": 1,
  "cart_id": "550e8400-e29b-41d4-a716-446655440000",
  "address_type": "shipping",
  "first_name": "John",
  "last_name": "Doe",
  "address_1": "123 Main Street",
  "address_2": "Apt 4B",
  "city": "New York",
  "state": "NY",
  "zip": "10001"
}
```

**Important Notes:**
- Can add multiple addresses (e.g., separate shipping and billing)
- Can only add addresses to active carts
- Both shipping and billing addresses can be added to the same cart

**Error Responses:**
- `400 Bad Request`: Invalid JSON or validation errors
- `404 Not Found`: Cart does not exist
- `400 Bad Request`: Cannot add address to non-active cart

---

### Payment Information

#### Set Payment Method
```
PUT /api/v1/carts/{cart_id}/payment
```

Sets or updates the credit card payment information. Replaces any existing payment method.

**Path Parameters:**
- `cart_id` (string, required): The UUID of the cart

**Request Body:**
```json
{
  "card_type": "visa",
  "card_number": "4111111111111111",
  "card_holder_name": "John Doe",
  "card_expires": "12/27",
  "card_cvv": "123"
}
```

**Field Requirements:**
- `card_type` (string, optional): Card type (visa, mastercard, amex, etc.)
- `card_number` (string, required): Credit card number
- `card_holder_name` (string, required): Name on card
- `card_expires` (string, required): Expiration date (format: MM/YY)
- `card_cvv` (string, required): Security code

**Response:** `204 No Content`

**Important Notes:**
- This replaces any existing payment information
- Required before checkout
- Can only set payment for active carts
- Card data is stored in the database (in a real system, this would use a tokenization service)

**Error Responses:**
- `400 Bad Request`: Invalid JSON or validation errors
- `404 Not Found`: Cart does not exist
- `400 Bad Request`: Cannot modify payment for non-active cart

---

### Checkout

#### Checkout Cart
```
POST /api/v1/carts/{cart_id}/checkout
```

Initiates the checkout process. Validates the cart is ready, updates totals, changes status to "checked_out", and publishes a checkout event.

**Path Parameters:**
- `cart_id` (string, required): The UUID of the cart

**Response (200 OK):**
```json
{
  "cart_id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": null,
  "current_status": "checked_out",
  "currency": "USD",
  "net_price": 299.97,
  "tax": 9.00,
  "shipping": 0,
  "total_price": 308.97,
  "created_at": "2026-01-15T10:30:00Z",
  "updated_at": "2026-01-15T11:00:00Z",
  "version": 2,
  "contact": { ... },
  "addresses": [ ... ],
  "credit_card": { ... },
  "items": [ ... ],
  "status_history": [
    { "cart_status": "active", ... },
    { "cart_status": "checked_out", "status_date_time": "2026-01-15T11:00:00Z" }
  ]
}
```

**Checkout Requirements:**
1. Cart must have `current_status: "active"`
2. Cart must have at least one item
3. Contact information must be set
4. Payment method must be set

**Important Notes:**
- Cart totals are recalculated before checkout
- Status changes from "active" to "checked_out"
- A `cart.checked_out` event is published to the event bus
- After checkout, the cart becomes immutable (no modifications allowed)

**Error Responses:**
- `400 Bad Request`: Missing cart ID
- `404 Not Found`: Cart does not exist
- `400 Bad Request`: Cart must be active to checkout
- `400 Bad Request`: Cart is empty (no items)
- `400 Bad Request`: Contact information required
- `400 Bad Request`: Payment method required
- `500 Internal Server Error`: Checkout processing failed

---

## Entity Reference

### Cart Entity

```typescript
interface Cart {
  cart_id: string;           // UUID v4
  customer_id?: string;      // Optional UUID (null for guest carts)
  current_status: "active" | "checked_out" | "completed" | "cancelled";
  currency: string;          // Default: "USD"
  net_price: number;         // Sum of all item totals
  tax: number;               // Calculated as net_price * 0.03 (3%)
  shipping: number;          // Currently always 0
  total_price: number;       // net_price + tax + shipping
  created_at: string;        // ISO 8601 timestamp
  updated_at: string;        // ISO 8601 timestamp
  version: number;           // Optimistic locking version
  
  // Relationships
  contact?: Contact;
  addresses: Address[];
  credit_card?: CreditCard;
  items: CartItem[];
  status_history: CartStatus[];
}
```

### CartItem Entity

```typescript
interface CartItem {
  id: number;               // Database ID (internal use)
  cart_id: string;          // Parent cart UUID
  line_number: string;      // 3-digit line number (001, 002, etc.)
  product_id: string;       // Product identifier from Product Service
  product_name: string;     // Denormalized product name
  unit_price: number;       // Denormalized unit price
  quantity: number;         // Item quantity
  total_price: number;      // unit_price * quantity
}
```

### Contact Entity

```typescript
interface Contact {
  id: number;               // Database ID (internal use)
  cart_id: string;          // Parent cart UUID
  email: string;            // Must contain "@"
  first_name: string;       // Required, non-empty
  last_name: string;        // Required, non-empty
  phone: string;            // Required, non-empty
}
```

### Address Entity

```typescript
interface Address {
  id: number;               // Database ID (internal use)
  cart_id: string;          // Parent cart UUID
  address_type: "shipping" | "billing";
  first_name: string;       // Recipient first name
  last_name: string;        // Recipient last name
  address_1: string;        // Street address line 1
  address_2?: string;       // Street address line 2 (optional)
  city: string;             // City
  state: string;            // State/Province
  zip: string;              // Postal/ZIP code
}
```

### CreditCard Entity

```typescript
interface CreditCard {
  id: number;               // Database ID (internal use)
  cart_id: string;          // Parent cart UUID
  card_type?: string;       // Card type (visa, mastercard, etc.)
  card_number: string;      // Full card number
  card_holder_name: string; // Name on card
  card_expires: string;     // Expiration date (MM/YY)
  card_cvv: string;         // Security code
}
```

### CartStatus Entity

```typescript
interface CartStatus {
  id: number;               // Database ID (internal use)
  cart_id: string;          // Parent cart UUID
  cart_status: string;      // Status value
  status_date_time: string; // ISO 8601 timestamp
}
```

---

## Cart Status Workflow

```
active → checked_out → completed
   ↓           ↓
cancelled  cancelled
```

**Status Transitions:**
- `active` → `checked_out`: Via checkout API call
- `active` → `cancelled`: Via cancellation (future feature)
- `checked_out` → `completed`: Via order completion event (handled by event consumer)
- `checked_out` → `cancelled`: Via cancellation (future feature)
- `completed` → (no transitions): Terminal state
- `cancelled` → (no transitions): Terminal state

**Important:** Once a cart leaves the "active" status, it cannot be modified. All modification operations (add item, update quantity, remove item, set contact, add address, set payment) will return errors.

---

## Typical Workflow Examples

### Guest Checkout Flow

```javascript
// 1. Create a guest cart
const createResponse = await fetch('/api/v1/carts', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({})  // No customer_id for guest
});
const cart = await createResponse.json();
const cartId = cart.cart_id;

// 2. Add items to cart
await fetch(`/api/v1/carts/${cartId}/items`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    product_id: 'prod-123',
    quantity: 2
  })
});

// 3. Set contact information
await fetch(`/api/v1/carts/${cartId}/contact`, {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    email: 'customer@example.com',
    first_name: 'Jane',
    last_name: 'Smith',
    phone: '+1-555-987-6543'
  })
});

// 4. Add shipping address
await fetch(`/api/v1/carts/${cartId}/addresses`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    address_type: 'shipping',
    first_name: 'Jane',
    last_name: 'Smith',
    address_1: '456 Oak Avenue',
    city: 'Los Angeles',
    state: 'CA',
    zip: '90210'
  })
});

// 5. Set payment method
await fetch(`/api/v1/carts/${cartId}/payment`, {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    card_type: 'visa',
    card_number: '4111111111111111',
    card_holder_name: 'Jane Smith',
    card_expires: '12/27',
    card_cvv: '123'
  })
});

// 6. Checkout
const checkoutResponse = await fetch(`/api/v1/carts/${cartId}/checkout`, {
  method: 'POST'
});
const checkedOutCart = await checkoutResponse.json();
```

### Modify Cart Items

```javascript
// Update quantity of line item "001"
await fetch(`/api/v1/carts/${cartId}/items/001`, {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ quantity: 5 })
});

// Remove line item "002"
await fetch(`/api/v1/carts/${cartId}/items/002`, {
  method: 'DELETE'
});
```

---

## Error Handling

### Standard Error Response Format

```json
{
  "type": "validation_error",
  "message": "Quantity must be positive",
  "code": 400
}
```

### Common Error Scenarios

| HTTP Status | Error Type | Scenario |
|-------------|------------|----------|
| 400 | invalid_request | Invalid JSON, missing required fields |
| 400 | validation | Field validation failed |
| 404 | not_found | Cart, item, or related entity does not exist |
| 422 | unprocessable | Cart not ready for checkout |
| 500 | internal | Server error |

---

## Summary for Frontend Development

### Key Points to Remember:

1. **Always store the cart_id** returned from cart creation
2. **Use line_number for item operations** - not the item's database id
3. **Follow the checkout sequence**: Create cart → Add items → Set contact → Add addresses → Set payment → Checkout
4. **Handle status changes** - Once checked out, the cart becomes read-only
5. **Poll for updates** - After checkout, the cart transitions from checked_out to completed
6. **Recalculate totals** - API automatically recalculates after item modifications
7. **Error handling** - Be prepared for validation errors around checkout requirements
8. **Guest vs Authenticated** - Guest carts don't require customer_id; authenticated limited to one per customer
9. **Product validation** - Adding items validates with Product Service; handle out-of-stock errors
10. **Tax display** - Always show the 3% tax calculation to users

### Recommended Frontend State Management:

```typescript
interface CartState {
  cartId: string | null;
  cart: Cart | null;
  isLoading: boolean;
  error: string | null;
}
```

This documentation provides everything needed to build a complete cart and checkout experience in the frontend application.
