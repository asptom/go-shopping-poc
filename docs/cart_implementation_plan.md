# Cart Service Implementation Plan

## Overview

This document provides a detailed implementation plan for the Cart Service in the go-shopping-poc application. The cart service follows Clean Architecture principles and integrates with the existing customer and product services.

**Key Features:**
- Support for both guest and authenticated users
- Real-time product validation via HTTP calls to product service
- Event-driven checkout flow
- Denormalized product data in cart items
- 3% tax calculation with $0 shipping (placeholders for future services)

**Product Service URL:** `http://product-svc.shopping.svc.cluster.local:80`

---

## Database Schema (001_init.sql)

### Complete Updated Schema

```sql
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
```

---

## Entity Layer (internal/service/cart/entity.go)

### Package Structure
Create file: `internal/service/cart/entity.go`

### Core Entities

#### Cart Entity
```go
type Cart struct {
    CartID       uuid.UUID     `json:"cart_id" db:"cart_id"`
    CustomerID   *uuid.UUID    `json:"customer_id,omitempty" db:"customer_id"`
    ContactID    *int64        `json:"-" db:"contact_id"`
    CreditCardID *int64        `json:"-" db:"credit_card_id"`
    CurrentStatus string       `json:"current_status" db:"current_status"`
    Currency     string        `json:"currency" db:"currency"`
    NetPrice     float64       `json:"net_price" db:"net_price"`
    Tax          float64       `json:"tax" db:"tax"`
    Shipping     float64       `json:"shipping" db:"shipping"`
    TotalPrice   float64       `json:"total_price" db:"total_price"`
    CreatedAt    time.Time     `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`
    Version      int           `json:"version" db:"version"`
    
    // Relationships
    Contact       *Contact       `json:"contact,omitempty"`
    Addresses     []Address      `json:"addresses,omitempty"`
    CreditCard    *CreditCard    `json:"credit_card,omitempty"`
    Items         []CartItem     `json:"items,omitempty"`
    StatusHistory []CartStatus   `json:"status_history,omitempty"`
}

// Validate performs domain validation
func (c *Cart) Validate() error {
    if c.Currency == "" {
        return errors.New("currency is required")
    }
    validStatuses := []string{"active", "checked_out", "completed", "cancelled"}
    if !contains(validStatuses, c.CurrentStatus) {
        return errors.New("invalid cart status")
    }
    return nil
}

// CalculateTotals computes all cart totals (3% tax, $0 shipping)
func (c *Cart) CalculateTotals() {
    c.NetPrice = 0
    for i := range c.Items {
        c.Items[i].CalculateLineTotal()
        c.NetPrice += c.Items[i].TotalPrice
    }
    c.Tax = c.calculateTax()
    c.Shipping = c.calculateShipping()
    c.TotalPrice = c.NetPrice + c.Tax + c.Shipping
}

func (c *Cart) calculateTax() float64 {
    return c.NetPrice * 0.03 // 3% tax rate
}

func (c *Cart) calculateShipping() float64 {
    return 0.0 // Placeholder for future shipping service
}

// CanCheckout validates cart is ready for checkout
func (c *Cart) CanCheckout() error {
    if c.CurrentStatus != "active" {
        return errors.New("cart must be active to checkout")
    }
    if len(c.Items) == 0 {
        return errors.New("cart must have at least one item")
    }
    if c.Contact == nil {
        return errors.New("contact information required")
    }
    if c.CreditCard == nil {
        return errors.New("payment method required")
    }
    return nil
}

// SetStatus updates cart status with validation
func (c *Cart) SetStatus(newStatus string) error {
    validTransitions := map[string][]string{
        "active":       {"checked_out", "cancelled"},
        "checked_out":  {"completed", "cancelled"},
        "completed":    {},
        "cancelled":    {},
    }
    
    allowed, ok := validTransitions[c.CurrentStatus]
    if !ok {
        return fmt.Errorf("unknown current status: %s", c.CurrentStatus)
    }
    
    if !contains(allowed, newStatus) {
        return fmt.Errorf("invalid status transition from %s to %s", c.CurrentStatus, newStatus)
    }
    
    c.CurrentStatus = newStatus
    c.StatusHistory = append(c.StatusHistory, CartStatus{
        CartID:    c.CartID,
        Status:    newStatus,
        ChangedAt: time.Now(),
    })
    return nil
}
```

#### CartItem Entity
```go
type CartItem struct {
    ID          int64     `json:"id" db:"id"`
    CartID      uuid.UUID `json:"cart_id" db:"cart_id"`
    LineNumber  string    `json:"line_number" db:"line_number"`
    ProductID   string    `json:"product_id" db:"product_id"`
    ProductName string    `json:"product_name" db:"product_name"`
    UnitPrice   float64   `json:"unit_price" db:"unit_price"`
    Quantity    int       `json:"quantity" db:"quantity"`
    TotalPrice  float64   `json:"total_price" db:"total_price"`
}

func (ci *CartItem) CalculateLineTotal() {
    ci.TotalPrice = float64(ci.Quantity) * ci.UnitPrice
}

func (ci *CartItem) Validate() error {
    if ci.ProductID == "" {
        return errors.New("product_id is required")
    }
    if ci.Quantity <= 0 {
        return errors.New("quantity must be positive")
    }
    if ci.UnitPrice < 0 {
        return errors.New("unit_price cannot be negative")
    }
    return nil
}
```

#### Contact Entity
```go
type Contact struct {
    ID        int64     `json:"id" db:"id"`
    CartID    uuid.UUID `json:"cart_id" db:"cart_id"`
    Email     string    `json:"email" db:"email"`
    FirstName string    `json:"first_name" db:"first_name"`
    LastName  string    `json:"last_name" db:"last_name"`
    Phone     string    `json:"phone" db:"phone"`
}

func (c *Contact) Validate() error {
    if strings.TrimSpace(c.Email) == "" {
        return errors.New("email is required")
    }
    if !strings.Contains(c.Email, "@") {
        return errors.New("invalid email format")
    }
    if strings.TrimSpace(c.FirstName) == "" {
        return errors.New("first_name is required")
    }
    if strings.TrimSpace(c.LastName) == "" {
        return errors.New("last_name is required")
    }
    if strings.TrimSpace(c.Phone) == "" {
        return errors.New("phone is required")
    }
    return nil
}
```

#### Address Entity
```go
type Address struct {
    ID          int64     `json:"id" db:"id"`
    CartID      uuid.UUID `json:"cart_id" db:"cart_id"`
    AddressType string    `json:"address_type" db:"address_type"`
    FirstName   string    `json:"first_name" db:"first_name"`
    LastName    string    `json:"last_name" db:"last_name"`
    Address1    string    `json:"address_1" db:"address_1"`
    Address2    string    `json:"address_2" db:"address_2"`
    City        string    `json:"city" db:"city"`
    State       string    `json:"state" db:"state"`
    Zip         string    `json:"zip" db:"zip"`
}

func (a *Address) Validate() error {
    validTypes := []string{"shipping", "billing"}
    if !contains(validTypes, a.AddressType) {
        return errors.New("address_type must be shipping or billing")
    }
    if strings.TrimSpace(a.Address1) == "" {
        return errors.New("address_1 is required")
    }
    if strings.TrimSpace(a.City) == "" {
        return errors.New("city is required")
    }
    if strings.TrimSpace(a.State) == "" {
        return errors.New("state is required")
    }
    if strings.TrimSpace(a.Zip) == "" {
        return errors.New("zip is required")
    }
    return nil
}
```

#### CreditCard Entity
```go
type CreditCard struct {
    ID             int64     `json:"id" db:"id"`
    CartID         uuid.UUID `json:"cart_id" db:"cart_id"`
    CardType       string    `json:"card_type" db:"card_type"`
    CardNumber     string    `json:"card_number" db:"card_number"`
    CardHolderName string    `json:"card_holder_name" db:"card_holder_name"`
    CardExpires    string    `json:"card_expires" db:"card_expires"`
    CardCVV        string    `json:"card_cvv" db:"card_cvv"`
}

func (cc *CreditCard) Validate() error {
    if strings.TrimSpace(cc.CardNumber) == "" {
        return errors.New("card_number is required")
    }
    if strings.TrimSpace(cc.CardHolderName) == "" {
        return errors.New("card_holder_name is required")
    }
    if strings.TrimSpace(cc.CardExpires) == "" {
        return errors.New("card_expires is required")
    }
    if strings.TrimSpace(cc.CardCVV) == "" {
        return errors.New("card_cvv is required")
    }
    return nil
}

func (cc *CreditCard) MaskedNumber() string {
    if len(cc.CardNumber) < 4 {
        return cc.CardNumber
    }
    return "****-****-****-" + cc.CardNumber[len(cc.CardNumber)-4:]
}
```

#### CartStatus Entity
```go
type CartStatus struct {
    ID        int64     `json:"id" db:"id"`
    CartID    uuid.UUID `json:"cart_id" db:"cart_id"`
    Status    string    `json:"cart_status" db:"cart_status"`
    ChangedAt time.Time `json:"status_date_time" db:"status_date_time"`
}
```

#### Helper Function
```go
func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

---

## Event Contracts (internal/contracts/events/cart.go)

### Package Structure
Create file: `internal/contracts/events/cart.go`

### Event Definitions
```go
package events

