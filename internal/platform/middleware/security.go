package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecureHeaders sets conservative security headers suitable for a JSON API.
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Next()
	}
}

// CORS allows the listed origins ("*" allows any). With an empty list no CORS
// headers are emitted, i.e. browsers enforce same-origin.
func CORS(allowed []string) gin.HandlerFunc {
	allowAll := false
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		if o == "*" {
			allowAll = true
		}
		set[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" || len(allowed) == 0 {
			c.Next()
			return
		}
		if _, ok := set[origin]; !ok && !allowAll {
			c.Next()
			return
		}

		h := c.Writer.Header()
		if allowAll {
			h.Set("Access-Control-Allow-Origin", "*")
		} else {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Add("Vary", "Origin")
		}
		h.Set("Access-Control-Expose-Headers", HeaderRequestID)

		if c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != "" {
			h.Set("Access-Control-Allow-Methods", strings.Join([]string{
				http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions,
			}, ", "))
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, "+HeaderRequestID)
			h.Set("Access-Control-Max-Age", "600")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
