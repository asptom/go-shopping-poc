package websocket

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// mockConfig implements the config fields needed for testing.
type mockConfig struct {
	ReadBuffer     int
	WriteBuffer    int
	AllowedOrigins []string
}

// TestWebSocketServer_Creation tests WebSocket server creation and configuration
func TestWebSocketServer_Creation(t *testing.T) {
	// Create temporary config files for testing
	tempDir := t.TempDir()
	configDir := tempDir + "/config"
	os.MkdirAll(configDir, 0755)

	tests := []struct {
		name        string
		configFile  string
		envVars     map[string]string
		expectError bool
	}{
		{
			name: "valid configuration",
			configFile: `WEBSOCKET_URL=ws://localhost:8080
WEBSOCKET_ALLOWED_ORIGINS=http://localhost:3000,https://example.com`,
			envVars:     map[string]string{},
			expectError: false,
		},
		{
			name:        "missing URL",
			configFile:  `WEBSOCKET_ALLOWED_ORIGINS=http://localhost:3000`,
			envVars:     map[string]string{},
			expectError: true,
		},
		{
			name:        "missing allowed origins",
			configFile:  `WEBSOCKET_URL=ws://localhost:8080`,
			envVars:     map[string]string{},
			expectError: true,
		},
		{
			name: "invalid timeout via env",
			configFile: `WEBSOCKET_URL=ws://localhost:8080
WEBSOCKET_ALLOWED_ORIGINS=http://localhost:3000`,
			envVars:     map[string]string{"WEBSOCKET_TIMEOUT": "-1s"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config file
			configFile := configDir + "/platform-websocket.env"
			err := os.WriteFile(configFile, []byte(tt.configFile), 0644)
			if err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			// Change to temp directory for config loading
			oldWd, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(oldWd)

			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			server, err := NewWebSocketServer()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if server == nil {
				t.Fatal("Server is nil")
			}

			if server.Clients == nil {
				t.Error("Clients map not initialized")
			}

			if server.Upgrader.ReadBufferSize <= 0 {
				t.Error("ReadBufferSize not set properly")
			}

			if server.Upgrader.WriteBufferSize <= 0 {
				t.Error("WriteBufferSize not set properly")
			}

			if server.Upgrader.CheckOrigin == nil {
				t.Error("CheckOrigin function not set")
			}
		})
	}
}

// TestCreateCheckOrigin_AllowedOrigins tests origin validation with allowed origins
func TestCreateCheckOrigin_AllowedOrigins(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		requestOrigin  string
		expected       bool
	}{
		{
			name:           "exact match",
			allowedOrigins: []string{"http://localhost:3000", "https://example.com"},
			requestOrigin:  "http://localhost:3000",
			expected:       true,
		},
		{
			name:           "second allowed origin",
			allowedOrigins: []string{"http://localhost:3000", "https://example.com"},
			requestOrigin:  "https://example.com",
			expected:       true,
		},
		{
			name:           "disallowed origin",
			allowedOrigins: []string{"http://localhost:3000", "https://example.com"},
			requestOrigin:  "http://evil.com",
			expected:       false,
		},
		{
			name:           "empty allowed origins",
			allowedOrigins: []string{},
			requestOrigin:  "http://localhost:3000",
			expected:       false,
		},
		{
			name:           "nil allowed origins",
			allowedOrigins: nil,
			requestOrigin:  "http://localhost:3000",
			expected:       false,
		},
		{
			name:           "case sensitive match",
			allowedOrigins: []string{"http://localhost:3000"},
			requestOrigin:  "HTTP://LOCALHOST:3000",
			expected:       false,
		},
		{
			name:           "missing origin header",
			allowedOrigins: []string{"http://localhost:3000"},
			requestOrigin:  "",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkOrigin := createCheckOrigin(tt.allowedOrigins)

			req, _ := http.NewRequest("GET", "/ws", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}

			result := checkOrigin(req)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for origin %s", tt.expected, result, tt.requestOrigin)
			}
		})
	}
}

