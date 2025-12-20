package eventreader

import (
	"os"
	"testing"
)

// TestLoadConfig_WriteTopicPopulation tests that WriteTopic is properly populated from environment variables
func TestLoadConfig_WriteTopicPopulation(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("event_reader_write_topic")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("event_reader_write_topic")
		} else {
			os.Setenv("event_reader_write_topic", originalEnv)
		}
	}()

	// Set test environment variable
	testWriteTopic := "test-customer-events"
	os.Setenv("event_reader_write_topic", testWriteTopic)

	// Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify WriteTopic is populated
	if cfg.WriteTopic != testWriteTopic {
		t.Errorf("Expected WriteTopic to be '%s', got '%s'", testWriteTopic, cfg.WriteTopic)
	}

	// Verify WriteTopic is not empty
	if cfg.WriteTopic == "" {
		t.Error("WriteTopic should not be empty")
	}
}

// TestLoadConfig_ReadTopicsPopulation tests that ReadTopics is properly populated from environment variables
func TestLoadConfig_ReadTopicsPopulation(t *testing.T) {
	// Save original environments
	originalReadTopics := os.Getenv("event_reader_read_topics")
	originalWriteTopic := os.Getenv("event_reader_write_topic")
	defer func() {
		if originalReadTopics == "" {
			os.Unsetenv("event_reader_read_topics")
		} else {
			os.Setenv("event_reader_read_topics", originalReadTopics)
		}
		if originalWriteTopic == "" {
			os.Unsetenv("event_reader_write_topic")
		} else {
			os.Setenv("event_reader_write_topic", originalWriteTopic)
		}
	}()

	// Set test environment variables
	testReadTopics := "topic1,topic2,topic3"
	os.Setenv("event_reader_read_topics", testReadTopics)
	os.Setenv("event_reader_write_topic", "dummy-write-topic") // Required for validation

	// Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify ReadTopics is populated
	expectedTopics := []string{"topic1", "topic2", "topic3"}
	if len(cfg.ReadTopics) != len(expectedTopics) {
		t.Errorf("Expected %d read topics, got %d", len(expectedTopics), len(cfg.ReadTopics))
		return
	}

	for i, expected := range expectedTopics {
		if cfg.ReadTopics[i] != expected {
			t.Errorf("Expected ReadTopic[%d] to be '%s', got '%s'", i, expected, cfg.ReadTopics[i])
		}
	}
}

// TestLoadConfig_GroupPopulation tests that Group is properly populated from environment variables
func TestLoadConfig_GroupPopulation(t *testing.T) {
	// Save original environments
	originalGroup := os.Getenv("event_reader_group")
	originalWriteTopic := os.Getenv("event_reader_write_topic")
	defer func() {
		if originalGroup == "" {
			os.Unsetenv("event_reader_group")
		} else {
			os.Setenv("event_reader_group", originalGroup)
		}
		if originalWriteTopic == "" {
			os.Unsetenv("event_reader_write_topic")
		} else {
			os.Setenv("event_reader_write_topic", originalWriteTopic)
		}
	}()

	// Set test environment variables
	testGroup := "test-consumer-group"
	os.Setenv("event_reader_group", testGroup)
	os.Setenv("event_reader_write_topic", "dummy-write-topic") // Required for validation

	// Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify Group is populated
	if cfg.Group != testGroup {
		t.Errorf("Expected Group to be '%s', got '%s'", testGroup, cfg.Group)
	}
}

// TestLoadConfig_Validation tests that configuration validation works correctly
func TestLoadConfig_Validation(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("event_reader_write_topic")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("event_reader_write_topic")
		} else {
			os.Setenv("event_reader_write_topic", originalEnv)
		}
	}()

	// Clear WriteTopic to test validation
	os.Unsetenv("event_reader_write_topic")

	// Load configuration - this should fail validation
	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected validation to fail when WriteTopic is empty")
	}

	// Verify the error message contains expected text
	if err != nil && !contains(err.Error(), "Field validation for 'WriteTopic' failed on the 'required' tag") {
		t.Errorf("Expected error to contain 'Field validation for WriteTopic failed on the required tag', got: %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
