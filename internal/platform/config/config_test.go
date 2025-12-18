/*
Package config_test provides comprehensive testing for the configuration package.

This package tests the core configuration loading functionality including:
- Environment variable parsing with various data types
- Configuration loading from .env files
- Environment file resolution based on APP_ENV
- Integration with godotenv library
- Error handling for invalid configurations

Test Categories:
1. Environment Variable Parsing - Testing getEnv, getEnvArray, getEnvInt, etc.
2. Environment File Resolution - Testing ResolveEnvFile logic
3. Configuration Loading - Testing Load() function with various scenarios
4. Integration Testing - Testing with real .env files and environment variables

Important Notes:
- Tests use isolated environment setup to avoid affecting other tests
- Temporary files are created and cleaned up automatically
- Tests validate both success and error scenarios
*/
package config

import (
	"os"
	"testing"
	"time"
)

// ===== TEST HELPERS =====

// setupTestEnv sets up environment variables for testing and cleans them up after
func setupTestEnv(t *testing.T, env map[string]string) {
	t.Helper()

	// Save original values
	originals := make(map[string]string)
	for key := range env {
		originals[key] = os.Getenv(key)
	}

	// Set test values
	for key, value := range env {
		os.Setenv(key, value)
	}

	// Cleanup function
	t.Cleanup(func() {
		for key, value := range originals {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	})
}

// ===== ENVIRONMENT VARIABLE PARSING =====

// TestGetEnv_BasicFunctionality tests basic getEnv functionality
func TestGetEnv_BasicFunctionality(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{"existing variable", "TEST_VAR", "test_value", "test_value"},
		{"non-existing variable", "NON_EXISTENT_VAR", "", "fallback_value"},
		{"empty string variable", "EMPTY_VAR", "", "fallback_value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestEnv(t, map[string]string{tt.key: tt.value})

			result := getEnv(tt.key, "fallback_value")
			if result != tt.expected {
				t.Errorf("getEnv(%s, fallback_value) = %s, expected %s", tt.key, result, tt.expected)
			}
		})
	}
}

// TestGetEnvArray_SingleValue tests parsing single array values
func TestGetEnvArray_SingleValue(t *testing.T) {
	setupTestEnv(t, map[string]string{"TEST_ARRAY": "topic1"})

	result := getEnvArray("TEST_ARRAY", []string{"default"})
	expected := []string{"topic1"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
		return
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, v)
		}
	}
}

// TestGetEnvArray_MultipleValues tests parsing multiple array values
func TestGetEnvArray_MultipleValues(t *testing.T) {
	setupTestEnv(t, map[string]string{"TEST_ARRAY": "topic1,topic2,topic3"})

	result := getEnvArray("TEST_ARRAY", []string{"default"})
	expected := []string{"topic1", "topic2", "topic3"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
		return
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, v)
		}
	}
}

// TestGetEnvArray_WithSpaces tests trimming spaces in array values
func TestGetEnvArray_WithSpaces(t *testing.T) {
	setupTestEnv(t, map[string]string{"TEST_ARRAY": " topic1 , topic2 "})

	result := getEnvArray("TEST_ARRAY", []string{"default"})
	expected := []string{"topic1", "topic2"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
		return
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, v)
		}
	}
}

// TestGetEnvArray_EmptyValues tests filtering empty values
func TestGetEnvArray_EmptyValues(t *testing.T) {
	setupTestEnv(t, map[string]string{"TEST_ARRAY": "topic1,,topic2,"})

	result := getEnvArray("TEST_ARRAY", []string{"default"})
	expected := []string{"topic1", "topic2"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
		return
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, v)
		}
	}
}

// TestGetEnvArray_EmptyString tests empty string fallback
func TestGetEnvArray_EmptyString(t *testing.T) {
	setupTestEnv(t, map[string]string{"TEST_ARRAY": ""})

	result := getEnvArray("TEST_ARRAY", []string{"default1", "default2"})
	expected := []string{"default1", "default2"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(result))
		return
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, v)
		}
	}
}

