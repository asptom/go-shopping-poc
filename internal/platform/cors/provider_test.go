package cors

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestNewCORSProvider_Success tests successful CORS provider creation
func TestNewCORSProvider_Success(t *testing.T) {
	// Set required environment variables
	envVars := map[string]string{
		"CORS_ALLOWED_ORIGINS":   "https://example.com,https://test.com",
		"CORS_ALLOWED_METHODS":   "GET,POST,PUT,DELETE",
		"CORS_ALLOWED_HEADERS":   "Content-Type,Authorization",
		"CORS_ALLOW_CREDENTIALS": "true",
		"CORS_MAX_AGE":           "3600",
	}

	// Set environment variables
	for key, value := range envVars {
		oldValue := os.Getenv(key)
		os.Setenv(key, value)
		defer func(k, v string) {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}(key, oldValue)
	}

	provider, err := NewCORSProvider()
	if err != nil {
		t.Fatalf("NewCORSProvider() failed: %v", err)
	}

	if provider == nil {
		t.Fatal("NewCORSProvider() returned nil provider")
	}

	// Test that we can get a CORS handler
	corsHandler := provider.GetCORSHandler()
	if corsHandler == nil {
		t.Fatal("GetCORSHandler() returned nil handler")
	}

	// Test the handler works
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := corsHandler(testHandler)

	req := httptest.NewRequest("GET", "https://api.example.com/test", nil)
	req.Header.Set("Origin", "https://example.com")

	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Check CORS headers are present
	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", origin, "https://example.com")
	}

	methods := w.Header().Get("Access-Control-Allow-Methods")
	expectedMethods := "GET,POST,PUT,DELETE"
	if methods != expectedMethods {
		t.Errorf("Access-Control-Allow-Methods = %q, want %q", methods, expectedMethods)
	}
}

// TestNewCORSProvider_ConfigLoadError tests provider creation with config load error
func TestNewCORSProvider_ConfigLoadError(t *testing.T) {
	// Clear required environment variables to simulate missing config
	envVars := []string{
		"CORS_ALLOWED_ORIGINS",
		"CORS_ALLOWED_METHODS",
		"CORS_ALLOWED_HEADERS",
	}

	// Clear environment variables
	for _, key := range envVars {
		oldValue := os.Getenv(key)
		os.Unsetenv(key)
		defer func(k, v string) {
			if v != "" {
				os.Setenv(k, v)
			}
		}(key, oldValue)
	}

	provider, err := NewCORSProvider()
	if err == nil {
		t.Fatal("NewCORSProvider() should have failed with config load error")
	}

	if provider != nil {
		t.Fatal("NewCORSProvider() should return nil provider on error")
	}

	expectedErrSubstring := "failed to load CORS config"
	if err.Error()[:len(expectedErrSubstring)] != expectedErrSubstring {
		t.Errorf("Error message = %q, want to start with %q", err.Error(), expectedErrSubstring)
	}
}

// TestNewCORSProvider_InvalidConfig tests provider creation with invalid config
func TestNewCORSProvider_InvalidConfig(t *testing.T) {
	// Set environment variables with invalid config (empty allowed origins)
	envVars := map[string]string{
		"CORS_ALLOWED_ORIGINS":   "", // Empty origins (invalid)
		"CORS_ALLOWED_METHODS":   "GET,POST",
		"CORS_ALLOWED_HEADERS":   "Content-Type",
		"CORS_ALLOW_CREDENTIALS": "true",
		"CORS_MAX_AGE":           "3600",
	}

	// Set environment variables
	for key, value := range envVars {
		oldValue := os.Getenv(key)
		os.Setenv(key, value)
		defer func(k, v string) {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}(key, oldValue)
	}

	provider, err := NewCORSProvider()
	if err == nil {
		t.Fatal("NewCORSProvider() should have failed with invalid config")
	}

	if provider != nil {
		t.Fatal("NewCORSProvider() should return nil provider on error")
	}

	expectedErrSubstring := "failed to load CORS config"
	if err.Error()[:len(expectedErrSubstring)] != expectedErrSubstring {
		t.Errorf("Error message = %q, want to start with %q", err.Error(), expectedErrSubstring)
	}
}

