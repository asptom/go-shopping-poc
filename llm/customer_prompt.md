# Customer Service API Specification

## Base URL
`https://pocstore.local` (Traefik handles TLS termination)

## Customer Management Endpoints

### Create Customer
- **POST** `/customers`
- **Request Body**: Customer object (without IDs - backend generates UUIDs)
- **Response**: 201 Created with full customer object including generated IDs
- **Note**: Do not send customer_id, address_id, or card_id - backend generates these

### Get Customer by Email  
- **GET** `/customers/{email}`
- **URL Parameter**: email (path-encoded)
- **Response**: 200 OK with customer object or 404 Not Found

### Update Customer
- **PUT** `/customers` 
- **Request Body**: Complete customer record (required for PUT)
- **Response**: 200 OK
- **Behavior**: Replaces entire customer record
  - **Validation**: Requires customer_id, username, and email
  - **Use Case**: When you need to replace the entire customer record
  - **All Fields**: Must include addresses, credit cards, and status history

### Patch Customer (NEW)
- **PATCH** `/customers/{id}` 
- **Request Body**: Partial update object with only fields to change
- **Response**: 200 OK with updated customer
- **Behavior**: Field-level partial updates
  - **Basic Info**: `"user_name"`, `"email"`, `"first_name"`, `"last_name"`, `"phone"`, `"customer_status"`
  - **Default Fields**: `"default_shipping_address_id"`, `"default_billing_address_id"`, `"default_credit_card_id"` (UUID strings or null)
  - **Addresses**: `"addresses"` - Array of address objects (replace only)
  - **Credit Cards**: `"credit_cards"` - Array of credit card objects (replace only)
  - **Use Case**: When you need to update specific fields without affecting others
  - **Preservation**: Unspecified fields are preserved automatically

### Address Management Endpoints
- **POST** `/customers/{id}/addresses` - Add new address
- **PUT** `/customers/addresses/{addressId}` - Update existing address  
- **DELETE** `/customers/addresses/{addressId}` - Delete address

### Credit Card Management Endpoints
- **POST** `/customers/{id}/credit-cards` - Add new credit card
- **PUT** `/customers/credit-cards/{cardId}` - Update existing credit card
- **DELETE** `/customers/credit-cards/{cardId}` - Delete credit card

### Default Management Endpoints (ENHANCED)
- **PUT** `/customers/{id}/default-shipping-address/{addressId}` - Set default shipping address
- **PUT** `/customers/{id}/default-billing-address/{addressId}` - Set default billing address  
- **PUT** `/customers/{id}/default-credit-card/{cardId}` - Set default credit card
- **DELETE** `/customers/{id}/default-shipping-address` - Clear default shipping address
- **DELETE** `/customers/{id}/default-billing-address` - Clear default billing address
- **DELETE** `/customers/{id}/default-credit-card` - Clear default credit card

### Usage Guidelines
- **Use PUT**: When creating or completely replacing a customer record
- **Use PATCH**: When updating specific fields (more efficient and RESTful)
- **Use Dedicated Endpoints**: For default address/credit card management (recommended)
- **UUID Handling**: Backend generates all UUIDs, frontend never specifies them

## Address Management Endpoints

### Add Address
- **POST** `/customers/{id}/addresses`
- **URL Parameter**: customer ID
- **Request Body**: Address object (without address_id - backend generates)
- **Response**: 201 Created, Location header with new address URL
- **Returns**: Address object with generated address_id

### Update Address
- **PUT** `/customers/addresses/{addressId}`
- **URL Parameter**: address ID  
- **Request Body**: Address object with updated fields
- **Response**: 204 No Content

### Delete Address
- **DELETE** `/customers/addresses/{addressId}`
- **URL Parameter**: address ID
- **Response**: 204 No Content
- **Note**: Automatically clears customer's default_*_address_id if this was the default

## Credit Card Management Endpoints

### Add Credit Card
- **POST** `/customers/{id}/credit-cards`
- **URL Parameter**: customer ID
- **Request Body**: CreditCard object (without card_id - backend generates)
- **Response**: 201 Created
- **Returns**: CreditCard object with generated card_id

### Update Credit Card  
- **PUT** `/customers/credit-cards/{cardId}`
- **URL Parameter**: card ID
- **Request Body**: CreditCard object with updated fields
- **Response**: 204 No Content

### Delete Credit Card
- **DELETE** `/customers/credit-cards/{cardId}`
- **URL Parameter**: card ID  
- **Response**: 204 No Content
- **Note**: Automatically clears customer's default_credit_card_id if this was the default

## Default Management Endpoints (NEW)

### Set Default Shipping Address
- **PUT** `/customers/{id}/default-shipping-address/{addressId}`
- **URL Parameters**: customer ID, address ID
- **Response**: 204 No Content
- **Event**: Emits `default.shipping_address.changed` event

### Set Default Billing Address  
- **PUT** `/customers/{id}/default-billing-address/{addressId}`
- **URL Parameters**: customer ID, address ID
- **Response**: 204 No Content
- **Event**: Emits `default.billing_address.changed` event

### Set Default Credit Card
- **PUT** `/customers/{id}/default-credit-card/{cardId}`  
- **URL Parameters**: customer ID, card ID
- **Response**: 204 No Content
- **Event**: Emits `default.credit_card.changed` event

