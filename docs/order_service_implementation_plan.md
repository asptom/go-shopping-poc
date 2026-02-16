# Order Service Implementation Plan

## Acknowledgment

This plan follows the patterns and conventions established in:
- **AGENTS.md**: Requires retrieval-led reasoning and adherence to .go-docs/ directory
- **.go-docs/00-LLMrules.md**: Emphasizes simplicity, surgical changes, and goal-driven execution
- **Customer and Cart services**: Provide excellent examples for service implementation
- **EventReader service**: Provides the pattern for event consumption

## Critical DDL Evaluation

### Issues with Current `001-init.sql`:

1. **Missing customer_id in OrderHead**: Cannot query orders by customer without denormalizing or joining through cart
2. **Missing order_number field**: Referenced in OrderEvent but not stored in database
3. **Missing timestamps**: OrderHead lacks `created_at` and `updated_at`
4. **No currency field**: Orders should store currency for historical accuracy
5. **Inefficient status tracking**: OrderStatus table lacks `is_current` flag for current status lookup
6. **Missing indexes**: No indexes on frequently queried fields (order_id, customer_id, cart_id)
7. **line_number not constrained**: Should be NOT NULL and unique per order
8. **1:1 relationships are limiting**: Contact and CreditCard currently 1:1, but order history might need multiple

### Proposed DDL Changes:

```sql
-- Orders Schema (REVISED)

DROP SCHEMA IF EXISTS orders CASCADE;
CREATE SCHEMA orders;

SET search_path TO orders;

-- Order Items Table
DROP TABLE IF EXISTS orders.OrderItem;
CREATE TABLE orders.OrderItem (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id uuid not null,
    line_number int not null,  -- Changed: NOT NULL constraint
    product_id uuid not null,  -- Changed: UUID type for consistency
    product_name text not null,
    unit_price numeric(19,4),  -- Changed: 4 decimal places for precision
    quantity int not null,     -- Changed: int type for quantities
    total_price numeric(19,4), -- Changed: 4 decimal places
    item_status text default 'pending', -- Added: default status
    item_status_date_time timestamp default CURRENT_TIMESTAMP,
    UNIQUE(order_id, line_number)  -- Added: enforce unique line numbers per order
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
    card_number text not null, -- Store masked: ****-****-****-1234
    card_holder_name text not null,
    card_expires text not null,
    card_cvv text not null     -- Store masked: ***
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

-- Order Header (Main Order Table) - REVISED
DROP TABLE IF EXISTS orders.OrderHead;
CREATE TABLE orders.OrderHead (
    order_id uuid not null,
    order_number text not null unique,  -- ADDED: Human-readable order number
    cart_id uuid not null,
    customer_id uuid,                   -- ADDED: Direct customer reference
    contact_id bigint not null,
    credit_card_id bigint not null,
    currency text default 'USD' not null, -- ADDED: Currency for historical accuracy
    net_price numeric(19,4) not null,
    tax numeric(19,4) not null,
    shipping numeric(19,4) not null,
    total_price numeric(19,4) not null,
    current_status text default 'created' not null check (current_status in ('created', 'confirmed', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded')),
    created_at timestamp default CURRENT_TIMESTAMP not null,  -- ADDED
    updated_at timestamp default CURRENT_TIMESTAMP not null,  -- ADDED
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
    seq_num := nextval('order_number_seq');
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
```

## Phase 1: Infrastructure Setup

### 1.1 Update services.mk
**File**: `resources/make/services.mk`

Add order service to the configuration:

```makefile
# Add 'order' to SERVICES variable
SERVICES := customer eventreader product product-admin cart order

# Add DB build flag
SERVICE_BUILDS_DB_order := true

# Add internal directory mapping
SERVICE_INTERNAL_DIR_order := order
```

**Verification**: Run `make help` to confirm order targets appear (order-build, order-test, order-deploy, etc.)

### 1.2 Create Deployment Configuration

Create directory: `deploy/k8s/service/order/`

