package logging

import (
	"context"
	"log/slog"
)

func IsDebugEnabled(logger *slog.Logger, ctx context.Context) bool {
	return logger.Enabled(ctx, slog.LevelDebug)
}

func IsInfoEnabled(logger *slog.Logger, ctx context.Context) bool {
	return logger.Enabled(ctx, slog.LevelInfo)
}

func IsWarnEnabled(logger *slog.Logger, ctx context.Context) bool {
	return logger.Enabled(ctx, slog.LevelWarn)
}

func IsErrorEnabled(logger *slog.Logger, ctx context.Context) bool {
	return logger.Enabled(ctx, slog.LevelError)
}