import (
    "encoding/json"
    "time"
    
    "github.com/google/uuid"
)

// Event Types
const (
    CartCreated    EventType = "cart.created"
    CartDeleted    EventType = "cart.deleted"
    CartCheckedOut EventType = "cart.checked_out"
)

// CartEvent represents cart domain events
type CartEvent struct {
    ID        string          `json:"id"`
    EventType EventType       `json:"type"`
    Timestamp time.Time       `json:"timestamp"`
    Payload   CartEventPayload `json:"payload"`
}

// CartEventPayload contains event data
type CartEventPayload struct {
    CartID     string   `json:"cart_id"`
    CustomerID *string  `json:"customer_id,omitempty"`
    TotalPrice float64  `json:"total_price,omitempty"`
    ItemCount  int      `json:"item_count,omitempty"`
    Details    map[string]string `json:"details,omitempty"`
}

// Event interface implementation
func (e CartEvent) Type() string        { return string(e.EventType) }
func (e CartEvent) Topic() string       { return "cart-events" }
func (e CartEvent) Payload() any        { return e.Payload }
func (e CartEvent) ToJSON() ([]byte, error) { return json.Marshal(e) }
func (e CartEvent) GetEntityID() string { return e.Payload.CartID }
func (e CartEvent) GetResourceID() string { return e.ID }

// CartEventFactory for event reconstruction
type CartEventFactory struct{}

func (f CartEventFactory) FromJSON(data []byte) (CartEvent, error) {
    var event CartEvent
    err := json.Unmarshal(data, &event)
    return event, err
}

// Convenience constructors
func NewCartEvent(cartID string, eventType EventType, customerID *string, totalPrice float64, itemCount int, details map[string]string) *CartEvent {
    payload := CartEventPayload{
        CartID:     cartID,
        CustomerID: customerID,
        TotalPrice: totalPrice,
        ItemCount:  itemCount,
        Details:    details,
    }
    
    return &CartEvent{
        ID:        uuid.New().String(),
        EventType: eventType,
        Timestamp: time.Now(),
        Payload:   payload,
    }
}

func NewCartCreatedEvent(cartID string, customerID *string) *CartEvent {
    return NewCartEvent(cartID, CartCreated, customerID, 0, 0, nil)
}

func NewCartDeletedEvent(cartID string, customerID *string) *CartEvent {
    return NewCartEvent(cartID, CartDeleted, customerID, 0, 0, nil)
}

func NewCartCheckedOutEvent(cartID string, customerID *string, totalPrice float64, itemCount int) *CartEvent {
    return NewCartEvent(cartID, CartCheckedOut, customerID, totalPrice, itemCount, nil)
}
```

---

## Repository Layer

### File: repository.go
```go
package cart

import (
    "context"
    "errors"
    
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/outbox"
)

// Domain-specific errors
var (
    ErrCartNotFound      = errors.New("cart not found")
    ErrCartItemNotFound  = errors.New("cart item not found")
    ErrAddressNotFound   = errors.New("address not found")
    ErrContactNotFound   = errors.New("contact not found")
    ErrCreditCardNotFound = errors.New("credit card not found")
    ErrInvalidUUID       = errors.New("invalid UUID format")
    ErrDatabaseOperation = errors.New("database operation failed")
    ErrTransactionFailed = errors.New("transaction failed")
    ErrDuplicateActiveCart = errors.New("customer already has an active cart")
)

// CartRepository defines the contract for cart data access
type CartRepository interface {
    // Cart CRUD
    CreateCart(ctx context.Context, cart *Cart) error
    GetCartByID(ctx context.Context, cartID string) (*Cart, error)
    UpdateCart(ctx context.Context, cart *Cart) error
    DeleteCart(ctx context.Context, cartID string) error
    GetActiveCartByCustomerID(ctx context.Context, customerID string) (*Cart, error)
    
    // Cart Items
    AddItem(ctx context.Context, cartID string, item *CartItem) error
    UpdateItemQuantity(ctx context.Context, cartID string, lineNumber string, quantity int) error
    RemoveItem(ctx context.Context, cartID string, lineNumber string) error
    GetCartItems(ctx context.Context, cartID string) ([]CartItem, error)
    
    // Contact & Addresses
    SetContact(ctx context.Context, cartID string, contact *Contact) error
    GetContact(ctx context.Context, cartID string) (*Contact, error)
    AddAddress(ctx context.Context, cartID string, address *Address) error
    GetAddresses(ctx context.Context, cartID string) ([]Address, error)
    UpdateAddress(ctx context.Context, addressID int64, address *Address) error
    RemoveAddress(ctx context.Context, addressID int64) error
    
    // Payment
    SetCreditCard(ctx context.Context, cartID string, card *CreditCard) error
    GetCreditCard(ctx context.Context, cartID string) (*CreditCard, error)
    RemoveCreditCard(ctx context.Context, cartID string) error
    
    // Status
    GetStatusHistory(ctx context.Context, cartID string) ([]CartStatus, error)
    AddStatusEntry(ctx context.Context, cartID string, status string) error
    
    // Checkout
    CheckoutCart(ctx context.Context, cartID string) (*Cart, error)
}

// cartRepository implements CartRepository
type cartRepository struct {
    db           database.Database
    outboxWriter *outbox.Writer
}

// NewCartRepository creates a new cart repository
func NewCartRepository(db database.Database, outbox *outbox.Writer) CartRepository {
    return &cartRepository{
        db:           db,
        outboxWriter: outbox,
    }
}

// Interface compliance check
var _ CartRepository = (*cartRepository)(nil)
```

### File: repository_crud.go
```go
package cart

import (
    "context"
    "fmt"
    "log"
    
    "github.com/google/uuid"
    
    events "go-shopping-poc/internal/contracts/events"
)

func (r *cartRepository) CreateCart(ctx context.Context, cart *Cart) error {
    log.Printf("[DEBUG] Repository: Creating new cart...")
    
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    committed := false
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()
    
    // Generate cart ID
    cart.CartID = uuid.New()
    cart.CurrentStatus = "active"
    cart.CreatedAt = time.Now()
    cart.UpdatedAt = time.Now()
    
    // Insert cart
    query := `
        INSERT INTO carts.Cart (
            cart_id, customer_id, contact_id, credit_card_id, current_status,
            currency, net_price, tax, shipping, total_price, created_at, updated_at
        ) VALUES (
            :cart_id, :customer_id, :contact_id, :credit_card_id, :current_status,
            :currency, :net_price, :tax, :shipping, :total_price, :created_at, :updated_at
        )
    `
    
    _, err = tx.NamedExecContext(ctx, query, cart)
    if err != nil {
        return fmt.Errorf("%w: failed to insert cart: %v", ErrDatabaseOperation, err)
    }
    
    // Add initial status
    if err := r.addStatusEntryTx(ctx, tx, cart.CartID.String(), "active"); err != nil {
        return err
    }
    
    // Write event to outbox
    var customerIDStr *string
    if cart.CustomerID != nil {
        id := cart.CustomerID.String()
        customerIDStr = &id
    }
    evt := events.NewCartCreatedEvent(cart.CartID.String(), customerIDStr)
    if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
        return fmt.Errorf("failed to write cart created event: %w", err)
    }
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
    }
    committed = true
    
    return nil
}

func (r *cartRepository) GetCartByID(ctx context.Context, cartID string) (*Cart, error) {
    id, err := uuid.Parse(cartID)
    if err != nil {
        return nil, fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    query := `
        SELECT cart_id, customer_id, contact_id, credit_card_id, current_status,
               currency, net_price, tax, shipping, total_price, created_at, updated_at, version
        FROM carts.Cart
        WHERE cart_id = $1
    `
    
    var cart Cart
    err = r.db.GetContext(ctx, &cart, query, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrCartNotFound
        }
        return nil, fmt.Errorf("%w: failed to get cart: %v", ErrDatabaseOperation, err)
    }
    
    // Load related data
    if err := r.loadCartRelations(ctx, &cart); err != nil {
        return nil, err
    }
    
    return &cart, nil
}

func (r *cartRepository) UpdateCart(ctx context.Context, cart *Cart) error {
    query := `
        UPDATE carts.Cart
        SET customer_id = :customer_id,
            contact_id = :contact_id,
            credit_card_id = :credit_card_id,
            current_status = :current_status,
            currency = :currency,
            net_price = :net_price,
            tax = :tax,
            shipping = :shipping,
            total_price = :total_price,
            version = version + 1
        WHERE cart_id = :cart_id
    `
    
    result, err := r.db.NamedExecContext(ctx, query, cart)
    if err != nil {
        return fmt.Errorf("%w: failed to update cart: %v", ErrDatabaseOperation, err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
    }
    if rows == 0 {
        return ErrCartNotFound
    }
    
    return nil
}

func (r *cartRepository) DeleteCart(ctx context.Context, cartID string) error {
    id, err := uuid.Parse(cartID)
    if err != nil {
        return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    committed := false
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()
    
    // Get customer ID for event
    var customerID *uuid.UUID
    query := `SELECT customer_id FROM carts.Cart WHERE cart_id = $1`
    err = tx.QueryRowContext(ctx, query, id).Scan(&customerID)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return ErrCartNotFound
        }
        return fmt.Errorf("%w: failed to get cart: %v", ErrDatabaseOperation, err)
    }
    
    // Write event before delete
    var customerIDStr *string
    if customerID != nil {
        id := customerID.String()
        customerIDStr = &id
    }
    evt := events.NewCartDeletedEvent(cartID, customerIDStr)
    if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
        return fmt.Errorf("failed to write cart deleted event: %w", err)
    }
    
    // Delete cart (cascade will handle related tables)
    query = `DELETE FROM carts.Cart WHERE cart_id = $1`
    result, err := tx.ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("%w: failed to delete cart: %v", ErrDatabaseOperation, err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
    }
    if rows == 0 {
        return ErrCartNotFound
    }
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
    }
    committed = true
    
    return nil
}

