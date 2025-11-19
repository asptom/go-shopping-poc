# Updated Postman JSON Request Bodies

After implementing the UUID generation fix, these JSON request bodies will work correctly and return proper UUIDs in responses.

## Customer Management Endpoints

### 1. Create Customer
**Method:** `POST`  
**URL:** `http://localhost:8080/customers`

```json
{
  "user_name": "johndoe",
  "email": "john.doe@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "phone": "555-123-4567",
  "addresses": [
    {
      "address_type": "shipping",
      "first_name": "John",
      "last_name": "Doe",
      "address_1": "123 Main St",
      "address_2": "Apt 4B",
      "city": "New York",
      "state": "NY",
      "zip": "10001"
    },
    {
      "address_type": "billing",
      "first_name": "John",
      "last_name": "Doe",
      "address_1": "456 Oak Ave",
      "address_2": "",
      "city": "New York",
      "state": "NY",
      "zip": "10002"
    }
  ],
  "credit_cards": [
    {
      "card_type": "visa",
      "card_number": "4111111111111111",
      "card_holder_name": "John Doe",
      "card_expires": "12/25",
      "card_cvv": "123"
    }
  ]
}
```

**Expected Response (with real UUIDs):**
```json
{
    "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
    "user_name": "johndoe",
    "email": "john.doe@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "phone": "555-123-4567",
    "customer_since": "2025-11-18T20:52:48.723473225Z",
    "customer_status": "active",
    "status_date_time": "2025-11-18T20:52:48.723473308Z",
    "addresses": [
        {
            "address_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
            "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
            "address_type": "shipping",
            "first_name": "John",
            "last_name": "Doe",
            "address_1": "123 Main St",
            "address_2": "Apt 4B",
            "city": "New York",
            "state": "NY",
            "zip": "10001"
        },
        {
            "address_id": "b2c3d4e5-f6g7-8901-bcde-f23456789012",
            "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
            "address_type": "billing",
            "first_name": "John",
            "last_name": "Doe",
            "address_1": "456 Oak Ave",
            "address_2": "",
            "city": "New York",
            "state": "NY",
            "zip": "10002"
        }
    ],
    "credit_cards": [
        {
            "card_id": "c3d4e5f6-g7h8-9012-cdef-345678901234",
            "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
            "card_type": "visa",
            "card_number": "4111111111111111",
            "card_holder_name": "John Doe",
            "card_expires": "12/25",
            "card_cvv": "123"
        }
    ]
}
```

### 2. Update Customer

#### 2a. Update Basic Info Only
**Method:** `PUT`  
**URL:** `http://localhost:8080/customers`

```json
{
  "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
  "user_name": "johndoe_updated",
  "email": "john.doe.updated@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "phone": "555-987-6543",
  "customer_status": "active",
  "status_date_time": "2023-11-18T14:20:00Z"
}
```
**Result**: Updates basic info, preserves existing addresses and credit cards

#### 2b. Update Defaults Only
**Method:** `PUT`  
**URL:** `http://localhost:8080/customers`

```json
{
  "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
  "default_shipping_address_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "default_billing_address_id": "b2c3d4e5-f6g7-8901-bcde-f23456789012",
  "default_credit_card_id": "c3d4e5f6-g7h8-9012-cdef-345678901234"
}
```
**Result**: Updates default references, preserves all other data

#### 2c. Full Customer Update (Replace All)
**Method:** `PUT`  
**URL:** `http://localhost:8080/customers`

```json
{
  "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
  "user_name": "johndoe_updated",
  "email": "john.doe.updated@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "phone": "555-987-6543",
  "default_shipping_address_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "default_billing_address_id": "b2c3d4e5-f6g7-8901-bcde-f23456789012",
  "default_credit_card_id": "c3d4e5f6-g7h8-9012-cdef-345678901234",
  "customer_since": "2023-01-15T10:30:00Z",
  "customer_status": "active",
  "status_date_time": "2023-11-18T14:20:00Z",
  "addresses": [
    {
      "address_type": "shipping",
      "first_name": "John",
      "last_name": "Doe",
      "address_1": "123 Updated Street",
      "address_2": "Apt 5C",
      "city": "New York",
      "state": "NY",
      "zip": "10003"
    }
  ],
  "credit_cards": [
    {
      "card_type": "amex",
      "card_number": "378282246310005",
      "card_holder_name": "John Doe",
      "card_expires": "11/27",
      "card_cvv": "789"
    }
  ]
}
```
**Result**: Replaces all addresses and credit cards with new data

#### 2d. Clear All Addresses
**Method:** `PUT`  
**URL:** `http://localhost:8080/customers`

```json
{
  "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
  "addresses": []
}
```
**Result**: Removes all addresses + address defaults, preserves credit cards and other data

#### 2e. Update Individual Default Fields

**Update Only Shipping Default**
**Method:** `PUT`  
**URL:** `http://localhost:8080/customers`

```json
{
  "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
  "default_shipping_address_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```
**Result**: Updates only shipping default, preserves all other data

**Update Only Billing Default**
**Method:** `PUT`  
**URL:** `http://localhost:8080/customers`

```json
{
  "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
  "default_billing_address_id": "b2c3d4e5-f6g7-8901-bcde-f23456789012"
}
```
**Result**: Updates only billing default, preserves all other data

**Update Only Credit Card Default**
**Method:** `PUT`  
**URL:** `http://localhost:8080/customers`

