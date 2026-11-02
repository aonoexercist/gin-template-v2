package auth

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/idtoken"
)

// OAuthProfile represents normalized user information extracted from any OAuth/OIDC provider.
type OAuthProfile struct {
	Provider       string
	ProviderUserID string
	Email          string
	Name           string
	AvatarURL      *string
}

// TokenVerifier defines the contract for verifying an identity token from an authentication provider.
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (*OAuthProfile, error)
}

// ProviderRegistry manages token verifiers for different identity providers (Google, GitHub, Apple, etc.).
type ProviderRegistry struct {
	verifiers map[string]TokenVerifier
}

// NewProviderRegistry initializes an empty provider registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		verifiers: make(map[string]TokenVerifier),
	}
}

// Register adds a verifier for a specific provider name (e.g. "google", "github").
func (r *ProviderRegistry) Register(provider string, verifier TokenVerifier) {
	r.verifiers[strings.ToLower(strings.TrimSpace(provider))] = verifier
}

// Get retrieves the verifier for the requested provider.
func (r *ProviderRegistry) Get(provider string) (TokenVerifier, error) {
	v, ok := r.verifiers[strings.ToLower(strings.TrimSpace(provider))]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedProvider, provider)
	}
	return v, nil
}

// GoogleVerifier validates Google ID tokens against Google's public keys.
type GoogleVerifier struct {
	clientID string
}

// NewGoogleVerifier creates a GoogleVerifier. If clientID is non-empty, audience is enforced.
func NewGoogleVerifier(clientID string) *GoogleVerifier {
	return &GoogleVerifier{clientID: strings.TrimSpace(clientID)}
}

// Verify validates a Google ID token JWT and parses the profile.
func (g *GoogleVerifier) Verify(ctx context.Context, token string) (*OAuthProfile, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrInvalidToken
	}

	payload, err := idtoken.Validate(ctx, token, g.clientID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if verified, ok := payload.Claims["email_verified"]; ok && !isEmailVerified(verified) {
		return nil, ErrEmailNotVerified
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, fmt.Errorf("%w: missing email in token", ErrInvalidToken)
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	var avatarURL *string
	picture = strings.TrimSpace(picture)
	if picture != "" {
		avatarURL = &picture
	}

	return &OAuthProfile{
		Provider:       "google",
		ProviderUserID: payload.Subject,
		Email:          email,
		Name:           name,
		AvatarURL:      avatarURL,
	}, nil
}

func isEmailVerified(val any) bool {
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	default:
		return false
	}
}
