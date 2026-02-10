// Package customer provides data access operations for customer entities.
//
// This file contains CRUD (Create, Read, Update, Delete) operations
// for customer entities, handling database operations with proper transaction
// management and validation.
package customer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"

	"go-shopping-poc/internal/platform/database"
)

// InsertCustomer creates a new customer record in the database.
func (r *customerRepository) InsertCustomer(ctx context.Context, customer *Customer) error {
	log.Printf("[DEBUG] Repository: Inserting new customer...")

	r.prepareCustomerDefaults(customer)

	if err := r.insertCustomerWithRelations(ctx, customer); err != nil {
		return fmt.Errorf("failed to insert customer with relations: %w", err)
	}

	return nil
}

// insertCustomerWithRelations handles the complete customer creation process
func (r *customerRepository) insertCustomerWithRelations(ctx context.Context, customer *Customer) error {
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

	if err := r.insertCustomerRecordInTransaction(ctx, tx, customer); err != nil {
		return err
	}

	customerID, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return fmt.Errorf("%w: invalid customer ID: %v", ErrInvalidUUID, err)
	}

	if len(customer.Addresses) > 0 {
		if err := r.insertAddresses(ctx, tx, customer.Addresses, customerID); err != nil {
			return err
		}
	}

	if len(customer.CreditCards) > 0 {
		if err := r.insertCreditCards(ctx, tx, customer.CreditCards, customerID); err != nil {
			return err
		}
	}

	initialStatus := []CustomerStatus{{
		CustomerID: customerID,
		OldStatus:  "",
		NewStatus:  customer.CustomerStatus,
		ChangedAt:  customer.StatusDateTime,
	}}
	if err := r.insertStatusHistory(ctx, tx, initialStatus, customerID); err != nil {
		return err
	}

	evt := events.NewCustomerCreatedEvent(customer.CustomerID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return fmt.Errorf("failed to publish customer created event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	if err := r.LoadCustomerRelations(ctx, customer); err != nil {
		return fmt.Errorf("failed to load customer relations: %w", err)
	}

	return nil
}

// insertCustomerRecordInTransaction inserts customer record within a transaction
func (r *customerRepository) insertCustomerRecordInTransaction(ctx context.Context, tx database.Tx, customer *Customer) error {
	customerQuery := `INSERT INTO customers.Customer (customer_id, user_name, email, first_name, last_name, phone, customer_since, customer_status, status_date_time) VALUES (:customer_id, :user_name, :email, :first_name, :last_name, :phone, :customer_since, :customer_status, :status_date_time)`

	_, err := tx.NamedExecContext(ctx, customerQuery, customer)
	if err != nil {
		return fmt.Errorf("%w: failed to execute insert query: %v", ErrDatabaseOperation, err)
	}
	return nil
}

func (r *customerRepository) prepareCustomerDefaults(customer *Customer) {
	newID := uuid.New()
	customer.CustomerID = newID.String()

	if customer.CustomerSince.IsZero() {
		customer.CustomerSince = time.Now()
	}
	if customer.CustomerStatus == "" {
		customer.CustomerStatus = "active"
	}
	if customer.StatusDateTime.IsZero() {
		customer.StatusDateTime = time.Now()
	}
}

func (r *customerRepository) UpdateCustomer(ctx context.Context, customer *Customer) error {
	log.Printf("[DEBUG] Repository: Updating customer (PUT - complete replace)...")

	if err := r.validateCustomerForUpdate(customer); err != nil {
		return err
	}

	id, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return fmt.Errorf("%w: invalid customer ID %s: %v", ErrInvalidUUID, customer.CustomerID, err)
	}

	return r.executeCustomerUpdate(ctx, customer, id)
}

func (r *customerRepository) validateCustomerForUpdate(customer *Customer) error {
	if customer.CustomerID == "" || customer.Username == "" || customer.Email == "" {
		return fmt.Errorf("PUT requires complete customer record with customer_id, username, and email")
	}
	return nil
}

