-- Add keycloak_sub column to Customer table for linking customers to Keycloak identities

ALTER TABLE customers.Customer ADD COLUMN keycloak_sub VARCHAR(255);
ALTER TABLE customers.Customer ADD CONSTRAINT uq_customers_keycloak_sub UNIQUE (keycloak_sub);
CREATE INDEX idx_customers_keycloak_sub ON customers.Customer(keycloak_sub);
