package connectors

import (
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type OAuthTokenStore interface {
	SaveState(state string, sessionKey string, expiresAt time.Time)
	ConsumeState(state string, now time.Time) (sessionKey string, ok bool)
	SaveToken(sessionKey string, token *oauth2.Token)
	Token(sessionKey string) (*oauth2.Token, bool)
}

type InMemoryOAuthTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*oauth2.Token
	states map[string]oauthState
}

type oauthState struct {
	sessionKey string
	expiresAt  time.Time
}

func NewInMemoryOAuthTokenStore() *InMemoryOAuthTokenStore {
	return &InMemoryOAuthTokenStore{
		tokens: make(map[string]*oauth2.Token),
		states: make(map[string]oauthState),
	}
}

func (s *InMemoryOAuthTokenStore) SaveState(state string, sessionKey string, expiresAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[state] = oauthState{
		sessionKey: sessionKey,
		expiresAt:  expiresAt,
	}
}

func (s *InMemoryOAuthTokenStore) ConsumeState(state string, now time.Time) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.states[state]
	if !ok {
		return "", false
	}
	delete(s.states, state)

	if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
		return "", false
	}

	return entry.sessionKey, true
}

func (s *InMemoryOAuthTokenStore) SaveToken(sessionKey string, token *oauth2.Token) {
	if token == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	copied := copyToken(token)
	if existing, ok := s.tokens[sessionKey]; ok {
		// Google may omit refresh token on subsequent exchanges/refresh responses.
		if copied.RefreshToken == "" {
			copied.RefreshToken = existing.RefreshToken
		}
	}
	s.tokens[sessionKey] = copied
}

func (s *InMemoryOAuthTokenStore) Token(sessionKey string) (*oauth2.Token, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, ok := s.tokens[sessionKey]
	if !ok {
		return nil, false
	}
	return copyToken(token), true
}

func copyToken(token *oauth2.Token) *oauth2.Token {
	if token == nil {
		return nil
	}
	copied := *token
	return &copied
}

var oauthStore OAuthTokenStore = NewInMemoryOAuthTokenStore()

func OAuthStore() OAuthTokenStore {
	return oauthStore
}

func SetOAuthStoreForTests(store OAuthTokenStore) func() {
	previous := oauthStore
	if store == nil {
		store = NewInMemoryOAuthTokenStore()
	}
	oauthStore = store
	return func() {
		oauthStore = previous
	}
}
