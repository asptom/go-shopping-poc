-- name: GetCustomer :one
SELECT * FROM customers.Customer
WHERE customerId = $1 LIMIT 1;

-- name: GetCustomerByUsername :one
SELECT * FROM customers.Customer
WHERE username = $1 LIMIT 1;

-- name: GetCustomers :many
SELECT * FROM customers.Customer
ORDER BY username;

-- name: GetCustomerAddresses :many
SELECT * FROM customers.Address
WHERE customerId = $1
ORDER BY isDefault DESC;

-- name: GetCustomerShippingAddresses :many
SELECT * FROM customers.Address
WHERE customerId = $1 and addressType = 'shipping'
ORDER BY isDefault DESC;

-- name: GetCustomerBillingAddresses :many
SELECT * FROM customers.Address
WHERE customerId = $1 and addressType = 'billing'
ORDER BY isDefault DESC;

-- name: GetCustomerCreditCards :many
SELECT * FROM customers.CreditCard
WHERE customerId = $1
ORDER BY isDefault DESC;

-- name: GetCustomerCurrentStatus :one
SELECT * FROM customers.CustomerStatus
WHERE customerId = $1
ORDER BY statusDateTime DESC
LIMIT 1;

-- name: GetCustomerStatuses :many
SELECT * FROM customers.CustomerStatus
WHERE customerId = $1
ORDER BY statusDateTime DESC;

-- name: AddCustomer :one

INSERT INTO customers.Customer (
    customerId,
    username,
    firstName,
    lastName,
    email,
    phone
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING customerId, username, firstName, lastName, email, phone;

-- name: AddCustomerAddress :one

INSERT INTO customers.Address (
    id,
    customerId,
    addressType,
    firstName,
    lastName,
    address_1,
    address_2,
    city,
    state,
    zip,
    isDefault
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING id;

-- name: AddCustomerCreditCard :one

INSERT INTO customers.CreditCard (
    id,
    customerId,
    cardType,
    cardNumber,
    cardHolderName,
    cardExpires,
    cardCVV,
    isDefault
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id;

-- name: AddCustomerStatus :one

INSERT INTO customers.CustomerStatus (
    id,
    customerId,
    customerStatus,
    statusDateTime
) VALUES (
  $1, $2, $3, NOW()
)
RETURNING id;