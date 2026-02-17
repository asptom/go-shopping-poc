-- Migration: Add cart item validation support
-- Adds status tracking and validation correlation to cart items

-- Add status column with default for existing items
ALTER TABLE carts.CartItem ADD COLUMN status VARCHAR(20) DEFAULT 'confirmed';
ALTER TABLE carts.CartItem ALTER COLUMN status SET NOT NULL;

-- Add validation correlation ID for tracking pending validations
ALTER TABLE carts.CartItem ADD COLUMN validation_id UUID;

-- Add backorder reason for failed validations
ALTER TABLE carts.CartItem ADD COLUMN backorder_reason VARCHAR(100);

-- Index for validation ID lookups (event handler needs this)
CREATE INDEX idx_cart_items_validation_id ON carts.CartItem(validation_id);

-- Index for status filtering (frontend filtering)
CREATE INDEX idx_cart_items_status ON carts.CartItem(status);

-- Composite index for cart + status queries
CREATE INDEX idx_cart_items_cart_status ON carts.CartItem(cart_id, status);
