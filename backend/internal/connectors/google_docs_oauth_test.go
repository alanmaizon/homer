package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestGoogleDocsOAuthStartAuth(t *testing.T) {
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "http://localhost/callback")
	t.Setenv("GOOGLE_OAUTH_AUTH_URL", "https://accounts.example.com/auth")
	t.Setenv("GOOGLE_OAUTH_TOKEN_URL", "https://accounts.example.com/token")

	store := NewInMemoryOAuthTokenStore()
	manager, err := NewGoogleDocsOAuthManagerFromEnv(store)
	if err != nil {
		t.Fatalf("expected manager, got error: %v", err)
	}

	result, err := manager.StartAuth()
	if err != nil {
		t.Fatalf("expected start success, got error: %v", err)
	}

	if result.SessionKey == "" {
		t.Fatalf("expected session key to be generated")
	}
	if result.AuthURL == "" {
		t.Fatalf("expected auth url to be generated")
	}

	parsed, err := url.Parse(result.AuthURL)
	if err != nil {
		t.Fatalf("expected valid auth url, got error: %v", err)
	}
	if parsed.Host != "accounts.example.com" {
		t.Fatalf("unexpected auth host: %s", parsed.Host)
	}
	state := strings.TrimSpace(parsed.Query().Get("state"))
	if state == "" {
		t.Fatalf("expected state query parameter in auth url")
	}
	if _, ok := store.ConsumeState(state, time.Now()); !ok {
		t.Fatalf("expected generated state to be stored")
	}
}

func TestGoogleDocsOAuthCompleteAuth(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-1",
			"refresh_token": "refresh-1",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "http://localhost/callback")
	t.Setenv("GOOGLE_OAUTH_AUTH_URL", tokenServer.URL+"/auth")
	t.Setenv("GOOGLE_OAUTH_TOKEN_URL", tokenServer.URL+"/token")

	store := NewInMemoryOAuthTokenStore()
	manager, err := NewGoogleDocsOAuthManagerFromEnv(store)
	if err != nil {
		t.Fatalf("expected manager, got error: %v", err)
	}

	store.SaveState("state-1", "session-1", time.Now().Add(time.Minute))

	result, err := manager.CompleteAuth(context.Background(), "state-1", "code-1")
	if err != nil {
		t.Fatalf("expected complete auth success, got error: %v", err)
	}
	if result.SessionKey != "session-1" {
		t.Fatalf("expected session-1, got %q", result.SessionKey)
	}

	token, ok := store.Token("session-1")
	if !ok {
		t.Fatalf("expected token to be stored")
	}
	if token.AccessToken != "access-1" {
		t.Fatalf("expected access-1, got %q", token.AccessToken)
	}
	if token.RefreshToken != "refresh-1" {
		t.Fatalf("expected refresh-1, got %q", token.RefreshToken)
	}
}

func TestGoogleDocsOAuthCompleteAuthInvalidState(t *testing.T) {
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "http://localhost/callback")

	store := NewInMemoryOAuthTokenStore()
	manager, err := NewGoogleDocsOAuthManagerFromEnv(store)
	if err != nil {
		t.Fatalf("expected manager, got error: %v", err)
	}

	_, err = manager.CompleteAuth(context.Background(), "missing", "code")
	if err == nil || !strings.Contains(err.Error(), ErrOAuthStateInvalid.Error()) {
		t.Fatalf("expected invalid state error, got %v", err)
	}
}

func TestSessionTokenSourceRefreshesAndPersists(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "refreshed-access",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "http://localhost/callback")
	t.Setenv("GOOGLE_OAUTH_AUTH_URL", tokenServer.URL+"/auth")
	t.Setenv("GOOGLE_OAUTH_TOKEN_URL", tokenServer.URL+"/token")

	store := NewInMemoryOAuthTokenStore()
	store.SaveToken("session-1", &oauth2.Token{
		AccessToken:  "expired-access",
		RefreshToken: "refresh-1",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Minute),
	})

	source, ok := newSessionTokenSource(context.Background(), "session-1", store)
	if !ok {
		t.Fatalf("expected token source for session")
	}

	token, err := source.Token()
	if err != nil {
		t.Fatalf("expected token refresh success, got error: %v", err)
	}
	if token.AccessToken != "refreshed-access" {
		t.Fatalf("expected refreshed-access, got %q", token.AccessToken)
	}

	stored, ok := store.Token("session-1")
	if !ok {
		t.Fatalf("expected token to remain stored")
	}
	if stored.AccessToken != "refreshed-access" {
		t.Fatalf("expected persisted refreshed token, got %q", stored.AccessToken)
	}
	if stored.RefreshToken != "refresh-1" {
		t.Fatalf("expected refresh token to be preserved, got %q", stored.RefreshToken)
	}
}
