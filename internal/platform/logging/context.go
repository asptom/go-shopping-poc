package logging

import (
	"context"
	"log/slog"
)

type contextKey int

const (
	loggerKey contextKey = iota
)

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	logger := FromContext(ctx)
	return WithLogger(ctx, logger.With("request_id", requestID))
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	logger := FromContext(ctx)
	return WithLogger(ctx, logger.With("trace_id", traceID))
}