// TestWebSocketServer_Handle tests the WebSocketServer.Handle method.
func TestWebSocketServer_Handle(t *testing.T) {
	// Use mockConfig for buffer sizes
	cfg := &mockConfig{
		ReadBuffer:  1024,
		WriteBuffer: 1024,
	}

	// Create server using buffer sizes from mock config
	server := &WebSocketServer{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  cfg.ReadBuffer,
			WriteBufferSize: cfg.WriteBuffer,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		Clients: make(map[*websocket.Conn]bool),
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// Handler echoes "hello" -> "world"
	handler := server.Handle(func(conn *websocket.Conn) {
		defer wg.Done()
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("ReadMessage error: %v", err)
			return
		}
		if mt != websocket.TextMessage || string(msg) != "hello" {
			t.Errorf("Unexpected message: type=%v msg=%s", mt, msg)
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte("world")); err != nil {
			t.Errorf("WriteMessage error: %v", err)
		}
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	u.Scheme = "ws"

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer ws.Close()

	if err := ws.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatalf("WriteMessage error: %v", err)
	}

	mt, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage error: %v", err)
	}
	if mt != websocket.TextMessage || string(msg) != "world" {
		t.Errorf("Unexpected response: type=%v msg=%s", mt, msg)
	}

	wg.Wait()
}

// TestWebSocketServer_ConnectionHandling tests connection management
func TestWebSocketServer_ConnectionHandling(t *testing.T) {
	server := &WebSocketServer{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		Clients: make(map[*websocket.Conn]bool),
	}

	initialClientCount := len(server.Clients)

	var wg sync.WaitGroup
	wg.Add(1)

	handler := server.Handle(func(conn *websocket.Conn) {
		defer wg.Done()
		// Verify connection is tracked
		if len(server.Clients) != initialClientCount+1 {
			t.Errorf("Expected %d clients, got %d", initialClientCount+1, len(server.Clients))
		}
		// Connection will be cleaned up by defer in Handle method
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	u.Scheme = "ws"

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer ws.Close()

	wg.Wait()

	// After connection closes, client should be removed
	time.Sleep(100 * time.Millisecond) // Allow cleanup to happen
	if len(server.Clients) != initialClientCount {
		t.Errorf("Expected %d clients after cleanup, got %d", initialClientCount, len(server.Clients))
	}
}

// TestWebSocketServer_OriginValidation tests origin validation during upgrades
func TestWebSocketServer_OriginValidation(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		requestOrigin  string
		expectUpgrade  bool
	}{
		{
			name:           "allowed origin",
			allowedOrigins: []string{"http://localhost:3000"},
			requestOrigin:  "http://localhost:3000",
			expectUpgrade:  true,
		},
		{
			name:           "disallowed origin",
			allowedOrigins: []string{"http://localhost:3000"},
			requestOrigin:  "http://evil.com",
			expectUpgrade:  false,
		},
		{
			name:           "multiple allowed origins",
			allowedOrigins: []string{"http://localhost:3000", "https://example.com", "http://app.local"},
			requestOrigin:  "https://example.com",
			expectUpgrade:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &WebSocketServer{
				Upgrader: websocket.Upgrader{
					CheckOrigin: createCheckOrigin(tt.allowedOrigins),
				},
				Clients: make(map[*websocket.Conn]bool),
			}

			handler := server.Handle(func(conn *websocket.Conn) {
				t.Log("Connection established successfully")
			})

			ts := httptest.NewServer(handler)
			defer ts.Close()

			u, _ := url.Parse(ts.URL)
			u.Scheme = "ws"

			header := http.Header{}
			if tt.requestOrigin != "" {
				header.Set("Origin", tt.requestOrigin)
			}

			ws, resp, err := websocket.DefaultDialer.Dial(u.String(), header)

			if tt.expectUpgrade {
				if err != nil {
					t.Fatalf("Expected successful upgrade, got error: %v", err)
				}
				if ws == nil {
					t.Fatal("Expected websocket connection, got nil")
				}
				ws.Close()
			} else {
				if err == nil {
					t.Error("Expected upgrade to fail, but it succeeded")
					if ws != nil {
						ws.Close()
					}
				}
				// Check for 403 Forbidden status when origin is not allowed
				if resp != nil && resp.StatusCode != http.StatusForbidden {
					t.Errorf("Expected status 403, got %d", resp.StatusCode)
				}
			}
		})
	}
}

