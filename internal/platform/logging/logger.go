package logging

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type LoggerConfig struct {
	ServiceName string
	Level       string
	Format      string
}

func DefaultLoggerConfig(serviceName string) LoggerConfig {
	return LoggerConfig{
		ServiceName: serviceName,
		Level:       getEnv("LOG_LEVEL", "info"),
		Format:      getEnv("LOG_FORMAT", "json"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

type LoggerProvider struct {
	logger *slog.Logger
}

func NewLoggerProvider(config LoggerConfig) (*LoggerProvider, error) {
	if config.ServiceName == "" {
		return nil, fmt.Errorf("service name is required")
	}

	level := parseLevel(config.Level)
	format := config.Format

	var handler slog.Handler

	handlerOptions := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "request_id" || a.Key == "trace_id" {
				return slog.String("context."+a.Key, a.Value.String())
			}
			return a
		},
	}

	if format == "text" || os.Getenv("ENVIRONMENT") == "development" {
		handler = slog.NewTextHandler(os.Stdout, handlerOptions)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, handlerOptions)
	}

	logger := slog.New(handler).With("service", config.ServiceName)

	return &LoggerProvider{logger: logger}, nil
}

func (p *LoggerProvider) Logger() *slog.Logger {
	return p.logger
}

func (p *LoggerProvider) With(attrs ...any) *slog.Logger {
	return p.logger.With(attrs...)
}

func parseLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