// TestCORSProviderImpl_GetCORSHandler tests the GetCORSHandler method
func TestCORSProviderImpl_GetCORSHandler(t *testing.T) {
	// Create a provider with a known config
	config := &Config{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           "3600",
	}

	corsHandler := NewFromConfig(config)
	provider := &CORSProviderImpl{
		corsHandler: corsHandler,
	}

	// Test GetCORSHandler returns the expected handler
	handler := provider.GetCORSHandler()
	if handler == nil {
		t.Fatal("GetCORSHandler() returned nil handler")
	}

	// Test the handler functionality
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := handler(testHandler)

	req := httptest.NewRequest("GET", "https://api.example.com/test", nil)
	req.Header.Set("Origin", "https://example.com")

	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Verify CORS headers
	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", origin, "https://example.com")
	}

	creds := w.Header().Get("Access-Control-Allow-Credentials")
	if creds != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", creds, "true")
	}
}

// TestCORSProviderImpl_InterfaceCompliance tests that CORSProviderImpl implements CORSProvider interface
func TestCORSProviderImpl_InterfaceCompliance(t *testing.T) {
	var _ CORSProvider = (*CORSProviderImpl)(nil)

	// Create a minimal provider to test interface compliance
	config := &Config{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           "3600",
	}

	corsHandler := NewFromConfig(config)
	provider := &CORSProviderImpl{
		corsHandler: corsHandler,
	}

	// Test that it implements the interface
	var iface CORSProvider = provider
	if iface == nil {
		t.Fatal("CORSProviderImpl does not implement CORSProvider interface")
	}

	// Test that GetCORSHandler method is available
	handler := iface.GetCORSHandler()
	if handler == nil {
		t.Fatal("GetCORSHandler() returned nil through interface")
	}
}

// TestNewCORSProvider_Integration tests full integration with real config loading
func TestNewCORSProvider_Integration(t *testing.T) {
	// Set environment variables for integration test
	envVars := map[string]string{
		"CORS_ALLOWED_ORIGINS":   "https://frontend.example.com,*.api.example.com",
		"CORS_ALLOWED_METHODS":   "GET,POST,PUT,DELETE,OPTIONS",
		"CORS_ALLOWED_HEADERS":   "Content-Type,Authorization,X-Requested-With",
		"CORS_ALLOW_CREDENTIALS": "true",
		"CORS_MAX_AGE":           "7200",
	}

	// Set environment variables
	for key, value := range envVars {
		oldValue := os.Getenv(key)
		os.Setenv(key, value)
		defer func(k, v string) {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}(key, oldValue)
	}

	provider, err := NewCORSProvider()
	if err != nil {
		t.Fatalf("NewCORSProvider() integration test failed: %v", err)
	}

	corsHandler := provider.GetCORSHandler()

	// Test various scenarios
	testCases := []struct {
		name         string
		origin       string
		method       string
		expectOrigin string
	}{
		{
			name:         "exact match origin",
			origin:       "https://frontend.example.com",
			method:       "GET",
			expectOrigin: "https://frontend.example.com",
		},
		{
			name:         "wildcard subdomain match",
			origin:       "https://v1.api.example.com",
			method:       "POST",
			expectOrigin: "https://v1.api.example.com",
		},
		{
			name:         "disallowed origin",
			origin:       "https://evil.com",
			method:       "GET",
			expectOrigin: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := corsHandler(testHandler)

			req := httptest.NewRequest(tc.method, "https://api.example.com/test", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			gotOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if gotOrigin != tc.expectOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", gotOrigin, tc.expectOrigin)
			}

			// Check that other headers are always present
			methods := w.Header().Get("Access-Control-Allow-Methods")
			if methods == "" {
				t.Error("Access-Control-Allow-Methods header should be present")
			}

			headers := w.Header().Get("Access-Control-Allow-Headers")
			if headers == "" {
				t.Error("Access-Control-Allow-Headers header should be present")
			}
		})
	}
}
