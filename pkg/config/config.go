package config

import (
	"go-shopping-poc/pkg/logging"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Kafka configuration
	EventBroker           string
	EventWriterWriteTopic string
	EventWriterReadTopics []string
	EventWriterGroup      string
	EventReaderWriteTopic string
	EventReaderReadTopics []string
	EventReaderGroup      string

	// WebSocket configuration
	WebSocketURL         string
	WebSocketTimeoutMs   int
	WebSocketReadBuffer  int
	WebSocketWriteBuffer int
	WebSocketPort        string

	// Customer service configuration
	CustomerDBURL          string
	CustomerDBURLLocal     string
	CustomerServicePort    string
	CustomerWriteTopic     string
	CustomerReadTopics     []string
	CustomerGroup          string
	CustomerOutboxInterval time.Duration

	// CORS configuration
	CORSAllowedOrigins   string
	CORSAllowedMethods   string
	CORSAllowedHeaders   string
	CORSAllowCredentials bool
	CORSMaxAge           string
}

func Load(envFile string) *Config {
	_ = godotenv.Load(envFile) // Loads the specified .env file

	return &Config{
		EventBroker: getEnv("EVENT_BROKER", "localhost:9092"),

		EventWriterWriteTopic: getEnv("EVENT_WRITER_WRITE_TOPIC", ""),
		EventWriterReadTopics: getEnvArray("EVENT_WRITER_READ_TOPICS", []string{}),
		EventWriterGroup:      getEnv("EVENT_WRITER_GROUP", ""),

		EventReaderWriteTopic: getEnv("EVENT_READER_WRITE_TOPIC", ""),
		EventReaderReadTopics: getEnvArray("EVENT_READER_READ_TOPICS", []string{}),
		EventReaderGroup:      getEnv("EVENT_READER_GROUP", ""),

		WebSocketURL:         getEnv("WEBSOCKET_URL", "ws://localhost:8080/ws"),
		WebSocketTimeoutMs:   getEnvInt("WEBSOCKET_TIMEOUT_MS", 5000),
		WebSocketReadBuffer:  getEnvInt("WEBSOCKET_READ_BUFFER", 1024),
		WebSocketWriteBuffer: getEnvInt("WEBSOCKET_WRITE_BUFFER", 1024),
		WebSocketPort:        getEnv("WEBSOCKET_PORT", ":8080"),

		CustomerDBURL:          getEnv("PSQL_CUSTOMER_DB_URL", "postgres://user:password@localhost:5432/customer_db?sslmode=disable"),
		CustomerDBURLLocal:     getEnv("PSQL_CUSTOMER_DB_URL_LOCAL", "postgres://user:password@localhost:5432/customer_db?sslmode=disable"),
		CustomerServicePort:    getEnv("CUSTOMER_SERVICE_PORT", ":80"),
		CustomerWriteTopic:     getEnv("CUSTOMER_WRITE_TOPIC", ""),
		CustomerReadTopics:     getEnvArray("CUSTOMER_READ_TOPICS", []string{}),
		CustomerGroup:          getEnv("CUSTOMER_GROUP", "CustomerEventGroup"),
		CustomerOutboxInterval: getEnvTimeDuration("CUSTOMER_OUTBOX_INTERVAL", (5 * time.Second)),

		// Populate CORS fields from env
		CORSAllowedOrigins:   getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:4200"),
		CORSAllowedMethods:   getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"),
		CORSAllowedHeaders:   getEnv("CORS_ALLOWED_HEADERS", "Content-Type,Authorization"),
		CORSAllowCredentials: getEnvBool("CORS_ALLOW_CREDENTIALS", true),
		CORSMaxAge:           getEnv("CORS_MAX_AGE", "3600"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvArray(key string, fallback []string) []string {
	if value := os.Getenv(key); value != "" {
		// Split the string by comma and trim spaces
		var parts []string
		for _, v := range strings.Split(value, ",") {
			// Only append non-empty values after trimming
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
		logging.Debug("Config: %s=%v", key, parts)
		return parts
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvTimeDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		l := strings.ToLower(value)
		return l == "1" || l == "true" || l == "yes"
	}
	return fallback
}

// Getters for the Websocket configuration

func (c *Config) WebSocketEnabled() bool {
	return c.WebSocketURL != ""
}
func (c *Config) GetWebSocketURL() string {
	return c.WebSocketURL
}
func (c *Config) WebSocketTimeout() time.Duration {
	return time.Duration(c.WebSocketTimeoutMs) * time.Millisecond
}
func (c *Config) WebSocketReadBufferSize() int {
	return c.WebSocketReadBuffer
}
func (c *Config) WebSocketWriteBufferSize() int {
	return c.WebSocketWriteBuffer
}
func (c *Config) GetWebSocketPort() string {
	return c.WebSocketPort
}

// Getters for CORS configuration

func (c *Config) GetCORSAllowedOrigins() string {
	return c.CORSAllowedOrigins
}

func (c *Config) GetCORSAllowedMethods() string {
	return c.CORSAllowedMethods
}

func (c *Config) GetCORSAllowedHeaders() string {
	return c.CORSAllowedHeaders
}

func (c *Config) GetCORSAllowCredentials() bool {
	return c.CORSAllowCredentials
}

func (c *Config) GetCORSMaxAge() string {
	return c.CORSMaxAge
}

// Getters for the Kafka configuration

func (c *Config) GetEventBroker() string {
	return c.EventBroker
}

// Getters for event writer and reader Kafka topics
func (c *Config) GetEventWriterWriteTopic() string {
	return c.EventWriterWriteTopic
}
func (c *Config) GetEventWriterReadTopics() []string {
	return c.EventWriterReadTopics
}
func (c *Config) GetEventWriterGroup() string {
	return c.EventWriterGroup
}
func (c *Config) GetEventReaderWriteTopic() string {
	return c.EventReaderWriteTopic
}
func (c *Config) GetEventReaderReadTopics() []string {
	return c.EventReaderReadTopics
}
func (c *Config) GetEventReaderGroup() string {
	return c.EventReaderGroup
}

// Getters for Customer services
func (c *Config) GetCustomerDBURL() string {
	return c.CustomerDBURL
}
func (c *Config) GetCustomerDBURLLocal() string {
	return c.CustomerDBURLLocal
}
func (c *Config) GetCustomerServicePort() string {
	return c.CustomerServicePort
}
func (c *Config) GetCustomerWriteTopic() string {
	return c.CustomerWriteTopic
}
func (c *Config) GetCustomerReadTopics() []string {
	return c.CustomerReadTopics
}
func (c *Config) GetCustomerGroup() string {
	return c.CustomerGroup
}
func (c *Config) GetCustomerOutboxInterval() time.Duration {
	return c.CustomerOutboxInterval
}
