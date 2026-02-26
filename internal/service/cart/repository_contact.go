package cart

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

func (r *cartRepository) SetContact(ctx context.Context, cartID string, contact *Contact) error {
	r.logger.Debug("Setting contact for cart",
		"cart_id", cartID,
		"contact", contact,
	)
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error("Failed to begin transaction for setting contact", "cart_id", cartID, "error", err.Error())
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	_, _ = tx.Exec(ctx, `DELETE FROM carts.Contact WHERE cart_id = $1`, cartUUID)

	contact.CartID = cartUUID
	err = tx.QueryRow(ctx, `
		INSERT INTO carts.Contact (cart_id, email, first_name, last_name, phone)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, cartUUID, contact.Email, contact.FirstName, contact.LastName, contact.Phone).Scan(&contact.ID)
	if err != nil {
		r.logger.Error("Failed to insert contact into database", "cart_id", cartID, "error", err.Error())
		return fmt.Errorf("%w: failed to insert contact: %v", ErrDatabaseOperation, err)
	}

	_, err = tx.Exec(ctx, `UPDATE carts.Cart SET contact_id = $1 WHERE cart_id = $2`, contact.ID, cartUUID)
	if err != nil {
		r.logger.Error("Failed to update cart with contact", "cart_id", cartID, "error", err.Error())
		return fmt.Errorf("%w: failed to update cart contact: %v", ErrDatabaseOperation, err)
	}

	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit transaction for setting contact", "cart_id", cartID, "error", err.Error())
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

	var contact Contact
	err = r.db.GetContext(ctx, &contact, `
		SELECT id, cart_id, email, first_name, last_name, phone
		FROM carts.Contact
		WHERE cart_id = $1
	`, cartUUID)
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
	_, err = r.db.Exec(ctx, `
		INSERT INTO carts.Address (cart_id, address_type, first_name, last_name, address_1, address_2, city, state, zip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, cartUUID, address.AddressType, address.FirstName, address.LastName, address.Address1, address.Address2, address.City, address.State, address.Zip)
	if err != nil {
		return fmt.Errorf("%w: failed to insert address: %v", ErrDatabaseOperation, err)
	}

	return nil
}

func (r *cartRepository) GetAddresses(ctx context.Context, cartID string) ([]Address, error) {
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cart ID: %v", ErrInvalidUUID, err)
	}

	var addresses []Address
	err = r.db.SelectContext(ctx, &addresses, `
		SELECT id, cart_id, address_type, first_name, last_name, address_1, address_2, city, state, zip
		FROM carts.Address
		WHERE cart_id = $1
		ORDER BY id
	`, cartUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get addresses: %v", ErrDatabaseOperation, err)
	}

	return addresses, nil
}

func (r *cartRepository) UpdateAddress(ctx context.Context, addressID int64, address *Address) error {
	address.ID = addressID
	_, err := r.db.Exec(ctx, `
		UPDATE carts.Address
		SET address_type = $1,
		    first_name = $2,
		    last_name = $3,
		    address_1 = $4,
		    address_2 = $5,
		    city = $6,
		    state = $7,
		    zip = $8
		WHERE id = $9
	`, address.AddressType, address.FirstName, address.LastName, address.Address1, address.Address2, address.City, address.State, address.Zip, addressID)
	if err != nil {
		return fmt.Errorf("%w: failed to update address: %v", ErrDatabaseOperation, err)
	}

	return nil
}

func (r *cartRepository) RemoveAddress(ctx context.Context, addressID int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM carts.Address WHERE id = $1`, addressID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete address: %v", ErrDatabaseOperation, err)
	}

	return nil
}
