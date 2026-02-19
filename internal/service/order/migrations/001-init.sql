-- Orders Schema

DROP SCHEMA IF EXISTS orders CASCADE;
CREATE SCHEMA orders;

SET search_path TO orders;

-- Order Items Table
DROP TABLE IF EXISTS orders.OrderItem;
CREATE TABLE orders.OrderItem (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id uuid not null,
    line_number int not null,
    product_id uuid not null,
    product_name text not null,
    unit_price numeric(19,4),
    quantity int not null,
    total_price numeric(19,4),
    item_status text default 'pending',
    item_status_date_time timestamp default CURRENT_TIMESTAMP,
    UNIQUE(order_id, line_number)
);

-- Order Contact Information (snapshot at order time)
DROP TABLE IF EXISTS orders.Contact;
CREATE TABLE orders.Contact (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id uuid not null,
    email text not null,
    first_name text not null,
    last_name text not null,
    phone text
);

-- Order Address (shipping and billing)
DROP TABLE IF EXISTS orders.Address;
CREATE TABLE orders.Address (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id uuid not null,
    address_type text not null check (address_type in ('shipping', 'billing')),
    first_name text,
    last_name text,
    address_1 text not null,
    address_2 text,
    city text not null,
    state text not null,
    zip text not null
);

-- Payment Information (snapshot at order time)
DROP TABLE IF EXISTS orders.CreditCard;
CREATE TABLE orders.CreditCard (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id uuid not null,
    card_type text not null,
    card_number text not null,
    card_holder_name text not null,
    card_expires text not null,
    card_cvv text not null
);

-- Order Status History
DROP TABLE IF EXISTS orders.OrderStatus;
CREATE TABLE orders.OrderStatus (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id uuid not null,
    order_status text not null check (order_status in ('created', 'confirmed', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded')),
    status_date_time timestamp default CURRENT_TIMESTAMP not null,
    notes text
);

-- Order Header (Main Order Table)
DROP TABLE IF EXISTS orders.OrderHead;
CREATE TABLE orders.OrderHead (
    order_id uuid not null,
    order_number text not null unique,
    cart_id uuid not null,
    customer_id uuid,
    contact_id bigint not null,
    credit_card_id bigint not null,
    currency text default 'USD' not null,
    net_price numeric(19,4) not null,
    tax numeric(19,4) not null,
    shipping numeric(19,4) not null,
    total_price numeric(19,4) not null,
    current_status text default 'created' not null check (current_status in ('created', 'confirmed', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded')),
    created_at timestamp default CURRENT_TIMESTAMP not null,
    updated_at timestamp default CURRENT_TIMESTAMP not null,
    primary key (order_id)
);

-- One-to-one relationships
ALTER TABLE IF EXISTS orders.OrderHead
    ADD CONSTRAINT fk_contact_id
        FOREIGN KEY (contact_id)
            REFERENCES orders.Contact(id) ON DELETE RESTRICT;

ALTER TABLE IF EXISTS orders.OrderHead
    ADD CONSTRAINT fk_credit_card_id
        FOREIGN KEY (credit_card_id)
            REFERENCES orders.CreditCard(id) ON DELETE RESTRICT;

-- One-to-many relationships
ALTER TABLE IF EXISTS orders.OrderItem
    ADD CONSTRAINT fk_order_id
        FOREIGN KEY (order_id)
            REFERENCES orders.OrderHead(order_id) ON DELETE CASCADE;

ALTER TABLE IF EXISTS orders.Address
    ADD CONSTRAINT fk_order_id
        FOREIGN KEY (order_id)
            REFERENCES orders.OrderHead(order_id) ON DELETE CASCADE;

ALTER TABLE IF EXISTS orders.OrderStatus
    ADD CONSTRAINT fk_order_id
        FOREIGN KEY (order_id)
            REFERENCES orders.OrderHead(order_id) ON DELETE CASCADE;

-- Indexes for performance
CREATE INDEX idx_orderhead_customer_id ON orders.OrderHead(customer_id);
CREATE INDEX idx_orderhead_cart_id ON orders.OrderHead(cart_id);
CREATE INDEX idx_orderhead_order_number ON orders.OrderHead(order_number);
CREATE INDEX idx_orderhead_created_at ON orders.OrderHead(created_at);
CREATE INDEX idx_orderhead_current_status ON orders.OrderHead(current_status);
CREATE INDEX idx_orderitem_order_id ON orders.OrderItem(order_id);
CREATE INDEX idx_address_order_id ON orders.Address(order_id);
CREATE INDEX idx_address_type ON orders.Address(address_type);
CREATE INDEX idx_orderstatus_order_id ON orders.OrderStatus(order_id);

-- Order Number Sequence
CREATE SEQUENCE order_number_seq START 1000 INCREMENT BY 1;

-- Function to generate order number
CREATE OR REPLACE FUNCTION generate_order_number()
RETURNS text AS $$
DECLARE
    year text;
    seq_num bigint;
BEGIN
    year := to_char(CURRENT_DATE, 'YYYY');
    seq_num := nextval('orders.order_number_seq');
    RETURN 'ORD-' || year || '-' || lpad(seq_num::text, 8, '0');
END;
$$ LANGUAGE plpgsql;

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
