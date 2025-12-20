package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestConfig_Validate tests the Config.Validate method
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				AllowedOrigins:   []string{"https://example.com"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
				MaxAge:           "3600",
			},
			wantErr: false,
		},
		{
			name: "missing allowed origins",
			config: Config{
				AllowedOrigins:   []string{},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
				MaxAge:           "3600",
			},
			wantErr: true,
			errMsg:  "at least one allowed origin is required",
		},
		{
			name: "missing allowed methods",
			config: Config{
				AllowedOrigins:   []string{"https://example.com"},
				AllowedMethods:   []string{},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
				MaxAge:           "3600",
			},
			wantErr: true,
			errMsg:  "at least one allowed method is required",
		},
		{
			name: "missing allowed headers",
			config: Config{
				AllowedOrigins:   []string{"https://example.com"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{},
				AllowCredentials: true,
				MaxAge:           "3600",
			},
			wantErr: true,
			errMsg:  "at least one allowed header is required",
		},
		{
			name: "wildcard origin allowed",
			config: Config{
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: false,
				MaxAge:           "3600",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Config.Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

// TestOriginAllowed tests the originAllowed helper function
func TestOriginAllowed(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		allowed  []string
		expected bool
	}{
		{
			name:     "exact match",
			origin:   "https://example.com",
			allowed:  []string{"https://example.com", "https://other.com"},
			expected: true,
		},
		{
			name:     "no match",
			origin:   "https://example.com",
			allowed:  []string{"https://other.com"},
			expected: false,
		},
		{
			name:     "empty allowed list",
			origin:   "https://example.com",
			allowed:  []string{},
			expected: false,
		},
		{
			name:     "wildcard subdomain match",
			origin:   "https://sub.example.com",
			allowed:  []string{"*.example.com"},
			expected: true,
		},
		{
			name:     "wildcard subdomain no match",
			origin:   "https://sub.other.com",
			allowed:  []string{"*.example.com"},
			expected: false,
		},
		{
			name:     "wildcard exact match",
			origin:   "https://example.com",
			allowed:  []string{"*.example.com"},
			expected: false,
		},
		{
			name:     "multiple wildcards",
			origin:   "https://sub.example.com",
			allowed:  []string{"*.other.com", "*.example.com"},
			expected: true,
		},
		{
			name:     "empty origin",
			origin:   "",
			allowed:  []string{"https://example.com"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := originAllowed(tt.origin, tt.allowed)
			if result != tt.expected {
				t.Errorf("originAllowed(%q, %v) = %v, want %v", tt.origin, tt.allowed, result, tt.expected)
			}
		})
	}
}

// TestNewFromConfig_Middleware tests the middleware functionality
func TestNewFromConfig_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		requestOrigin  string
		requestMethod  string
		expectedOrigin string
		expectCreds    bool
	}{
		{
			name: "allowed origin with credentials",
			config: Config{
				AllowedOrigins:   []string{"https://example.com"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
				MaxAge:           "3600",
			},
			requestOrigin:  "https://example.com",
			requestMethod:  "GET",
			expectedOrigin: "https://example.com",
			expectCreds:    true,
		},
		{
			name: "disallowed origin",
			config: Config{
				AllowedOrigins:   []string{"https://example.com"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
				MaxAge:           "3600",
			},
			requestOrigin:  "https://other.com",
			requestMethod:  "GET",
			expectedOrigin: "",
			expectCreds:    true,
		},
		{
			name: "wildcard origin",
			config: Config{
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: false,
				MaxAge:           "3600",
			},
			requestOrigin:  "https://any.com",
			requestMethod:  "GET",
			expectedOrigin: "*",
			expectCreds:    false,
		},
		{
			name: "wildcard subdomain",
			config: Config{
				AllowedOrigins:   []string{"*.example.com"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
				MaxAge:           "3600",
			},
			requestOrigin:  "https://sub.example.com",
			requestMethod:  "GET",
			expectedOrigin: "https://sub.example.com",
			expectCreds:    true,
		},
		{
			name: "no origin header",
			config: Config{
				AllowedOrigins:   []string{"https://example.com"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				AllowCredentials: true,
				MaxAge:           "3600",
			},
			requestOrigin:  "",
			requestMethod:  "GET",
			expectedOrigin: "",
			expectCreds:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewFromConfig(&tt.config)

			// Create a test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with middleware
			wrappedHandler := middleware(testHandler)

			// Create request
			req := httptest.NewRequest(tt.requestMethod, "https://api.example.com/test", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(w, req)

			// Check Access-Control-Allow-Origin header
			gotOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if gotOrigin != tt.expectedOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", gotOrigin, tt.expectedOrigin)
			}

			// Check Access-Control-Allow-Credentials header
			gotCreds := w.Header().Get("Access-Control-Allow-Credentials")
			if tt.expectCreds {
				if gotCreds != "true" {
					t.Errorf("Access-Control-Allow-Credentials = %q, want %q", gotCreds, "true")
				}
			} else {
				if gotCreds != "" {
					t.Errorf("Access-Control-Allow-Credentials = %q, want empty", gotCreds)
				}
			}

			// Check other required headers
			expectedMethods := "GET,POST"
			gotMethods := w.Header().Get("Access-Control-Allow-Methods")
			if gotMethods != expectedMethods {
				t.Errorf("Access-Control-Allow-Methods = %q, want %q", gotMethods, expectedMethods)
			}

			expectedHeaders := "Content-Type"
			gotHeaders := w.Header().Get("Access-Control-Allow-Headers")
			if gotHeaders != expectedHeaders {
				t.Errorf("Access-Control-Allow-Headers = %q, want %q", gotHeaders, expectedHeaders)
			}

			expectedMaxAge := "3600"
			gotMaxAge := w.Header().Get("Access-Control-Max-Age")
			if gotMaxAge != expectedMaxAge {
				t.Errorf("Access-Control-Max-Age = %q, want %q", gotMaxAge, expectedMaxAge)
			}
		})
	}
}

// TestNewFromConfig_Preflight tests preflight request handling
func TestNewFromConfig_Preflight(t *testing.T) {
	config := Config{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           "3600",
	}

	middleware := NewFromConfig(&config)

	// Create a test handler that should not be called for preflight
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS preflight request")
	})

	wrappedHandler := middleware(testHandler)

	// Test preflight request
	req := httptest.NewRequest("OPTIONS", "https://api.example.com/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Preflight status code = %d, want %d", w.Code, http.StatusOK)
	}

	// Check CORS headers are present
	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", origin, "https://example.com")
	}

	methods := w.Header().Get("Access-Control-Allow-Methods")
	expectedMethods := "GET,POST,PUT"
	if methods != expectedMethods {
		t.Errorf("Access-Control-Allow-Methods = %q, want %q", methods, expectedMethods)
	}

	headers := w.Header().Get("Access-Control-Allow-Headers")
	expectedHeaders := "Content-Type,Authorization"
	if headers != expectedHeaders {
		t.Errorf("Access-Control-Allow-Headers = %q, want %q", headers, expectedHeaders)
	}

	creds := w.Header().Get("Access-Control-Allow-Credentials")
	if creds != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", creds, "true")
	}

	maxAge := w.Header().Get("Access-Control-Max-Age")
	if maxAge != "3600" {
		t.Errorf("Access-Control-Max-Age = %q, want %q", maxAge, "3600")
	}
}

