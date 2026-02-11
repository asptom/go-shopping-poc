-- Carts Schema
-- This schema is used to manage cart-related data

DROP SCHEMA IF EXISTS carts CASCADE;
CREATE SCHEMA carts;

SET search_path TO carts;

-- Contact information table
DROP TABLE IF EXISTS carts.Contact CASCADE;
CREATE TABLE carts.Contact (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cart_id uuid not null,
    email text not null,
    first_name text not null,
    last_name text not null,
    phone text not null
);

-- Address table for shipping and billing
DROP TABLE IF EXISTS carts.Address CASCADE;
CREATE TABLE carts.Address (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cart_id uuid not null,
    address_type text not null,
    first_name text not null,
    last_name text not null,
    address_1 text not null,
    address_2 text,  -- Made nullable
    city text not null,
    state text not null,
    zip text not null,
    CONSTRAINT chk_address_type CHECK (address_type IN ('shipping', 'billing'))
);

-- Credit card table
DROP TABLE IF EXISTS carts.CreditCard CASCADE;
CREATE TABLE carts.CreditCard (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cart_id uuid not null,
    card_type text,
    card_number text,
    card_holder_name text,
    card_expires text,
    card_cvv text
);

-- Cart items (denormalized product data)
DROP TABLE IF EXISTS carts.CartItem CASCADE;
CREATE TABLE carts.CartItem (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cart_id uuid not null,
    line_number text not null,
    product_id text not null,
    product_name text,
    unit_price numeric(19,2),
    quantity numeric(19),
    total_price numeric (19,2),
    CONSTRAINT chk_quantity_positive CHECK (quantity > 0),
    CONSTRAINT unique_line_number_per_cart UNIQUE (cart_id, line_number)
);

-- Cart status history
DROP TABLE IF EXISTS carts.CartStatus CASCADE;
CREATE TABLE carts.CartStatus (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cart_id uuid not null,
    cart_status text not null,
    status_date_time timestamp DEFAULT CURRENT_TIMESTAMP not null
);

-- Main cart table
DROP TABLE IF EXISTS carts.Cart CASCADE;
CREATE TABLE carts.Cart (
    cart_id uuid not null,
    customer_id uuid,  -- Nullable for guest carts
    contact_id bigint,  -- Made nullable
    credit_card_id bigint,  -- Made nullable
    current_status text NOT NULL DEFAULT 'active',  -- Track current status
    currency text DEFAULT 'USD',
    net_price numeric(19,2) DEFAULT 0,
    tax numeric(19,2) DEFAULT 0,
    shipping numeric(19,2) DEFAULT 0,
    total_price numeric(19,2) DEFAULT 0,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP not null,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP not null,
    version integer DEFAULT 1,  -- For optimistic locking
    primary key (cart_id),
    CONSTRAINT chk_status CHECK (current_status IN ('active', 'checked_out', 'completed', 'cancelled'))
);

-- One-to-one relationships
ALTER TABLE carts.Cart
    ADD CONSTRAINT fk_contact_id
        FOREIGN KEY (contact_id)
            REFERENCES carts.Contact(id)
            ON DELETE SET NULL;

ALTER TABLE carts.Cart
    ADD CONSTRAINT fk_credit_card_id
        FOREIGN KEY (credit_card_id)
            REFERENCES carts.CreditCard(id)
            ON DELETE SET NULL;

-- One-to-many relationships
ALTER TABLE carts.Contact
    ADD CONSTRAINT fk_contact_cart_id
        FOREIGN KEY (cart_id)
            REFERENCES carts.Cart(cart_id)
            ON DELETE CASCADE;

ALTER TABLE carts.Address
    ADD CONSTRAINT fk_address_cart_id
        FOREIGN KEY (cart_id)
            REFERENCES carts.Cart(cart_id)
            ON DELETE CASCADE;

ALTER TABLE carts.CartStatus
    ADD CONSTRAINT fk_status_cart_id
        FOREIGN KEY (cart_id)
            REFERENCES carts.Cart(cart_id)
            ON DELETE CASCADE;

ALTER TABLE carts.CartItem
    ADD CONSTRAINT fk_item_cart_id
        FOREIGN KEY (cart_id)
            REFERENCES carts.Cart(cart_id)
            ON DELETE CASCADE;

ALTER TABLE carts.CreditCard
    ADD CONSTRAINT fk_creditcard_cart_id
        FOREIGN KEY (cart_id)
            REFERENCES carts.Cart(cart_id)
            ON DELETE CASCADE;

-- Indexes for performance
CREATE INDEX idx_cart_customer_id ON carts.Cart(customer_id);
CREATE INDEX idx_cart_current_status ON carts.Cart(current_status);
CREATE INDEX idx_cart_created_at ON carts.Cart(created_at);
CREATE INDEX idx_cart_status_history ON carts.CartStatus(cart_id, status_date_time);
CREATE INDEX idx_cart_item_cart_id ON carts.CartItem(cart_id);

-- Partial unique index: one active cart per customer
CREATE UNIQUE INDEX idx_one_active_cart_per_customer 
    ON carts.Cart (customer_id) 
    WHERE customer_id IS NOT NULL AND current_status = 'active';

-- Sequence for generating line numbers
CREATE SEQUENCE cart_sequence START 1 INCREMENT BY 1;

-- Trigger to auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_cart_updated_at 
    BEFORE UPDATE ON carts.Cart 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Outbox pattern for event sourcing
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