func (r *cartRepository) GetActiveCartByCustomerID(ctx context.Context, customerID string) (*Cart, error) {
    id, err := uuid.Parse(customerID)
    if err != nil {
        return nil, fmt.Errorf("%w: invalid customer ID: %v", ErrInvalidUUID, err)
    }
    
    query := `
        SELECT cart_id, customer_id, contact_id, credit_card_id, current_status,
               currency, net_price, tax, shipping, total_price, created_at, updated_at, version
        FROM carts.Cart
        WHERE customer_id = $1 AND current_status = 'active'
    `
    
    var cart Cart
    err = r.db.GetContext(ctx, &cart, query, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrCartNotFound
        }
        return nil, fmt.Errorf("%w: failed to get cart: %v", ErrDatabaseOperation, err)
    }
    
    if err := r.loadCartRelations(ctx, &cart); err != nil {
        return nil, err
    }
    
    return &cart, nil
}

func (r *cartRepository) loadCartRelations(ctx context.Context, cart *Cart) error {
    var err error
    
    // Load contact
    cart.Contact, err = r.GetContact(ctx, cart.CartID.String())
    if err != nil && !errors.Is(err, ErrContactNotFound) {
        return fmt.Errorf("failed to load contact: %w", err)
    }
    
    // Load addresses
    cart.Addresses, err = r.GetAddresses(ctx, cart.CartID.String())
    if err != nil {
        return fmt.Errorf("failed to load addresses: %w", err)
    }
    
    // Load credit card
    cart.CreditCard, err = r.GetCreditCard(ctx, cart.CartID.String())
    if err != nil && !errors.Is(err, ErrCreditCardNotFound) {
        return fmt.Errorf("failed to load credit card: %w", err)
    }
    
    // Load items
    cart.Items, err = r.GetCartItems(ctx, cart.CartID.String())
    if err != nil {
        return fmt.Errorf("failed to load items: %w", err)
    }
    
    // Load status history
    cart.StatusHistory, err = r.GetStatusHistory(ctx, cart.CartID.String())
    if err != nil {
        return fmt.Errorf("failed to load status history: %w", err)
    }
    
    return nil
}
```

### File: repository_items.go
```go
package cart

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    
    "github.com/google/uuid"
)

func (r *cartRepository) AddItem(ctx context.Context, cartID string, item *CartItem) error {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    committed := false
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()
    
    // Generate line number using sequence
    var nextLine int
    err = tx.QueryRowContext(ctx, `SELECT nextval('carts.cart_sequence')`).Scan(&nextLine)
    if err != nil {
        return fmt.Errorf("%w: failed to generate line number: %v", ErrDatabaseOperation, err)
    }
    item.LineNumber = fmt.Sprintf("%03d", nextLine)
    item.CartID = cartUUID
    item.CalculateLineTotal()
    
    // Insert item
    query := `
        INSERT INTO carts.CartItem (
            cart_id, line_number, product_id, product_name, unit_price, quantity, total_price
        ) VALUES (
            :cart_id, :line_number, :product_id, :product_name, :unit_price, :quantity, :total_price
        )
        RETURNING id
    `
    
    rows, err := tx.NamedQueryContext(ctx, query, item)
    if err != nil {
        return fmt.Errorf("%w: failed to insert item: %v", ErrDatabaseOperation, err)
    }
    defer rows.Close()
    
    if rows.Next() {
        err = rows.Scan(&item.ID)
        if err != nil {
            return fmt.Errorf("%w: failed to scan item ID: %v", ErrDatabaseOperation, err)
        }
    }
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
    }
    committed = true
    
    return nil
}

func (r *cartRepository) UpdateItemQuantity(ctx context.Context, cartID string, lineNumber string, quantity int) error {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    if quantity <= 0 {
        return errors.New("quantity must be positive")
    }
    
    // Get current item to calculate new total
    query := `
        SELECT id, unit_price FROM carts.CartItem 
        WHERE cart_id = $1 AND line_number = $2
    `
    var item CartItem
    err = r.db.GetContext(ctx, &item, query, cartUUID, lineNumber)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return ErrCartItemNotFound
        }
        return fmt.Errorf("%w: failed to get item: %v", ErrDatabaseOperation, err)
    }
    
    newTotal := float64(quantity) * item.UnitPrice
    
    query = `
        UPDATE carts.CartItem
        SET quantity = $1, total_price = $2
        WHERE cart_id = $3 AND line_number = $4
    `
    result, err := r.db.ExecContext(ctx, query, quantity, newTotal, cartUUID, lineNumber)
    if err != nil {
        return fmt.Errorf("%w: failed to update item: %v", ErrDatabaseOperation, err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
    }
    if rows == 0 {
        return ErrCartItemNotFound
    }
    
    return nil
}

func (r *cartRepository) RemoveItem(ctx context.Context, cartID string, lineNumber string) error {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    query := `DELETE FROM carts.CartItem WHERE cart_id = $1 AND line_number = $2`
    result, err := r.db.ExecContext(ctx, query, cartUUID, lineNumber)
    if err != nil {
        return fmt.Errorf("%w: failed to delete item: %v", ErrDatabaseOperation, err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
    }
    if rows == 0 {
        return ErrCartItemNotFound
    }
    
    return nil
}

func (r *cartRepository) GetCartItems(ctx context.Context, cartID string) ([]CartItem, error) {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return nil, fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    query := `
        SELECT id, cart_id, line_number, product_id, product_name, unit_price, quantity, total_price
        FROM carts.CartItem
        WHERE cart_id = $1
        ORDER BY line_number
    `
    
    var items []CartItem
    err = r.db.SelectContext(ctx, &items, query, cartUUID)
    if err != nil {
        return nil, fmt.Errorf("%w: failed to get items: %v", ErrDatabaseOperation, err)
    }
    
    return items, nil
}
```

### File: repository_contact.go
```go
package cart

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    
    "github.com/google/uuid"
)

func (r *cartRepository) SetContact(ctx context.Context, cartID string, contact *Contact) error {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    committed := false
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()
    
    // Delete existing contact if any
    query := `DELETE FROM carts.Contact WHERE cart_id = $1`
    _, _ = tx.ExecContext(ctx, query, cartUUID)
    
    // Insert new contact
    contact.CartID = cartUUID
    query = `
        INSERT INTO carts.Contact (cart_id, email, first_name, last_name, phone)
        VALUES (:cart_id, :email, :first_name, :last_name, :phone)
        RETURNING id
    `
    
    rows, err := tx.NamedQueryContext(ctx, query, contact)
    if err != nil {
        return fmt.Errorf("%w: failed to insert contact: %v", ErrDatabaseOperation, err)
    }
    defer rows.Close()
    
    if rows.Next() {
        err = rows.Scan(&contact.ID)
        if err != nil {
            return fmt.Errorf("%w: failed to scan contact ID: %v", ErrDatabaseOperation, err)
        }
    }
    
    // Update cart with contact_id
    query = `UPDATE carts.Cart SET contact_id = $1 WHERE cart_id = $2`
    _, err = tx.ExecContext(ctx, query, contact.ID, cartUUID)
    if err != nil {
        return fmt.Errorf("%w: failed to update cart contact: %v", ErrDatabaseOperation, err)
    }
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
    }
    committed = true
    
    return nil
}

func (r *cartRepository) GetContact(ctx context.Context, cartID string) (*Contact, error) {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return nil, fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    query := `
        SELECT id, cart_id, email, first_name, last_name, phone
        FROM carts.Contact
        WHERE cart_id = $1
    `
    
    var contact Contact
    err = r.db.GetContext(ctx, &contact, query, cartUUID)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrContactNotFound
        }
        return nil, fmt.Errorf("%w: failed to get contact: %v", ErrDatabaseOperation, err)
    }
    
    return &contact, nil
}

func (r *cartRepository) AddAddress(ctx context.Context, cartID string, address *Address) error {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    address.CartID = cartUUID
    query := `
        INSERT INTO carts.Address (cart_id, address_type, first_name, last_name, address_1, address_2, city, state, zip)
        VALUES (:cart_id, :address_type, :first_name, :last_name, :address_1, :address_2, :city, :state, :zip)
        RETURNING id
    `
    
    rows, err := r.db.NamedQueryContext(ctx, query, address)
    if err != nil {
        return fmt.Errorf("%w: failed to insert address: %v", ErrDatabaseOperation, err)
    }
    defer rows.Close()
    
    if rows.Next() {
        err = rows.Scan(&address.ID)
        if err != nil {
            return fmt.Errorf("%w: failed to scan address ID: %v", ErrDatabaseOperation, err)
        }
    }
    
    return nil
}

