-- Carts Schema

-- This schema is used to manage cart-related data

-- Credentials are managed outside of source control in the .ENV file
-- There is a corresponding .db_template script that is used to initialize the database
-- This script is used to create the carts schema in PostgreSQL

-- This script is also used to generate Go structs for the cart domain using sqlc

-- The following lines will be used to set the DB, USER, and PGPASSWORD used to 
-- run the psqlclient that executes this script
-- DB: $PSQL_CART_DB
-- USER: $PSQL_CART_ROLE
-- PGPASSWORD: $PSQL_CART_PASSWORD

DROP SCHEMA IF EXISTS carts CASCADE;
CREATE SCHEMA carts;

SET search_path TO carts;

DROP TABLE IF EXISTS carts.EventLog;
CREATE TABLE carts.EventLog (
    eventid uuid not null,
    eventdomain varchar(255) not null,
    eventtype varchar(255) not null, 
    timeprocessed timestamp not null,
    primary key (eventid)
);

DROP TABLE IF EXISTS carts.Contact;
CREATE TABLE carts.Contact (
    id int not null,
    cartId uuid not null,
    email varchar(255) not null,
    firstName varchar(255) not null,
    lastName varchar(255) not null,
    phone varchar(255) not null,
    primary key (id)
);

DROP TABLE IF EXISTS carts.Address;
CREATE TABLE carts.Address (
    id int not null,
    cartId uuid not null,
    addressType varchar(255) not null,
    firstName varchar(255) not null,
    lastName varchar(255) not null,
    address_1 varchar(255) not null,
    address_2 varchar(255) not null,
    city varchar(255) not null,
    state varchar(255) not null,
    zip varchar(255) not null,
    primary key (id)
);

DROP TABLE IF EXISTS carts.CreditCard;
CREATE TABLE carts.CreditCard (
    id int not null,
    cartId uuid not null,
    cardType varchar(255),
    cardNumber varchar(255),
    cardHolderName varchar(255),
    cardExpires varchar(255),
    cardCVV varchar(255),
    primary key (id)
);

DROP TABLE IF EXISTS carts.ShoppingCartItem;
CREATE TABLE carts.ShoppingCartItem (
    id int not null,
    cartId uuid not null,
    lineNumber varchar(255) not null,
    productId varchar(255) not null,
    productName varchar(255),
    unitPrice numeric(19,2),
    quantity numeric(19),
    totalPrice numeric (19,2),
    primary key (id)
);

DROP TABLE IF EXISTS carts.cartstatus;
CREATE TABLE carts.cartstatus (
    id int not null,
    cartId uuid not null,
    status varchar(255),
    statusDateTime timestamp,
    primary key (id)
);

DROP TABLE IF EXISTS carts.ShoppingCart;
CREATE TABLE carts.ShoppingCart (
    cartId uuid not null,
    contactId int not null,
    creditCardId int not null,
    netPrice numeric(19,2),
    tax numeric(19,2),
    shipping numeric(19,2),
    totalPrice numeric(19,2),
    primary key (cartId)
);

-- One-to-one relationships

ALTER TABLE IF EXISTS carts.ShoppingCart
    ADD CONSTRAINT fk_contactId
        FOREIGN KEY (contactId)
            REFERENCES carts.Contact;

ALTER TABLE IF EXISTS carts.ShoppingCart
    ADD CONSTRAINT fk_creditCardId
        FOREIGN KEY (creditCardId)
            REFERENCES carts.CreditCard;

-- One-to-many relationships

ALTER TABLE IF EXISTS carts.Address
    ADD CONSTRAINT fk_cartId
        FOREIGN KEY (cartId)
            REFERENCES carts.ShoppingCart;

ALTER TABLE IF EXISTS carts.cartstatus
    ADD CONSTRAINT fk_cartId
        FOREIGN KEY (cartId)
            REFERENCES carts.ShoppingCart;

ALTER TABLE IF EXISTS carts.ShoppingCartItem
    ADD CONSTRAINT fk_cartId
        FOREIGN KEY (cartId)
            REFERENCES carts.ShoppingCart;

DROP TABLE IF EXISTS carts.OutboxEvent;
CREATE TABLE carts.OutboxEvent (
    id uuid not null,
    aggregatetype varchar(255) not null,
    aggregateid varchar(255) not null,
    type varchar(255) not null,
    timestamp timestamp not null,
    payload varchar(7500),
    primary key (id)
);

CREATE SEQUENCE cart_sequence START 1 INCREMENT BY 1;