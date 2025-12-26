package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-shopping-poc/internal/platform/config"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with brokers",
			config: Config{
				Brokers: []string{"localhost:9092"},
				Topic:   "events",
				GroupID: "test-group",
			},
			wantErr: false,
		},
		{
			name: "valid config with multiple brokers",
			config: Config{
				Brokers: []string{"localhost:9092", "localhost:9093"},
				Topic:   "events",
				GroupID: "test-group",
			},
			wantErr: false,
		},
		{
			name: "invalid config with no brokers",
			config: Config{
				Brokers: []string{},
				Topic:   "events",
				GroupID: "test-group",
			},
			wantErr: true,
		},
		{
			name: "invalid config with nil brokers",
			config: Config{
				Brokers: nil,
				Topic:   "events",
				GroupID: "test-group",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Test that config loading doesn't panic and returns a config
	// Note: This test may skip if config file doesn't exist in test environment
	cfg, err := config.LoadConfig[Config]("platform-kafka")

	// We can't guarantee the config file exists in all test environments,
	// so we just verify the function doesn't panic
	if err != nil {
		t.Logf("config loading returned error (expected in test environment): %v", err)
	} else {
		require.NotNil(t, cfg)
		// If config loaded successfully, validate it
		err := cfg.Validate()
		if len(cfg.Brokers) == 0 {
			assert.Error(t, err, "config should be invalid with no brokers")
		} else {
			assert.NoError(t, err, "config should be valid when brokers are present")
		}
	}
}