func (r *cartRepository) GetAddresses(ctx context.Context, cartID string) ([]Address, error) {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return nil, fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    query := `
        SELECT id, cart_id, address_type, first_name, last_name, address_1, address_2, city, state, zip
        FROM carts.Address
        WHERE cart_id = $1
        ORDER BY id
    `
    
    var addresses []Address
    err = r.db.SelectContext(ctx, &addresses, query, cartUUID)
    if err != nil {
        return nil, fmt.Errorf("%w: failed to get addresses: %v", ErrDatabaseOperation, err)
    }
    
    return addresses, nil
}

func (r *cartRepository) UpdateAddress(ctx context.Context, addressID int64, address *Address) error {
    address.ID = addressID
    query := `
        UPDATE carts.Address
        SET address_type = :address_type,
            first_name = :first_name,
            last_name = :last_name,
            address_1 = :address_1,
            address_2 = :address_2,
            city = :city,
            state = :state,
            zip = :zip
        WHERE id = :id
    `
    
    result, err := r.db.NamedExecContext(ctx, query, address)
    if err != nil {
        return fmt.Errorf("%w: failed to update address: %v", ErrDatabaseOperation, err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
    }
    if rows == 0 {
        return ErrAddressNotFound
    }
    
    return nil
}

func (r *cartRepository) RemoveAddress(ctx context.Context, addressID int64) error {
    query := `DELETE FROM carts.Address WHERE id = $1`
    result, err := r.db.ExecContext(ctx, query, addressID)
    if err != nil {
        return fmt.Errorf("%w: failed to delete address: %v", ErrDatabaseOperation, err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
    }
    if rows == 0 {
        return ErrAddressNotFound
    }
    
    return nil
}
```

### File: repository_payment.go
```go
package cart

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    
    "github.com/google/uuid"
)

func (r *cartRepository) SetCreditCard(ctx context.Context, cartID string, card *CreditCard) error {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    committed := false
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()
    
    // Delete existing credit card if any
    query := `DELETE FROM carts.CreditCard WHERE cart_id = $1`
    _, _ = tx.ExecContext(ctx, query, cartUUID)
    
    // Insert new credit card
    card.CartID = cartUUID
    query = `
        INSERT INTO carts.CreditCard (
            cart_id, card_type, card_number, card_holder_name, card_expires, card_cvv
        ) VALUES (
            :cart_id, :card_type, :card_number, :card_holder_name, :card_expires, :card_cvv
        )
        RETURNING id
    `
    
    rows, err := tx.NamedQueryContext(ctx, query, card)
    if err != nil {
        return fmt.Errorf("%w: failed to insert credit card: %v", ErrDatabaseOperation, err)
    }
    defer rows.Close()
    
    if rows.Next() {
        err = rows.Scan(&card.ID)
        if err != nil {
            return fmt.Errorf("%w: failed to scan credit card ID: %v", ErrDatabaseOperation, err)
        }
    }
    
    // Update cart with credit_card_id
    query = `UPDATE carts.Cart SET credit_card_id = $1 WHERE cart_id = $2`
    _, err = tx.ExecContext(ctx, query, card.ID, cartUUID)
    if err != nil {
        return fmt.Errorf("%w: failed to update cart credit card: %v", ErrDatabaseOperation, err)
    }
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
    }
    committed = true
    
    return nil
}

func (r *cartRepository) GetCreditCard(ctx context.Context, cartID string) (*CreditCard, error) {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return nil, fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    query := `
        SELECT id, cart_id, card_type, card_number, card_holder_name, card_expires, card_cvv
        FROM carts.CreditCard
        WHERE cart_id = $1
    `
    
    var card CreditCard
    err = r.db.GetContext(ctx, &card, query, cartUUID)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrCreditCardNotFound
        }
        return nil, fmt.Errorf("%w: failed to get credit card: %v", ErrDatabaseOperation, err)
    }
    
    return &card, nil
}

func (r *cartRepository) RemoveCreditCard(ctx context.Context, cartID string) error {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    committed := false
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()
    
    // Remove credit card_id from cart first
    query := `UPDATE carts.Cart SET credit_card_id = NULL WHERE cart_id = $1`
    _, err = tx.ExecContext(ctx, query, cartUUID)
    if err != nil {
        return fmt.Errorf("%w: failed to update cart: %v", ErrDatabaseOperation, err)
    }
    
    // Delete credit card
    query = `DELETE FROM carts.CreditCard WHERE cart_id = $1`
    _, err = tx.ExecContext(ctx, query, cartUUID)
    if err != nil {
        return fmt.Errorf("%w: failed to delete credit card: %v", ErrDatabaseOperation, err)
    }
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
    }
    committed = true
    
    return nil
}
```

### File: repository_checkout.go
```go
package cart

import (
    "context"
    "fmt"
    
    "github.com/google/uuid"
    
    events "go-shopping-poc/internal/contracts/events"
)

func (r *cartRepository) CheckoutCart(ctx context.Context, cartID string) (*Cart, error) {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return nil, fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    committed := false
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()
    
    // Get cart with current data
    cart, err := r.getCartByIDTx(ctx, tx, cartUUID)
    if err != nil {
        return nil, err
    }
    
    // Load relations within transaction
    if err := r.loadCartRelationsTx(ctx, tx, cart); err != nil {
        return nil, err
    }
    
    // Validate cart can checkout
    if err := cart.CanCheckout(); err != nil {
        return nil, fmt.Errorf("cart not ready for checkout: %w", err)
    }
    
    // Update status
    if err := cart.SetStatus("checked_out"); err != nil {
        return nil, err
    }
    
    // Update cart
    query := `
        UPDATE carts.Cart
        SET current_status = $1,
            net_price = $2,
            tax = $3,
            shipping = $4,
            total_price = $5,
            version = version + 1
        WHERE cart_id = $6
    `
    _, err = tx.ExecContext(ctx, query, 
        cart.CurrentStatus, cart.NetPrice, cart.Tax, 
        cart.Shipping, cart.TotalPrice, cart.CartID)
    if err != nil {
        return nil, fmt.Errorf("%w: failed to update cart: %v", ErrDatabaseOperation, err)
    }
    
    // Add status entry
    if err := r.addStatusEntryTx(ctx, tx, cartID, "checked_out"); err != nil {
        return nil, err
    }
    
    // Write checkout event
    var customerIDStr *string
    if cart.CustomerID != nil {
        id := cart.CustomerID.String()
        customerIDStr = &id
    }
    evt := events.NewCartCheckedOutEvent(cartID, customerIDStr, cart.TotalPrice, len(cart.Items))
    if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
        return nil, fmt.Errorf("failed to write checkout event: %w", err)
    }
    
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
    }
    committed = true
    
    return cart, nil
}

func (r *cartRepository) GetStatusHistory(ctx context.Context, cartID string) ([]CartStatus, error) {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return nil, fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    query := `
        SELECT id, cart_id, cart_status, status_date_time
        FROM carts.CartStatus
        WHERE cart_id = $1
        ORDER BY status_date_time DESC
    `
    
    var history []CartStatus
    err = r.db.SelectContext(ctx, &history, query, cartUUID)
    if err != nil {
        return nil, fmt.Errorf("%w: failed to get status history: %v", ErrDatabaseOperation, err)
    }
    
    return history, nil
}

func (r *cartRepository) AddStatusEntry(ctx context.Context, cartID string, status string) error {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    query := `
        INSERT INTO carts.CartStatus (cart_id, cart_status, status_date_time)
        VALUES ($1, $2, CURRENT_TIMESTAMP)
    `
    _, err = r.db.ExecContext(ctx, query, cartUUID, status)
    if err != nil {
        return fmt.Errorf("%w: failed to add status entry: %v", ErrDatabaseOperation, err)
    }
    
    return nil
}

func (r *cartRepository) addStatusEntryTx(ctx context.Context, tx database.Tx, cartID string, status string) error {
    cartUUID, err := uuid.Parse(cartID)
    if err != nil {
        return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
    }
    
    query := `
        INSERT INTO carts.CartStatus (cart_id, cart_status, status_date_time)
        VALUES ($1, $2, CURRENT_TIMESTAMP)
    `
    _, err = tx.ExecContext(ctx, query, cartUUID, status)
    if err != nil {
        return fmt.Errorf("%w: failed to add status entry: %v", ErrDatabaseOperation, err)
    }
    
    return nil
}

func (r *cartRepository) getCartByIDTx(ctx context.Context, tx database.Tx, cartID uuid.UUID) (*Cart, error) {
    query := `
        SELECT cart_id, customer_id, contact_id, credit_card_id, current_status,
               currency, net_price, tax, shipping, total_price, created_at, updated_at, version
        FROM carts.Cart
        WHERE cart_id = $1
    `
    
    var cart Cart
    err := tx.QueryRowContext(ctx, query, cartID).Scan(
        &cart.CartID, &cart.CustomerID, &cart.ContactID, &cart.CreditCardID, &cart.CurrentStatus,
        &cart.Currency, &cart.NetPrice, &cart.Tax, &cart.Shipping, &cart.TotalPrice,
        &cart.CreatedAt, &cart.UpdatedAt, &cart.Version,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrCartNotFound
        }
        return nil, fmt.Errorf("%w: failed to get cart: %v", ErrDatabaseOperation, err)
    }
    
    return &cart, nil
}

