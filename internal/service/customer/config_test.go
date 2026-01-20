package customer_test

import (
	"os"
	"testing"

	"go-shopping-poc/internal/service/customer"
)

// setValidTestEnv sets valid environment variables for testing
func setValidTestEnv(t *testing.T) {
	t.Helper()

	os.Setenv("db_url", "postgres://localhost:5432/test")
	os.Setenv("CUSTOMER_SERVICE_PORT", "8080")
	os.Setenv("CUSTOMER_WRITE_TOPIC", "CustomerEvents")
	os.Setenv("CUSTOMER_GROUP", "CustomerGroup")
}

// cleanupTestEnv unsets test environment variables
func cleanupTestEnv() {
	os.Unsetenv("db_url")
	os.Unsetenv("CUSTOMER_SERVICE_PORT")
	os.Unsetenv("CUSTOMER_WRITE_TOPIC")
	os.Unsetenv("CUSTOMER_GROUP")
}

func TestLoadConfigSuccess(t *testing.T) {
	t.Parallel()

	setValidTestEnv(t)
	defer cleanupTestEnv()

	cfg, err := customer.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if cfg.DatabaseURL == "" || cfg.ServicePort == "" || cfg.WriteTopic == "" {
		t.Error("required fields should be set")
	}
}

func TestConfigValidateSuccess(t *testing.T) {
	t.Parallel()

	cfg := &customer.Config{
		DatabaseURL: "postgres://localhost:5432/test",
		ServicePort: "8080",
		WriteTopic:  "CustomerEvents",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("valid config should pass validation: %v", err)
	}
}

func TestConfigValidateMissingRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       *customer.Config
		wantError bool
	}{
		{
			name:      "missing database URL",
			cfg:       &customer.Config{ServicePort: "8080", WriteTopic: "CustomerEvents"},
			wantError: true,
		},
		{
			name:      "missing service port",
			cfg:       &customer.Config{DatabaseURL: "postgres://localhost:5432/test", WriteTopic: "CustomerEvents"},
			wantError: true,
		},
		{
			name:      "missing write topic",
			cfg:       &customer.Config{DatabaseURL: "postgres://localhost:5432/test", ServicePort: "8080"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