// TestNewFromConfig_PreflightDisallowedOrigin tests preflight with disallowed origin
func TestNewFromConfig_PreflightDisallowedOrigin(t *testing.T) {
	config := Config{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           "3600",
	}

	middleware := NewFromConfig(&config)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS preflight request")
	})

	wrappedHandler := middleware(testHandler)

	// Test preflight request with disallowed origin
	req := httptest.NewRequest("OPTIONS", "https://api.example.com/test", nil)
	req.Header.Set("Origin", "https://disallowed.com")

	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Preflight status code = %d, want %d", w.Code, http.StatusOK)
	}

	// Check that Access-Control-Allow-Origin is not set for disallowed origin
	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want empty for disallowed origin", origin)
	}

	// Other headers should still be set
	methods := w.Header().Get("Access-Control-Allow-Methods")
	expectedMethods := "GET,POST"
	if methods != expectedMethods {
		t.Errorf("Access-Control-Allow-Methods = %q, want %q", methods, expectedMethods)
	}
}

// TestNewFromConfig_ActualRequest tests actual requests (non-preflight)
func TestNewFromConfig_ActualRequest(t *testing.T) {
	config := Config{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           "3600",
	}

	middleware := NewFromConfig(&config)

	// Create a test handler that should be called for actual requests
	called := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	wrappedHandler := middleware(testHandler)

	// Test actual POST request
	req := httptest.NewRequest("POST", "https://api.example.com/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Check that handler was called
	if !called {
		t.Error("Handler was not called for actual request")
	}

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}

	// Check response body
	if w.Body.String() != "success" {
		t.Errorf("Response body = %q, want %q", w.Body.String(), "success")
	}

	// Check CORS headers are present
	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", origin, "https://example.com")
	}
}