#### 1.2.1 ConfigMap
**File**: `deploy/k8s/service/order/order-configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: order-config
  namespace: shopping
  labels:
    app: go-shopping-poc
    component: order
data:
  ORDER_SERVICE_PORT: ":8083"
  ORDER_WRITE_TOPIC: "OrderEvents"
  ORDER_READ_TOPICS: "CartEvents"
  ORDER_GROUP: "OrderGroup"
```

**Critical**: Port 8083 to avoid conflicts with existing services (customer:8080, product:8081, cart:8082)

#### 1.2.2 Deployment
**File**: `deploy/k8s/service/order/order-deploy.yaml`

Follow exact pattern from `customer-deploy.yaml`:

- Use StatefulSet (not Deployment) for database consistency
- Include all platform config env vars (LOG_LEVEL, DATABASE_*, KAFKA_BROKERS, OUTBOX_*, CORS_*)
- Include service-specific env vars from order-config ConfigMap
- Reference `order-db-secret` for DB_URL
- Expose port 8083
- Include readiness probe on /health endpoint
- Include Ingress for `/api/v1/orders` path

**Verification**: Check against customer-deploy.yaml line-by-line to ensure consistency

#### 1.2.3 Database Jobs
Create subdirectory: `deploy/k8s/service/order/db/`

**File**: `deploy/k8s/service/order/db/order-db-create.yaml`
- Copy from `customer/db/customer-db-create.yaml`
- Change all references from "customer" to "order"

**File**: `deploy/k8s/service/order/db/order-db-migrate.yaml`
- Copy from `customer/db/customer-db-migrate.yaml`
- Change all references from "customer" to "order"

**Note**: Do NOT create init SQL ConfigMap - services.mk generates this automatically

## Phase 2: Core Service Implementation

### 2.1 Create Service Directory Structure
```
internal/service/order/
├── config.go           # Service configuration
├── entity.go           # Domain entities (Order, OrderItem, etc.)
├── repository.go       # Repository interface and implementation
├── repository_crud.go  # CRUD operations
├── service.go          # Business logic
├── handler.go          # HTTP handlers
└── migrations/
    └── 001-init.sql    # Already exists - use revised version above
```

### 2.2 Configuration
**File**: `internal/service/order/config.go`

Follow pattern from `cart/config.go`:

```go
package order

import (
    "errors"
    "go-shopping-poc/internal/platform/config"
)

type Config struct {
    DatabaseURL string   `mapstructure:"db_url" validate:"required"`
    ServicePort string   `mapstructure:"order_service_port" validate:"required"`
    WriteTopic  string   `mapstructure:"order_write_topic" validate:"required"`
    ReadTopics  []string `mapstructure:"order_read_topics"`
    Group       string   `mapstructure:"order_group"`
}

func LoadConfig() (*Config, error) {
    return config.LoadConfig[Config]("order")
}

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

### 2.3 Domain Entities
**File**: `internal/service/order/entity.go`

Define entities matching the revised DDL:

```go
package order

import (
    "errors"
    "time"
    "github.com/google/uuid"
)