### Clear Default Shipping Address
- **DELETE** `/customers/{id}/default-shipping-address`
- **URL Parameter**: customer ID
- **Response**: 204 No Content
- **Event**: Emits `default.shipping_address.changed` event with empty address_id

### Clear Default Billing Address
- **DELETE** `/customers/{id}/default-billing-address`  
- **URL Parameter**: customer ID
- **Response**: 204 No Content
- **Event**: Emits `default.billing_address.changed` event with empty address_id

### Clear Default Credit Card
- **DELETE** `/customers/{id}/default-credit-card`
- **URL Parameter**: customer ID  
- **Response**: 204 No Content
- **Event**: Emits `default.credit_card.changed` event with empty card_id

## Data Models

### Customer
```json
{
  "customer_id": "string (UUID)",
  "user_name": "string",
  "email": "string", 
  "first_name": "string",
  "last_name": "string",
  "phone": "string",
  "default_shipping_address_id": "string (UUID) or null",
  "default_billing_address_id": "string (UUID) or null", 
  "default_credit_card_id": "string (UUID) or null",
  "customer_since": "string (ISO8601)",
  "customer_status": "string",
  "status_date_time": "string (ISO8601)",
  "addresses": [Address],
  "credit_cards": [CreditCard],
  "status_history": [CustomerStatus]
}
```

### Address
```json
{
  "address_id": "string (UUID)",
  "customer_id": "string (UUID)", 
  "address_type": "string",
  "first_name": "string",
  "last_name": "string", 
  "address_1": "string",
  "address_2": "string",
  "city": "string",
  "state": "string", 
  "zip": "string"
}
```

### CreditCard
```json
{
  "card_id": "string (UUID)",
  "customer_id": "string (UUID)",
  "card_type": "string", 
  "card_number": "string",
  "card_holder_name": "string",
  "card_expires": "string",
  "card_cvv": "string"
}
```

### CustomerStatusHistory
```json
{
  "id": "integer",
  "customer_id": "string (UUID)",
  "old_status": "string",
  "new_status": "string",
  "changed_at": "string (ISO8601)"
}
```

## Common Workflows

### Add New Address and Set as Default
1. POST `/customers/{id}/addresses` → Get address_id from response
2. PUT `/customers/{id}/default-shipping-address/{addressId}` (or billing)

### Add New Credit Card and Set as Default  
1. POST `/customers/{id}/credit-cards` → Get card_id from response
2. PUT `/customers/{id}/default-credit-card/{cardId}`

### Replace Default Address
1. POST `/customers/{id}/addresses` → Add new address
2. PUT `/customers/{id}/default-shipping-address/{newAddressId}`
3. DELETE `/customers/addresses/{oldAddressId}` (optional cleanup)

### Clear All Defaults
1. DELETE `/customers/{id}/default-shipping-address`
2. DELETE `/customers/{id}/default-billing-address`
3. DELETE `/customers/{id}/default-credit-card`

## Event Types for Real-time Updates

The customer service emits these events that your frontend can subscribe to via WebSocket:

### Customer Events
- `customer.created` - New customer created
- `customer.updated` - Customer details updated

### Address Events
- `address.add` - Address added to customer
- `address.update` - Address updated
- `address.delete` - Address deleted

### Credit Card Events
- `card.add` - Credit card added to customer
- `card.update` - Credit card updated
- `card.delete` - Credit card deleted

### Default Change Events (NEW)
- `default.shipping_address.changed` - Default shipping address changed or cleared
- `default.billing_address.changed` - Default billing address changed or cleared
- `default.credit_card.changed` - Default credit card changed or cleared

## Error Handling
- **400 Bad Request**: Invalid input data, missing required fields, invalid UUID format
- **404 Not Found**: Customer/address/card not found  
- **500 Internal Server Error**: Backend processing error, database errors

## Important Notes

### UUID Generation
- **Backend generates all UUIDs** (customer_id, address_id, card_id)
- **Frontend should never generate or specify UUIDs** in requests
- Default fields are nullable UUIDs (can be null/unset)

### Database Constraints
- Deleting a default address/card automatically clears the default reference
- Foreign key constraints ensure referential integrity
- All operations are transactional for data consistency

### Performance Considerations
- Use dedicated default-setting endpoints instead of full customer updates
- Default-setting operations are efficient single-field updates
- Events are published asynchronously for real-time updates

### Security Notes
- All endpoints require proper authentication/authorization
- Credit card CVV data should be handled with appropriate security measures
- Customer data is protected by privacy regulations

## Example API Calls

### Add Address and Set as Default
```bash
# Step 1: Add address
curl -X POST https://pocstore.local/customers/123e4567-e89b-12d3-a456-426614174000/addresses \
  -H "Content-Type: application/json" \
  -d '{
    "address_type": "shipping",
    "first_name": "John",
    "last_name": "Doe",
    "address_1": "123 Main St",
    "city": "Anytown",
    "state": "CA",
    "zip": "12345"
  }'

# Response: {"address_id": "456e7890-e89b-12d3-a456-426614174001", ...}

# Step 2: Set as default
curl -X PUT https://pocstore.local/customers/123e4567-e89b-12d3-a456-426614174000/default-shipping-address/456e7890-e89b-12d3-a456-426614174001
```

### Clear Default Credit Card
```bash
curl -X DELETE https://pocstore.local/customers/123e4567-e89b-12d3-a456-426614174000/default-credit-card
```

This specification provides complete guidance for implementing frontend customer management functionality with the new default address/credit card features.