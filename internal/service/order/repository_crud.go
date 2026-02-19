package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/database"
)

func (r *orderRepository) CreateOrder(ctx context.Context, order *Order) error {
	log.Printf("[DEBUG] Repository: Creating new order...")

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

	order.OrderID = uuid.New()
	order.CurrentStatus = "created"
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	var orderNumber string
	err = tx.QueryRow(ctx, "SELECT orders.generate_order_number()").Scan(&orderNumber)
	if err != nil {
		return fmt.Errorf("%w: failed to generate order number: %v", ErrDatabaseOperation, err)
	}
	order.OrderNumber = orderNumber

	contactID, err := r.insertContactTx(ctx, tx, order)
	if err != nil {
		return err
	}
	order.ContactID = contactID

	creditCardID, err := r.insertCreditCardTx(ctx, tx, order)
	if err != nil {
		return err
	}
	order.CreditCardID = creditCardID

	query := `
		INSERT INTO orders.OrderHead (
			order_id, order_number, cart_id, customer_id, contact_id, credit_card_id,
			currency, net_price, tax, shipping, total_price, current_status,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14
		)
	`

	_, err = tx.ExecContext(ctx, query,
		order.OrderID, order.OrderNumber, order.CartID, order.CustomerID, order.ContactID, order.CreditCardID,
		order.Currency, order.NetPrice, order.Tax, order.Shipping, order.TotalPrice, order.CurrentStatus,
		order.CreatedAt, order.UpdatedAt)
	if err != nil {
		return fmt.Errorf("%w: failed to insert order: %v", ErrDatabaseOperation, err)
	}

	for i := range order.Items {
		order.Items[i].OrderID = order.OrderID
		if err := r.insertOrderItemTx(ctx, tx, &order.Items[i]); err != nil {
			return err
		}
	}

	for i := range order.Addresses {
		order.Addresses[i].OrderID = order.OrderID
		if err := r.insertAddressTx(ctx, tx, &order.Addresses[i]); err != nil {
			return err
		}
	}

	if err := r.addStatusEntryTx(ctx, tx, order.OrderID.String(), "created", ""); err != nil {
		return err
	}

	var customerIDStr *string
	if order.CustomerID != nil {
		id := order.CustomerID.String()
		customerIDStr = &id
	}
	evt := events.NewOrderCreatedEvent(order.OrderID.String(), order.OrderNumber, order.CartID.String(), customerIDStr, order.TotalPrice)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return fmt.Errorf("failed to write order created event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

func (r *orderRepository) insertContactTx(ctx context.Context, tx database.Tx, order *Order) (int64, error) {
	if order.Contact == nil {
		return 0, errors.New("contact is required")
	}

	query := `
		INSERT INTO orders.Contact (order_id, email, first_name, last_name, phone)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var contactID int64
	err := tx.QueryRow(ctx, query,
		order.OrderID, order.Contact.Email, order.Contact.FirstName, order.Contact.LastName, order.Contact.Phone,
	).Scan(&contactID)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to insert contact: %v", ErrDatabaseOperation, err)
	}

	return contactID, nil
}

func (r *orderRepository) insertCreditCardTx(ctx context.Context, tx database.Tx, order *Order) (int64, error) {
	if order.CreditCard == nil {
		return 0, errors.New("credit card is required")
	}

	query := `
		INSERT INTO orders.CreditCard (order_id, card_type, card_number, card_holder_name, card_expires, card_cvv)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var cardID int64
	err := tx.QueryRow(ctx, query,
		order.OrderID, order.CreditCard.CardType, order.CreditCard.CardNumber,
		order.CreditCard.CardHolderName, order.CreditCard.CardExpires, order.CreditCard.CardCVV,
	).Scan(&cardID)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to insert credit card: %v", ErrDatabaseOperation, err)
	}

	return cardID, nil
}

func (r *orderRepository) insertOrderItemTx(ctx context.Context, tx database.Tx, item *OrderItem) error {
	query := `
		INSERT INTO orders.OrderItem (order_id, line_number, product_id, product_name, unit_price, quantity, total_price, item_status, item_status_date_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := tx.ExecContext(ctx, query,
		item.OrderID, item.LineNumber, item.ProductID, item.ProductName,
		item.UnitPrice, item.Quantity, item.TotalPrice, item.ItemStatus, item.ItemStatusDate,
	)
	if err != nil {
		return fmt.Errorf("%w: failed to insert order item: %v", ErrDatabaseOperation, err)
	}

	return nil
}

func (r *orderRepository) insertAddressTx(ctx context.Context, tx database.Tx, address *Address) error {
	query := `
		INSERT INTO orders.Address (order_id, address_type, first_name, last_name, address_1, address_2, city, state, zip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := tx.ExecContext(ctx, query,
		address.OrderID, address.AddressType, address.FirstName, address.LastName,
		address.Address1, address.Address2, address.City, address.State, address.Zip,
	)
	if err != nil {
		return fmt.Errorf("%w: failed to insert address: %v", ErrDatabaseOperation, err)
	}

	return nil
}

func (r *orderRepository) GetOrderByID(ctx context.Context, orderID string) (*Order, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid order ID: %v", ErrInvalidUUID, err)
	}

	query := `
		SELECT order_id, order_number, cart_id, customer_id, contact_id, credit_card_id,
		       currency, net_price, tax, shipping, total_price, current_status,
		       created_at, updated_at
		FROM orders.OrderHead
		WHERE order_id = $1
	`

	var order Order
	err = r.db.GetContext(ctx, &order, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("%w: failed to get order: %v", ErrDatabaseOperation, err)
	}

	if err := r.loadOrderRelations(ctx, &order); err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *orderRepository) GetOrdersByCustomerID(ctx context.Context, customerID string) ([]Order, error) {
	id, err := uuid.Parse(customerID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid customer ID: %v", ErrInvalidUUID, err)
	}

	query := `
		SELECT order_id, order_number, cart_id, customer_id, contact_id, credit_card_id,
		       currency, net_price, tax, shipping, total_price, current_status,
		       created_at, updated_at
		FROM orders.OrderHead
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`

	var orders []Order
	err = r.db.SelectContext(ctx, &orders, query, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get orders: %v", ErrDatabaseOperation, err)
	}

	for i := range orders {
		if err := r.loadOrderRelations(ctx, &orders[i]); err != nil {
			return nil, err
		}
	}

	return orders, nil
}

func (r *orderRepository) GetOrdersByCartID(ctx context.Context, cartID string) ([]Order, error) {
	id, err := uuid.Parse(cartID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
	}

	query := `
		SELECT order_id, order_number, cart_id, customer_id, contact_id, credit_card_id,
		       currency, net_price, tax, shipping, total_price, current_status,
		       created_at, updated_at
		FROM orders.OrderHead
		WHERE cart_id = $1
		ORDER BY created_at DESC
	`

	var orders []Order
	err = r.db.SelectContext(ctx, &orders, query, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get orders: %v", ErrDatabaseOperation, err)
	}

	for i := range orders {
		if err := r.loadOrderRelations(ctx, &orders[i]); err != nil {
			return nil, err
		}
	}

	return orders, nil
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, orderID string, newStatus string, notes string) error {
	order, err := r.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	if err := order.SetStatus(newStatus); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidStatusTransition, err)
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

	query := `
		UPDATE orders.OrderHead
		SET current_status = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE order_id = $2
	`
	_, err = tx.ExecContext(ctx, query, newStatus, order.OrderID)
	if err != nil {
		return fmt.Errorf("%w: failed to update order status: %v", ErrDatabaseOperation, err)
	}

	if err := r.addStatusEntryTx(ctx, tx, orderID, newStatus, notes); err != nil {
		return err
	}

	if newStatus == "cancelled" {
		var customerIDStr *string
		if order.CustomerID != nil {
			id := order.CustomerID.String()
			customerIDStr = &id
		}
		evt := events.NewOrderUpdatedEvent(order.OrderID.String(), order.OrderNumber, order.CartID.String(), customerIDStr, order.TotalPrice)
		if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
			return fmt.Errorf("failed to write order cancelled event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

func (r *orderRepository) loadOrderRelations(ctx context.Context, order *Order) error {
	var err error

	order.Contact, err = r.getContactByID(ctx, order.ContactID)
	if err != nil && !errors.Is(err, ErrOrderNotFound) {
		return fmt.Errorf("failed to load contact: %w", err)
	}

	order.CreditCard, err = r.getCreditCardByID(ctx, order.CreditCardID)
	if err != nil && !errors.Is(err, ErrOrderNotFound) {
		return fmt.Errorf("failed to load credit card: %w", err)
	}

	order.Addresses, err = r.getAddressesByOrderID(ctx, order.OrderID)
	if err != nil {
		return fmt.Errorf("failed to load addresses: %w", err)
	}

	order.Items, err = r.getOrderItemsByOrderID(ctx, order.OrderID)
	if err != nil {
		return fmt.Errorf("failed to load items: %w", err)
	}

	order.StatusHistory, err = r.GetStatusHistory(ctx, order.OrderID.String())
	if err != nil {
		return fmt.Errorf("failed to load status history: %w", err)
	}

	return nil
}

func (r *orderRepository) getContactByID(ctx context.Context, contactID int64) (*Contact, error) {
	query := `SELECT id, order_id, email, first_name, last_name, phone FROM orders.Contact WHERE id = $1`

	var contact Contact
	err := r.db.GetContext(ctx, &contact, query, contactID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("%w: failed to get contact: %v", ErrDatabaseOperation, err)
	}

	return &contact, nil
}

func (r *orderRepository) getCreditCardByID(ctx context.Context, cardID int64) (*CreditCard, error) {
	query := `SELECT id, order_id, card_type, card_number, card_holder_name, card_expires, card_cvv FROM orders.CreditCard WHERE id = $1`

	var card CreditCard
	err := r.db.GetContext(ctx, &card, query, cardID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("%w: failed to get credit card: %v", ErrDatabaseOperation, err)
	}

	return &card, nil
}

func (r *orderRepository) getAddressesByOrderID(ctx context.Context, orderID uuid.UUID) ([]Address, error) {
	query := `
		SELECT id, order_id, address_type, first_name, last_name, address_1, address_2, city, state, zip
		FROM orders.Address
		WHERE order_id = $1
	`

	var addresses []Address
	err := r.db.SelectContext(ctx, &addresses, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get addresses: %v", ErrDatabaseOperation, err)
	}

	return addresses, nil
}

func (r *orderRepository) getOrderItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]OrderItem, error) {
	query := `
		SELECT id, order_id, line_number, product_id, product_name, unit_price, quantity, total_price, item_status, item_status_date_time
		FROM orders.OrderItem
		WHERE order_id = $1
		ORDER BY line_number
	`

	var items []OrderItem
	err := r.db.SelectContext(ctx, &items, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get order items: %v", ErrDatabaseOperation, err)
	}

	return items, nil
}

func (r *orderRepository) GetStatusHistory(ctx context.Context, orderID string) ([]OrderStatus, error) {
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid order ID: %v", ErrInvalidUUID, err)
	}

	var history []OrderStatus
	err = r.db.SelectContext(ctx, &history, `
		SELECT id, order_id, order_status, status_date_time, notes
		FROM orders.OrderStatus
		WHERE order_id = $1
		ORDER BY status_date_time DESC
	`, orderUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get status history: %v", ErrDatabaseOperation, err)
	}

	return history, nil
}

func (r *orderRepository) AddStatusEntry(ctx context.Context, orderID string, status string, notes string) error {
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return fmt.Errorf("%w: invalid order ID: %v", ErrInvalidUUID, err)
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO orders.OrderStatus (order_id, order_status, status_date_time, notes)
		VALUES ($1, $2, CURRENT_TIMESTAMP, $3)
	`, orderUUID, status, notes)
	if err != nil {
		return fmt.Errorf("%w: failed to add status entry: %v", ErrDatabaseOperation, err)
	}

	return nil
}

func (r *orderRepository) addStatusEntryTx(ctx context.Context, tx database.Tx, orderID string, status string, notes string) error {
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return fmt.Errorf("%w: invalid order ID: %v", ErrInvalidUUID, err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO orders.OrderStatus (order_id, order_status, status_date_time, notes)
		VALUES ($1, $2, CURRENT_TIMESTAMP, $3)
	`, orderUUID, status, notes)
	if err != nil {
		return fmt.Errorf("%w: failed to add status entry: %v", ErrDatabaseOperation, err)
	}

	return nil
}