// Order represents a customer order
type Order struct {
    OrderID      uuid.UUID    `json:"order_id" db:"order_id"`
    OrderNumber  string       `json:"order_number" db:"order_number"`
    CartID       uuid.UUID    `json:"cart_id" db:"cart_id"`
    CustomerID   *uuid.UUID   `json:"customer_id,omitempty" db:"customer_id"`
    ContactID    int64        `json:"-" db:"contact_id"`
    CreditCardID int64        `json:"-" db:"credit_card_id"`
    Currency     string       `json:"currency" db:"currency"`
    NetPrice     float64      `json:"net_price" db:"net_price"`
    Tax          float64      `json:"tax" db:"tax"`
    Shipping     float64      `json:"shipping" db:"shipping"`
    TotalPrice   float64      `json:"total_price" db:"total_price"`
    CurrentStatus string      `json:"current_status" db:"current_status"`
    CreatedAt    time.Time    `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time    `json:"updated_at" db:"updated_at"`
    
    Contact     *Contact      `json:"contact,omitempty"`
    CreditCard  *CreditCard   `json:"credit_card,omitempty"`
    Addresses   []Address     `json:"addresses,omitempty"`
    Items       []OrderItem   `json:"items,omitempty"`
    StatusHistory []OrderStatus `json:"status_history,omitempty"`
}

// OrderItem represents an item in an order
type OrderItem struct {
    ID             int64     `json:"id" db:"id"`
    OrderID        uuid.UUID `json:"order_id" db:"order_id"`
    LineNumber     int       `json:"line_number" db:"line_number"`
    ProductID      string    `json:"product_id" db:"product_id"`
    ProductName    string    `json:"product_name" db:"product_name"`
    UnitPrice      float64   `json:"unit_price" db:"unit_price"`
    Quantity       int       `json:"quantity" db:"quantity"`
    TotalPrice     float64   `json:"total_price" db:"total_price"`
    ItemStatus     string    `json:"item_status" db:"item_status"`
    ItemStatusDate time.Time `json:"item_status_date_time" db:"item_status_date_time"`
}

// Contact represents contact information snapshot at order time
type Contact struct {
    ID        int64     `json:"id" db:"id"`
    OrderID   uuid.UUID `json:"order_id" db:"order_id"`
    Email     string    `json:"email" db:"email"`
    FirstName string    `json:"first_name" db:"first_name"`
    LastName  string    `json:"last_name" db:"last_name"`
    Phone     string    `json:"phone" db:"phone"`
}

// Address represents a shipping or billing address
type Address struct {
    ID          int64     `json:"id" db:"id"`
    OrderID     uuid.UUID `json:"order_id" db:"order_id"`
    AddressType string    `json:"address_type" db:"address_type"`
    FirstName   string    `json:"first_name" db:"first_name"`
    LastName    string    `json:"last_name" db:"last_name"`
    Address1    string    `json:"address_1" db:"address_1"`
    Address2    string    `json:"address_2" db:"address_2"`
    City        string    `json:"city" db:"city"`
    State       string    `json:"state" db:"state"`
    Zip         string    `json:"zip" db:"zip"`
}

// CreditCard represents payment information snapshot at order time
type CreditCard struct {
    ID             int64     `json:"id" db:"id"`
    OrderID        uuid.UUID `json:"order_id" db:"order_id"`
    CardType       string    `json:"card_type" db:"card_type"`
    CardNumber     string    `json:"card_number" db:"card_number"` // Masked
    CardHolderName string    `json:"card_holder_name" db:"card_holder_name"`
    CardExpires    string    `json:"card_expires" db:"card_expires"`
    CardCVV        string    `json:"card_cvv" db:"card_cvv"` // Masked
}

// OrderStatus represents an order status history entry
type OrderStatus struct {
    ID        int64     `json:"id" db:"id"`
    OrderID   uuid.UUID `json:"order_id" db:"order_id"`
    Status    string    `json:"order_status" db:"order_status"`
    ChangedAt time.Time `json:"status_date_time" db:"status_date_time"`
    Notes     string    `json:"notes,omitempty" db:"notes"`
}

// Validate performs domain validation on Order
func (o *Order) Validate() error {
    validStatuses := []string{"created", "confirmed", "processing", "shipped", "delivered", "cancelled", "refunded"}
    if !contains(validStatuses, o.CurrentStatus) {
        return errors.New("invalid order status")
    }
    if o.Currency == "" {
        return errors.New("currency is required")
    }
    return nil
}

// CanCancel checks if order can be cancelled
func (o *Order) CanCancel() error {
    // Orders can only be cancelled if not already shipped, delivered, or cancelled
    nonCancellableStatuses := []string{"shipped", "delivered", "cancelled", "refunded"}
    if contains(nonCancellableStatuses, o.CurrentStatus) {
        return errors.New("order cannot be cancelled in current status: " + o.CurrentStatus)
    }
    return nil
}

// SetStatus updates order status with validation
func (o *Order) SetStatus(newStatus string) error {
    validTransitions := map[string][]string{
        "created":     {"confirmed", "cancelled"},
        "confirmed":   {"processing", "cancelled"},
        "processing":  {"shipped", "cancelled"},
        "shipped":     {"delivered"},
        "delivered":   {"refunded"},
        "cancelled":   {},
        "refunded":    {},
    }
    
    allowed, ok := validTransitions[o.CurrentStatus]
    if !ok {
        return errors.New("unknown current status: " + o.CurrentStatus)
    }
    
    if !contains(allowed, newStatus) {
        return errors.New("invalid status transition from " + o.CurrentStatus + " to " + newStatus)
    }
    
    o.CurrentStatus = newStatus
    o.StatusHistory = append(o.StatusHistory, OrderStatus{
        OrderID:   o.OrderID,
        Status:    newStatus,
        ChangedAt: time.Now(),
    })
    return nil
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

### 2.4 Repository Interface
**File**: `internal/service/order/repository.go`

Follow pattern from `cart/repository.go`:

```go
package order

import (
    "context"
    "errors"
    
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/outbox"
)

var (
    ErrOrderNotFound       = errors.New("order not found")
    ErrOrderItemNotFound   = errors.New("order item not found")
    ErrInvalidUUID         = errors.New("invalid UUID format")
    ErrDatabaseOperation   = errors.New("database operation failed")
    ErrTransactionFailed   = errors.New("transaction failed")
    ErrInvalidStatusTransition = errors.New("invalid order status transition")
    ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled in current status")
)

type OrderRepository interface {
    // Order CRUD operations
    CreateOrder(ctx context.Context, order *Order) error
    GetOrderByID(ctx context.Context, orderID string) (*Order, error)
    GetOrdersByCustomerID(ctx context.Context, customerID string) ([]Order, error)
    UpdateOrderStatus(ctx context.Context, orderID string, newStatus string) error
    
    // Status history
    GetStatusHistory(ctx context.Context, orderID string) ([]OrderStatus, error)
    AddStatusEntry(ctx context.Context, orderID string, status string, notes string) error
}

type orderRepository struct {
    db           database.Database
    outboxWriter *outbox.Writer
}

func NewOrderRepository(db database.Database, outbox *outbox.Writer) OrderRepository {
    return &orderRepository{
        db:           db,
        outboxWriter: outbox,
    }
}

var _ OrderRepository = (*orderRepository)(nil)
```

### 2.5 Repository Implementation
**File**: `internal/service/order/repository_crud.go`

Implement CRUD operations following patterns from `customer/repository_crud.go`:

Key methods to implement:
1. `CreateOrder` - Insert order with all related entities in transaction, write outbox event
2. `GetOrderByID` - Get order with all relations (contact, addresses, items, credit card)
3. `GetOrdersByCustomerID` - List orders for customer (with pagination support)
4. `UpdateOrderStatus` - Update status with validation and history tracking

**Critical Pattern**: Follow the transaction pattern from repository-pattern.md:
- Begin transaction
- Set deferred rollback with `committed` flag
- Perform all operations within transaction
- Write outbox event within same transaction
- Commit transaction

### 2.6 Service Layer
**File**: `internal/service/order/service.go`

Follow pattern from `cart/service.go`:

```go
package order

import (
    "context"
    "fmt"
    "net/http"
    
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/platform/outbox"
    "go-shopping-poc/internal/platform/service"
)

type OrderInfrastructure struct {
    Database        database.Database
    EventBus        bus.Bus
    OutboxWriter    *outbox.Writer
    OutboxPublisher *outbox.Publisher
    CORSHandler     func(http.Handler) http.Handler
}

func NewOrderInfrastructure(
    db database.Database,
    eventBus bus.Bus,
    outboxWriter *outbox.Writer,
    outboxPublisher *outbox.Publisher,
    corsHandler func(http.Handler) http.Handler,
) *OrderInfrastructure {
    return &OrderInfrastructure{
        Database:        db,
        EventBus:        eventBus,
        OutboxWriter:    outboxWriter,
        OutboxPublisher: outboxPublisher,
        CORSHandler:     corsHandler,
    }
}

type OrderService struct {
    *service.EventServiceBase
    repo           OrderRepository
    infrastructure *OrderInfrastructure
    config         *Config
}

func NewOrderService(infrastructure *OrderInfrastructure, config *Config) *OrderService {
    repo := NewOrderRepository(infrastructure.Database, infrastructure.OutboxWriter)
    
    return &OrderService{
        EventServiceBase: service.NewEventServiceBase("order", infrastructure.EventBus),
        repo:             repo,
        infrastructure:   infrastructure,
        config:           config,
    }
}

// GetOrder retrieves a single order by ID
func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*Order, error) {
    return s.repo.GetOrderByID(ctx, orderID)
}

