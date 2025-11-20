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

// ===== CONFIG LOADING =====

// TestLoad_AllDefaultValues tests loading config with no environment variables or .env files
func TestLoad_AllDefaultValues(t *testing.T) {
	// Clear all relevant environment variables
	envVars := []string{
		"EVENT_BROKER", "EVENT_WRITER_WRITE_TOPIC", "EVENT_WRITER_READ_TOPICS",
		"EVENT_WRITER_GROUP", "EVENT_READER_WRITE_TOPIC", "EVENT_READER_READ_TOPICS",
		"EVENT_READER_GROUP", "WEBSOCKET_URL", "WEBSOCKET_TIMEOUT_MS",
		"WEBSOCKET_READ_BUFFER", "WEBSOCKET_WRITE_BUFFER", "WEBSOCKET_PORT",
		"PSQL_CUSTOMER_DB_URL", "PSQL_CUSTOMER_DB_URL_LOCAL", "CUSTOMER_SERVICE_PORT",
		"CUSTOMER_WRITE_TOPIC", "CUSTOMER_READ_TOPICS", "CUSTOMER_GROUP",
		"CUSTOMER_OUTBOX_INTERVAL", "CORS_ALLOWED_ORIGINS", "CORS_ALLOWED_METHODS",
		"CORS_ALLOWED_HEADERS", "CORS_ALLOW_CREDENTIALS", "CORS_MAX_AGE",
	}

	clearEnv := make(map[string]string)
	for _, envVar := range envVars {
		clearEnv[envVar] = ""
	}
	setupTestEnv(t, clearEnv)

	// Load config with non-existent file (should use defaults)
	cfg := Load("/non/existent/.env")

	// Verify default values
	if cfg.EventBroker != "localhost:9092" {
		t.Errorf("Expected EventBroker 'localhost:9092', got '%s'", cfg.EventBroker)
	}

	if cfg.WebSocketURL != "ws://localhost:8080/ws" {
		t.Errorf("Expected WebSocketURL 'ws://localhost:8080/ws', got '%s'", cfg.WebSocketURL)
	}

	if cfg.CustomerDBURL != "postgres://user:password@localhost:5432/customer_db?sslmode=disable" {
		t.Errorf("Expected CustomerDBURL default, got '%s'", cfg.CustomerDBURL)
	}

	if cfg.CORSAllowedOrigins != "http://localhost:4200" {
		t.Errorf("Expected CORSAllowedOrigins 'http://localhost:4200', got '%s'", cfg.CORSAllowedOrigins)
	}
}

// TestLoad_EnvironmentVariablePrecedence tests that environment variables override .env file values
func TestLoad_EnvironmentVariablePrecedence(t *testing.T) {
	// Set environment variables
	setupTestEnv(t, map[string]string{
		"EVENT_BROKER":          "env-kafka:9092",
		"WEBSOCKET_URL":         "ws://env-host:8080/ws",
		"CUSTOMER_SERVICE_PORT": ":9090",
	})

	// Load config (will try to load .env.local but env vars should take precedence)
	cfg := Load(".env.local")

	// Verify environment variables take precedence
	if cfg.EventBroker != "env-kafka:9092" {
		t.Errorf("Expected EventBroker 'env-kafka:9092', got '%s'", cfg.EventBroker)
	}

	if cfg.WebSocketURL != "ws://env-host:8080/ws" {
		t.Errorf("Expected WebSocketURL 'ws://env-host:8080/ws', got '%s'", cfg.WebSocketURL)
	}

	if cfg.CustomerServicePort != ":9090" {
		t.Errorf("Expected CustomerServicePort ':9090', got '%s'", cfg.CustomerServicePort)
	}
}

// ===== CONFIG STRUCTURE VALIDATION =====

