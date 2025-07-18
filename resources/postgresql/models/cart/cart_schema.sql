-- Carts Schema

-- This schema is used to manage cart-related data

-- Credentials are managed outside of source control in the .ENV file
-- There is a corresponding <domain>_db.sql script that is used to initialize the database
-- This script is used to create the carts schema in PostgreSQL

-- The following lines will be used to set the DB, USER, and PGPASSWORD used to 
-- run the psqlclient that executes this script
-- DB: $PSQL_CART_DB
-- USER: $PSQL_CART_ROLE
-- PGPASSWORD: $PSQL_CART_PASSWORD

DROP SCHEMA IF EXISTS carts CASCADE;
CREATE SCHEMA carts;

SET search_path TO carts;

DROP TABLE IF EXISTS carts.Contact;
CREATE TABLE carts.Contact (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cart_id uuid not null,
    email text not null,
    first_name text not null,
    last_name text not null,
    phone text not null
);

DROP TABLE IF EXISTS carts.Address;
CREATE TABLE carts.Address (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cart_id uuid not null,
    address_type text not null,
    first_name text not null,
    last_name text not null,
    address_1 text not null,
    address_2 text not null,
    city text not null,
    state text not null,
    zip text not null
);

DROP TABLE IF EXISTS carts.CreditCard;
CREATE TABLE carts.CreditCard (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cart_id uuid not null,
    card_type text,
    card_number text,
    card_holder_name text,
    card_expires text,
    card_cvv text
);

DROP TABLE IF EXISTS carts.CartItem;
CREATE TABLE carts.CartItem (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cart_id uuid not null,
    line_number text not null,
    product_id text not null,
    product_name text,
    unit_price numeric(19,2),
    quantity numeric(19),
    total_price numeric (19,2)
);

DROP TABLE IF EXISTS carts.CartStatus;
CREATE TABLE carts.CartStatus (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cart_id uuid not null,
    cart_status text,
    status_date_time timestamp
);

DROP TABLE IF EXISTS carts.Cart;
CREATE TABLE carts.Cart (
    cart_id uuid not null,
    contact_id bigint not null,
    credit_card_id bigint not null,
    net_price numeric(19,2),
    tax numeric(19,2),
    shipping numeric(19,2),
    total_price numeric(19,2),
    primary key (cart_id)
);

-- One-to-one relationships

ALTER TABLE IF EXISTS carts.Cart
    ADD CONSTRAINT fk_contact_id
        FOREIGN KEY (contact_id)
            REFERENCES carts.Contact;

ALTER TABLE IF EXISTS carts.Cart
    ADD CONSTRAINT fk_credit_card_id
        FOREIGN KEY (credit_card_id)
            REFERENCES carts.CreditCard;

-- One-to-many relationships

ALTER TABLE IF EXISTS carts.Address
    ADD CONSTRAINT fk_cart_id
        FOREIGN KEY (cart_id)
            REFERENCES carts.Cart;

ALTER TABLE IF EXISTS carts.CartStatus
    ADD CONSTRAINT fk_cart_id
        FOREIGN KEY (cart_id)
            REFERENCES carts.Cart;

ALTER TABLE IF EXISTS carts.CartItem
    ADD CONSTRAINT fk_cart_id
        FOREIGN KEY (cart_id)
            REFERENCES carts.Cart;


CREATE SEQUENCE cart_sequence START 1 INCREMENT BY 1;

-- Outbox pattern for event sourcing
-- This schema is used to manage outbox events for this database

DROP SCHEMA IF EXISTS outbox CASCADE;
CREATE SCHEMA outbox;

DROP TABLE IF EXISTS outbox.processed_events;
CREATE TABLE outbox.processed_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id uuid not null,
    event_type text not null,
    time_processed timestamp not null
);

DROP TABLE IF EXISTS outbox.outbox;
CREATE TABLE outbox.outbox (
    id uuid PRIMARY KEY,
    created_at timestamp not null,
    event_type text not null,
    event_payload text,
    times_attempted integer DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_outbox_created_at ON outbox.outbox (created_at);