// TestGetEnvInt_ValidInteger tests parsing valid integers
func TestGetEnvInt_ValidInteger(t *testing.T) {
	setupTestEnv(t, map[string]string{"TEST_INT": "8080"})

	result := getEnvInt("TEST_INT", 3000)
	expected := 8080

	if result != expected {
		t.Errorf("getEnvInt(TEST_INT, 3000) = %d, expected %d", result, expected)
	}
}

// TestGetEnvInt_InvalidInteger tests invalid integer fallback
func TestGetEnvInt_InvalidInteger(t *testing.T) {
	setupTestEnv(t, map[string]string{"TEST_INT": "not-a-number"})

	result := getEnvInt("TEST_INT", 3000)
	expected := 3000

	if result != expected {
		t.Errorf("getEnvInt(TEST_INT, 3000) = %d, expected %d", result, expected)
	}
}

// TestGetEnvInt_EmptyString tests empty string fallback
func TestGetEnvInt_EmptyString(t *testing.T) {
	setupTestEnv(t, map[string]string{"TEST_INT": ""})

	result := getEnvInt("TEST_INT", 3000)
	expected := 3000

	if result != expected {
		t.Errorf("getEnvInt(TEST_INT, 3000) = %d, expected %d", result, expected)
	}
}

// TestGetEnvTimeDuration_ValidDuration tests parsing valid durations
func TestGetEnvTimeDuration_ValidDuration(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected time.Duration
	}{
		{"seconds", "30s", 30 * time.Second},
		{"minutes", "5m", 5 * time.Minute},
		{"hours", "1h", 1 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestEnv(t, map[string]string{"TEST_DURATION": tt.value})

			result := getEnvTimeDuration("TEST_DURATION", 10*time.Second)
			if result != tt.expected {
				t.Errorf("getEnvTimeDuration(TEST_DURATION, 10s) = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestGetEnvTimeDuration_InvalidDuration tests invalid duration fallback
func TestGetEnvTimeDuration_InvalidDuration(t *testing.T) {
	setupTestEnv(t, map[string]string{"TEST_DURATION": "not-a-duration"})

	result := getEnvTimeDuration("TEST_DURATION", 10*time.Second)
	expected := 10 * time.Second

	if result != expected {
		t.Errorf("getEnvTimeDuration(TEST_DURATION, 10s) = %v, expected %v", result, expected)
	}
}

// TestGetEnvBool_TrueValues tests various true values
func TestGetEnvBool_TrueValues(t *testing.T) {
	trueValues := []string{"true", "TRUE", "1", "yes", "YES"}

	for _, value := range trueValues {
		t.Run("true_"+value, func(t *testing.T) {
			setupTestEnv(t, map[string]string{"TEST_BOOL": value})

			result := getEnvBool("TEST_BOOL", false)
			if result != true {
				t.Errorf("getEnvBool(TEST_BOOL, false) = %v, expected true for value %s", result, value)
			}
		})
	}
}

// TestGetEnvBool_FalseValues tests various false values
func TestGetEnvBool_FalseValues(t *testing.T) {
	falseValues := []string{"false", "FALSE", "0", "no", "anything-else", "invalid"}

	for _, value := range falseValues {
		t.Run("false_"+value, func(t *testing.T) {
			setupTestEnv(t, map[string]string{"TEST_BOOL": value})

			result := getEnvBool("TEST_BOOL", true)
			if result != false {
				t.Errorf("getEnvBool(TEST_BOOL, true) = %v, expected false for value %s", result, value)
			}
		})
	}
}

// TestGetEnvBool_EmptyString tests empty string fallback
func TestGetEnvBool_EmptyString(t *testing.T) {
	setupTestEnv(t, map[string]string{"TEST_BOOL": ""})

	result := getEnvBool("TEST_BOOL", true)
	expected := true

	if result != expected {
		t.Errorf("getEnvBool(TEST_BOOL, true) = %v, expected %v", result, expected)
	}
}

// ===== UTILITY FUNCTION TESTS ONLY =====
// All old monolithic config tests have been removed as the system now uses
// service-specific configurations with the new LoadConfig[T] architecture.
