package providers

import (
	"log/slog"
	"os"
)

var (
	logger *slog.Logger
)

func init() {
	logger = slog.New(slog.NewJSONHandler(os.Stderr, nil)).
		With("platform", "outbox_provider")
}

func Logger() *slog.Logger {
	return logger
}

func SetLogger(l *slog.Logger) {
	logger = l
}
