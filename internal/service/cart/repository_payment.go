package cart

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

func (r *cartRepository) SetCreditCard(ctx context.Context, cartID string, card *CreditCard) error {
	r.logger.Debug("Setting credit card for cart",
		"cart_id", cartID,
		"card", card,
	)
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		r.logger.Error("Failed to parse cart ID for setting credit card", "cart_id", cartID, "error", err.Error())
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

	_, _ = tx.Exec(ctx, `DELETE FROM carts.CreditCard WHERE cart_id = $1`, cartUUID)

	card.CartID = cartUUID
	err = tx.QueryRow(ctx, `
		INSERT INTO carts.CreditCard (
			cart_id, card_type, card_number, card_holder_name, card_expires, card_cvv
		) VALUES (
			$1, $2, $3, $4, $5, $6)
		RETURNING id
	`, cartUUID, card.CardType, card.CardNumber, card.CardHolderName, card.CardExpires, card.CardCVV).Scan(&card.ID)
	if err != nil {
		r.logger.Error("Failed to insert credit card", "cart_id", cartID, "error", err.Error())
		return fmt.Errorf("%w: failed to insert credit card: %v", ErrDatabaseOperation, err)
	}

	_, err = tx.Exec(ctx, `UPDATE carts.Cart SET credit_card_id = $1 WHERE cart_id = $2`, card.ID, cartUUID)
	if err != nil {
		r.logger.Error("Failed to update cart with credit card", "cart_id", cartID, "error", err.Error())
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

	var card CreditCard
	err = r.db.GetContext(ctx, &card, `
		SELECT id, cart_id, card_type, card_number, card_holder_name, card_expires, card_cvv
		FROM carts.CreditCard
		WHERE cart_id = $1
	`, cartUUID)
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

	_, err = tx.Exec(ctx, `UPDATE carts.Cart SET credit_card_id = NULL WHERE cart_id = $1`, cartUUID)
	if err != nil {
		return fmt.Errorf("%w: failed to update cart: %v", ErrDatabaseOperation, err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM carts.CreditCard WHERE cart_id = $1`, cartUUID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete credit card: %v", ErrDatabaseOperation, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}
