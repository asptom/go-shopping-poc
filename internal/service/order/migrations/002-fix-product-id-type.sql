-- Fix product_id type mismatch: cart sends product_id as string (e.g., "40616315"), not UUID

ALTER TABLE orders.OrderItem 
ALTER COLUMN product_id TYPE text;