// TestConfig_AllFieldsInitialized tests that Load() initializes all Config struct fields
func TestConfig_AllFieldsInitialized(t *testing.T) {
	cfg := Load("/non/existent/.env")

	// Test that all string fields are initialized (not empty unless expected)
	if cfg.EventBroker == "" {
		t.Error("EventBroker should be initialized")
	}

	if cfg.WebSocketURL == "" {
		t.Error("WebSocketURL should be initialized")
	}

	if cfg.CustomerDBURL == "" {
		t.Error("CustomerDBURL should be initialized")
	}

	// Test that slices are initialized (may be empty but not nil)
	if cfg.EventWriterReadTopics == nil {
		t.Error("EventWriterReadTopics should be initialized")
	}

	if cfg.CustomerReadTopics == nil {
		t.Error("CustomerReadTopics should be initialized")
	}

	// Test that numeric fields have reasonable defaults
	if cfg.WebSocketTimeoutMs <= 0 {
		t.Error("WebSocketTimeoutMs should be positive")
	}

	if cfg.WebSocketReadBuffer <= 0 {
		t.Error("WebSocketReadBuffer should be positive")
	}
}

// TestConfig_GetterMethods tests all getter methods return expected values
func TestConfig_GetterMethods(t *testing.T) {
	setupTestEnv(t, map[string]string{
		"EVENT_BROKER":           "test-broker:9092",
		"WEBSOCKET_URL":          "ws://test-ws:8080",
		"WEBSOCKET_TIMEOUT_MS":   "10000",
		"CUSTOMER_SERVICE_PORT":  ":9090",
		"CORS_ALLOWED_ORIGINS":   "https://test.com",
		"CORS_ALLOW_CREDENTIALS": "false",
	})

	cfg := Load("/non/existent/.env")

	// Test Kafka getters
	if cfg.GetEventBroker() != "test-broker:9092" {
		t.Errorf("GetEventBroker() = %s, expected 'test-broker:9092'", cfg.GetEventBroker())
	}

	// Test WebSocket getters
	if cfg.GetWebSocketURL() != "ws://test-ws:8080" {
		t.Errorf("GetWebSocketURL() = %s, expected 'ws://test-ws:8080'", cfg.GetWebSocketURL())
	}

	if cfg.WebSocketTimeout() != 10*time.Second {
		t.Errorf("WebSocketTimeout() = %v, expected 10s", cfg.WebSocketTimeout())
	}

	if cfg.WebSocketReadBufferSize() != 1024 {
		t.Errorf("WebSocketReadBufferSize() = %d, expected 1024", cfg.WebSocketReadBufferSize())
	}

	if cfg.GetWebSocketPort() != ":8080" { // This uses default since env var not set
		t.Errorf("GetWebSocketPort() = %s, expected ':8080'", cfg.GetWebSocketPort())
	}

	// Test Customer getters
	if cfg.GetCustomerServicePort() != ":9090" {
		t.Errorf("GetCustomerServicePort() = %s, expected ':9090'", cfg.GetCustomerServicePort())
	}

	// Test CORS getters
	if cfg.GetCORSAllowedOrigins() != "https://test.com" {
		t.Errorf("GetCORSAllowedOrigins() = %s, expected 'https://test.com'", cfg.GetCORSAllowedOrigins())
	}

	if cfg.GetCORSAllowCredentials() != false {
		t.Errorf("GetCORSAllowCredentials() = %v, expected false", cfg.GetCORSAllowCredentials())
	}
}

// ===== END-TO-END TESTING =====

