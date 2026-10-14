// Package middleware contains the cross-cutting Gin middleware.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gin-template-v2/internal/platform/httpx"
)

const HeaderRequestID = "X-Request-ID"

// RequestID propagates an incoming X-Request-ID (if sane) or generates one,
// stores it in the gin context and echoes it on the response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if !validRequestID(id) {
			id = uuid.NewString()
		}
		c.Set(httpx.RequestIDKey, id)
		c.Header(HeaderRequestID, id)
		c.Next()
	}
}

// validRequestID guards against log injection and oversized values.
func validRequestID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}