func (r *cartRepository) loadCartRelationsTx(ctx context.Context, tx database.Tx, cart *Cart) error {
    // Load items
    query := `
        SELECT id, cart_id, line_number, product_id, product_name, unit_price, quantity, total_price
        FROM carts.CartItem
        WHERE cart_id = $1
        ORDER BY line_number
    `
    err := tx.SelectContext(ctx, &cart.Items, query, cart.CartID)
    if err != nil {
        return fmt.Errorf("failed to load items: %w", err)
    }
    
    // Load contact
    if cart.ContactID != nil {
        query = `SELECT id, cart_id, email, first_name, last_name, phone FROM carts.Contact WHERE id = $1`
        cart.Contact = &Contact{}
        err = tx.GetContext(ctx, cart.Contact, query, *cart.ContactID)
        if err != nil && !errors.Is(err, sql.ErrNoRows) {
            return fmt.Errorf("failed to load contact: %w", err)
        }
    }
    
    // Load credit card
    if cart.CreditCardID != nil {
        query = `SELECT id, cart_id, card_type, card_number, card_holder_name, card_expires, card_cvv FROM carts.CreditCard WHERE id = $1`
        cart.CreditCard = &CreditCard{}
        err = tx.GetContext(ctx, cart.CreditCard, query, *cart.CreditCardID)
        if err != nil && !errors.Is(err, sql.ErrNoRows) {
            return fmt.Errorf("failed to load credit card: %w", err)
        }
    }
    
    return nil
}
```

---

## Product Service Client (internal/service/cart/product_client.go)

```go
package cart

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

// ProductClient interface for product service communication
type ProductClient interface {
    GetProduct(ctx context.Context, productID string) (*ProductInfo, error)
}

// ProductInfo represents product data from product service
type ProductInfo struct {
    ID         string  `json:"id"`
    Name       string  `json:"name"`
    FinalPrice float64 `json:"final_price"`
    Currency   string  `json:"currency"`
    InStock    bool    `json:"in_stock"`
}

// HTTPProductClient implements ProductClient using HTTP
type HTTPProductClient struct {
    baseURL    string
    httpClient *http.Client
}

// NewProductClient creates a new product client
func NewProductClient(baseURL string) ProductClient {
    return &HTTPProductClient{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 5 * time.Second,
        },
    }
}

// GetProduct fetches product information from product service
func (c *HTTPProductClient) GetProduct(ctx context.Context, productID string) (*ProductInfo, error) {
    url := fmt.Sprintf("%s/api/v1/products/%s", c.baseURL, productID)
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to call product service: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == http.StatusNotFound {
        return nil, fmt.Errorf("product not found: %s", productID)
    }
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("product service returned status %d", resp.StatusCode)
    }
    
    var product ProductInfo
    if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
        return nil, fmt.Errorf("failed to decode product response: %w", err)
    }
    
    return &product, nil
}
```

---

## Service Layer (internal/service/cart/service.go)

### Infrastructure Structure
```go
package cart

import (
    "context"
    "fmt"
    "net/http"
    
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/platform/outbox"
    "go-shopping-poc/internal/platform/service"
)

// CartInfrastructure defines infrastructure components
type CartInfrastructure struct {
    Database        database.Database
    EventBus        bus.Bus
    OutboxWriter    *outbox.Writer
    OutboxPublisher *outbox.Publisher
    ProductClient   ProductClient
    CORSHandler     func(http.Handler) http.Handler
}

func NewCartInfrastructure(
    db database.Database,
    eventBus bus.Bus,
    outboxWriter *outbox.Writer,
    outboxPublisher *outbox.Publisher,
    productClient ProductClient,
    corsHandler func(http.Handler) http.Handler,
) *CartInfrastructure {
    return &CartInfrastructure{
        Database:        db,
        EventBus:        eventBus,
        OutboxWriter:    outboxWriter,
        OutboxPublisher: outboxPublisher,
        ProductClient:   productClient,
        CORSHandler:     corsHandler,
    }
}

// CartService orchestrates cart business operations
type CartService struct {
    *service.BaseService
    repo           CartRepository
    infrastructure *CartInfrastructure
    config         *Config
}

// NewCartService creates a new cart service
func NewCartService(infrastructure *CartInfrastructure, config *Config) *CartService {
    repo := NewCartRepository(infrastructure.Database, infrastructure.OutboxWriter)
    
    return &CartService{
        BaseService:    service.NewBaseService("cart"),
        repo:           repo,
        infrastructure: infrastructure,
        config:         config,
    }
}

// NewCartServiceWithRepo creates service with custom repository (for testing)
func NewCartServiceWithRepo(repo CartRepository, infrastructure *CartInfrastructure, config *Config) *CartService {
    return &CartService{
        BaseService:    service.NewBaseService("cart"),
        repo:           repo,
        infrastructure: infrastructure,
        config:         config,
    }
}
```

### Service Methods

```go
// CreateCart creates a new cart (guest or authenticated)
func (s *CartService) CreateCart(ctx context.Context, customerID *string) (*Cart, error) {
    cart := &Cart{
        Currency:      "USD",
        CurrentStatus: "active",
    }
    
    // Parse customer ID if provided
    if customerID != nil && *customerID != "" {
        id, err := uuid.Parse(*customerID)
        if err != nil {
            return nil, fmt.Errorf("invalid customer ID: %w", err)
        }
        cart.CustomerID = &id
    }
    
    if err := s.repo.CreateCart(ctx, cart); err != nil {
        return nil, fmt.Errorf("failed to create cart: %w", err)
    }
    
    return cart, nil
}

// GetCart retrieves a cart by ID with all relations
func (s *CartService) GetCart(ctx context.Context, cartID string) (*Cart, error) {
    cart, err := s.repo.GetCartByID(ctx, cartID)
    if err != nil {
        if errors.Is(err, ErrCartNotFound) {
            return nil, err
        }
        return nil, fmt.Errorf("failed to get cart: %w", err)
    }
    return cart, nil
}

// DeleteCart removes a cart and publishes event
func (s *CartService) DeleteCart(ctx context.Context, cartID string) error {
    if err := s.repo.DeleteCart(ctx, cartID); err != nil {
        if errors.Is(err, ErrCartNotFound) {
            return err
        }
        return fmt.Errorf("failed to delete cart: %w", err)
    }
    return nil
}

// AddItem adds a product to the cart with validation
func (s *CartService) AddItem(ctx context.Context, cartID string, productID string, quantity int) (*CartItem, error) {
    if quantity <= 0 {
        return nil, errors.New("quantity must be positive")
    }
    
    // Get cart
    cart, err := s.repo.GetCartByID(ctx, cartID)
    if err != nil {
        return nil, fmt.Errorf("failed to get cart: %w", err)
    }
    
    if cart.CurrentStatus != "active" {
        return nil, errors.New("cannot add items to non-active cart")
    }
    
    // Validate product with product service
    product, err := s.infrastructure.ProductClient.GetProduct(ctx, productID)
    if err != nil {
        return nil, fmt.Errorf("failed to validate product: %w", err)
    }
    
    if !product.InStock {
        return nil, errors.New("product is out of stock")
    }
    
    // Create cart item with denormalized data
    item := &CartItem{
        ProductID:   productID,
        ProductName: product.Name,
        UnitPrice:   product.FinalPrice,
        Quantity:    quantity,
    }
    item.CalculateLineTotal()
    
    // Add item to cart
    if err := s.repo.AddItem(ctx, cartID, item); err != nil {
        return nil, fmt.Errorf("failed to add item: %w", err)
    }
    
    // Recalculate cart totals
    cart.Items = append(cart.Items, *item)
    cart.CalculateTotals()
    
    // Update cart with new totals
    if err := s.repo.UpdateCart(ctx, cart); err != nil {
        return nil, fmt.Errorf("failed to update cart totals: %w", err)
    }
    
    return item, nil
}

// UpdateItemQuantity updates item quantity and recalculates totals
func (s *CartService) UpdateItemQuantity(ctx context.Context, cartID string, lineNumber string, quantity int) error {
    if quantity <= 0 {
        return errors.New("quantity must be positive")
    }
    
    // Get cart
    cart, err := s.repo.GetCartByID(ctx, cartID)
    if err != nil {
        return fmt.Errorf("failed to get cart: %w", err)
    }
    
    if cart.CurrentStatus != "active" {
        return errors.New("cannot modify items in non-active cart")
    }
    
    // Update item quantity
    if err := s.repo.UpdateItemQuantity(ctx, cartID, lineNumber, quantity); err != nil {
        return fmt.Errorf("failed to update item: %w", err)
    }
    
    // Recalculate totals
    for i := range cart.Items {
        if cart.Items[i].LineNumber == lineNumber {
            cart.Items[i].Quantity = quantity
            break
        }
    }
    cart.CalculateTotals()
    
    // Update cart
    if err := s.repo.UpdateCart(ctx, cart); err != nil {
        return fmt.Errorf("failed to update cart totals: %w", err)
    }
    
    return nil
}

