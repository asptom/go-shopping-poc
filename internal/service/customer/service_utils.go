// Package customer provides business logic for customer management operations.
//
// This package implements the service layer for customer domain operations including
// CRUD operations, validation, business rule enforcement, and event processing utilities.
package customer

import (
	"fmt"
	"log/slog"
)

// CustomerServiceUtils provides utility functions for customer service operations
// This contains common patterns and helpers used across customer service methods
type CustomerServiceUtils struct {
	logger *slog.Logger
}

// NewCustomerServiceUtils creates a new customer service utilities instance
func NewCustomerServiceUtils() *CustomerServiceUtils {
	return &CustomerServiceUtils{
		logger: slog.Default().With("component", "customer_service_utils"),
	}
}

// LogCustomerOperation logs customer-related operations with consistent formatting
func (u *CustomerServiceUtils) LogCustomerOperation(operation string, customerID string, details map[string]interface{}) {
	u.logger.Info(operation, "customer_id", customerID)
	if len(details) > 0 {
		for key, value := range details {
			u.logger.Debug(operation+" detail", "key", key, "value", value)
		}
	}
}

// ValidateCustomerID performs common customer ID validation
func (u *CustomerServiceUtils) ValidateCustomerID(customerID string) error {
	if customerID == "" {
		return NewCustomerError("customer ID cannot be empty", nil)
	}
	if len(customerID) < 3 {
		return NewCustomerError("customer ID must be at least 3 characters", nil)
	}
	return nil
}

var utilsLogger = slog.Default().With("component", "customer_service_utils")

// NewCustomerError creates a standardized customer service error
func NewCustomerError(message string, cause error) error {
	if cause != nil {
		utilsLogger.Error(message, "error", cause.Error())
		return fmt.Errorf("%s: %w", message, cause)
	}
	utilsLogger.Error(message)
	return fmt.Errorf("%s", message)
}
