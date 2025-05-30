package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	KafkaBroker          string
	KafkaTopic           string
	KafkaGroupID         string
	webSocketURL         string
	WebSocketTimeoutMs   int
	WebSocketReadBuffer  int
	WebSocketWriteBuffer int
	webSocketPort        string
}

func Load(envFile string) *Config {
	_ = godotenv.Load(envFile) // Loads the specified .env file

	return &Config{
		KafkaBroker:          getEnv("KAFKA_BROKER", "localhost:9092"),
		KafkaTopic:           getEnv("KAFKA_TOPIC", "orders"),
		KafkaGroupID:         getEnv("KAFKA_GROUP_ID", "order-service"),
		webSocketURL:         getEnv("WEBSOCKET_URL", "ws://localhost:8080/ws"),
		WebSocketTimeoutMs:   getEnvInt("WEBSOCKET_TIMEOUT_MS", 5000),
		WebSocketReadBuffer:  getEnvInt("WEBSOCKET_READ_BUFFER", 1024),
		WebSocketWriteBuffer: getEnvInt("WEBSOCKET_WRITE_BUFFER", 1024),
		webSocketPort:        getEnv("WEBSOCKET_PORT", ":8080"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
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

func (c *Config) WebSocketURL() string {
	return c.webSocketURL
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
func (c *Config) WebSocketPort() string {
	return c.webSocketPort
}