func (r *customerRepository) executeCustomerUpdate(ctx context.Context, customer *Customer, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := r.updateCustomerRecord(ctx, tx, customer); err != nil {
		return err
	}

	if err := r.replaceCustomerRelations(ctx, tx, customer, id); err != nil {
		return err
	}

	if err := r.publishCustomerUpdateEvent(ctx, tx, customer); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

func (r *customerRepository) updateCustomerRecord(ctx context.Context, tx database.Tx, customer *Customer) error {
	customerQuery := `UPDATE customers.Customer
		SET user_name = :user_name, email = :email, first_name = :first_name,
		last_name = :last_name, phone = :phone, customer_since = :customer_since,
		customer_status = :customer_status, status_date_time = :status_date_time,
		default_shipping_address_id = :default_shipping_address_id,
		default_billing_address_id = :default_billing_address_id,
		default_credit_card_id = :default_credit_card_id
		WHERE customer_id = :customer_id`

	result, err := tx.NamedExecContext(ctx, customerQuery, customer)
	if err != nil {
		return fmt.Errorf("%w: failed to update customer record: %v", ErrDatabaseOperation, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: customer not found: %s", ErrCustomerNotFound, customer.CustomerID)
	}

	return nil
}

func (r *customerRepository) replaceCustomerRelations(ctx context.Context, tx database.Tx, customer *Customer, id uuid.UUID) error {
	if err := r.deleteExistingRelations(ctx, tx, id); err != nil {
		return err
	}

	if err := r.insertNewRelations(ctx, tx, customer, id); err != nil {
		return err
	}

	return nil
}

func (r *customerRepository) deleteExistingRelations(ctx context.Context, tx database.Tx, customerID uuid.UUID) error {
	queries := []string{
		`DELETE FROM customers.Address WHERE customer_id = $1`,
		`DELETE FROM customers.CreditCard WHERE customer_id = $1`,
		`DELETE FROM customers.CustomerStatusHistory WHERE customer_id = $1`,
	}

	for _, query := range queries {
		if _, err := tx.ExecContext(ctx, query, customerID); err != nil {
			return fmt.Errorf("%w: failed to delete existing relations: %v", ErrDatabaseOperation, err)
		}
	}

	return nil
}

func (r *customerRepository) insertNewRelations(ctx context.Context, tx database.Tx, customer *Customer, id uuid.UUID) error {
	if err := r.insertAddresses(ctx, tx, customer.Addresses, id); err != nil {
		return err
	}

	if err := r.insertCreditCards(ctx, tx, customer.CreditCards, id); err != nil {
		return err
	}

	if err := r.insertStatusHistory(ctx, tx, customer.StatusHistory, id); err != nil {
		return err
	}

	return nil
}

func (r *customerRepository) insertAddresses(ctx context.Context, tx database.Tx, addresses []Address, customerID uuid.UUID) error {
	addressQuery := `INSERT INTO customers.Address (
		address_id, customer_id, address_type, first_name, last_name,
		address_1, address_2, city, state, zip
	) VALUES (
		:address_id, :customer_id, :address_type, :first_name, :last_name,
		:address_1, :address_2, :city, :state, :zip
	)`

	for i := range addresses {
		addresses[i].CustomerID = customerID
		addresses[i].AddressID = uuid.New()
		if _, err := tx.NamedExecContext(ctx, addressQuery, &addresses[i]); err != nil {
			return fmt.Errorf("%w: failed to insert address: %v", ErrDatabaseOperation, err)
		}
	}

	return nil
}

func (r *customerRepository) insertCreditCards(ctx context.Context, tx database.Tx, cards []CreditCard, customerID uuid.UUID) error {
	cardQuery := `INSERT INTO customers.CreditCard (
		card_id, customer_id, card_type, card_number, card_holder_name,
		card_expires, card_cvv
	) VALUES (
		:card_id, :customer_id, :card_type, :card_number, :card_holder_name,
		:card_expires, :card_cvv
	)`

	for i := range cards {
		cards[i].CustomerID = customerID
		cards[i].CardID = uuid.New()
		if _, err := tx.NamedExecContext(ctx, cardQuery, &cards[i]); err != nil {
			return fmt.Errorf("%w: failed to insert credit card: %v", ErrDatabaseOperation, err)
		}
	}

	return nil
}

// PatchCustomer applies partial updates to customer data (PATCH semantics).
func (r *customerRepository) PatchCustomer(ctx context.Context, customerID string, patchData *PatchCustomerRequest) error {
	log.Printf("[DEBUG] Repository: Patching customer %s", customerID)

	existing, err := r.GetCustomerByID(ctx, customerID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	updated := *existing

	if patchData.UserName != nil {
		updated.Username = *patchData.UserName
	}
	if patchData.Email != nil {
		updated.Email = *patchData.Email
	}
	if patchData.FirstName != nil {
		updated.FirstName = *patchData.FirstName
	}
	if patchData.LastName != nil {
		updated.LastName = *patchData.LastName
	}
	if patchData.Phone != nil {
		updated.Phone = *patchData.Phone
	}
	if patchData.CustomerStatus != nil {
		updated.CustomerStatus = *patchData.CustomerStatus
	}

	if patchData.DefaultShippingAddressID != nil {
		if *patchData.DefaultShippingAddressID == "" {
			updated.DefaultShippingAddressID = nil
		} else {
			if uuid, err := uuid.Parse(*patchData.DefaultShippingAddressID); err == nil {
				updated.DefaultShippingAddressID = &uuid
			}
		}
	}
	if patchData.DefaultBillingAddressID != nil {
		if *patchData.DefaultBillingAddressID == "" {
			updated.DefaultBillingAddressID = nil
		} else {
			if uuid, err := uuid.Parse(*patchData.DefaultBillingAddressID); err == nil {
				updated.DefaultBillingAddressID = &uuid
			}
		}
	}
	if patchData.DefaultCreditCardID != nil {
		if *patchData.DefaultCreditCardID == "" {
			updated.DefaultCreditCardID = nil
		} else {
			if uuid, err := uuid.Parse(*patchData.DefaultCreditCardID); err == nil {
				updated.DefaultCreditCardID = &uuid
			}
		}
	}

	var newAddresses []Address
	if patchData.Addresses != nil {
		for _, patchAddr := range patchData.Addresses {
			addr := Address{
				AddressType: patchAddr.AddressType,
				FirstName:   patchAddr.FirstName,
				LastName:    patchAddr.LastName,
				Address1:    patchAddr.Address1,
				Address2:    patchAddr.Address2,
				City:        patchAddr.City,
				State:       patchAddr.State,
				Zip:         patchAddr.Zip,
			}
			newAddresses = append(newAddresses, addr)
		}
	}

	var newCreditCards []CreditCard
	if patchData.CreditCards != nil {
		for _, patchCard := range patchData.CreditCards {
			card := CreditCard{
				CardType:       patchCard.CardType,
				CardNumber:     patchCard.CardNumber,
				CardHolderName: patchCard.CardHolderName,
				CardExpires:    patchCard.CardExpires,
				CardCVV:        patchCard.CardCVV,
			}
			newCreditCards = append(newCreditCards, card)
		}
	}

	id, err := uuid.Parse(customerID)
	if err != nil {
		return fmt.Errorf("%w: invalid customer ID %s: %v", ErrInvalidUUID, customerID, err)
	}

	return r.executePatchCustomer(ctx, &updated, id, newAddresses, newCreditCards)
}

func (r *customerRepository) executePatchCustomer(ctx context.Context, customer *Customer, id uuid.UUID, newAddresses []Address, newCreditCards []CreditCard) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := r.updateCustomerRecord(ctx, tx, customer); err != nil {
		return err
	}

	if len(newAddresses) > 0 {
		if err := r.deleteAddresses(ctx, tx, id); err != nil {
			return err
		}
		if err := r.insertAddresses(ctx, tx, newAddresses, id); err != nil {
			return err
		}
	}

	if len(newCreditCards) > 0 {
		if err := r.deleteCreditCards(ctx, tx, id); err != nil {
			return err
		}
		if err := r.insertCreditCards(ctx, tx, newCreditCards, id); err != nil {
			return err
		}
	}

	if err := r.publishCustomerUpdateEvent(ctx, tx, customer); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

func (r *customerRepository) deleteAddresses(ctx context.Context, tx database.Tx, customerID uuid.UUID) error {
	query := `DELETE FROM customers.Address WHERE customer_id = $1`
	_, err := tx.ExecContext(ctx, query, customerID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete addresses: %v", ErrDatabaseOperation, err)
	}
	return nil
}

func (r *customerRepository) deleteCreditCards(ctx context.Context, tx database.Tx, customerID uuid.UUID) error {
	query := `DELETE FROM customers.CreditCard WHERE customer_id = $1`
	_, err := tx.ExecContext(ctx, query, customerID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete credit cards: %v", ErrDatabaseOperation, err)
	}
	return nil
}

// LoadCustomerRelations loads addresses, credit cards, and status history for the customer
func (r *customerRepository) LoadCustomerRelations(ctx context.Context, customer *Customer) error {
	id, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return fmt.Errorf("invalid customer ID: %w", err)
	}

	addresses, err := r.getAddressesByCustomerID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load addresses: %w", err)
	}
	customer.Addresses = addresses

	creditCards, err := r.getCreditCardsByCustomerID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load credit cards: %w", err)
	}
	customer.CreditCards = creditCards

	statusHistory, err := r.getStatusHistoryByCustomerID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load status history: %w", err)
	}
	customer.StatusHistory = statusHistory

	return nil
}
