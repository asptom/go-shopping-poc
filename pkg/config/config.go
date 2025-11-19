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
	EventBroker string

	EventWriter_Write_Topic string
	EventWriter_Read_Topics []string
	EventWriter_Group       string

	EventReader_Write_Topic string
	EventReader_Read_Topics []string
	EventReader_Group       string

	webSocket_URL         string
	WebSocket_TimeoutMs   int
	WebSocket_ReadBuffer  int
	WebSocket_WriteBuffer int
	webSocket_Port        string

	Customer_DB_URL          string
	Customer_DB_URL_Local    string
	Customer_Service_Port    string
	Customer_Write_Topic     string
	Customer_Read_Topics     []string
	Customer_Group           string
	Customer_Outbox_Interval time.Duration

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

		EventWriter_Write_Topic: getEnv("EVENT_WRITER_WRITE_TOPIC", ""),
		EventWriter_Read_Topics: getEnvArray("EVENT_WRITER_READ_TOPICS", []string{}),
		EventWriter_Group:       getEnv("EVENT_WRITER_GROUP", ""),

		EventReader_Write_Topic: getEnv("EVENT_READER_WRITE_TOPIC", ""),
		EventReader_Read_Topics: getEnvArray("EVENT_READER_READ_TOPICS", []string{}),
		EventReader_Group:       getEnv("EVENT_READER_GROUP", ""),

		webSocket_URL:         getEnv("WEBSOCKET_URL", "ws://localhost:8080/ws"),
		WebSocket_TimeoutMs:   getEnvInt("WEBSOCKET_TIMEOUT_MS", 5000),
		WebSocket_ReadBuffer:  getEnvInt("WEBSOCKET_READ_BUFFER", 1024),
		WebSocket_WriteBuffer: getEnvInt("WEBSOCKET_WRITE_BUFFER", 1024),
		webSocket_Port:        getEnv("WEBSOCKET_PORT", ":8080"),

		Customer_DB_URL:          getEnv("PSQL_CUSTOMER_DB_URL", "postgres://user:password@localhost:5432/customer_db?sslmode=disable"),
		Customer_DB_URL_Local:    getEnv("PSQL_CUSTOMER_DB_URL_LOCAL", "postgres://user:password@localhost:5432/customer_db?sslmode=disable"),
		Customer_Service_Port:    getEnv("CUSTOMER_SERVICE_PORT", ":80"),
		Customer_Write_Topic:     getEnv("CUSTOMER_WRITE_TOPIC", ""),
		Customer_Read_Topics:     getEnvArray("CUSTOMER_READ_TOPICS", []string{}),
		Customer_Group:           getEnv("CUSTOMER_GROUP", "CustomerEventGroup"),
		Customer_Outbox_Interval: getEnvTimeDuration("CUSTOMER_OUTBOX_INTERVAL", (5 * time.Second)),

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
		parts := []string{}
		for _, v := range splitAndTrim(value, ",") {
			// Only append non-empty values
			// This is a simple check, you might want to handle more complex cases
			//logging.Info("Config: %s=%s", key, v)

			if v != "" {
				parts = append(parts, v)
			}
		}
		logging.Debug("Config: %s=%v", key, parts)
		return parts
	}
	return fallback
}

// splitAndTrim splits a string by sep and trims spaces from each element.
func splitAndTrim(s, sep string) []string {
	raw := []string{}
	for _, part := range split(s, sep) {
		raw = append(raw, trim(part))
	}
	return raw
}

// split splits a string by sep.
func split(s, sep string) []string {
	return strings.Split(s, sep)
}

// trim trims spaces from a string.
func trim(s string) string {
	return strings.TrimSpace(s)
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
	return c.webSocket_URL != ""
}
func (c *Config) WebSocketURL() string {
	return c.webSocket_URL
}
func (c *Config) WebSocketTimeout() time.Duration {
	return time.Duration(c.WebSocket_TimeoutMs) * time.Millisecond
}
func (c *Config) WebSocketReadBufferSize() int {
	return c.WebSocket_ReadBuffer
}
func (c *Config) WebSocketWriteBufferSize() int {
	return c.WebSocket_WriteBuffer
}
func (c *Config) WebSocketPort() string {
	return c.webSocket_Port
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
	return c.EventWriter_Write_Topic
}
func (c *Config) GetEventWriterReadTopics() []string {
	return c.EventWriter_Read_Topics
}
func (c *Config) GetEventWriterGroup() string {
	return c.EventWriter_Group
}
func (c *Config) GetEventReaderWriteTopic() string {
	return c.EventReader_Write_Topic
}
func (c *Config) GetEventReaderReadTopics() []string {
	return c.EventReader_Read_Topics
}
func (c *Config) GetEventReaderGroup() string {
	return c.EventReader_Group
}

// Getters for Customer services
func (c *Config) GetCustomerDBURL() string {
	return c.Customer_DB_URL
}
func (c *Config) GetCustomerDBURLLocal() string {
	return c.Customer_DB_URL_Local
}
func (c *Config) GetCustomerServicePort() string {
	return c.Customer_Service_Port
}
func (c *Config) GetCustomerWriteTopic() string {
	return c.Customer_Write_Topic
}
func (c *Config) GetCustomerReadTopics() []string {
	return c.Customer_Read_Topics
}
func (c *Config) GetCustomerGroup() string {
	return c.Customer_Group
}
func (c *Config) GetCustomerOutboxInterval() time.Duration {
	return c.Customer_Outbox_Interval
}