// GetOrdersByCustomer retrieves all orders for a customer
func (s *OrderService) GetOrdersByCustomer(ctx context.Context, customerID string) ([]Order, error) {
    return s.repo.GetOrdersByCustomerID(ctx, customerID)
}

// CancelOrder cancels an order if it's in a valid state
func (s *OrderService) CancelOrder(ctx context.Context, orderID string) error {
    // Get current order
    order, err := s.repo.GetOrderByID(ctx, orderID)
    if err != nil {
        return fmt.Errorf("failed to get order: %w", err)
    }
    
    // Validate cancellation is allowed
    if err := order.CanCancel(); err != nil {
        return fmt.Errorf("cannot cancel order: %w", err)
    }
    
    // Update status
    return s.repo.UpdateOrderStatus(ctx, orderID, "cancelled")
}
```

### 2.7 HTTP Handlers
**File**: `internal/service/order/handler.go`

Follow pattern from `customer/handler.go`:

```go
package order

import (
    "encoding/json"
    "net/http"
    
    "go-shopping-poc/internal/platform/errors"
    "github.com/go-chi/chi/v5"
)

type OrderHandler struct {
    service *OrderService
}

func NewOrderHandler(service *OrderService) *OrderHandler {
    return &OrderHandler{service: service}
}

// GetOrder - GET /api/v1/orders/{id}
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
    orderID := chi.URLParam(r, "id")
    if orderID == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing order ID")
        return
    }
    
    order, err := h.service.GetOrder(r.Context(), orderID)
    if err != nil {
        if errors.Is(err, ErrOrderNotFound) {
            errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Order not found")
            return
        }
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to get order")
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(order)
}

