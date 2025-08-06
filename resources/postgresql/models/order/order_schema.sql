-- Orders Schema

-- This schema is used to manage order-related data

-- Credentials are managed outside of source control in the .ENV file
-- There is a corresponding <domain>_db.sql script that is used to initialize the database
-- This script is used to create the orders schema in PostgreSQL

-- The following lines will be used to set the DB, USER, and PGPASSWORD used to 
-- run the psqlclient that executes this script
-- DB: $PSQL_ORDER_DB
-- USER: $PSQL_ORDER_ROLE
-- PGPASSWORD: $PSQL_ORDER_PASSWORD

DROP SCHEMA IF EXISTS orders CASCADE;
CREATE SCHEMA orders;

SET search_path TO orders;

DROP TABLE IF EXISTS orders.OrderItem;
CREATE TABLE orders.OrderItem (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id uuid not null,
    line_number text,
    product_id text,
    product_name text,
    unit_price numeric(19,2),
    quantity numeric(19),
    total_price numeric (19,2),
    item_status text,
    item_status_date_time timestamp
);

DROP TABLE IF EXISTS orders.Contact;
CREATE TABLE orders.Contact (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id uuid not null,
    email text,
    first_name text,
    last_name text,
    phone text
);

DROP TABLE IF EXISTS orders.Address;
CREATE TABLE orders.Address (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id uuid not null,
    address_type text,
    first_name text,
    last_name text,
    address_1 text,
    address_2 text,
    city text,
    state text,
    zip text
);

DROP TABLE IF EXISTS orders.CreditCard;
CREATE TABLE orders.CreditCard (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id uuid not null,
    card_type text,
    card_number text,
    card_holder_name text,
    card_expires text,
    card_cvv text
);

DROP TABLE IF EXISTS orders.OrderStatus;
CREATE TABLE orders.OrderStatus (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id uuid not null,
    order_status text,
    status_date_time timestamp
);

DROP TABLE IF EXISTS orders.OrderHead;
CREATE TABLE orders.OrderHead (
    order_id uuid not null,
    cart_id uuid not null,
    contact_id bigint not null,
    credit_card_id bigint not null,
    net_price numeric(19,2),
    tax numeric(19,2),
    shipping numeric(19,2),
    total_price numeric(19,2),
    primary key (order_id)
);

-- One-to-one relationships

ALTER TABLE IF EXISTS orders.OrderHead
    ADD CONSTRAINT fk_contact_id
        FOREIGN KEY (contact_id)
            REFERENCES orders.Contact;

ALTER TABLE IF EXISTS orders.OrderHead
    ADD CONSTRAINT fk_credit_card_id
        FOREIGN KEY (credit_card_id)
            REFERENCES orders.CreditCard;

-- One-to-many relationships

ALTER TABLE IF EXISTS orders.OrderItem
    ADD CONSTRAINT fk_order_id
        FOREIGN KEY (order_id)
            REFERENCES orders.OrderHead;

ALTER TABLE IF EXISTS orders.Address
    ADD CONSTRAINT fk_order_id
        FOREIGN KEY (order_id)
            REFERENCES orders.OrderHead;

ALTER TABLE IF EXISTS orders.OrderStatus
    ADD CONSTRAINT fk_order_id
        FOREIGN KEY (order_id)
            REFERENCES orders.OrderHead;

CREATE SEQUENCE order_sequence START 1 INCREMENT BY 1;

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