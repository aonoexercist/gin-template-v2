// Package logger builds the application's structured logger (log/slog) and
// carries a request-scoped logger through context.Context.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

// New creates a slog.Logger. level: debug|info|warn|error; format: text|json.
func New(level, format string, w io.Writer) (*slog.Logger, error) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info", "":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log level %q", level)
	}

	opts := &slog.HandlerOptions{Level: lvl}
	switch format {
	case "json", "":
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	case "text":
		return slog.New(slog.NewTextHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("invalid log format %q", format)
	}
}

type ctxKey struct{}

// WithContext returns a context carrying l.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the request-scoped logger, or slog.Default() if none.
// Use it in handlers/services so every line carries the request_id.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}
