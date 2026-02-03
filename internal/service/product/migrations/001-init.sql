-- Products Schema

-- This schema is used to manage product-related data

DROP SCHEMA IF EXISTS products CASCADE;
CREATE SCHEMA products;

SET search_path TO products;

-- Product Image Ingestion Database Schema
-- Supports product data and image metadata storage

-- Products table - core product information
DROP TABLE IF EXISTS products.Products;
CREATE TABLE products.Products (
    id BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    initial_price DECIMAL(10,2),
    final_price DECIMAL(10,2),
    currency VARCHAR(3) DEFAULT 'USD',
    in_stock BOOLEAN DEFAULT true,
    color VARCHAR(100),
    size VARCHAR(100),
    -- REMOVED: main_image column - now using is_main flag in product_images table
    country_code VARCHAR(2),
    image_count INTEGER,
    model_number VARCHAR(100),
    root_category VARCHAR(100),
    category VARCHAR(100),
    brand VARCHAR(100),
    other_attributes TEXT,
    all_available_sizes JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Product images table - image metadata and MinIO references
CREATE TABLE products.Product_images (
    id SERIAL PRIMARY KEY,
    product_id BIGINT REFERENCES products.Products(id) ON DELETE CASCADE,
    minio_object_name VARCHAR(500) NOT NULL,  -- Required for Minio-only storage
    is_main BOOLEAN DEFAULT false,
    image_order INTEGER DEFAULT 0,
    file_size BIGINT,
    content_type VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, minio_object_name)
);

-- Performance indexes
CREATE INDEX idx_products_brand ON products.Products(brand);
CREATE INDEX idx_products_category ON products.Products(category);
CREATE INDEX idx_product_images_product_id ON products.Product_images(product_id);
CREATE INDEX idx_product_images_main ON products.Product_images(product_id, is_main);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_products_updated_at 
    BEFORE UPDATE ON products.Products 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Trigger to auto-update image_count on products table
CREATE OR REPLACE FUNCTION update_image_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE products.Products SET image_count = COALESCE(image_count, 0) + 1 WHERE id = NEW.product_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE products.Products SET image_count = GREATEST(COALESCE(image_count, 0) - 1, 0) WHERE id = OLD.product_id;
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' AND NEW.product_id != OLD.product_id THEN
        UPDATE products.Products SET image_count = GREATEST(COALESCE(image_count, 0) - 1, 0) WHERE id = OLD.product_id;
        UPDATE products.Products SET image_count = COALESCE(image_count, 0) + 1 WHERE id = NEW.product_id;
        RETURN NEW;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_product_image_count
    AFTER INSERT OR DELETE OR UPDATE ON products.Product_images
    FOR EACH ROW EXECUTE FUNCTION update_image_count();

-- ==========================================================
-- Outbox pattern for event sourcing
-- This schema is used to manage outbox events for this database
-- ==========================================================

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

