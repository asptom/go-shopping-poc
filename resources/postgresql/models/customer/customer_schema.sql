-- Customers Schema

-- This schema is used to manage customer-related data

-- Credentials are managed outside of source control in the .ENV file
-- There is a corresponding .db_template script that is used to initialize the database
-- This script is used to create the customers schema in PostgreSQL

-- This script is also used to generate Go structs for the customer domain using sqlc

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
    customerId uuid not null,
    username text not null,
    email text,
    firstName text,
    lastName text,
    phone text,
    primary key (customerId)
);

--CREATE DOMAIN address_type AS text CHECK (VALUE IN ('shipping', 'billing'));

DROP TABLE IF EXISTS customers.Address;
CREATE TABLE customers.Address (
    id bigint not null,
    customerId uuid not null,
    addressType text not null,
    firstName text,
    lastName text,
    address_1 text,
    address_2 text,
    city text,
    state text,
    zip text,
    isDefault boolean,
    primary key (id)
);

DROP TABLE IF EXISTS customers.CreditCard;
CREATE TABLE customers.CreditCard (
    id bigint not null,
    customerId uuid not null,
    cardType text,
    cardNumber text,
    cardHolderName text,
    cardExpires text,
    cardCVV text,
    isDefault boolean,
    primary key (id)
);

--CREATE DOMAIN customer_status AS text CHECK (VALUE IN ('active', 'inactive', 'suspended', 'deleted'));

DROP TABLE IF EXISTS customers.CustomerStatus;
CREATE TABLE customers.CustomerStatus (
    id bigint not null,
    customerId uuid not null,
    customerStatus text not null,
    statusDateTime timestamp,
    primary key (id)
);

DROP TABLE IF EXISTS customers.EventLog;
CREATE TABLE customers.EventLog (
    eventid uuid not null,
    eventdomain text not null,
    eventtype text not null, 
    timeprocessed timestamp not null,
    primary key (eventid)
);

-- One-to-many relationships

ALTER TABLE IF EXISTS customers.CreditCard
    ADD CONSTRAINT fk_customerId
        FOREIGN KEY (customerId)
            REFERENCES customers.Customer;

ALTER TABLE IF EXISTS customers.Address
    ADD CONSTRAINT fk_customerId
        FOREIGN KEY (customerId)
            REFERENCES customers.Customer;

ALTER TABLE IF EXISTS customers.CustomerStatus
    ADD CONSTRAINT fk_customerId
        FOREIGN KEY (customerId)
            REFERENCES customers.Customer;

DROP TABLE IF EXISTS customers.OutboxEvent;
CREATE TABLE customers.OutboxEvent (
    id uuid not null,
    aggregatetype text not null,
    aggregateid text not null,
    event_type text not null,
    time_stamp timestamp not null,
    payload text,
    primary key (id)
);

CREATE SEQUENCE customer_sequence START 1 INCREMENT BY 1;