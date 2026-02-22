package connectors

import (
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestInMemoryOAuthTokenStoreConsumeState(t *testing.T) {
	store := NewInMemoryOAuthTokenStore()
	now := time.Now()

	store.SaveState("state-1", "session-1", now.Add(time.Minute))

	sessionKey, ok := store.ConsumeState("state-1", now)
	if !ok {
		t.Fatalf("expected state to be consumed")
	}
	if sessionKey != "session-1" {
		t.Fatalf("expected session-1, got %q", sessionKey)
	}

	if _, ok := store.ConsumeState("state-1", now); ok {
		t.Fatalf("expected state to be one-time use")
	}
}

func TestInMemoryOAuthTokenStoreConsumeStateExpired(t *testing.T) {
	store := NewInMemoryOAuthTokenStore()
	now := time.Now()

	store.SaveState("state-1", "session-1", now.Add(-time.Second))

	if _, ok := store.ConsumeState("state-1", now); ok {
		t.Fatalf("expected expired state to fail")
	}
}

func TestInMemoryOAuthTokenStorePreservesRefreshToken(t *testing.T) {
	store := NewInMemoryOAuthTokenStore()

	store.SaveToken("session-1", &oauth2.Token{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
	})
	store.SaveToken("session-1", &oauth2.Token{
		AccessToken: "access-2",
	})

	token, ok := store.Token("session-1")
	if !ok {
		t.Fatalf("expected token to be present")
	}
	if token.AccessToken != "access-2" {
		t.Fatalf("expected refreshed access token access-2, got %q", token.AccessToken)
	}
	if token.RefreshToken != "refresh-1" {
		t.Fatalf("expected refresh token to be preserved, got %q", token.RefreshToken)
	}
}
