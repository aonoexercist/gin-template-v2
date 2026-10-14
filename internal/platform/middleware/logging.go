package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"gin-template-v2/internal/platform/httpx"
	"gin-template-v2/internal/platform/logger"
)

// RequestLogger attaches a request-scoped logger (with request_id) to the
// request context and emits one structured access-log line per request.
// Paths in skipPaths (e.g. health probes) are not logged.
// Must run after RequestID.
func RequestLogger(base *slog.Logger, skipPaths ...string) gin.HandlerFunc {
	skip := make(map[string]struct{}, len(skipPaths))
	for _, p := range skipPaths {
		skip[p] = struct{}{}
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		reqLog := base.With("request_id", c.GetString(httpx.RequestIDKey))
		c.Request = c.Request.WithContext(logger.WithContext(c.Request.Context(), reqLog))

		c.Next()

		if _, ok := skip[path]; ok {
			return
		}

		status := c.Writer.Status()
		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"route", c.FullPath(),
			"status", status,
			"latency_ms", float64(time.Since(start).Microseconds()) / 1000,
			"bytes", c.Writer.Size(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		switch {
		case status >= 500:
			reqLog.Error("http request", attrs...)
		case status >= 400:
			reqLog.Warn("http request", attrs...)
		default:
			reqLog.Info("http request", attrs...)
		}
	}
}