```json
{
  "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
  "default_credit_card_id": "c3d4e5f6-g7h8-9012-cdef-345678901234"
}
```
**Result**: Updates only credit card default, preserves all other data

## Address Management Endpoints

### 3. Add Address
**Method:** `POST`  
**URL:** `http://localhost:8080/customers/583ca09f-1e13-4272-a0a6-b707e489e46d/addresses`

```json
{
  "address_type": "shipping",
  "first_name": "Jane",
  "last_name": "Doe",
  "address_1": "789 Pine Street",
  "address_2": "Suite 200",
  "city": "Los Angeles",
  "state": "CA",
  "zip": "90210"
}
```

**Expected Response:**
```json
{
    "address_id": "d4e5f6g7-h8i9-0123-defg-456789012345",
    "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
    "address_type": "shipping",
    "first_name": "Jane",
    "last_name": "Doe",
    "address_1": "789 Pine Street",
    "address_2": "Suite 200",
    "city": "Los Angeles",
    "state": "CA",
    "zip": "90210"
}
```

### 4. Update Address
**Method:** `PUT`  
**URL:** `http://localhost:8080/customers/addresses/a1b2c3d4-e5f6-7890-abcd-ef1234567890`

```json
{
  "address_type": "billing",
  "first_name": "John",
  "last_name": "Doe",
  "address_1": "123 Updated Street",
  "address_2": "Apt 5C",
  "city": "New York",
  "state": "NY",
  "zip": "10003"
}
```

## Credit Card Management Endpoints

### 5. Add Credit Card
**Method:** `POST`  
**URL:** `http://localhost:8080/customers/583ca09f-1e13-4272-a0a6-b707e489e46d/credit-cards`

```json
{
  "card_type": "mastercard",
  "card_number": "5555555555554444",
  "card_holder_name": "John Doe",
  "card_expires": "08/26",
  "card_cvv": "456"
}
```

**Expected Response:**
```json
{
    "card_id": "e5f6g7h8-i9j0-1234-efgh-567890123456",
    "customer_id": "583ca09f-1e13-4272-a0a6-b707e489e46d",
    "card_type": "mastercard",
    "card_number": "5555555555554444",
    "card_holder_name": "John Doe",
    "card_expires": "08/26",
    "card_cvv": "456"
}
```

### 6. Update Credit Card
**Method:** `PUT`  
**URL:** `http://localhost:8080/customers/credit-cards/c3d4e5f6-g7h8-9012-cdef-345678901234`

```json
{
  "card_type": "amex",
  "card_holder_name": "John Doe Jr.",
  "card_expires": "11/27",
  "card_cvv": "789"
}
```

## Default Management Endpoints (NEW)

### 7. Set Default Shipping Address
**Method:** `PUT`  
**URL:** `http://localhost:8080/customers/583ca09f-1e13-4272-a0a6-b707e489e46d/default-shipping-address/d4e5f6g7-h8i9-0123-defg-456789012345`
**Request Body:** None (empty)

### 8. Set Default Billing Address
**Method:** `PUT`  
**URL:** `http://localhost:8080/customers/583ca09f-1e13-4272-a0a6-b707e489e46d/default-billing-address/d4e5f6g7-h8i9-0123-defg-456789012345`
**Request Body:** None (empty)

### 9. Set Default Credit Card
**Method:** `PUT`  
**URL:** `http://localhost:8080/customers/583ca09f-1e13-4272-a0a6-b707e489e46d/default-credit-card/e5f6g7h8-i9j0-1234-efgh-567890123456`
**Request Body:** None (empty)

### 10. Clear Default Shipping Address
**Method:** `DELETE`  
**URL:** `http://localhost:8080/customers/583ca09f-1e13-4272-a0a6-b707e489e46d/default-shipping-address`
**Request Body:** None (empty)

### 11. Clear Default Billing Address
**Method:** `DELETE`  
**URL:** `http://localhost:8080/customers/583ca09f-1e13-4272-a0a6-b707e489e46d/default-billing-address`
**Request Body:** None (empty)

### 12. Clear Default Credit Card
**Method:** `DELETE`  
**URL:** `http://localhost:8080/customers/583ca09f-1e13-4272-a0a6-b707e489e46d/default-credit-card`
**Request Body:** None (empty)

## Query Endpoint

### 13. Get Customer by Email
**Method:** `GET`  
**URL:** `http://localhost:8080/customers/john.doe%40example.com` (URL-encoded email)
**Request Body:** None (empty)

## Important Notes for Postman Testing

1. **UUIDs are now properly generated** - All responses will contain real UUIDs
2. **Use returned UUIDs** for subsequent API calls (update, delete, set default)
3. **Base URL**: Use `http://localhost:8080` for local testing
4. **Headers**: Set `Content-Type: application/json` for POST/PUT requests
5. **URL Encoding**: Email addresses in GET requests must be URL-encoded (`@` becomes `%40`)
6. **Default Endpoints**: New default-setting endpoints don't require request bodies

## Complete Workflow Example

1. **Create Customer** → Get customer_id, address_id, card_id from response
2. **Set Default Shipping** → Use returned address_id in URL
3. **Set Default Credit Card** → Use returned card_id in URL
4. **Add New Address** → Get new address_id from response  
5. **Update Default** → Use new address_id to replace default

All UUIDs in responses are now real, generated values that can be used for subsequent API operations.