// RemoveItem removes an item and recalculates totals
func (s *CartService) RemoveItem(ctx context.Context, cartID string, lineNumber string) error {
    // Get cart
    cart, err := s.repo.GetCartByID(ctx, cartID)
    if err != nil {
        return fmt.Errorf("failed to get cart: %w", err)
    }
    
    if cart.CurrentStatus != "active" {
        return errors.New("cannot remove items from non-active cart")
    }
    
    // Remove item
    if err := s.repo.RemoveItem(ctx, cartID, lineNumber); err != nil {
        return fmt.Errorf("failed to remove item: %w", err)
    }
    
    // Recalculate totals
    var newItems []CartItem
    for _, item := range cart.Items {
        if item.LineNumber != lineNumber {
            newItems = append(newItems, item)
        }
    }
    cart.Items = newItems
    cart.CalculateTotals()
    
    // Update cart
    if err := s.repo.UpdateCart(ctx, cart); err != nil {
        return fmt.Errorf("failed to update cart totals: %w", err)
    }
    
    return nil
}

// SetContact sets contact information for the cart
func (s *CartService) SetContact(ctx context.Context, cartID string, contact *Contact) error {
    if err := contact.Validate(); err != nil {
        return fmt.Errorf("invalid contact: %w", err)
    }
    
    cart, err := s.repo.GetCartByID(ctx, cartID)
    if err != nil {
        return fmt.Errorf("failed to get cart: %w", err)
    }
    
    if cart.CurrentStatus != "active" {
        return errors.New("cannot modify contact for non-active cart")
    }
    
    if err := s.repo.SetContact(ctx, cartID, contact); err != nil {
        return fmt.Errorf("failed to set contact: %w", err)
    }
    
    return nil
}

// AddAddress adds an address to the cart
func (s *CartService) AddAddress(ctx context.Context, cartID string, address *Address) error {
    if err := address.Validate(); err != nil {
        return fmt.Errorf("invalid address: %w", err)
    }
    
    cart, err := s.repo.GetCartByID(ctx, cartID)
    if err != nil {
        return fmt.Errorf("failed to get cart: %w", err)
    }
    
    if cart.CurrentStatus != "active" {
        return errors.New("cannot add address to non-active cart")
    }
    
    if err := s.repo.AddAddress(ctx, cartID, address); err != nil {
        return fmt.Errorf("failed to add address: %w", err)
    }
    
    return nil
}

// SetCreditCard sets payment method for the cart
func (s *CartService) SetCreditCard(ctx context.Context, cartID string, card *CreditCard) error {
    if err := card.Validate(); err != nil {
        return fmt.Errorf("invalid credit card: %w", err)
    }
    
    cart, err := s.repo.GetCartByID(ctx, cartID)
    if err != nil {
        return fmt.Errorf("failed to get cart: %w", err)
    }
    
    if cart.CurrentStatus != "active" {
        return errors.New("cannot modify payment for non-active cart")
    }
    
    if err := s.repo.SetCreditCard(ctx, cartID, card); err != nil {
        return fmt.Errorf("failed to set credit card: %w", err)
    }
    
    return nil
}

// Checkout validates cart and initiates checkout
func (s *CartService) Checkout(ctx context.Context, cartID string) (*Cart, error) {
    // Get fresh cart data
    cart, err := s.repo.GetCartByID(ctx, cartID)
    if err != nil {
        return nil, fmt.Errorf("failed to get cart: %w", err)
    }
    
    // Final calculation before checkout
    cart.CalculateTotals()
    
    // Update cart with final totals
    if err := s.repo.UpdateCart(ctx, cart); err != nil {
        return nil, fmt.Errorf("failed to update cart totals: %w", err)
    }
    
    // Perform checkout
    checkedOutCart, err := s.repo.CheckoutCart(ctx, cartID)
    if err != nil {
        return nil, fmt.Errorf("checkout failed: %w", err)
    }
    
    return checkedOutCart, nil
}

// CalculateTax placeholder for future tax service integration
func (s *CartService) CalculateTax(ctx context.Context, netPrice float64) (float64, error) {
    // Currently: 3% tax rate
    // Future: call external tax service
    return netPrice * 0.03, nil
}

// CalculateShipping placeholder for future shipping service integration
func (s *CartService) CalculateShipping(ctx context.Context, cart *Cart) (float64, error) {
    // Currently: $0 shipping
    // Future: call external shipping service
    return 0.0, nil
}
```

---

## HTTP Handler (internal/service/cart/handler.go)

### Request/Response Types
```go
package cart

import (
    "encoding/json"
    "net/http"
    
    "github.com/go-chi/chi/v5"
    "go-shopping-poc/internal/platform/errors"
)

// Request types
type CreateCartRequest struct {
    CustomerID *string `json:"customer_id,omitempty"`
}

type AddItemRequest struct {
    ProductID string `json:"product_id"`
    Quantity  int    `json:"quantity"`
}

type UpdateItemRequest struct {
    Quantity int `json:"quantity"`
}

type SetContactRequest struct {
    Email     string `json:"email"`
    FirstName string `json:"first_name"`
    LastName  string `json:"last_name"`
    Phone     string `json:"phone"`
}

type AddAddressRequest struct {
    AddressType string `json:"address_type"`
    FirstName   string `json:"first_name"`
    LastName    string `json:"last_name"`
    Address1    string `json:"address_1"`
    Address2    string `json:"address_2,omitempty"`
    City        string `json:"city"`
    State       string `json:"state"`
    Zip         string `json:"zip"`
}

type SetPaymentRequest struct {
    CardType       string `json:"card_type"`
    CardNumber     string `json:"card_number"`
    CardHolderName string `json:"card_holder_name"`
    CardExpires    string `json:"card_expires"`
    CardCVV        string `json:"card_cvv"`
}
```

### Handler Implementation
```go
// CartHandler handles HTTP requests for cart operations
type CartHandler struct {
    service *CartService
}

func NewCartHandler(service *CartService) *CartHandler {
    return &CartHandler{service: service}
}

// CreateCart - POST /api/v1/carts
func (h *CartHandler) CreateCart(w http.ResponseWriter, r *http.Request) {
    var req CreateCartRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON")
        return
    }
    
    cart, err := h.service.CreateCart(r.Context(), req.CustomerID)
    if err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to create cart")
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(cart)
}

// GetCart - GET /api/v1/carts/{id}
func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
    cartID := chi.URLParam(r, "id")
    if cartID == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
        return
    }
    
    cart, err := h.service.GetCart(r.Context(), cartID)
    if err != nil {
        if errors.Is(err, ErrCartNotFound) {
            errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Cart not found")
            return
        }
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to get cart")
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(cart)
}

// DeleteCart - DELETE /api/v1/carts/{id}
func (h *CartHandler) DeleteCart(w http.ResponseWriter, r *http.Request) {
    cartID := chi.URLParam(r, "id")
    if cartID == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
        return
    }
    
    if err := h.service.DeleteCart(r.Context(), cartID); err != nil {
        if errors.Is(err, ErrCartNotFound) {
            errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Cart not found")
            return
        }
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to delete cart")
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}

// AddItem - POST /api/v1/carts/{id}/items
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
    cartID := chi.URLParam(r, "id")
    if cartID == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
        return
    }
    
    var req AddItemRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON")
        return
    }
    
    if req.ProductID == "" || req.Quantity <= 0 {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Invalid product_id or quantity")
        return
    }
    
    item, err := h.service.AddItem(r.Context(), cartID, req.ProductID, req.Quantity)
    if err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to add item")
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(item)
}

// UpdateItem - PUT /api/v1/carts/{id}/items/{line}
func (h *CartHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
    cartID := chi.URLParam(r, "id")
    lineNumber := chi.URLParam(r, "line")
    
    if cartID == "" || lineNumber == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID or line number")
        return
    }
    
    var req UpdateItemRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON")
        return
    }
    
    if req.Quantity <= 0 {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Quantity must be positive")
        return
    }
    
    if err := h.service.UpdateItemQuantity(r.Context(), cartID, lineNumber, req.Quantity); err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to update item")
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}

// RemoveItem - DELETE /api/v1/carts/{id}/items/{line}
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
    cartID := chi.URLParam(r, "id")
    lineNumber := chi.URLParam(r, "line")
    
    if cartID == "" || lineNumber == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID or line number")
        return
    }
    
    if err := h.service.RemoveItem(r.Context(), cartID, lineNumber); err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to remove item")
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}

// SetContact - PUT /api/v1/carts/{id}/contact
func (h *CartHandler) SetContact(w http.ResponseWriter, r *http.Request) {
    cartID := chi.URLParam(r, "id")
    if cartID == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
        return
    }
    
    var req SetContactRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON")
        return
    }
    
    contact := &Contact{
        Email:     req.Email,
        FirstName: req.FirstName,
        LastName:  req.LastName,
        Phone:     req.Phone,
    }
    
    if err := h.service.SetContact(r.Context(), cartID, contact); err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to set contact")
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}

