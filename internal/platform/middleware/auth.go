package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-template-v2/internal/features/auth"
	"gin-template-v2/internal/platform/httpx"
)

const (
	ContextUserKey    = "auth_user"
	ContextSessionKey = "auth_session"
	ContextTokenKey   = "auth_token"
)

// Authenticate verifies the session cookie and populates user and session in the gin context.
func Authenticate(svc auth.Service, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			httpx.AbortError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}

		user, session, err := svc.ValidateSession(c.Request.Context(), token)
		if err != nil {
			httpx.AbortError(c, http.StatusUnauthorized, "unauthorized", "Invalid or expired session")
			return
		}

		c.Set(ContextUserKey, user)
		c.Set(ContextSessionKey, session)
		c.Set(ContextTokenKey, token)
		c.Next()
	}
}

// CurrentUser retrieves the authenticated user from the request context.
func CurrentUser(c *gin.Context) (auth.User, bool) {
	val, ok := c.Get(ContextUserKey)
	if !ok {
		return auth.User{}, false
	}
	u, ok := val.(auth.User)
	return u, ok
}

// CurrentSession retrieves the active session from the request context.
func CurrentSession(c *gin.Context) (auth.Session, bool) {
	val, ok := c.Get(ContextSessionKey)
	if !ok {
		return auth.Session{}, false
	}
	s, ok := val.(auth.Session)
	return s, ok
}

// CurrentRawToken retrieves the raw session token from the request context.
func CurrentRawToken(c *gin.Context) (string, bool) {
	val, ok := c.Get(ContextTokenKey)
	if !ok {
		return "", false
	}
	t, ok := val.(string)
	return t, ok
}