// GetOrdersByCustomer - GET /api/v1/orders?customer_id={customer_id}
func (h *OrderHandler) GetOrdersByCustomer(w http.ResponseWriter, r *http.Request) {
    customerID := r.URL.Query().Get("customer_id")
    if customerID == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing customer_id parameter")
        return
    }
    
    orders, err := h.service.GetOrdersByCustomer(r.Context(), customerID)
    if err != nil {
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to get orders")
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(orders)
}

// CancelOrder - POST /api/v1/orders/{id}/cancel
func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
    orderID := chi.URLParam(r, "id")
    if orderID == "" {
        errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeInvalidRequest, "Missing order ID")
        return
    }
    
    err := h.service.CancelOrder(r.Context(), orderID)
    if err != nil {
        if errors.Is(err, ErrOrderNotFound) {
            errors.SendError(w, http.StatusNotFound, errors.ErrorTypeNotFound, "Order not found")
            return
        }
        if errors.Is(err, ErrOrderCannotBeCancelled) {
            errors.SendError(w, http.StatusBadRequest, errors.ErrorTypeValidation, err.Error())
            return
        }
        errors.SendError(w, http.StatusInternalServerError, errors.ErrorTypeInternal, "Failed to cancel order")
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}
```

## Phase 3: Event Handling

### 3.1 Event Handler for Cart Checkout
**File**: `internal/service/order/eventhandlers/on_cart_checked_out.go`

Follow pattern from `eventreader/eventhandlers/on_customer_created.go`:

```go
package eventhandlers

import (
    "context"
    "log"
    
    events "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/platform/event/handler"
    "go-shopping-poc/internal/service/order"
)

// OnCartCheckedOut handles cart.checked_out events to create orders
type OnCartCheckedOut struct {
    service *order.OrderService
}