// AddAddress - POST /api/v1/carts/{id}/addresses
func (h *CartHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
    cartID := chi.URLParam(r, "id")
    if cartID == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
        return
    }
    
    var req AddAddressRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON")
        return
    }
    
    address := &Address{
        AddressType: req.AddressType,
        FirstName:   req.FirstName,
        LastName:    req.LastName,
        Address1:    req.Address1,
        Address2:    req.Address2,
        City:        req.City,
        State:       req.State,
        Zip:         req.Zip,
    }
    
    if err := h.service.AddAddress(r.Context(), cartID, address); err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to add address")
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(address)
}

// SetPayment - PUT /api/v1/carts/{id}/payment
func (h *CartHandler) SetPayment(w http.ResponseWriter, r *http.Request) {
    cartID := chi.URLParam(r, "id")
    if cartID == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
        return
    }
    
    var req SetPaymentRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Invalid JSON")
        return
    }
    
    card := &CreditCard{
        CardType:       req.CardType,
        CardNumber:     req.CardNumber,
        CardHolderName: req.CardHolderName,
        CardExpires:    req.CardExpires,
        CardCVV:        req.CardCVV,
    }
    
    if err := h.service.SetCreditCard(r.Context(), cartID, card); err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to set payment")
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}

// Checkout - POST /api/v1/carts/{id}/checkout
func (h *CartHandler) Checkout(w http.ResponseWriter, r *http.Request) {
    cartID := chi.URLParam(r, "id")
    if cartID == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing cart ID")
        return
    }
    
    cart, err := h.service.Checkout(r.Context(), cartID)
    if err != nil {
        if err.Error() == "cart not ready for checkout: cart must be active to checkout" {
            errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, err.Error())
            return
        }
        if err.Error() == "cart not ready for checkout: cart must have at least one item" {
            errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Cart is empty")
            return
        }
        if err.Error() == "cart not ready for checkout: contact information required" {
            errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Contact information required")
            return
        }
        if err.Error() == "cart not ready for checkout: payment method required" {
            errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, "Payment method required")
            return
        }
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Checkout failed")
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(cart)
}
```

---

## Configuration (internal/service/cart/config.go)

```go
package cart

import (
    "errors"
    
    "go-shopping-poc/internal/platform/config"
)

// Config defines cart service configuration
type Config struct {
    DatabaseURL       string `mapstructure:"db_url" validate:"required"`
    ServicePort       string `mapstructure:"cart_service_port" validate:"required"`
    WriteTopic        string `mapstructure:"cart_write_topic" validate:"required"`
    ProductServiceURL string `mapstructure:"product_service_url"` // "http://product-svc.shopping.svc.cluster.local:80"
    Group             string `mapstructure:"cart_group"`
}

// LoadConfig loads cart service configuration
func LoadConfig() (*Config, error) {
    return config.LoadConfig[Config]("cart")
}

// Validate performs cart service specific validation
func (c *Config) Validate() error {
    if c.DatabaseURL == "" {
        return errors.New("database URL is required")
    }
    if c.ServicePort == "" {
        return errors.New("service port is required")
    }
    if c.WriteTopic == "" {
        return errors.New("write topic is required")
    }
    return nil
}
```

---

## Main Entry Point (cmd/cart/main.go)

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "go-shopping-poc/internal/platform/cors"
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/event"
    "go-shopping-poc/internal/platform/outbox/providers"
    "go-shopping-poc/internal/service/cart"
    
    "github.com/go-chi/chi/v5"
)

func main() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("[ERROR] Panic recovered in cart service: %v", r)
        }
    }()
    
    log.SetFlags(log.LstdFlags)
    log.Printf("[INFO] Cart: Cart service started...")
    
    // Load configuration
    cfg, err := cart.LoadConfig()
    if err != nil {
        log.Fatalf("Cart: Failed to load config: %v", err)
    }
    
    log.Printf("[DEBUG] Cart: Configuration loaded successfully")
    
    // Get database URL
    dbURL := cfg.DatabaseURL
    if dbURL == "" {
        log.Fatalf("Cart: Database URL is required")
    }
    
    // Create database provider
    log.Printf("[DEBUG] Cart: Creating database provider")
    dbProvider, err := database.NewDatabaseProvider(dbURL)
    if err != nil {
        log.Fatalf("Cart: Failed to create database provider: %v", err)
    }
    db := dbProvider.GetDatabase()
    defer func() {
        if err := db.Close(); err != nil {
            log.Printf("[ERROR] Cart: Error closing database: %v", err)
        }
    }()
    
    // Create event bus provider
    log.Printf("[DEBUG] Cart: Creating event bus provider")
    eventBusConfig := event.EventBusConfig{
        WriteTopic: cfg.WriteTopic,
        GroupID:    cfg.Group,
    }
    eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
    if err != nil {
        log.Fatalf("Cart: Failed to create event bus provider: %v", err)
    }
    eventBus := eventBusProvider.GetEventBus()
    
    // Create outbox providers
    log.Printf("[DEBUG] Cart: Creating outbox providers")
    writerProvider := providers.NewWriterProvider(db)
    publisherProvider := providers.NewPublisherProvider(db, eventBus)
    if publisherProvider == nil {
        log.Fatalf("Cart: Failed to create publisher provider")
    }
    outboxPublisher := publisherProvider.GetPublisher()
    outboxPublisher.Start()
    defer outboxPublisher.Stop()
    outboxWriter := writerProvider.GetWriter()
    
    // Create CORS provider
    log.Printf("[DEBUG] Cart: Creating CORS provider")
    corsProvider, err := cors.NewCORSProvider()
    if err != nil {
        log.Fatalf("Cart: Failed to create CORS provider: %v", err)
    }
    corsHandler := corsProvider.GetCORSHandler()
    
    // Create product service client
    productServiceURL := cfg.ProductServiceURL
    if productServiceURL == "" {
        // Default to Kubernetes service URL
        productServiceURL = "http://product-svc.shopping.svc.cluster.local:80"
    }
    log.Printf("[DEBUG] Cart: Product service URL: %s", productServiceURL)
    productClient := cart.NewProductClient(productServiceURL)
    
    // Create cart infrastructure
    log.Printf("[DEBUG] Cart: Creating cart infrastructure")
    infrastructure := cart.NewCartInfrastructure(
        db, eventBus, outboxWriter, outboxPublisher, productClient, corsHandler,
    )
    
    // Create cart service
    log.Printf("[DEBUG] Cart: Creating cart service")
    service := cart.NewCartService(infrastructure, cfg)
    
    // Create cart handler
    log.Printf("[DEBUG] Cart: Creating cart handler")
    handler := cart.NewCartHandler(service)
    
    // Setup router
    log.Printf("[DEBUG] Cart: Setting up HTTP router")
    router := chi.NewRouter()
    router.Use(corsHandler)
    
    // Health check
    router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    })
    
    // Cart routes
    cartRouter := chi.NewRouter()
    cartRouter.Post("/carts", handler.CreateCart)
    cartRouter.Get("/carts/{id}", handler.GetCart)
    cartRouter.Delete("/carts/{id}", handler.DeleteCart)
    
    // Item routes
    cartRouter.Post("/carts/{id}/items", handler.AddItem)
    cartRouter.Put("/carts/{id}/items/{line}", handler.UpdateItem)
    cartRouter.Delete("/carts/{id}/items/{line}", handler.RemoveItem)
    
    // Contact route
    cartRouter.Put("/carts/{id}/contact", handler.SetContact)
    
    // Address routes
    cartRouter.Post("/carts/{id}/addresses", handler.AddAddress)
    
    // Payment route
    cartRouter.Put("/carts/{id}/payment", handler.SetPayment)
    
    // Checkout route
    cartRouter.Post("/carts/{id}/checkout", handler.Checkout)
    
    router.Mount("/api/v1", cartRouter)
    
    // Start server
    serverAddr := "0.0.0.0" + cfg.ServicePort
    server := &http.Server{
        Addr:         serverAddr,
        Handler:      router,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }
    
    done := make(chan bool, 1)
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        log.Printf("[INFO] Cart: Starting HTTP server on %s", serverAddr)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Cart: Failed to start HTTP server: %v", err)
        }
    }()
    
    <-quit
    log.Printf("[INFO] Cart: Shutting down server...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("Cart: Server forced to shutdown: %v", err)
    }
    
    close(done)
    log.Printf("[INFO] Cart: Server exited")
}
```

---

## Event Handler (internal/service/eventreader/eventhandlers/on_order_created.go)

