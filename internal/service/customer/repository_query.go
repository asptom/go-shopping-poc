// Package customer provides data access operations for customer entities.
//
// This file contains query operations for fetching customers and their related data.
package customer

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
)

// GetCustomerByID retrieves a customer by ID with all related data.
func (r *customerRepository) GetCustomerByID(ctx context.Context, customerID string) (*Customer, error) {
	log.Printf("[DEBUG] Repository: Fetching customer by ID...")

	id, err := uuid.Parse(customerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrInvalidUUID, customerID, err)
	}

	query := `select * from customers.customer where customers.customer.customer_id = $1`
	var customer Customer
	if err := r.db.GetContext(ctx, &customer, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Printf("[ERROR] Error fetching customer by ID: %v", err)
		return nil, fmt.Errorf("%w: failed to fetch customer %s: %v", ErrDatabaseOperation, customerID, err)
	}

	if err := r.LoadCustomerRelations(ctx, &customer); err != nil {
		return nil, fmt.Errorf("failed to load customer relations for %s: %w", customerID, err)
	}

	return &customer, nil
}

// GetCustomerByEmail retrieves a customer by email with all related data.
func (r *customerRepository) GetCustomerByEmail(ctx context.Context, email string) (*Customer, error) {
	log.Printf("[DEBUG] Repository: Fetching customer by email...")

	query := `SELECT * FROM customers.Customer WHERE email = $1`
	var customer Customer
	if err := r.db.GetContext(ctx, &customer, query, email); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Printf("[ERROR] Error fetching customer by email: %v", err)
		return nil, fmt.Errorf("%w: failed to fetch customer by email %s: %v", ErrDatabaseOperation, email, err)
	}

	if err := r.LoadCustomerRelations(ctx, &customer); err != nil {
		return nil, fmt.Errorf("failed to load customer relations for %s: %w", email, err)
	}

	return &customer, nil
}

// getAddressesByCustomerID retrieves all addresses for a customer.
func (r *customerRepository) getAddressesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]Address, error) {
	query := `SELECT * FROM customers.Address WHERE customer_id = $1`
	var addresses []Address
	if err := r.db.SelectContext(ctx, &addresses, query, customerID); err != nil {
		return nil, fmt.Errorf("%w: failed to fetch addresses for customer %s: %v", ErrDatabaseOperation, customerID, err)
	}
	return addresses, nil
}

// getCreditCardsByCustomerID retrieves all credit cards for a customer.
func (r *customerRepository) getCreditCardsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]CreditCard, error) {
	query := `SELECT * FROM customers.CreditCard WHERE customer_id = $1`
	var creditCards []CreditCard
	if err := r.db.SelectContext(ctx, &creditCards, query, customerID); err != nil {
		return nil, fmt.Errorf("%w: failed to fetch credit cards for customer %s: %v", ErrDatabaseOperation, customerID, err)
	}
	return creditCards, nil
}

// getStatusHistoryByCustomerID retrieves status history for a customer.
func (r *customerRepository) getStatusHistoryByCustomerID(ctx context.Context, customerID uuid.UUID) ([]CustomerStatus, error) {
	query := `SELECT * FROM customers.CustomerStatusHistory WHERE customer_id = $1`
	var statusHistory []CustomerStatus
	if err := r.db.SelectContext(ctx, &statusHistory, query, customerID); err != nil {
		return nil, fmt.Errorf("%w: failed to fetch status history for customer %s: %v", ErrDatabaseOperation, customerID, err)
	}
	return statusHistory, nil
}
