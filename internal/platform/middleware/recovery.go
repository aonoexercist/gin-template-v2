package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"gin-template-v2/internal/platform/httpx"
	"gin-template-v2/internal/platform/logger"
)

// Recovery turns panics into a logged 500 JSON response. Place it after
// RequestLogger so the panic is logged with the request_id.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec) // intentional abort; let net/http handle it
			}
			logger.FromContext(c.Request.Context()).Error("panic recovered",
				"panic", fmt.Sprint(rec),
				"stack", string(debug.Stack()),
			)
			httpx.AbortError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
		}()
		c.Next()
	}
}
