package cart

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"go-shopping-poc/internal/platform/database"

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

	cart, err := r.getCartByIDTx(ctx, tx, cartUUID)
	if err != nil {
		return nil, err
	}

	if err := r.loadCartRelationsTx(ctx, tx, cart); err != nil {
		return nil, err
	}

	if err := cart.CanCheckout(); err != nil {
		return nil, fmt.Errorf("cart not ready for checkout: %w", err)
	}

	if err := cart.SetStatus("checked_out"); err != nil {
		return nil, err
	}

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
	_, err = tx.Exec(ctx, query,
		cart.CurrentStatus, cart.NetPrice, cart.Tax,
		cart.Shipping, cart.TotalPrice, cart.CartID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to update cart: %v", ErrDatabaseOperation, err)
	}

	if err := r.addStatusEntryTx(ctx, tx, cartID, "checked_out"); err != nil {
		return nil, err
	}

	var customerIDStr *string
	if cart.CustomerID != nil {
		id := cart.CustomerID.String()
		customerIDStr = &id
	}

	snapshot := createCartSnapshot(cart)
	evt := events.NewCartCheckedOutEventWithSnapshot(cartID, customerIDStr, snapshot)
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

	var history []CartStatus
	err = r.db.SelectContext(ctx, &history, `
		SELECT id, cart_id, cart_status, status_date_time
		FROM carts.CartStatus
		WHERE cart_id = $1
		ORDER BY status_date_time DESC
	`, cartUUID)
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

	_, err = r.db.Exec(ctx, `
		INSERT INTO carts.CartStatus (cart_id, cart_status, status_date_time)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
	`, cartUUID, status)
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

	_, err = tx.Exec(ctx, `
		INSERT INTO carts.CartStatus (cart_id, cart_status, status_date_time)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
	`, cartUUID, status)
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
	err := tx.QueryRow(ctx, query, cartID).Scan(
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

	if cart.ContactID != nil {
		query = `SELECT id, cart_id, email, first_name, last_name, phone FROM carts.Contact WHERE id = $1`
		cart.Contact = &Contact{}
		err = tx.GetContext(ctx, cart.Contact, query, *cart.ContactID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to load contact: %w", err)
		}
	}

	if cart.CreditCardID != nil {
		query = `SELECT id, cart_id, card_type, card_number, card_holder_name, card_expires, card_cvv FROM carts.CreditCard WHERE id = $1`
		cart.CreditCard = &CreditCard{}
		err = tx.GetContext(ctx, cart.CreditCard, query, *cart.CreditCardID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to load credit card: %w", err)
		}
	}

	query = `
		SELECT id, cart_id, address_type, first_name, last_name, address_1, address_2, city, state, zip
		FROM carts.Address
		WHERE cart_id = $1
	`
	err = tx.SelectContext(ctx, &cart.Addresses, query, cart.CartID)
	if err != nil {
		return fmt.Errorf("failed to load addresses: %w", err)
	}

	return nil
}

func createCartSnapshot(cart *Cart) *events.CartSnapshot {
	snapshot := &events.CartSnapshot{
		Currency:   cart.Currency,
		NetPrice:   cart.NetPrice,
		Tax:        cart.Tax,
		Shipping:   cart.Shipping,
		TotalPrice: cart.TotalPrice,
		Items:      make([]events.SnapshotItem, len(cart.Items)),
		Addresses:  make([]events.SnapshotAddress, 0),
	}

	if cart.Contact != nil {
		snapshot.Contact = &events.SnapshotContact{
			Email:     cart.Contact.Email,
			FirstName: cart.Contact.FirstName,
			LastName:  cart.Contact.LastName,
			Phone:     cart.Contact.Phone,
		}
	}

	if cart.CreditCard != nil {
		snapshot.CreditCard = &events.SnapshotPayment{
			CardType:       cart.CreditCard.CardType,
			CardNumber:     cart.CreditCard.CardNumber,
			CardHolderName: cart.CreditCard.CardHolderName,
			CardExpires:    cart.CreditCard.CardExpires,
			CardCVV:        cart.CreditCard.CardCVV,
		}
	}

	for i, item := range cart.Items {
		snapshot.Items[i] = events.SnapshotItem{
			LineNumber:  item.LineNumber,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    item.Quantity,
			TotalPrice:  item.TotalPrice,
		}
	}

	for _, addr := range cart.Addresses {
		snapshot.Addresses = append(snapshot.Addresses, events.SnapshotAddress{
			AddressType: addr.AddressType,
			FirstName:   addr.FirstName,
			LastName:    addr.LastName,
			Address1:    addr.Address1,
			Address2:    addr.Address2,
			City:        addr.City,
			State:       addr.State,
			Zip:         addr.Zip,
		})
	}

	return snapshot
}
