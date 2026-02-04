// Package product provides data access operations for product entities.
//
// This file contains utility functions for product repository operations,
// including helper methods for validation, default values, and error handling.
package product

import (
	"fmt"
	"strings"
	"time"
)

// prepareProductDefaults sets default values for missing product fields
func (r *productRepository) prepareProductDefaults(product *Product) {
	if product.CreatedAt.IsZero() {
		product.CreatedAt = time.Now()
	}
	if product.UpdatedAt.IsZero() {
		product.UpdatedAt = time.Now()
	}
	if product.Currency == "" {
		product.Currency = "USD"
	}
	if !product.InStock {
		product.InStock = true
	}
}

// isDuplicateError checks if an error is a duplicate key violation
func isDuplicateError(err error) bool {
	errStr := fmt.Sprintf("%v", err)
	return err != nil &&
		(strings.Contains(errStr, "duplicate") ||
			strings.Contains(errStr, "unique"))
}
