package logging

import (
	"log/slog"
	"time"
)

func ErrorAttr(err error) slog.Attr {
	return slog.String("error", err.Error())
}

func ErrorStackAttr(err error) slog.Attr {
	if st, ok := err.(interface{ StackTrace() string }); ok {
		return slog.String("stack_trace", st.StackTrace())
	}
	return slog.String("error", err.Error())
}

func DurationAttr(d time.Duration) slog.Attr {
	return slog.Duration("duration", d)
}

func Int64Attr(key string, value int64) slog.Attr {
	return slog.Int64(key, value)
}

func Uint64Attr(key string, value uint64) slog.Attr {
	return slog.Uint64(key, value)
}
