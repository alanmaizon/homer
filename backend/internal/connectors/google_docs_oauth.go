package connectors

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
)

var ErrOAuthUnavailable = errors.New("google docs oauth unavailable")
var ErrOAuthStateInvalid = errors.New("google docs oauth state is invalid")
var ErrOAuthExchangeFailed = errors.New("google docs oauth code exchange failed")

const defaultGoogleOAuthStateTTL = 10 * time.Minute

type GoogleDocsAuthStart struct {
	SessionKey     string
	AuthURL        string
	StateExpiresAt time.Time
}

type GoogleDocsAuthCallback struct {
	SessionKey string
	ExpiresAt  *time.Time
}

type GoogleDocsOAuthManager struct {
	config       *oauth2.Config
	store        OAuthTokenStore
	now          func() time.Time
	stateTTL     time.Duration
	randomString func(int) (string, error)
}

func NewGoogleDocsOAuthManagerFromEnv(store OAuthTokenStore) (*GoogleDocsOAuthManager, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: token store is not configured", ErrOAuthUnavailable)
	}

	config, err := newGoogleDocsOAuthConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUnavailable, err)
	}

	return &GoogleDocsOAuthManager{
		config:       config,
		store:        store,
		now:          time.Now,
		stateTTL:     googleOAuthStateTTLFromEnv(),
		randomString: secureRandomString,
	}, nil
}

func (m *GoogleDocsOAuthManager) StartAuth() (GoogleDocsAuthStart, error) {
	if m == nil || m.config == nil || m.store == nil {
		return GoogleDocsAuthStart{}, fmt.Errorf("%w: manager is not initialized", ErrOAuthUnavailable)
	}

	sessionKey, err := m.randomString(32)
	if err != nil {
		return GoogleDocsAuthStart{}, fmt.Errorf("%w: %v", ErrOAuthUnavailable, err)
	}

	state, err := m.randomString(32)
	if err != nil {
		return GoogleDocsAuthStart{}, fmt.Errorf("%w: %v", ErrOAuthUnavailable, err)
	}

	stateExpiresAt := m.now().Add(m.stateTTL)
	m.store.SaveState(state, sessionKey, stateExpiresAt)

	authURL := m.config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	return GoogleDocsAuthStart{
		SessionKey:     sessionKey,
		AuthURL:        authURL,
		StateExpiresAt: stateExpiresAt,
	}, nil
}

func (m *GoogleDocsOAuthManager) CompleteAuth(ctx context.Context, state string, code string) (GoogleDocsAuthCallback, error) {
	if m == nil || m.config == nil || m.store == nil {
		return GoogleDocsAuthCallback{}, fmt.Errorf("%w: manager is not initialized", ErrOAuthUnavailable)
	}

	state = strings.TrimSpace(state)
	code = strings.TrimSpace(code)
	if state == "" {
		return GoogleDocsAuthCallback{}, ErrOAuthStateInvalid
	}
	if code == "" {
		return GoogleDocsAuthCallback{}, fmt.Errorf("%w: missing code", ErrOAuthExchangeFailed)
	}

	sessionKey, ok := m.store.ConsumeState(state, m.now())
	if !ok {
		return GoogleDocsAuthCallback{}, ErrOAuthStateInvalid
	}

	token, err := m.config.Exchange(ctx, code)
	if err != nil {
		return GoogleDocsAuthCallback{}, fmt.Errorf("%w: %v", ErrOAuthExchangeFailed, err)
	}
	m.store.SaveToken(sessionKey, token)

	result := GoogleDocsAuthCallback{
		SessionKey: sessionKey,
	}
	if !token.Expiry.IsZero() {
		expiresAt := token.Expiry.UTC()
		result.ExpiresAt = &expiresAt
	}

	return result, nil
}

func hasGoogleDocsOAuthConfig() bool {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"))
	redirectURL := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"))
	return clientID != "" && clientSecret != "" && redirectURL != ""
}

func newGoogleDocsOAuthConfigFromEnv() (*oauth2.Config, error) {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"))
	redirectURL := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"))

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, errors.New("GOOGLE_OAUTH_CLIENT_ID, GOOGLE_OAUTH_CLIENT_SECRET, and GOOGLE_OAUTH_REDIRECT_URL are required")
	}

	scopes := parseGoogleOAuthScopes(os.Getenv("GOOGLE_OAUTH_SCOPES"))
	endpoint := google.Endpoint

	if authURL := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_AUTH_URL")); authURL != "" {
		endpoint.AuthURL = authURL
	}
	if tokenURL := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_TOKEN_URL")); tokenURL != "" {
		endpoint.TokenURL = tokenURL
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}, nil
}

func parseGoogleOAuthScopes(rawScopes string) []string {
	cleaned := strings.TrimSpace(rawScopes)
	if cleaned == "" {
		return []string{docs.DocumentsScope}
	}

	parts := strings.FieldsFunc(cleaned, func(r rune) bool {
		return r == ',' || r == ' '
	})
	unique := make(map[string]struct{}, len(parts))
	scopes := make([]string, 0, len(parts))

	for _, part := range parts {
		scope := strings.TrimSpace(part)
		if scope == "" {
			continue
		}
		if _, exists := unique[scope]; exists {
			continue
		}
		unique[scope] = struct{}{}
		scopes = append(scopes, scope)
	}

	if len(scopes) == 0 {
		return []string{docs.DocumentsScope}
	}
	return scopes
}

func googleOAuthStateTTLFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_STATE_TTL"))
	if raw == "" {
		return defaultGoogleOAuthStateTTL
	}

	ttl, err := time.ParseDuration(raw)
	if err != nil || ttl <= 0 {
		return defaultGoogleOAuthStateTTL
	}
	return ttl
}

func secureRandomString(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("length must be positive")
	}

	data := make([]byte, length)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}
