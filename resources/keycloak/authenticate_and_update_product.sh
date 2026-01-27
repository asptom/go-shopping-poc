#!/bin/bash

# Script to authenticate with Keycloak and update a product via product-admin service
# Usage: ./authenticate_and_update_product.sh

set -e

# Configuration
KEYCLOAK_URL="http://keycloak.local/realms/pocstore-realm/protocol/openid-connect/token"
CLIENT_ID="product-admin"
CLIENT_SECRET="VVEXlDcyC2GLGldvpxTZAC905L4kYCkL"
USERNAME="product-admin-user"
PASSWORD="admin123"
PRODUCT_ID="39664424"
PRODUCT_ADMIN_URL="http://pocstore.local/api/v1/admin/products/$PRODUCT_ID"

echo "Authenticating with Keycloak..."

# # Get access token
# TOKEN_RESPONSE=$(curl -s -X POST "$KEYCLOAK_URL" \
#   -H "Content-Type: application/x-www-form-urlencoded" \
#   -d "grant_type=password&client_id=$CLIENT_ID&client_secret=$CLIENT_SECRET&username=$USERNAME&password=$PASSWORD")

# Get access token
TOKEN_RESPONSE=$(curl -s -X POST "$KEYCLOAK_URL" \
  -d grant_type=password -d client_id=$CLIENT_ID \
  -d username=$USERNAME -d password=$PASSWORD -d scope=openid \
  -d client_secret=$CLIENT_SECRET)


# Check if token request succeeded
if echo "$TOKEN_RESPONSE" | grep -q "error"; then
  echo "Authentication failed. Response: $TOKEN_RESPONSE"
  exit 1
fi

# Extract access token
ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.access_token')

if [ "$ACCESS_TOKEN" = "null" ] || [ -z "$ACCESS_TOKEN" ]; then
  echo "Failed to extract access token. Response: $TOKEN_RESPONSE"
  exit 1
fi

echo "Authentication successful. Updating product..."

# Product update payload (complete product record required)
PRODUCT_DATA='{
  "id": 39664424,
  "name": "Sample Product",
  "description": "A sample product for testing",
  "initial_price": 29.99,
  "final_price": 24.99,
  "currency": "USD",
  "in_stock": true,
  "color": "Blue",
  "size": "M",
  "main_image": "",
  "country_code": "US",
  "image_count": 0,
  "model_number": "SP-001",
  "other_attributes": "",
  "root_category": "Clothing",
  "category": "T-Shirts",
  "brand": "Test Brand",
  "all_available_sizes": ["S", "M", "L", "XL"]
}'

# Update product
UPDATE_RESPONSE=$(curl -s -X PUT "$PRODUCT_ADMIN_URL" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d "$PRODUCT_DATA")

# Check response
if echo "$UPDATE_RESPONSE" | grep -q "invalid\|expired\|error\|Error"; then
  echo "Product update failed. Response: $UPDATE_RESPONSE"
  exit 1
else
  echo "Product updated successfully!"
  echo "Response: $UPDATE_RESPONSE"
fi