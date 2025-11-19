-- Customers Schema

-- This schema is used to manage customer-related data

-- Credentials are managed outside of source control in the .ENV file
-- There is a corresponding <domain>_db.sql script that is used to initialize the database
-- This script is used to create the customers schema in PostgreSQL

-- The following lines will be used to set the DB, USER, and PGPASSWORD used to 
-- run the psqlclient that executes this script
-- DB: $PSQL_CUSTOMER_DB
-- USER: $PSQL_CUSTOMER_ROLE
-- PGPASSWORD: $PSQL_CUSTOMER_PASSWORD

DROP SCHEMA IF EXISTS customers CASCADE;
CREATE SCHEMA customers;

SET search_path TO customers;

DROP TABLE IF EXISTS customers.Customer;
CREATE TABLE customers.Customer (
    customer_id uuid not null,
    user_name text not null,
    email text,
    first_name text,
    last_name text,
    phone text,
    default_shipping_address_id uuid NULL,
    default_billing_address_id uuid NULL,
    default_credit_card_id uuid NULL,
    customer_since timestamp not null default CURRENT_TIMESTAMP,
    customer_status text not null default 'active',
    status_date_time timestamp not null default CURRENT_TIMESTAMP,
    primary key (customer_id)
);

-- Index to quickly retrieve customer by username (optional but common)
CREATE INDEX idx_customer_user_name
    ON customers.Customer(user_name);

-- Index to quickly retrieve customer by email (optional but common)
CREATE INDEX idx_customer_email
    ON customers.Customer(email);

--CREATE DOMAIN address_type AS text CHECK (VALUE IN ('shipping', 'billing'));

DROP TABLE IF EXISTS customers.Address;
CREATE TABLE customers.Address (
    address_id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    customer_id uuid not null references customers.Customer(customer_id) ON DELETE CASCADE,
    address_type text not null,
    first_name text,
    last_name text,
    address_1 text,
    address_2 text,
    city text,
    state text,
    zip text
);

-- Index to quickly fetch all addresses for a customer
CREATE INDEX idx_address_customer
    ON customers.Address(customer_id);

DROP TABLE IF EXISTS customers.CreditCard;
CREATE TABLE customers.CreditCard (
    card_id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    customer_id uuid not null references customers.Customer(customer_id) ON DELETE CASCADE,
    card_type text,
    card_number text,
    card_holder_name text,
    card_expires text,
    card_cvv text
);

-- Index for customer card lookup
CREATE INDEX idx_creditcard_customer
    ON customers.CreditCard(customer_id);

--CREATE DOMAIN customer_status AS text CHECK (VALUE IN ('active', 'inactive', 'suspended', 'deleted'));

DROP TABLE IF EXISTS customers.CustomerStatusHistory;
CREATE TABLE customers.CustomerStatusHistory (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id uuid not null references customers.Customer(customer_id),
    old_status text not null,
    new_status text not null,
    changed_at timestamp not null default CURRENT_TIMESTAMP
);

-- Index to query history by customer
CREATE INDEX idx_customer_status_history_customer
    ON customers.CustomerStatusHistory(customer_id, changed_at DESC);

-- One-to-one relationships

ALTER TABLE customers.Customer
ADD CONSTRAINT fk_default_shipping_address
    FOREIGN KEY (default_shipping_address_id) REFERENCES customers.Address(address_id)
    ON DELETE SET NULL;

ALTER TABLE customers.Customer
ADD CONSTRAINT fk_default_billing_address
    FOREIGN KEY (default_billing_address_id) REFERENCES customers.Address(address_id)
    ON DELETE SET NULL;

ALTER TABLE customers.Customer
ADD CONSTRAINT fk_default_card
    FOREIGN KEY (default_credit_card_id) REFERENCES customers.CreditCard(card_id)
    ON DELETE SET NULL;

-- One-to-many relationships

ALTER TABLE IF EXISTS customers.CreditCard
    ADD CONSTRAINT fk_customer_id
        FOREIGN KEY (customer_id)
            REFERENCES customers.Customer;

ALTER TABLE IF EXISTS customers.Address
    ADD CONSTRAINT fk_customer_id
        FOREIGN KEY (customer_id)
            REFERENCES customers.Customer;

ALTER TABLE IF EXISTS customers.CustomerStatusHistory
    ADD CONSTRAINT fk_customer_id
        FOREIGN KEY (customer_id)
            REFERENCES customers.Customer;

CREATE SEQUENCE customer_sequence START 1 INCREMENT BY 1;

-- Outbox pattern for event sourcing
-- This schema is used to manage outbox events for this database

DROP SCHEMA IF EXISTS outbox CASCADE;
CREATE SCHEMA outbox;
SET search_path TO outbox;

DROP TABLE IF EXISTS outbox.outbox;
CREATE TABLE outbox.outbox (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_type text not null,
    topic text not null,
    event_payload jsonb not null,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP not null,
    times_attempted integer DEFAULT 0,
    published_at timestamp
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox.outbox (published_at) WHERE published_at IS NULL;