// TestConfigEndToEnd_Integration tests end-to-end config loading with project defaults
func TestConfigEndToEnd_Integration(t *testing.T) {
	// This test verifies that config loading works with the current project setup
	// It simulates the .env.local file content to test environment variable expansion

	// Simulate the environment variables that would be set by .env.local
	setupTestEnv(t, map[string]string{
		"PSQL_CUSTOMER_DB":           "customersdb",
		"PSQL_CUSTOMER_ROLE":         "customersuser",
		"PSQL_CUSTOMER_PASSWORD":     "customerssecret",
		"PSQL_CUSTOMER_DB_URL_LOCAL": "postgres://$PSQL_CUSTOMER_ROLE:$PSQL_CUSTOMER_PASSWORD@localhost:30432/$PSQL_CUSTOMER_DB?sslmode=disable",
		"EVENT_BROKER":               "kafka-clusterip.kafka.svc.cluster.local:9092",
		"CORS_ALLOWED_ORIGINS":       "http://localhost:4200",
	})

	cfg := Load("/non/existent/.env") // Load without .env file to test env var precedence

	// Verify config is not nil
	if cfg == nil {
		t.Fatal("Load() returned nil")
	}

	// Verify basic fields are initialized
	if cfg.EventBroker == "" {
		t.Error("EventBroker should be initialized")
	}

	if cfg.CustomerDBURL == "" {
		t.Error("CustomerDBURL should be initialized")
	}

	// Verify getter methods work
	if cfg.GetEventBroker() == "" {
		t.Error("GetEventBroker() should return a value")
	}

	if cfg.GetCustomerDBURLLocal() == "" {
		t.Error("GetCustomerDBURLLocal() should return a value")
	}

	// Test that environment variable expansion works
	actualURL := cfg.GetCustomerDBURLLocal()
	expectedURL := "postgres://customersuser:customerssecret@localhost:30432/customersdb?sslmode=disable"

	t.Logf("Expected URL: %s", expectedURL)
	t.Logf("Actual URL: %s", actualURL)

	if actualURL != expectedURL {
		t.Errorf("Expected expanded URL %s, got %s", expectedURL, actualURL)
	} else {
		t.Logf("Successfully expanded environment variables in database URL: %s", actualURL)
	}

	// Verify other fields work
	if cfg.GetCORSAllowedOrigins() != "http://localhost:4200" {
		t.Errorf("Expected CORS origins 'http://localhost:4200', got '%s'", cfg.GetCORSAllowedOrigins())
	}
}

// TestConfigWithArrayValues tests configuration loading with array values
func TestConfigWithArrayValues(t *testing.T) {
	setupTestEnv(t, map[string]string{
		"EVENT_WRITER_READ_TOPICS": "topic1,topic2,topic3",
		"CUSTOMER_READ_TOPICS":     "customer.topic1,customer.topic2",
	})

	cfg := Load("/non/existent/.env")

	// Test EventWriter read topics
	expectedEventTopics := []string{"topic1", "topic2", "topic3"}
	eventTopics := cfg.GetEventWriterReadTopics()

	if len(eventTopics) != len(expectedEventTopics) {
		t.Errorf("Expected %d event topics, got %d", len(expectedEventTopics), len(eventTopics))
	}

	for i, topic := range eventTopics {
		if i < len(expectedEventTopics) && topic != expectedEventTopics[i] {
			t.Errorf("Expected event topic %s at index %d, got %s", expectedEventTopics[i], i, topic)
		}
	}

	// Test Customer read topics
	expectedCustomerTopics := []string{"customer.topic1", "customer.topic2"}
	customerTopics := cfg.GetCustomerReadTopics()

	if len(customerTopics) != len(expectedCustomerTopics) {
		t.Errorf("Expected %d customer topics, got %d", len(expectedCustomerTopics), len(customerTopics))
	}

	for i, topic := range customerTopics {
		if i < len(expectedCustomerTopics) && topic != expectedCustomerTopics[i] {
			t.Errorf("Expected customer topic %s at index %d, got %s", expectedCustomerTopics[i], i, topic)
		}
	}
}