// TestWebSocketServer_ErrorScenarios tests error handling scenarios
func TestWebSocketServer_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		headers     map[string]string
		expectError bool
	}{
		{
			name:        "invalid HTTP method",
			method:      "POST",
			headers:     map[string]string{"Upgrade": "websocket", "Connection": "Upgrade"},
			expectError: true,
		},
		{
			name:        "missing upgrade header",
			method:      "GET",
			headers:     map[string]string{"Connection": "Upgrade"},
			expectError: true,
		},
		{
			name:        "missing connection header",
			method:      "GET",
			headers:     map[string]string{"Upgrade": "websocket"},
			expectError: true,
		},
		{
			name:        "invalid upgrade header",
			method:      "GET",
			headers:     map[string]string{"Upgrade": "http", "Connection": "Upgrade"},
			expectError: true,
		},
		{
			name:        "valid websocket request",
			method:      "GET",
			headers:     map[string]string{"Upgrade": "websocket", "Connection": "Upgrade"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &WebSocketServer{
				Upgrader: websocket.Upgrader{
					CheckOrigin: func(r *http.Request) bool { return true },
				},
				Clients: make(map[*websocket.Conn]bool),
			}

			handler := server.Handle(func(conn *websocket.Conn) {
				t.Log("Connection established")
			})

			ts := httptest.NewServer(handler)
			defer ts.Close()

			u, _ := url.Parse(ts.URL)
			u.Scheme = "ws"

			req, _ := http.NewRequest(tt.method, u.String(), nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			client := &http.Client{}
			resp, err := client.Do(req)

			if tt.expectError {
				if err == nil && resp.StatusCode < 400 {
					t.Errorf("Expected error or error status code, got status %d", resp.StatusCode)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if resp.StatusCode >= 400 {
					t.Errorf("Expected success status, got %d", resp.StatusCode)
				}
			}

			if resp != nil {
				resp.Body.Close()
			}
		})
	}
}

// TestWebSocketServer_SecurityValidation tests security aspects
func TestWebSocketServer_SecurityValidation(t *testing.T) {
	tests := []struct {
		name          string
		origin        string
		userAgent     string
		expectUpgrade bool
	}{
		{
			name:          "normal browser request",
			origin:        "http://localhost:3000",
			userAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expectUpgrade: true,
		},
		{
			name:          "suspicious user agent",
			origin:        "http://localhost:3000",
			userAgent:     "curl/7.68.0",
			expectUpgrade: true, // Should still work if origin is allowed
		},
		{
			name:          "no user agent",
			origin:        "http://localhost:3000",
			userAgent:     "",
			expectUpgrade: true,
		},
		{
			name:          "malformed origin",
			origin:        "not-a-valid-origin",
			userAgent:     "Mozilla/5.0",
			expectUpgrade: false,
		},
		{
			name:          "origin with path",
			origin:        "http://localhost:3000/path",
			userAgent:     "Mozilla/5.0",
			expectUpgrade: false, // Exact match required
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &WebSocketServer{
				Upgrader: websocket.Upgrader{
					CheckOrigin: createCheckOrigin([]string{"http://localhost:3000"}),
				},
				Clients: make(map[*websocket.Conn]bool),
			}

			handler := server.Handle(func(conn *websocket.Conn) {
				t.Log("Connection established")
			})

			ts := httptest.NewServer(handler)
			defer ts.Close()

			u, _ := url.Parse(ts.URL)
			u.Scheme = "ws"

			header := http.Header{}
			if tt.origin != "" {
				header.Set("Origin", tt.origin)
			}
			if tt.userAgent != "" {
				header.Set("User-Agent", tt.userAgent)
			}

			ws, _, err := websocket.DefaultDialer.Dial(u.String(), header)

			if tt.expectUpgrade {
				if err != nil {
					t.Errorf("Expected successful upgrade, got error: %v", err)
				}
				if ws != nil {
					ws.Close()
				}
			} else {
				if err == nil {
					t.Error("Expected upgrade to fail, but it succeeded")
					if ws != nil {
						ws.Close()
					}
				}
			}
		})
	}
}

