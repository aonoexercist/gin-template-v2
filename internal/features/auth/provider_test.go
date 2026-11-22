package auth_test

import (
	"context"
	"errors"
	"testing"

	"gin-template-v2/internal/features/auth"
)

type mockVerifier struct {
	profile *auth.OAuthProfile
	err     error
}

func (m *mockVerifier) Verify(ctx context.Context, token string) (*auth.OAuthProfile, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.profile, nil
}

func TestProviderRegistry(t *testing.T) {
	reg := auth.NewProviderRegistry()

	mockGoogle := &mockVerifier{
		profile: &auth.OAuthProfile{
			Provider:       "google",
			ProviderUserID: "sub-123",
			Email:          "user@example.com",
			Name:           "Test User",
		},
	}

	reg.Register("google", mockGoogle)

	// Test case-insensitivity
	for _, name := range []string{"google", "Google", "GOOGLE", " google "} {
		v, err := reg.Get(name)
		if err != nil {
			t.Fatalf("expected provider %q to be found, got error: %v", name, err)
		}
		profile, err := v.Verify(context.Background(), "dummy-token")
		if err != nil {
			t.Fatalf("unexpected verify error: %v", err)
		}
		if profile.Email != "user@example.com" {
			t.Errorf("got email %q, want user@example.com", profile.Email)
		}
	}

	// Test unsupported provider
	_, err := reg.Get("github")
	if !errors.Is(err, auth.ErrUnsupportedProvider) {
		t.Fatalf("expected ErrUnsupportedProvider, got %v", err)
	}
}

func TestGoogleVerifier_EmptyToken(t *testing.T) {
	v := auth.NewGoogleVerifier("test-client-id")
	_, err := v.Verify(context.Background(), "")
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for empty token, got %v", err)
	}
}
