-- Migration: Add image_url to cart items
-- Stores product image URL as a snapshot when item is added to cart

ALTER TABLE carts.CartItem ADD COLUMN image_url VARCHAR(500);

CREATE INDEX idx_cart_items_image_url ON carts.CartItem(image_url);