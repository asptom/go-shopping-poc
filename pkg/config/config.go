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
	KafkaBroker string
	// KafkaUsername           string
	// KafkaPassword           string
	KafkaTopic_EventExample string
	KafkaGroup_EventExample string

	EventWriter_KafkaWriteTopics []string
	EventWriter_KafkaReadTopics  []string
	EventWriter_KafkaGroupID     string
	EventReader_KafkaWriteTopics []string
	EventReader_KafkaReadTopics  []string
	EventReader_KafkaGroupID     string

	webSocketURL         string
	WebSocketTimeoutMs   int
	WebSocketReadBuffer  int
	WebSocketWriteBuffer int
	webSocketPort        string
}

func Load(envFile string) *Config {
	_ = godotenv.Load(envFile) // Loads the specified .env file

	return &Config{
		KafkaBroker: getEnv("KAFKA_BROKER", "localhost:9092"),
		// KafkaUsername:                getEnv("KAFKA_USERNAME", ""),
		// KafkaPassword:                getEnv("KAFKA_PASSWORD", ""),
		KafkaTopic_EventExample:      getEnv("KAFKA_TOPIC_EVENT_EXAMPLE", "EventExampleTopic"),
		KafkaGroup_EventExample:      getEnv("KAFKA_GROUP_EVENT_EXAMPLE", "event-example-group"),
		EventWriter_KafkaWriteTopics: getEnvArray("EVENT_WRITER_KAFKA_WRITE_TOPICS", []string{"WriteTopic1", "WriteTopic2"}),
		EventWriter_KafkaReadTopics:  getEnvArray("EVENT_WRITER_KAFKA_READ_TOPICS", []string{}),
		EventWriter_KafkaGroupID:     getEnv("EVENT_EXAMPLE_KAFKA_GROUP_ID", "event-example-writer-group"),
		EventReader_KafkaWriteTopics: getEnvArray("EVENT_READER_KAFKA_WRITE_TOPICS", []string{}),
		EventReader_KafkaReadTopics:  getEnvArray("EVENT_READER_KAFKA_READ_TOPICS", []string{"ReadTopic1", "ReadTopic2"}),
		EventReader_KafkaGroupID:     getEnv("EVENT_EXAMPLE_KAFKA_GROUP_ID", "event-example-reader-group"),
		webSocketURL:                 getEnv("WEBSOCKET_URL", "ws://localhost:8080/ws"),
		WebSocketTimeoutMs:           getEnvInt("WEBSOCKET_TIMEOUT_MS", 5000),
		WebSocketReadBuffer:          getEnvInt("WEBSOCKET_READ_BUFFER", 1024),
		WebSocketWriteBuffer:         getEnvInt("WEBSOCKET_WRITE_BUFFER", 1024),
		webSocketPort:                getEnv("WEBSOCKET_PORT", ":8080"),
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
			logging.Info("Config: %s=%s", key, v)

			if v != "" {
				parts = append(parts, v)
			}
		}
		logging.Info("Config: %s=%v", key, parts)
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

// Getters for the Websocket configuration

func (c *Config) WebSocketEnabled() bool {
	return c.webSocketURL != ""
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

// Getters for the Kafka configuration

func (c *Config) GetKafkaBroker() string {
	return c.KafkaBroker
}

// func (c *Config) GetKafkaUsername() string {
// 	return c.KafkaUsername
// }
// func (c *Config) GetKafkaPassword() string {
// 	return c.KafkaPassword
// }

// Getters for event examples
func (c *Config) GetKafkaTopicEventExample() string {
	return c.KafkaTopic_EventExample
}
func (c *Config) GetKafkaGroupEventExample() string {
	return c.KafkaGroup_EventExample
}

// Getters for event writer and reader Kafka topics
func (c *Config) GetEventWriterKafkaWriteTopics() []string {
	return c.EventWriter_KafkaWriteTopics
}
func (c *Config) GetEventWriterKafkaReadTopics() []string {
	return c.EventWriter_KafkaReadTopics
}
func (c *Config) GetEventWriterKafkaGroupId() string {
	return c.EventWriter_KafkaGroupID
}
func (c *Config) GetEventReaderKafkaWriteTopics() []string {
	return c.EventReader_KafkaWriteTopics
}
func (c *Config) GetEventReaderKafkaReadTopics() []string {
	return c.EventReader_KafkaReadTopics
}
func (c *Config) GetEventReaderKafkaGroupId() string {
	return c.EventReader_KafkaGroupID
}
