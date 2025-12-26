package customer

import (
	"strings"
	"testing"
)

func TestCustomerServiceUtils_LogCustomerOperation(t *testing.T) {
	utils := NewCustomerServiceUtils()

	// This test mainly ensures the method doesn't panic
	// In a real scenario, we'd capture log output
	details := map[string]interface{}{
		"operation_id": "test-123",
		"timestamp":    "2023-01-01T00:00:00Z",
	}

	utils.LogCustomerOperation("test_operation", "customer-123", details)
	// Method should complete without error
}

func TestCustomerServiceUtils_ValidateCustomerID(t *testing.T) {
	utils := NewCustomerServiceUtils()

	// Test valid customer ID
	err := utils.ValidateCustomerID("valid-customer-123")
	if err != nil {
		t.Errorf("Expected valid customer ID to pass validation, got error: %v", err)
	}

	// Test empty customer ID
	err = utils.ValidateCustomerID("")
	if err == nil {
		t.Error("Expected error for empty customer ID")
	}
	if !strings.Contains(err.Error(), "customer ID cannot be empty") {
		t.Errorf("Expected empty customer ID validation error, got: %v", err)
	}

	// Test short customer ID
	err = utils.ValidateCustomerID("ab")
	if err == nil {
		t.Error("Expected error for short customer ID")
	}
	if !strings.Contains(err.Error(), "customer ID must be at least 3 characters") {
		t.Errorf("Expected short customer ID validation error, got: %v", err)
	}

	// Test minimum length customer ID
	err = utils.ValidateCustomerID("abc")
	if err != nil {
		t.Errorf("Expected minimum length customer ID to pass validation, got error: %v", err)
	}
}
