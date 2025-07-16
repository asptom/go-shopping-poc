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
    primary key (customer_id)
);

--CREATE DOMAIN address_type AS text CHECK (VALUE IN ('shipping', 'billing'));

DROP TABLE IF EXISTS customers.Address;
CREATE TABLE customers.Address (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id uuid not null,
    address_type text not null,
    first_name text,
    last_name text,
    address_1 text,
    address_2 text,
    city text,
    state text,
    zip text,
    is_default boolean
);

DROP TABLE IF EXISTS customers.CreditCard;
CREATE TABLE customers.CreditCard (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id uuid not null,
    card_type text,
    card_number text,
    card_holder_name text,
    card_expires text,
    card_cvv text,
    is_default boolean
);

--CREATE DOMAIN customer_status AS text CHECK (VALUE IN ('active', 'inactive', 'suspended', 'deleted'));

DROP TABLE IF EXISTS customers.CustomerStatus;
CREATE TABLE customers.CustomerStatus (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id uuid not null,
    customer_status text not null,
    status_date_time timestamp not null
);

DROP TABLE IF EXISTS customers.EventLog;
CREATE TABLE customers.EventLog (
    event_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_domain text not null,
    event_type text not null, 
    time_processed timestamp not null
);

DROP TABLE IF EXISTS customers.OutboxEvent;
CREATE TABLE customers.OutboxEvent (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    aggregate_type text not null,
    aggregate_id text not null,
    event_type text not null,
    time_stamp timestamp not null,
    payload text
);

-- One-to-many relationships

ALTER TABLE IF EXISTS customers.CreditCard
    ADD CONSTRAINT fk_customer_id
        FOREIGN KEY (customer_id)
            REFERENCES customers.Customer;

ALTER TABLE IF EXISTS customers.Address
    ADD CONSTRAINT fk_customer_id
        FOREIGN KEY (customer_id)
            REFERENCES customers.Customer;

ALTER TABLE IF EXISTS customers.CustomerStatus
    ADD CONSTRAINT fk_customer_id
        FOREIGN KEY (customer_id)
            REFERENCES customers.Customer;

CREATE SEQUENCE customer_sequence START 1 INCREMENT BY 1;