// TestNewFromConfig_EdgeCases tests edge cases and error scenarios
func TestNewFromConfig_EdgeCases(t *testing.T) {
	t.Run("empty config slices", func(t *testing.T) {
		// This should not panic, but validation should catch it
		config := Config{
			AllowedOrigins:   []string{},
			AllowedMethods:   []string{},
			AllowedHeaders:   []string{},
			AllowCredentials: false,
			MaxAge:           "3600",
		}

		// Validate should fail
		err := config.Validate()
		if err == nil {
			t.Error("Expected validation error for empty config slices")
		}

		// But middleware creation should not panic
		middleware := NewFromConfig(&config)
		if middleware == nil {
			t.Error("Middleware should not be nil even with invalid config")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		// This should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("NewFromConfig panicked with nil config: %v", r)
			}
		}()

		middleware := NewFromConfig(nil)
		if middleware == nil {
			t.Error("Middleware should not be nil even with nil config")
		}
	})

	t.Run("malformed origin header", func(t *testing.T) {
		config := Config{
			AllowedOrigins:   []string{"https://example.com"},
			AllowedMethods:   []string{"GET"},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: false,
			MaxAge:           "3600",
		}

		middleware := NewFromConfig(&config)
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		wrappedHandler := middleware(testHandler)

		req := httptest.NewRequest("GET", "https://api.example.com/test", nil)
		req.Header.Set("Origin", "not-a-valid-url")

		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		// Should not set Access-Control-Allow-Origin for invalid origin
		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin != "" {
			t.Errorf("Access-Control-Allow-Origin = %q, want empty for invalid origin", origin)
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		config := Config{
			AllowedOrigins:   []string{"https://Example.Com"},
			AllowedMethods:   []string{"GET"},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: false,
			MaxAge:           "3600",
		}

		middleware := NewFromConfig(&config)
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		wrappedHandler := middleware(testHandler)

		req := httptest.NewRequest("GET", "https://api.example.com/test", nil)
		req.Header.Set("Origin", "https://example.com") // different case

		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		// Should not match due to case sensitivity
		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin != "" {
			t.Errorf("Access-Control-Allow-Origin = %q, want empty for case mismatch", origin)
		}
	})

	t.Run("multiple origins in config", func(t *testing.T) {
		config := Config{
			AllowedOrigins:   []string{"https://example.com", "https://other.com", "*.test.com"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: true,
			MaxAge:           "3600",
		}

		middleware := NewFromConfig(&config)
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		wrappedHandler := middleware(testHandler)

		tests := []struct {
			requestOrigin string
			shouldAllow   bool
		}{
			{"https://example.com", true},
			{"https://other.com", true},
			{"https://sub.test.com", true},
			{"https://notallowed.com", false},
		}

		for _, tt := range tests {
			req := httptest.NewRequest("GET", "https://api.example.com/test", nil)
			req.Header.Set("Origin", tt.requestOrigin)

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			origin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.shouldAllow {
				if origin != tt.requestOrigin {
					t.Errorf("Origin %q should be allowed, got %q", tt.requestOrigin, origin)
				}
			} else {
				if origin != "" {
					t.Errorf("Origin %q should not be allowed, got %q", tt.requestOrigin, origin)
				}
			}
		}
	})
}

// TestNewFromConfig_Integration tests full middleware integration
func TestNewFromConfig_Integration(t *testing.T) {
	config := Config{
		AllowedOrigins:   []string{"https://frontend.example.com", "*.api.example.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           "7200",
	}

	middleware := NewFromConfig(&config)

	// Chain multiple middlewares to simulate real usage
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	})

	wrappedHandler := middleware(finalHandler)

	// Test various scenarios
	scenarios := []struct {
		name            string
		method          string
		origin          string
		expectOrigin    string
		expectPreflight bool
	}{
		{
			name:            "GET request from allowed origin",
			method:          "GET",
			origin:          "https://frontend.example.com",
			expectOrigin:    "https://frontend.example.com",
			expectPreflight: false,
		},
		{
			name:            "POST request from wildcard subdomain",
			method:          "POST",
			origin:          "https://v1.api.example.com",
			expectOrigin:    "https://v1.api.example.com",
			expectPreflight: false,
		},
		{
			name:            "OPTIONS preflight from allowed origin",
			method:          "OPTIONS",
			origin:          "https://frontend.example.com",
			expectOrigin:    "https://frontend.example.com",
			expectPreflight: true,
		},
		{
			name:            "PUT request from disallowed origin",
			method:          "PUT",
			origin:          "https://evil.com",
			expectOrigin:    "",
			expectPreflight: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			req := httptest.NewRequest(scenario.method, "https://api.example.com/users", nil)
			if scenario.origin != "" {
				req.Header.Set("Origin", scenario.origin)
			}
			if scenario.method == "OPTIONS" {
				req.Header.Set("Access-Control-Request-Method", "POST")
				req.Header.Set("Access-Control-Request-Headers", "Content-Type")
			}

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			// Check CORS origin header
			gotOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if gotOrigin != scenario.expectOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", gotOrigin, scenario.expectOrigin)
			}

			// Check other CORS headers are always present
			methods := w.Header().Get("Access-Control-Allow-Methods")
			expectedMethods := "GET,POST,PUT,DELETE,OPTIONS"
			if methods != expectedMethods {
				t.Errorf("Access-Control-Allow-Methods = %q, want %q", methods, expectedMethods)
			}

			headers := w.Header().Get("Access-Control-Allow-Headers")
			expectedHeaders := "Content-Type,Authorization,X-Requested-With"
			if headers != expectedHeaders {
				t.Errorf("Access-Control-Allow-Headers = %q, want %q", headers, expectedHeaders)
			}

			creds := w.Header().Get("Access-Control-Allow-Credentials")
			if creds != "true" {
				t.Errorf("Access-Control-Allow-Credentials = %q, want %q", creds, "true")
			}

			maxAge := w.Header().Get("Access-Control-Max-Age")
			if maxAge != "7200" {
				t.Errorf("Access-Control-Max-Age = %q, want %q", maxAge, "7200")
			}

			// Check response based on preflight expectation
			if scenario.expectPreflight {
				if w.Code != http.StatusOK {
					t.Errorf("Preflight status code = %d, want %d", w.Code, http.StatusOK)
				}
				if w.Body.Len() != 0 {
					t.Errorf("Preflight response body should be empty, got %q", w.Body.String())
				}
			} else {
				if w.Code != http.StatusOK {
					t.Errorf("Request status code = %d, want %d", w.Code, http.StatusOK)
				}
				expectedBody := `{"message": "success"}`
				if w.Body.String() != expectedBody {
					t.Errorf("Response body = %q, want %q", w.Body.String(), expectedBody)
				}
			}
		})
	}
}