// NewOnCartCheckedOut creates a new cart checked out handler
func NewOnCartCheckedOut(service *order.OrderService) *OnCartCheckedOut {
    return &OnCartCheckedOut{service: service}
}

// Handle processes CartCheckedOut events
func (h *OnCartCheckedOut) Handle(ctx context.Context, event events.Event) error {
    cartEvent, ok := event.(events.CartEvent)
    if !ok {
        log.Printf("[ERROR] Order: Expected CartEvent, got %T", event)
        return nil
    }
    
    if cartEvent.EventType != events.CartCheckedOut {
        log.Printf("[DEBUG] Order: Ignoring non-CartCheckedOut event: %s", cartEvent.EventType)
        return nil
    }
    
    utils := handler.NewEventUtils()
    utils.LogEventProcessing(ctx, string(cartEvent.EventType),
        cartEvent.EventPayload.CartID,
        "")
    
    // Create order from cart data
    err := h.createOrderFromCart(ctx, cartEvent)
    
    utils.LogEventCompletion(ctx, string(cartEvent.EventType),
        cartEvent.EventPayload.CartID, err)
    
    return err
}

// createOrderFromCart creates a new order when a cart is checked out
func (h *OnCartCheckedOut) createOrderFromCart(ctx context.Context, cartEvent events.CartEvent) error {
    // Implementation will:
    // 1. Fetch cart details from cart service (or get from event payload if enriched)
    // 2. Create Order entity with cart data
    // 3. Call service.CreateOrderFromCart
    // 4. Emit order.created event via outbox
    
    cartID := cartEvent.EventPayload.CartID
    log.Printf("[INFO] Order: Creating order from cart %s", cartID)
    
    // TODO: Fetch full cart details (items, contact, addresses, payment)
    // For now, create basic order structure
    
    return nil
}

// EventType returns the event type this handler processes
func (h *OnCartCheckedOut) EventType() string {
    return string(events.CartCheckedOut)
}

// CreateFactory returns the event factory for this handler
func (h *OnCartCheckedOut) CreateFactory() events.EventFactory[events.CartEvent] {
    return events.CartEventFactory{}
}

// CreateHandler returns the handler function
func (h *OnCartCheckedOut) CreateHandler() bus.HandlerFunc[events.CartEvent] {
    return func(ctx context.Context, event events.CartEvent) error {
        return h.Handle(ctx, event)
    }
}

