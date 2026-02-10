// Package customer provides data access operations for customer entities.
//
// This file contains address-related operations including CRUD and default
// address management.
package customer

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"
)

// AddAddress adds a new address to a customer.
func (r *customerRepository) AddAddress(ctx context.Context, customerID string, addr *Address) (*Address, error) {
	log.Printf("[DEBUG] Repository: Adding address for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	addr.CustomerID = custUUID
	addr.AddressID = uuid.New()

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
        INSERT INTO customers.Address (
            address_id, customer_id, address_type, first_name, last_name,
            address_1, address_2, city, state, zip
        ) VALUES (
            :address_id, :customer_id, :address_type, :first_name, :last_name,
            :address_1, :address_2, :city, :state, :zip
        )`

	params := map[string]any{
		"address_id":   addr.AddressID,
		"customer_id":  addr.CustomerID,
		"address_type": addr.AddressType,
		"first_name":   addr.FirstName,
		"last_name":    addr.LastName,
		"address_1":    addr.Address1,
		"address_2":    addr.Address2,
		"city":         addr.City,
		"state":        addr.State,
		"zip":          addr.Zip,
	}

	if _, err := tx.NamedExecContext(ctx, query, params); err != nil {
		return nil, err
	}

	evt := events.NewAddressAddedEvent(customerID, addr.AddressID.String(), map[string]string{"address_type": addr.AddressType})
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return addr, nil
}

// UpdateAddress updates an existing address.
func (r *customerRepository) UpdateAddress(ctx context.Context, addressID string, addr *Address) error {
	log.Printf("[DEBUG] Repository: Updating address %s", addressID)

	id, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}
	addr.AddressID = id

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
        UPDATE customers.Address
        SET first_name = :first_name,
            last_name = :last_name,
            address_1 = :address_1,
            address_2 = :address_2,
            city = :city,
            state = :state,
            zip = :zip,
			address_type = :address_type
        WHERE address_id = :address_id`

	res, err := tx.NamedExecContext(ctx, query, addr)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("address not found: %s", addressID)
	}

	evt := events.NewAddressUpdatedEvent(addr.CustomerID.String(), addr.AddressID.String(), nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// DeleteAddress removes an address.
func (r *customerRepository) DeleteAddress(ctx context.Context, addressID string) error {
	log.Printf("[DEBUG] Repository: Deleting address with ID %s", addressID)

	id, err := uuid.Parse(addressID)
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

	res, err := tx.ExecContext(ctx, `DELETE FROM customers.Address WHERE address_id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("address not found: %s", addressID)
	}

	evt := events.NewAddressDeletedEvent("", addressID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// UpdateDefaultShippingAddress sets the default shipping address for a customer.
func (r *customerRepository) UpdateDefaultShippingAddress(ctx context.Context, customerID, addressID string) error {
	log.Printf("[DEBUG] Repository: Setting default shipping address %s for customer %s", addressID, customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	addrUUID, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_shipping_address_id = $1 WHERE customer_id = $2`

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

	result, err := tx.ExecContext(ctx, query, addrUUID, custUUID)
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

	evt := events.NewDefaultShippingAddressChangedEvent(customerID, addressID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// UpdateDefaultBillingAddress sets the default billing address for a customer.
func (r *customerRepository) UpdateDefaultBillingAddress(ctx context.Context, customerID, addressID string) error {
	log.Printf("[DEBUG] Repository: Setting default billing address %s for customer %s", addressID, customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	addrUUID, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_billing_address_id = $1 WHERE customer_id = $2`

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

	result, err := tx.ExecContext(ctx, query, addrUUID, custUUID)
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

	evt := events.NewDefaultBillingAddressChangedEvent(customerID, addressID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// ClearDefaultShippingAddress clears the default shipping address for a customer.
func (r *customerRepository) ClearDefaultShippingAddress(ctx context.Context, customerID string) error {
	log.Printf("[DEBUG] Repository: Clearing default shipping address for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_shipping_address_id = NULL WHERE customer_id = $1`

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

	evt := events.NewDefaultShippingAddressChangedEvent(customerID, "", nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// ClearDefaultBillingAddress clears the default billing address for a customer.
func (r *customerRepository) ClearDefaultBillingAddress(ctx context.Context, customerID string) error {
	log.Printf("[DEBUG] Repository: Clearing default billing address for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_billing_address_id = NULL WHERE customer_id = $1`

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

	evt := events.NewDefaultBillingAddressChangedEvent(customerID, "", nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}
