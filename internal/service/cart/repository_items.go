package cart

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
)

func (r *cartRepository) AddItem(ctx context.Context, cartID string, item *CartItem) error {
	log.Printf("[DEBUG] CartRepository: Adding item to cart %s: product_id=%s, quantity=%d", cartID, item.ProductID, item.Quantity)
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[DEBUG] CartRepository: failed to begin transaction for adding item to cart %s: %v", cartID, err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var nextLine int
	err = r.db.QueryRow(ctx, `SELECT nextval('carts.cart_sequence')`).Scan(&nextLine)
	if err != nil {
		log.Printf("[DEBUG] CartRepository: failed to get next line number for cart %s: %v", cartID, err)
		return fmt.Errorf("%w: failed to generate line number: %v", ErrDatabaseOperation, err)
	}
	item.LineNumber = fmt.Sprintf("%03d", nextLine)
	item.CartID = cartUUID
	item.CalculateLineTotal()

	query := `
		INSERT INTO carts.CartItem (
			cart_id, line_number, product_id, product_name, unit_price, quantity, total_price
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
	`

	_, err = tx.ExecContext(ctx, query,
		item.CartID, item.LineNumber, item.ProductID, item.ProductName, item.UnitPrice, item.Quantity, item.TotalPrice)
	if err != nil {
		log.Printf("[DEBUG] CartRepository: failed to insert item into database for cart %s: %v", cartID, err)
		return fmt.Errorf("%w: failed to insert item: %v", ErrDatabaseOperation, err)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[DEBUG] CartRepository: failed to commit transaction for adding item to cart %s: %v", cartID, err)
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

	var item CartItem
	err = r.db.GetContext(ctx, &item, `
		SELECT id, unit_price FROM carts.CartItem 
		WHERE cart_id = $1 AND line_number = $2
	`, cartUUID, lineNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCartItemNotFound
		}
		return fmt.Errorf("%w: failed to get item: %v", ErrDatabaseOperation, err)
	}

	newTotal := float64(quantity) * item.UnitPrice

	_, err = r.db.Exec(ctx, `
		UPDATE carts.CartItem
		SET quantity = $1, total_price = $2
		WHERE cart_id = $3 AND line_number = $4
	`, quantity, newTotal, cartUUID, lineNumber)
	if err != nil {
		return fmt.Errorf("%w: failed to update item: %v", ErrDatabaseOperation, err)
	}

	return nil
}

func (r *cartRepository) RemoveItem(ctx context.Context, cartID string, lineNumber string) error {
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
	}

	_, err = r.db.Exec(ctx, `DELETE FROM carts.CartItem WHERE cart_id = $1 AND line_number = $2`, cartUUID, lineNumber)
	if err != nil {
		return fmt.Errorf("%w: failed to delete item: %v", ErrDatabaseOperation, err)
	}

	return nil
}

func (r *cartRepository) GetCartItems(ctx context.Context, cartID string) ([]CartItem, error) {
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
	}

	var items []CartItem
	err = r.db.SelectContext(ctx, &items, `
		SELECT id, cart_id, line_number, product_id, product_name, unit_price, quantity, total_price
		FROM carts.CartItem
		WHERE cart_id = $1
		ORDER BY line_number
	`, cartUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get items: %v", ErrDatabaseOperation, err)
	}

	return items, nil
}
