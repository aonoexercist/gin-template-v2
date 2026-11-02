package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gin-template-v2/internal/config"
	"gin-template-v2/internal/platform/httpx"
)

// Handler handles HTTP requests for authentication and session lifecycle.
type Handler struct {
	svc          Service
	cookieName   string
	cookieDomain string
	cookieSecure bool
	sameSite     http.SameSite
	maxAge       time.Duration
}

// NewHandler creates a new auth Handler.
func NewHandler(svc Service, cfg config.AuthConfig) *Handler {
	return &Handler{
		svc:          svc,
		cookieName:   cfg.CookieName,
		cookieDomain: cfg.CookieDomain,
		cookieSecure: cfg.CookieSecure,
		sameSite:     parseSameSite(cfg.CookieSameSite),
		maxAge:       cfg.SessionDuration,
	}
}

// LoginGoogle authenticates a user using a Google ID token.
func (h *Handler) LoginGoogle(c *gin.Context) {
	h.handleProviderLogin(c, "google")
}

// Login authenticates a user using email and password.
func (h *Handler) Login(c *gin.Context) {
	var req EmailLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}

	clientInfo := ClientInfo{
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	}

	rawToken, session, user, err := h.svc.LoginWithPassword(c.Request.Context(), req.Email, req.Password, clientInfo)
	if err != nil {
		h.fail(c, err)
		return
	}

	h.setSessionCookie(c, rawToken)

	httpx.JSON(c, http.StatusOK, AuthResponse{
		User:    newUserResponse(user),
		Session: newSessionResponse(session),
	})
}

// LoginProvider authenticates a user using any registered provider (e.g. google, github).
func (h *Handler) LoginProvider(c *gin.Context) {
	provider := c.Param("provider")
	h.handleProviderLogin(c, provider)
}

func (h *Handler) handleProviderLogin(c *gin.Context, provider string) {
	var req OAuthLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}

	clientInfo := ClientInfo{
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	}

	rawToken, session, user, err := h.svc.LoginWithProvider(c.Request.Context(), provider, req.Token, clientInfo)
	if err != nil {
		h.fail(c, err)
		return
	}

	h.setSessionCookie(c, rawToken)

	httpx.JSON(c, http.StatusOK, AuthResponse{
		User:    newUserResponse(user),
		Session: newSessionResponse(session),
	})
}

// Me returns the currently authenticated user's profile and active session.
func (h *Handler) Me(c *gin.Context) {
	userVal, ok := c.Get("auth_user")
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	user := userVal.(User)

	sessVal, _ := c.Get("auth_session")
	var sess Session
	if sessVal != nil {
		sess = sessVal.(Session)
	}

	httpx.JSON(c, http.StatusOK, AuthResponse{
		User:    newUserResponse(user),
		Session: newSessionResponse(sess),
	})
}

// Logout invalidates the current session and clears the cookie.
func (h *Handler) Logout(c *gin.Context) {
	rawToken, _ := c.Get("auth_token")
	if tokenStr, ok := rawToken.(string); ok && tokenStr != "" {
		_ = h.svc.Logout(c.Request.Context(), tokenStr)
	}
	h.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

// LogoutAll invalidates all active sessions for the authenticated user and clears the cookie.
func (h *Handler) LogoutAll(c *gin.Context) {
	userVal, ok := c.Get("auth_user")
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	user := userVal.(User)

	if err := h.svc.RevokeAllSessions(c.Request.Context(), user.ID); err != nil {
		h.fail(c, err)
		return
	}

	h.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

// ListSessions lists all active sessions for the authenticated user.
func (h *Handler) ListSessions(c *gin.Context) {
	userVal, ok := c.Get("auth_user")
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	user := userVal.(User)

	rawToken, _ := c.Get("auth_token")
	tokenStr, _ := rawToken.(string)

	sessions, err := h.svc.ListSessions(c.Request.Context(), user.ID, tokenStr)
	if err != nil {
		h.fail(c, err)
		return
	}

	out := make([]SessionResponse, len(sessions))
	for i, s := range sessions {
		out[i] = newSessionResponse(s)
	}

	httpx.JSON(c, http.StatusOK, out)
}

// RevokeSession revokes a specific session by ID.
func (h *Handler) RevokeSession(c *gin.Context) {
	userVal, ok := c.Get("auth_user")
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	user := userVal.(User)

	sessionID, ok := parseUUID(c, "id")
	if !ok {
		return
	}

	// Check if user is revoking their current session
	sessVal, _ := c.Get("auth_session")
	if sessVal != nil {
		currentSess := sessVal.(Session)
		if currentSess.ID == sessionID {
			h.clearSessionCookie(c)
		}
	}

	if err := h.svc.RevokeSession(c.Request.Context(), sessionID, user.ID); err != nil {
		h.fail(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) setSessionCookie(c *gin.Context, rawToken string) {
	c.SetSameSite(h.sameSite)
	c.SetCookie(
		h.cookieName,
		rawToken,
		int(h.maxAge.Seconds()),
		"/",
		h.cookieDomain,
		h.cookieSecure,
		true, // HttpOnly
	)
}

func (h *Handler) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(h.sameSite)
	c.SetCookie(
		h.cookieName,
		"",
		-1,
		"/",
		h.cookieDomain,
		h.cookieSecure,
		true, // HttpOnly
	)
}

func parseSameSite(s string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return http.SameSiteLaxMode
	}
}
