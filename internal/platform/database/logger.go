package database

import (
	"log/slog"
	"os"
)

var (
	logger *slog.Logger
)

func init() {
	logger = slog.New(slog.NewJSONHandler(os.Stderr, nil)).
		With("platform", "database", "component", "postgresql")
}

func Logger() *slog.Logger {
	return logger
}

func SetLogger(l *slog.Logger) {
	logger = l
}

func InitLogger(serviceName string, level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	})).With("platform", "database", "component", "postgresql", "service", serviceName)
}
