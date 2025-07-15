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
CREATE SCHEMA orders AUTHORIZATION $PSQL_ORDERS_ROLE;

DROP TABLE IF EXISTS orders.OrderItem;
CREATE TABLE orders.OrderItem (
    id int not null,
    order_id uuid not null,
    line_number varchar(255),
    product_id varchar(255),
    product_name varchar(255),
    unit_price numeric(19,2),
    quantity numeric(19),
    total_price numeric (19,2),
    item_status varchar(255),
    item_status_date_time timestamp,
    primary key (id)
);

DROP TABLE IF EXISTS orders.Contact;
CREATE TABLE orders.Contact (
    id int not null,
    order_id uuid not null,
    email varchar(255),
    first_name varchar(255),
    last_name varchar(255),
    phone varchar(255),
    primary key (id)
);

DROP TABLE IF EXISTS orders.Address;
CREATE TABLE orders.Address (
    id int not null,
    order_id uuid not null,
    address_type varchar(255),
    first_name varchar(255),
    last_name varchar(255),
    address_1 varchar(255),
    address_2 varchar(255),
    city varchar(255),
    state varchar(255),
    zip varchar(255),
    primary key (id)
);

DROP TABLE IF EXISTS orders.CreditCard;
CREATE TABLE orders.CreditCard (
    id int not null,
    order_id uuid not null,
    card_type varchar(255),
    card_number varchar(255),
    card_holder_name varchar(255),
    card_expires varchar(255),
    card_cvv varchar(255),
    primary key (id)
);

DROP TABLE IF EXISTS orders.OrderStatus;
CREATE TABLE orders.OrderStatus (
    id int not null,
    order_id uuid not null,
    order_status varchar(255),
    status_date_time timestamp,
    primary key (id)
);

DROP TABLE IF EXISTS orders.OrderHead;
CREATE TABLE orders.OrderHead (
    order_id uuid not null,
    cart_id uuid not null,
    contact_id int not null,
    credit_card_id int not null,
    net_price numeric(19,2),
    tax numeric(19,2),
    shipping numeric(19,2),
    total_price numeric(19,2),
    primary key (order_id)
);

DROP TABLE IF EXISTS orders.EventLog;
CREATE TABLE orders.EventLog (
    event_id uuid not null,
    event_domain text not null,
    event_type text not null, 
    time_processed timestamp not null,
    primary key (event_id)
);

DROP TABLE IF EXISTS orders.OutboxEvent;
CREATE TABLE orders.OutboxEvent (
    id uuid not null,
    aggregate_type text not null,
    aggregate_id text not null,
    event_type text not null,
    time_stamp timestamp not null,
    payload text,
    primary key (id)
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

SET search_path TO orders;

GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA orders TO $PSQL_ORDERS_ROLE;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA orders TO $PSQL_ORDERS_ROLE;

CREATE SEQUENCE order_sequence START 1 INCREMENT BY 1;
