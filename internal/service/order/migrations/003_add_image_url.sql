-- Migration: Add image_url to order items
-- Stores product image URL snapshot from cart item

ALTER TABLE orders.OrderItem ADD COLUMN image_url VARCHAR(500);

CREATE INDEX idx_order_items_image_url ON orders.OrderItem(image_url);