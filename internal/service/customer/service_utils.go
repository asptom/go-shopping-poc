// Package customer provides business logic for customer management operations.
//
// This package implements the service layer for customer domain operations including
// CRUD operations, validation, business rule enforcement, and event processing utilities.
package customer

import (
	"log"
)

// CustomerServiceUtils provides utility functions for customer service operations
// This contains common patterns and helpers used across customer service methods
type CustomerServiceUtils struct{}

// NewCustomerServiceUtils creates a new customer service utilities instance
func NewCustomerServiceUtils() *CustomerServiceUtils {
	return &CustomerServiceUtils{}
}

// LogCustomerOperation logs customer-related operations with consistent formatting
func (u *CustomerServiceUtils) LogCustomerOperation(operation string, customerID string, details map[string]interface{}) {
	log.Printf("[INFO] CustomerService: %s for customer %s", operation, customerID)
	if len(details) > 0 {
		for key, value := range details {
			log.Printf("[DEBUG] CustomerService: %s detail - %s: %v", operation, key, value)
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

// NewCustomerError creates a standardized customer service error
func NewCustomerError(message string, cause error) error {
	if cause != nil {
		log.Printf("[ERROR] CustomerService: %s: %v", message, cause)
		return cause // In a real implementation, you'd wrap with domain-specific error
	}
	log.Printf("[ERROR] CustomerService: %s", message)
	return cause // In a real implementation, you'd return a domain-specific error
}