// Ensure OnCartCheckedOut implements the shared interfaces
var _ handler.EventHandler = (*OnCartCheckedOut)(nil)
var _ handler.HandlerFactory[events.CartEvent] = (*OnCartCheckedOut)(nil)
```

**Note**: This handler requires fetching cart details. Two approaches:
1. **Option A**: Enrich cart.checked_out event with full cart data (items, contact, etc.)
2. **Option B**: Cart service provides API to fetch cart by ID (internal communication)

**Recommended**: Option B - keep events lightweight, fetch data via service client

### 3.2 Add Service Method for Order Creation
Add to `service.go`:

```go
// CreateOrderFromCart creates a new order from a checked-out cart
func (s *OrderService) CreateOrderFromCart(ctx context.Context, cartID string, customerID *string, totalPrice float64) (*Order, error) {
    // Implementation:
    // 1. Fetch cart details (items, contact, addresses, payment)
    // 2. Create Order entity
    // 3. Call repo.CreateOrder (which writes order.created event to outbox)
    // 4. Return created order
    return nil, nil
}
```

### 3.3 Add Cart Client (Optional)
If cart data needs to be fetched, create:
**File**: `internal/service/order/cart_client.go`

Follow pattern from `cart/product_client.go`

## Phase 4: Main Application

### 4.1 Create Main Entry Point
**File**: `cmd/order/main.go`

Follow pattern from `cmd/cart/main.go`:

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
    
    "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/cors"
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/event"
    "go-shopping-poc/internal/platform/outbox/providers"
    "go-shopping-poc/internal/service/order"
    "go-shopping-poc/internal/service/order/eventhandlers"
    
    "github.com/go-chi/chi/v5"
)

func main() {
    log.SetFlags(log.LstdFlags)
    log.Printf("[INFO] Order: Order service started...")
    
    cfg, err := order.LoadConfig()
    if err != nil {
        log.Fatalf("Order: Failed to load config: %v", err)
    }
    
    // Database setup
    dbURL := cfg.DatabaseURL
    if dbURL == "" {
        log.Fatalf("Order: Database URL is required")
    }
    
    dbProvider, err := database.NewDatabaseProvider(dbURL)
    if err != nil {
        log.Fatalf("Order: Failed to create database provider: %v", err)
    }
    db := dbProvider.GetDatabase()
    defer db.Close()
    
    // Event bus setup
    eventBusConfig := event.EventBusConfig{
        WriteTopic: cfg.WriteTopic,
        GroupID:    cfg.Group,
    }
    eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
    if err != nil {
        log.Fatalf("Order: Failed to create event bus provider: %v", err)
    }
    eventBus := eventBusProvider.GetEventBus()
    
    // Outbox setup
    writerProvider := providers.NewWriterProvider(db)
    publisherProvider := providers.NewPublisherProvider(db, eventBus)
    outboxPublisher := publisherProvider.GetPublisher()
    outboxPublisher.Start()
    defer outboxPublisher.Stop()
    outboxWriter := writerProvider.GetWriter()
    
    // CORS setup
    corsProvider, err := cors.NewCORSProvider()
    if err != nil {
        log.Fatalf("Order: Failed to create CORS provider: %v", err)
    }
    corsHandler := corsProvider.GetCORSHandler()
    
    // Infrastructure and service setup
    infrastructure := order.NewOrderInfrastructure(
        db, eventBus, outboxWriter, outboxPublisher, corsHandler,
    )
    
    service := order.NewOrderService(infrastructure, cfg)
    
    // Register event handlers
    if err := registerEventHandlers(service); err != nil {
        log.Fatalf("Order: Failed to register event handlers: %v", err)
    }
    
    handler := order.NewOrderHandler(service)
    
    // Router setup
    router := chi.NewRouter()
    router.Use(corsHandler)
    
    router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    })
    
    orderRouter := chi.NewRouter()
    orderRouter.Get("/orders/{id}", handler.GetOrder)
    orderRouter.Get("/orders", handler.GetOrdersByCustomer)
    orderRouter.Post("/orders/{id}/cancel", handler.CancelOrder)
    
    router.Mount("/api/v1", orderRouter)
    
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
        log.Printf("[INFO] Order: Starting HTTP server on %s", serverAddr)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Order: Failed to start HTTP server: %v", err)
        }
    }()
    
    <-quit
    log.Printf("[INFO] Order: Shutting down server...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("Order: Server forced to shutdown: %v", err)
    }
    
    close(done)
    log.Printf("[INFO] Order: Server exited")
}

func registerEventHandlers(service *order.OrderService) error {
    log.Printf("[INFO] Order: Registering event handlers...")
    
    // Register CartCheckedOut handler
    cartCheckedOutHandler := eventhandlers.NewOnCartCheckedOut(service)
    
    if err := order.RegisterHandler(
        service,
        cartCheckedOutHandler.CreateFactory(),
        cartCheckedOutHandler.CreateHandler(),
    ); err != nil {
        return err
    }
    
    log.Printf("[INFO] Order: Successfully registered CartCheckedOut handler")
    return nil
}
```

## Phase 5: Dockerfile

**File**: `cmd/order/Dockerfile`

Copy from `cmd/customer/Dockerfile` and change:
- `cmd/customer` → `cmd/order`
- `/customer` → `/order`
- Port 8080 → 8083

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY ../../go.mod ../../go.sum ./
RUN go mod download

COPY ../../ ./

WORKDIR /app/cmd/order
RUN CGO_ENABLED=0 GOOS=linux go build -o /order

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /
COPY --from=builder /order .
COPY --from=builder /etc/passwd /etc/passwd

EXPOSE 8083