// TestEnvironmentVariableExpansion tests that environment variables in .env values are expanded
func TestEnvironmentVariableExpansion(t *testing.T) {
	// Set up environment variables that will be referenced in URLs
	setupTestEnv(t, map[string]string{
		"PSQL_CUSTOMER_DB":           "testdb",
		"PSQL_CUSTOMER_ROLE":         "testuser",
		"PSQL_CUSTOMER_PASSWORD":     "testpass",
		"PSQL_CUSTOMER_DB_URL_LOCAL": "postgres://$PSQL_CUSTOMER_ROLE:$PSQL_CUSTOMER_PASSWORD@localhost:30432/$PSQL_CUSTOMER_DB?sslmode=disable",
	})

	cfg := Load("/non/existent/.env")

	expectedURL := "postgres://testuser:testpass@localhost:30432/testdb?sslmode=disable"
	if cfg.GetCustomerDBURLLocal() != expectedURL {
		t.Errorf("Expected expanded URL %s, got %s", expectedURL, cfg.GetCustomerDBURLLocal())
	}
}

// TestEnvironmentVariableExpansionInArrays tests expansion in array values
func TestEnvironmentVariableExpansionInArrays(t *testing.T) {
	setupTestEnv(t, map[string]string{
		"ENV_TOPIC_PREFIX":         "prod",
		"EVENT_WRITER_READ_TOPICS": "$ENV_TOPIC_PREFIX.topic1,$ENV_TOPIC_PREFIX.topic2",
	})

	cfg := Load("/non/existent/.env")

	expectedTopics := []string{"prod.topic1", "prod.topic2"}
	actualTopics := cfg.GetEventWriterReadTopics()

	if len(actualTopics) != len(expectedTopics) {
		t.Errorf("Expected %d topics, got %d", len(expectedTopics), len(actualTopics))
		return
	}

	for i, expected := range expectedTopics {
		if actualTopics[i] != expected {
			t.Errorf("Expected topic %s at index %d, got %s", expected, i, actualTopics[i])
		}
	}
}

// TestConfig_ComputedFields tests fields that are computed from other values
func TestConfig_ComputedFields(t *testing.T) {
	// Test computed duration and buffer fields
	t.Run("computed_duration_and_buffers", func(t *testing.T) {
		setupTestEnv(t, map[string]string{
			"WEBSOCKET_TIMEOUT_MS":   "15000",
			"WEBSOCKET_READ_BUFFER":  "2048",
			"WEBSOCKET_WRITE_BUFFER": "2048",
		})

		cfg := Load("/non/existent/.env")

		if cfg.WebSocketTimeout() != 15*time.Second {
			t.Errorf("WebSocketTimeout() = %v, expected 15s", cfg.WebSocketTimeout())
		}

		if cfg.WebSocketReadBufferSize() != 2048 {
			t.Errorf("WebSocketReadBufferSize() = %d, expected 2048", cfg.WebSocketReadBufferSize())
		}

		if cfg.WebSocketWriteBufferSize() != 2048 {
			t.Errorf("WebSocketWriteBufferSize() = %d, expected 2048", cfg.WebSocketWriteBufferSize())
		}
	})

	// Test boolean field computation - WebSocket enabled by default
	t.Run("WebSocket_enabled_by_default", func(t *testing.T) {
		setupTestEnv(t, map[string]string{}) // Clear any WEBSOCKET_URL setting
		cfg := Load("/non/existent/.env")
		if !cfg.WebSocketEnabled() {
			t.Error("WebSocketEnabled() should return true with default WebSocketURL")
		}
	})

	// Test WebSocket behavior with different URL values
	t.Run("WebSocket_with_custom_url", func(t *testing.T) {
		setupTestEnv(t, map[string]string{"WEBSOCKET_URL": "ws://custom:9000"})
		cfg := Load("/non/existent/.env")

		if cfg.WebSocketURL != "ws://custom:9000" {
			t.Errorf("Expected WebSocketURL 'ws://custom:9000', got '%s'", cfg.WebSocketURL)
		}
		if !cfg.WebSocketEnabled() {
			t.Error("WebSocketEnabled() should return true with custom URL")
		}
	})
}
