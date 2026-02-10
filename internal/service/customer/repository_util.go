// Package customer provides data access operations for customer entities.
//
// This file contains utility functions for customer repository operations,
// including helper methods for status history and event publishing.
package customer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"

	"go-shopping-poc/internal/platform/database"
)

// insertStatusHistory inserts new status history records.
func (r *customerRepository) insertStatusHistory(ctx context.Context, tx database.Tx, history []CustomerStatus, customerID uuid.UUID) error {
	statusQuery := `INSERT INTO customers.CustomerStatusHistory (
		customer_id, old_status, new_status, changed_at
	) VALUES (
		:customer_id, :old_status, :new_status, :changed_at
	)`

	for _, status := range history {
		status.CustomerID = customerID
		if status.ChangedAt.IsZero() {
			status.ChangedAt = time.Now()
		}
		if _, err := tx.NamedExecContext(ctx, statusQuery, status); err != nil {
			return fmt.Errorf("%w: failed to insert status history: %v", ErrDatabaseOperation, err)
		}
	}

	return nil
}

// publishCustomerUpdateEvent publishes the customer update event.
func (r *customerRepository) publishCustomerUpdateEvent(ctx context.Context, tx database.Tx, customer *Customer) error {
	customerEvent := events.NewCustomerUpdatedEvent(customer.CustomerID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, customerEvent); err != nil {
		return fmt.Errorf("failed to publish customer update event: %w", err)
	}
	return nil
}