USER nobody

CMD ["/order"]
```

## Phase 6: Testing Strategy

### 6.1 Unit Tests
Create tests following patterns in `customer/service_test.go`:

**File**: `internal/service/order/service_test.go`

- Test `GetOrder` with mock repository
- Test `GetOrdersByCustomer` with mock repository  
- Test `CancelOrder` with various statuses
- Test status transitions

### 6.2 Integration Tests
Create tests following patterns in `customer/config_test.go`:

**File**: `internal/service/order/config_test.go`

- Test configuration loading
- Test validation

## Phase 7: Build and Deployment Verification

### 7.1 Build Commands
```bash
# Build the service
make order-build

# Run tests
make order-test

# Build Docker image
make order-docker

# Deploy database
make order-db-create
make order-db-migrate

# Deploy service
make order-deploy
```

### 7.2 Verification Checklist

- [ ] Database schema applied correctly
- [ ] Service starts without errors
- [ ] Health endpoint returns 200
- [ ] GET /api/v1/orders/{id} returns order
- [ ] GET /api/v1/orders?customer_id={id} returns orders
- [ ] POST /api/v1/orders/{id}/cancel cancels order
- [ ] Cart checkout creates order
- [ ] order.created event emitted

## API Endpoints Summary

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/orders/{id} | Get single order by ID |
| GET | /api/v1/orders?customer_id={id} | Get all orders for customer |
| POST | /api/v1/orders/{id}/cancel | Cancel order (if valid state) |

## Event Subscriptions

| Event | Handler | Action |
|-------|---------|--------|
| cart.checked_out | OnCartCheckedOut | Create order from cart, emit order.created |

## Event Emissions

| Event | Trigger | Payload |
|-------|---------|---------|
| order.created | Cart checkout | OrderID, OrderNumber, CartID, CustomerID, Total |
| order.cancelled | Order cancellation | OrderID, OrderNumber, CartID, CustomerID, Total |

## Key Design Decisions

1. **Order Number**: Auto-generated using PostgreSQL sequence with format `ORD-YYYY-XXXXXXXX`
2. **Status Machine**: Orders flow: created → confirmed → processing → shipped → delivered → (refunded)
3. **Cancellation**: Allowed until order enters "shipped" state
4. **Data Snapshots**: Contact, address, and payment data are snapshotted at order time (not references)
5. **Currency**: Stored per order for historical accuracy
6. **Event-Driven**: Order creation triggered by cart.checked_out event, order.created event emitted
7. **Outbox Pattern**: All events written to outbox table within transaction for reliability

## File Structure Summary

```
resources/make/services.mk                          # Add order service config
deploy/k8s/service/order/
├── order-configmap.yaml                           # Service-specific config
├── order-deploy.yaml                              # StatefulSet and Service
└── db/
    ├── order-db-create.yaml                       # DB creation job
    └── order-db-migrate.yaml                      # DB migration job
internal/service/order/
├── config.go                                      # Configuration
├── entity.go                                      # Domain entities
├── repository.go                                  # Repository interface
├── repository_crud.go                             # CRUD operations
├── service.go                                     # Business logic
├── handler.go                                     # HTTP handlers
└── eventhandlers/
    └── on_cart_checked_out.go                     # Event handler
cmd/order/
├── main.go                                        # Application entry
└── Dockerfile                                     # Container image
internal/service/order/migrations/001-init.sql     # Database schema (REVISED)
```

## Notes for Implementing LLM

1. **Follow existing patterns exactly** - Do not create new architectural patterns
2. **Use customer and cart services as primary references** - Copy-paste-adapt approach
3. **Ensure transaction safety** - All DB operations in transactions, outbox writes within transactions
4. **Handle errors properly** - Use platform errors package, return appropriate HTTP status codes
5. **Log consistently** - Use [INFO], [DEBUG], [ERROR] prefixes with service name
6. **Test thoroughly** - Write unit tests for service layer
7. **Verify build** - Run make commands to ensure everything compiles
8. **No breaking changes** - This is a new service, no backward compatibility needed
