package cart

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

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

	cart.CartID = uuid.New()
	cart.CurrentStatus = "active"
	cart.CreatedAt = time.Now()
	cart.UpdatedAt = time.Now()

	query := `
		INSERT INTO carts.Cart (
			cart_id, customer_id, contact_id, credit_card_id, current_status,
			currency, net_price, tax, shipping, total_price, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12
		)
	`

	_, err = tx.ExecContext(ctx, query,
		cart.CartID, cart.CustomerID, cart.ContactID, cart.CreditCardID, cart.CurrentStatus,
		cart.Currency, cart.NetPrice, cart.Tax, cart.Shipping, cart.TotalPrice, cart.CreatedAt, cart.UpdatedAt)
	if err != nil {
		return fmt.Errorf("%w: failed to insert cart: %v", ErrDatabaseOperation, err)
	}

	if err := r.addStatusEntryTx(ctx, tx, cart.CartID.String(), "active"); err != nil {
		return err
	}

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
		log.Printf("[ERROR] Repository: Invalid cart ID format: %v", err)
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
		log.Printf("[ERROR] Repository: Failed to get cart by ID: %v", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("%w: failed to get cart: %v", ErrDatabaseOperation, err)
	}

	if err := r.loadCartRelations(ctx, &cart); err != nil {
		log.Printf("[ERROR] Repository: Failed to load cart relations: %v", err)
		return nil, err
	}

	return &cart, nil
}

func (r *cartRepository) UpdateCart(ctx context.Context, cart *Cart) error {
	query := `
		UPDATE carts.Cart
		SET customer_id = $1,
		    contact_id = $2,
		    credit_card_id = $3,
		    current_status = $4,
		    currency = $5,
		    net_price = $6,
		    tax = $7,
		    shipping = $8,
		    total_price = $9,
		    version = version + 1
		WHERE cart_id = $10
	`

	result, err := r.db.Exec(ctx, query,
		cart.CustomerID, cart.ContactID, cart.CreditCardID, cart.CurrentStatus,
		cart.Currency, cart.NetPrice, cart.Tax, cart.Shipping, cart.TotalPrice, cart.CartID)
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

	var customerID *uuid.UUID
	query := `SELECT customer_id FROM carts.Cart WHERE cart_id = $1`
	err = r.db.QueryRow(ctx, query, id).Scan(&customerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCartNotFound
		}
		return fmt.Errorf("%w: failed to get cart: %v", ErrDatabaseOperation, err)
	}

	var customerIDStr *string
	if customerID != nil {
		id := customerID.String()
		customerIDStr = &id
	}
	evt := events.NewCartDeletedEvent(cartID, customerIDStr)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return fmt.Errorf("failed to write cart deleted event: %w", err)
	}

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

	cart.Contact, err = r.GetContact(ctx, cart.CartID.String())
	if err != nil && !errors.Is(err, ErrContactNotFound) {
		log.Printf("[ERROR] Repository: Failed to load contact for cart: %v", err)
		return fmt.Errorf("failed to load contact: %w", err)
	}

	cart.Addresses, err = r.GetAddresses(ctx, cart.CartID.String())
	if err != nil {
		log.Printf("[ERROR] Repository: Failed to load addresses for cart: %v", err)
		return fmt.Errorf("failed to load addresses: %w", err)
	}

	cart.CreditCard, err = r.GetCreditCard(ctx, cart.CartID.String())
	if err != nil && !errors.Is(err, ErrCreditCardNotFound) {
		log.Printf("[ERROR] Repository: Failed to load credit card for cart: %v", err)
		return fmt.Errorf("failed to load credit card: %w", err)
	}

	cart.Items, err = r.GetCartItems(ctx, cart.CartID.String())
	if err != nil {
		log.Printf("[ERROR] Repository: Failed to load items for cart: %v", err)
		return fmt.Errorf("failed to load items: %w", err)
	}

	cart.StatusHistory, err = r.GetStatusHistory(ctx, cart.CartID.String())
	if err != nil {
		log.Printf("[ERROR] Repository: Failed to load status history for cart: %v", err)
		return fmt.Errorf("failed to load status history: %w", err)
	}

	return nil
}
