package cart

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go-shopping-poc/internal/platform/database"

	"github.com/google/uuid"
)

func (r *cartRepository) AddItem(ctx context.Context, cartID string, item *CartItem) error {
	r.logger.Debug("Adding item to cart",
		"cart_id", cartID,
		"product_id", item.ProductID,
		"quantity", item.Quantity,
	)
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return fmt.Errorf("%w: invalid cart ID: %w", ErrInvalidUUID, err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to begin transaction for adding item", "cart_id", cartID, "error", err.Error())
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
		r.logger.Error("Failed to get next line number", "cart_id", cartID, "error", err.Error())
		return fmt.Errorf("%w: failed to generate line number: %w", ErrDatabaseOperation, err)
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
		r.logger.Error("Failed to insert item into database", "cart_id", cartID, "error", err.Error())
		return fmt.Errorf("%w: failed to insert item: %w", ErrDatabaseOperation, err)
	}

	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit transaction for adding item", "cart_id", cartID, "error", err.Error())
		return fmt.Errorf("%w: failed to commit transaction: %w", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

func (r *cartRepository) UpdateItemQuantity(ctx context.Context, cartID string, lineNumber string, quantity int) error {
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return fmt.Errorf("%w: invalid cart ID: %w", ErrInvalidUUID, err)
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
		return fmt.Errorf("%w: failed to get item: %w", ErrDatabaseOperation, err)
	}

	newTotal := float64(quantity) * item.UnitPrice

	_, err = r.db.Exec(ctx, `
		UPDATE carts.CartItem
		SET quantity = $1, total_price = $2
		WHERE cart_id = $3 AND line_number = $4
	`, quantity, newTotal, cartUUID, lineNumber)
	if err != nil {
		return fmt.Errorf("%w: failed to update item: %w", ErrDatabaseOperation, err)
	}

	return nil
}

func (r *cartRepository) RemoveItem(ctx context.Context, cartID string, lineNumber string) error {
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return fmt.Errorf("%w: invalid cart ID: %w", ErrInvalidUUID, err)
	}

	_, err = r.db.Exec(ctx, `DELETE FROM carts.CartItem WHERE cart_id = $1 AND line_number = $2`, cartUUID, lineNumber)
	if err != nil {
		return fmt.Errorf("%w: failed to delete item: %w", ErrDatabaseOperation, err)
	}

	return nil
}

func (r *cartRepository) GetCartItems(ctx context.Context, cartID string) ([]CartItem, error) {
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cart ID: %w", ErrInvalidUUID, err)
	}

	var items []CartItem
	err = r.db.SelectContext(ctx, &items, `
		SELECT id, cart_id, line_number, product_id, product_name, unit_price, quantity, total_price, status, validation_id, backorder_reason
		FROM carts.CartItem
		WHERE cart_id = $1
		ORDER BY line_number
	`, cartUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get items: %w", ErrDatabaseOperation, err)
	}

	return items, nil
}

// AddItemTx adds an item within an existing transaction (for outbox pattern)
func (r *cartRepository) AddItemTx(ctx context.Context, tx database.Tx, cartID string, item *CartItem) error {
	r.logger.Debug("Adding item to cart (transactional)",
		"cart_id", cartID,
		"product_id", item.ProductID,
		"quantity", item.Quantity,
	)
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return fmt.Errorf("%w: invalid cart ID: %w", ErrInvalidUUID, err)
	}

	var nextLine int
	err = tx.QueryRow(ctx, `SELECT nextval('carts.cart_sequence')`).Scan(&nextLine)
	if err != nil {
		r.logger.Error("Failed to get next line number", "cart_id", cartID, "error", err.Error())
		return fmt.Errorf("%w: failed to generate line number: %w", ErrDatabaseOperation, err)
	}
	item.LineNumber = fmt.Sprintf("%03d", nextLine)
	item.CartID = cartUUID
	item.CalculateLineTotal()

	query := `
		INSERT INTO carts.CartItem (
			cart_id, line_number, product_id, product_name, unit_price, quantity, total_price, status, validation_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	_, err = tx.Exec(ctx, query,
		item.CartID, item.LineNumber, item.ProductID, item.ProductName, item.UnitPrice, item.Quantity, item.TotalPrice, item.Status, item.ValidationID)
	if err != nil {
		r.logger.Error("Failed to insert item into database (transactional)", "cart_id", cartID, "error", err.Error())
		return fmt.Errorf("%w: failed to insert item: %w", ErrDatabaseOperation, err)
	}

	return nil
}

// GetItemByValidationID finds a cart item by its validation correlation ID
func (r *cartRepository) GetItemByValidationID(ctx context.Context, validationID string) (*CartItem, error) {
	var item CartItem
	err := r.db.GetContext(ctx, &item, `
		SELECT id, cart_id, line_number, product_id, product_name, unit_price, quantity, total_price, status, validation_id, backorder_reason
		FROM carts.CartItem
		WHERE validation_id = $1
	`, validationID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCartItemNotFound
		}
		return nil, fmt.Errorf("%w: failed to get item by validation ID: %w", ErrDatabaseOperation, err)
	}

	return &item, nil
}

// GetItemByProductID finds a cart item by cart ID and product ID (for duplicate detection)
func (r *cartRepository) GetItemByProductID(ctx context.Context, cartID, productID string) (*CartItem, error) {
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cart ID: %w", ErrInvalidUUID, err)
	}

	var item CartItem
	err = r.db.GetContext(ctx, &item, `
		SELECT id, cart_id, line_number, product_id, product_name, unit_price, quantity, total_price, status, validation_id, backorder_reason
		FROM carts.CartItem
		WHERE cart_id = $1 AND product_id = $2
	`, cartUUID, productID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCartItemNotFound
		}
		return nil, fmt.Errorf("%w: failed to get item by product ID: %w", ErrDatabaseOperation, err)
	}

	return &item, nil
}

// UpdateItemStatus updates item status and details after validation
func (r *cartRepository) UpdateItemStatus(ctx context.Context, item *CartItem) error {
	_, err := r.db.Exec(ctx, `
		UPDATE carts.CartItem
		SET product_name = $1, unit_price = $2, total_price = $3, status = $4, backorder_reason = $5
		WHERE id = $6
	`, item.ProductName, item.UnitPrice, item.TotalPrice, item.Status, item.BackorderReason, item.ID)

	if err != nil {
		return fmt.Errorf("%w: failed to update item status: %w", ErrDatabaseOperation, err)
	}

	return nil
}
