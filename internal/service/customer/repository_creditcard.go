// Package customer provides data access operations for customer entities.
//
// This file contains credit card-related operations including CRUD and default
// credit card management.
package customer

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"
)

// AddCreditCard adds a new credit card to a customer.
func (r *customerRepository) AddCreditCard(ctx context.Context, customerID string, card *CreditCard) (*CreditCard, error) {
	log.Printf("[DEBUG] Repository: Adding credit card for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	card.CustomerID = custUUID
	card.CardID = uuid.New()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	query := `
        INSERT INTO customers.CreditCard (
            card_id, customer_id, card_number, card_type, card_holder_name,
            card_expires, card_cvv
        ) VALUES (
            :card_id, :customer_id, :card_number, :card_type, :card_holder_name,
            :card_expires, :card_cvv
        )`

	params := map[string]any{
		"card_id":          card.CardID,
		"customer_id":      card.CustomerID,
		"card_number":      card.CardNumber,
		"card_type":        card.CardType,
		"card_holder_name": card.CardHolderName,
		"card_expires":     card.CardExpires,
		"card_cvv":         card.CardCVV,
	}

	if _, err := tx.NamedExecContext(ctx, query, params); err != nil {
		return nil, err
	}

	evt := events.NewCardAddedEvent(card.CustomerID.String(), card.CardID.String(), map[string]string{
		"card_number": card.CardNumber,
	})
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return card, nil
}

// UpdateCreditCard updates an existing credit card.
func (r *customerRepository) UpdateCreditCard(ctx context.Context, cardID string, card *CreditCard) error {
	log.Printf("[DEBUG] Repository: Updating credit card %s", cardID)

	id, err := uuid.Parse(cardID)
	if err != nil {
		return err
	}
	card.CardID = id

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	query := `
        UPDATE customers.CreditCard
        SET card_type = :card_type,
            card_holder_name = :card_holder_name,
            card_expires = :card_expires,
            card_cvv = :card_cvv
        WHERE card_id = :card_id`

	res, err := tx.NamedExecContext(ctx, query, card)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("credit card not found: %s", cardID)
	}

	evt := events.NewCardUpdatedEvent(card.CustomerID.String(), card.CardID.String(), nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// DeleteCreditCard removes a credit card.
func (r *customerRepository) DeleteCreditCard(ctx context.Context, cardID string) error {
	log.Printf("[DEBUG] Repository: Deleting credit card with ID %s", cardID)

	id, err := uuid.Parse(cardID)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.ExecContext(ctx, `DELETE FROM customers.CreditCard WHERE card_id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("credit card not found: %s", cardID)
	}

	evt := events.NewCardDeletedEvent("", cardID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// UpdateDefaultCreditCard sets the default credit card for a customer.
func (r *customerRepository) UpdateDefaultCreditCard(ctx context.Context, customerID, cardID string) error {
	log.Printf("[DEBUG] Repository: Setting default credit card %s for customer %s", cardID, customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	cardUUID, err := uuid.Parse(cardID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_credit_card_id = $1 WHERE customer_id = $2`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	result, err := tx.ExecContext(ctx, query, cardUUID, custUUID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	evt := events.NewDefaultCreditCardChangedEvent(customerID, cardID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// ClearDefaultCreditCard clears the default credit card for a customer.
func (r *customerRepository) ClearDefaultCreditCard(ctx context.Context, customerID string) error {
	log.Printf("[DEBUG] Repository: Clearing default credit card for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_credit_card_id = NULL WHERE customer_id = $1`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	result, err := tx.ExecContext(ctx, query, custUUID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	evt := events.NewDefaultCreditCardChangedEvent(customerID, "", nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}