// TestWebSocketServer_BufferSizes tests buffer size configuration
func TestWebSocketServer_BufferSizes(t *testing.T) {
	tests := []struct {
		name        string
		readBuffer  int
		writeBuffer int
		expectError bool
	}{
		{
			name:        "valid buffer sizes",
			readBuffer:  1024,
			writeBuffer: 2048,
			expectError: false,
		},
		{
			name:        "zero read buffer",
			readBuffer:  0,
			writeBuffer: 1024,
			expectError: true,
		},
		{
			name:        "zero write buffer",
			readBuffer:  1024,
			writeBuffer: 0,
			expectError: true,
		},
		{
			name:        "negative buffers",
			readBuffer:  -1,
			writeBuffer: -1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for buffer sizes
			os.Setenv("WEBSOCKET_READ_BUFFER", string(rune(tt.readBuffer)))
			os.Setenv("WEBSOCKET_WRITE_BUFFER", string(rune(tt.writeBuffer)))
			os.Setenv("WEBSOCKET_URL", "ws://localhost:8080")
			os.Setenv("WEBSOCKET_ALLOWED_ORIGINS", "http://localhost:3000")
			defer func() {
				os.Unsetenv("WEBSOCKET_READ_BUFFER")
				os.Unsetenv("WEBSOCKET_WRITE_BUFFER")
				os.Unsetenv("WEBSOCKET_URL")
				os.Unsetenv("WEBSOCKET_ALLOWED_ORIGINS")
			}()

			server, err := NewWebSocketServer()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error due to invalid buffer sizes")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if server.Upgrader.ReadBufferSize != tt.readBuffer {
				t.Errorf("Expected ReadBufferSize %d, got %d", tt.readBuffer, server.Upgrader.ReadBufferSize)
			}

			if server.Upgrader.WriteBufferSize != tt.writeBuffer {
				t.Errorf("Expected WriteBufferSize %d, got %d", tt.writeBuffer, server.Upgrader.WriteBufferSize)
			}
		})
	}
}

// TestWebSocketClient_BasicOperations tests WebSocketClient basic operations
func TestWebSocketClient_BasicOperations(t *testing.T) {
	// Create a test server
	server := &WebSocketServer{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		Clients: make(map[*websocket.Conn]bool),
	}

	var receivedMessage string
	var wg sync.WaitGroup
	wg.Add(1)

	handler := server.Handle(func(conn *websocket.Conn) {
		defer wg.Done()
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("Server read error: %v", err)
			return
		}
		if mt == websocket.TextMessage {
			receivedMessage = string(msg)
			conn.WriteMessage(websocket.TextMessage, []byte("echo: "+receivedMessage))
		}
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Test client connection (this would normally use NewWebSocketClient, but we need to mock config)
	// For this test, we'll use the gorilla websocket client directly
	u, _ := url.Parse(ts.URL)
	u.Scheme = "ws"

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Client dial error: %v", err)
	}
	defer ws.Close()

	// Test sending message
	testMessage := "hello websocket"
	err = ws.WriteMessage(websocket.TextMessage, []byte(testMessage))
	if err != nil {
		t.Fatalf("Client write error: %v", err)
	}

	// Test receiving response
	mt, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Client read error: %v", err)
	}

	expectedResponse := "echo: " + testMessage
	if mt != websocket.TextMessage || string(msg) != expectedResponse {
		t.Errorf("Expected response %q, got %q (type: %v)", expectedResponse, string(msg), mt)
	}

	wg.Wait()
}

// TestConfig_Validation tests configuration validation
func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				URL:            "ws://localhost:8080",
				Timeout:        30 * time.Second,
				ReadBuffer:     1024,
				WriteBuffer:    1024,
				AllowedOrigins: []string{"http://localhost:3000"},
			},
			expectError: false,
		},
		{
			name: "empty URL",
			config: Config{
				URL:            "",
				Timeout:        30 * time.Second,
				ReadBuffer:     1024,
				WriteBuffer:    1024,
				AllowedOrigins: []string{"http://localhost:3000"},
			},
			expectError: true,
		},
		{
			name: "zero timeout",
			config: Config{
				URL:            "ws://localhost:8080",
				Timeout:        0,
				ReadBuffer:     1024,
				WriteBuffer:    1024,
				AllowedOrigins: []string{"http://localhost:3000"},
			},
			expectError: true,
		},
		{
			name: "negative timeout",
			config: Config{
				URL:            "ws://localhost:8080",
				Timeout:        -1 * time.Second,
				ReadBuffer:     1024,
				WriteBuffer:    1024,
				AllowedOrigins: []string{"http://localhost:3000"},
			},
			expectError: true,
		},
		{
			name: "zero read buffer",
			config: Config{
				URL:            "ws://localhost:8080",
				Timeout:        30 * time.Second,
				ReadBuffer:     0,
				WriteBuffer:    1024,
				AllowedOrigins: []string{"http://localhost:3000"},
			},
			expectError: true,
		},
		{
			name: "zero write buffer",
			config: Config{
				URL:            "ws://localhost:8080",
				Timeout:        30 * time.Second,
				ReadBuffer:     1024,
				WriteBuffer:    0,
				AllowedOrigins: []string{"http://localhost:3000"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected validation error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}