```go
package eventhandlers

import (
    "context"
    "encoding/json"
    "log"
    
    "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/handler"
)

// OrderEventPayload represents order event data
type OrderEventPayload struct {
    OrderID    string  `json:"order_id"`
    CartID     string  `json:"cart_id"`
    CustomerID *string `json:"customer_id,omitempty"`
    Total      float64 `json:"total"`
}

// OrderEvent represents an order event
type OrderEvent struct {
    ID        string          `json:"id"`
    EventType string          `json:"type"`
    Timestamp string          `json:"timestamp"`
    Payload   OrderEventPayload `json:"payload"`
}

// OnOrderCreated handles order.created events
type OnOrderCreated struct {
    cartRepo interface{} // Replace with actual cart repository interface
}

func NewOnOrderCreated() *OnOrderCreated {
    return &OnOrderCreated{}
}

func (h *OnOrderCreated) Handle(ctx context.Context, event events.Event) error {
    // Extract order data from event
    var orderEvent OrderEvent
    
    switch e := event.(type) {
    case OrderEvent:
        orderEvent = e
    case *OrderEvent:
        orderEvent = *e
    default:
        log.Printf("[ERROR] Expected OrderEvent, got %T", event)
        return nil
    }
    
    // Only process order.created events
    if orderEvent.EventType != "order.created" {
        log.Printf("[DEBUG] Ignoring event type: %s", orderEvent.EventType)
        return nil
    }
    
    log.Printf("[INFO] Processing order created: %s for cart: %s", 
        orderEvent.Payload.OrderID, orderEvent.Payload.CartID)
    
    // Update cart status from checked_out to completed
    // This would call the cart repository to update the status
    // Implementation depends on how eventreader service is structured
    
    return h.updateCartStatus(ctx, orderEvent.Payload.CartID)
}

func (h *OnOrderCreated) updateCartStatus(ctx context.Context, cartID string) error {
    // TODO: Implement cart status update
    // This requires access to the cart repository
    // The cart service should update status: checked_out -> completed
    log.Printf("[INFO] Updating cart %s status to completed", cartID)
    return nil
}

// CreateFactory implements HandlerFactory interface
func (h *OnOrderCreated) CreateFactory() events.EventFactory[OrderEvent] {
    return OrderEventFactory{}
}

// OrderEventFactory creates OrderEvent from JSON
type OrderEventFactory struct{}

func (f OrderEventFactory) FromJSON(data []byte) (OrderEvent, error) {
    var event OrderEvent
    err := json.Unmarshal(data, &event)
    return event, err
}

// Interface compliance
var _ handler.EventHandler = (*OnOrderCreated)(nil)
```

---

## Kubernetes Deployment (deploy/k8s/service/cart/cart-deploy.yaml)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: cart-svc
  namespace: shopping
  labels:
    app.kubernetes.io/name: cart-svc
    app.kubernetes.io/version: v1.0
spec:
  type: ClusterIP
  selector:
    app.kubernetes.io/name: cart
    app.kubernetes.io/version: v1.0
  ports:
    - name: http
      port: 80
      targetPort: 8082
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: cart
  namespace: shopping
  labels:
    app.kubernetes.io/name: cart
    app.kubernetes.io/version: v1.0
spec:
  serviceName: cart-svc
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: cart
      app.kubernetes.io/version: v1.0
  template:
    metadata:
      labels:
        app.kubernetes.io/name: cart
        app.kubernetes.io/version: v1.0
    spec:
      containers:
        - name: cart
          image: localhost:5000/go-shopping-poc/cart:1.0
          imagePullPolicy: Always
          env:
            - name: LOG_LEVEL
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: LOG_LEVEL
            - name: DATABASE_MAX_OPEN_CONNS
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: DATABASE_MAX_OPEN_CONNS
            - name: DATABASE_MAX_IDLE_CONNS
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: DATABASE_MAX_IDLE_CONNS
            - name: DATABASE_CONN_MAX_LIFETIME
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: DATABASE_CONN_MAX_LIFETIME
            - name: DATABASE_CONN_MAX_IDLE_TIME
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: DATABASE_CONN_MAX_IDLE_TIME
            - name: DATABASE_CONNECT_TIMEOUT
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: DATABASE_CONNECT_TIMEOUT
            - name: DATABASE_QUERY_TIMEOUT
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: DATABASE_QUERY_TIMEOUT
            - name: DATABASE_HEALTH_CHECK_INTERVAL
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: DATABASE_HEALTH_CHECK_INTERVAL
            - name: DATABASE_HEALTH_CHECK_TIMEOUT
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: DATABASE_HEALTH_CHECK_TIMEOUT
            - name: KAFKA_BROKERS
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: KAFKA_BROKERS
            - name: OUTBOX_BATCH_SIZE
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: OUTBOX_BATCH_SIZE
            - name: OUTBOX_PROCESS_INTERVAL
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: OUTBOX_PROCESS_INTERVAL
            - name: OUTBOX_OPERATION_TIMEOUT
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: OUTBOX_OPERATION_TIMEOUT
            - name: OUTBOX_MAX_RETRIES
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: OUTBOX_MAX_RETRIES
            - name: CORS_ALLOWED_ORIGINS
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: CORS_ALLOWED_ORIGINS
            - name: CORS_ALLOWED_METHODS
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: CORS_ALLOWED_METHODS
            - name: CORS_ALLOWED_HEADERS
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: CORS_ALLOWED_HEADERS
            - name: CORS_ALLOW_CREDENTIALS
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: CORS_ALLOW_CREDENTIALS
            - name: CORS_MAX_AGE
              valueFrom:
                configMapKeyRef:
                  name: platform-config
                  key: CORS_MAX_AGE
            - name: CART_SERVICE_PORT
              valueFrom:
                configMapKeyRef:
                  name: cart-config
                  key: CART_SERVICE_PORT
            - name: CART_WRITE_TOPIC
              valueFrom:
                configMapKeyRef:
                  name: cart-config
                  key: CART_WRITE_TOPIC
            - name: CART_GROUP
              valueFrom:
                configMapKeyRef:
                  name: cart-config
                  key: CART_GROUP
            - name: PRODUCT_SERVICE_URL
              valueFrom:
                configMapKeyRef:
                  name: cart-config
                  key: PRODUCT_SERVICE_URL
            - name: DB_URL
              valueFrom:
                secretKeyRef:
                  name: cart-db-secret
                  key: DB_URL
          ports:
            - containerPort: 8082
          readinessProbe:
            httpGet:
              path: /health
              port: 8082
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cart-ingress
  namespace: shopping
  labels:
    app.kubernetes.io/name: cart-ingress
    app.kubernetes.io/version: v1.0
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: "websecure,web"
    traefik.ingress.kubernetes.io/enable-rules: "true"
    nginx.ingress.kubernetes.io/enable-cors: "true"
spec:
  rules:
  - host: pocstore.local
    http:
      paths:
        - path: /api/v1/carts
          pathType: Prefix
          backend:
            service:
              name: cart-svc
              port:
                number: 80
  tls:
    - hosts:
      - pocstore.local
      secretName: pocstorelocal-tls
```

---

## Testing Strategy

### Unit Tests (internal/service/cart/)

**entity_test.go:**
- Test Cart.CalculateTotals()
- Test Cart.CanCheckout() with various states
- Test Cart.SetStatus() with valid/invalid transitions
- Test validation methods for Contact, Address, CreditCard

**service_test.go:**
- Mock CartRepository
- Test CreateCart with/without customer ID
- Test AddItem with product service mocking
- Test Checkout flow
- Test error handling

**handler_test.go:**
- Test HTTP endpoints with mock service
- Test request validation
- Test error responses

### Integration Tests
- Database operations with testcontainer
- Product service client with HTTP mocking
- Event publishing verification

---

## Implementation Checklist

### Phase 1: Foundation
- [ ] Update `001_init.sql` with improved schema
- [ ] Create `entity.go` with all domain models
- [ ] Create `internal/contracts/events/cart.go`

### Phase 2: Data Layer
- [ ] Create `repository.go` with interface
- [ ] Create `repository_crud.go`
- [ ] Create `repository_items.go`
- [ ] Create `repository_contact.go`
- [ ] Create `repository_payment.go`
- [ ] Create `repository_checkout.go`

### Phase 3: Service Layer
- [ ] Create `product_client.go`
- [ ] Create `service.go` with infrastructure
- [ ] Implement all service methods

### Phase 4: API Layer
- [ ] Create `handler.go` with all endpoints
- [ ] Create `config.go`

### Phase 5: Entry Point
- [ ] Create `cmd/cart/main.go`

### Phase 6: Event Handling
- [ ] Create `on_order_created.go`

### Phase 7: Deployment
- [ ] Create `cart-deploy.yaml`
- [ ] Create ConfigMap: `cart-config`
- [ ] Create Secret: `cart-db-secret`

### Phase 8: Testing
- [ ] Create unit tests
- [ ] Run integration tests

---

## Future Enhancements

1. **Tax Service Integration**: Replace 3% hardcoded tax with external tax calculation service
2. **Shipping Service Integration**: Replace $0 shipping with real shipping cost calculation
3. **Cart Expiration**: Add automatic cleanup of abandoned carts after X days
4. **Optimistic Locking**: Handle concurrent cart modifications using version field
5. **Inventory Check**: Verify product availability before checkout
6. **Promotions/Discounts**: Add coupon code and discount logic
7. **Payment Integration**: Integrate with real payment gateway instead of storing cards

---

## Dependencies

**Internal Dependencies:**
- `internal/platform/database`
- `internal/platform/event/bus`
- `internal/platform/outbox`
- `internal/platform/cors`
- `internal/platform/errors`
- `internal/platform/service`
- `internal/contracts/events`

**External Dependencies:**
- `github.com/google/uuid`
- `github.com/go-chi/chi/v5`
- `github.com/jmoiron/sqlx` (via platform/database)

---

**End of Implementation Plan**
