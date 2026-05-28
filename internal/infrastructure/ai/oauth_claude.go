package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"git.pauloholanda.com.br/paulorh/sdk-claude-oauth-go/oauth"
	"git.pauloholanda.com.br/paulorh/sdk-claude-oauth-go/pkce"
	"git.pauloholanda.com.br/paulorh/sdk-claude-oauth-go/storage"
)

const pendingAuthTTL = 15 * time.Minute

// OAuthTokens is the result of an authorization-code exchange or refresh.
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// ClaudeOAuth drives the Claude subscription OAuth flow via the claude-oauth-go
// SDK. Pending verifiers are held in memory keyed by state; they are short-lived
// and safe to lose on restart since the flow completes within minutes.
type ClaudeOAuth struct {
	client  *oauth.Client
	mu      sync.Mutex
	pending map[string]pendingAuth
}

type pendingAuth struct {
	verifier  string
	createdAt time.Time
}

func NewClaudeOAuth() *ClaudeOAuth {
	store := &memoryStore{}
	client, _ := oauth.New(oauth.Config{Store: store})
	return &ClaudeOAuth{
		client:  client,
		pending: make(map[string]pendingAuth),
	}
}

// BuildAuthorizeURL generates a PKCE challenge, stores the verifier keyed by the
// returned state, and returns the URL the user must open to authorize.
func (o *ClaudeOAuth) BuildAuthorizeURL() (authURL, state string, err error) {
	state = pkce.GenerateState()
	verifier := pkce.GenerateCodeVerifier()
	o.storePending(state, verifier)
	authURL = o.client.AuthorizationURL(state, verifier)
	return authURL, state, nil
}

// Exchange swaps an authorization code for tokens. The verifier is looked up by
// the state issued in BuildAuthorizeURL.
func (o *ClaudeOAuth) Exchange(state, pastedCode string) (*OAuthTokens, error) {
	verifier, ok := o.take(state)
	if !ok {
		return nil, fmt.Errorf("ai: unknown or expired oauth state")
	}

	token, err := o.client.Exchange(context.Background(), pastedCode, verifier, state)
	if err != nil {
		return nil, fmt.Errorf("ai: oauth exchange: %w", err)
	}

	return &OAuthTokens{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.ExpiresAt,
	}, nil
}

// Refresh obtains a new access token from a refresh token.
func (o *ClaudeOAuth) Refresh(refreshToken string) (*OAuthTokens, error) {
	store := &memoryStore{token: &storage.Token{
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(-time.Hour),
	}}
	client, _ := oauth.New(oauth.Config{Store: store})

	token, err := client.Refresh(context.Background())
	if err != nil {
		return nil, fmt.Errorf("ai: oauth refresh: %w", err)
	}

	return &OAuthTokens{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.ExpiresAt,
	}, nil
}

func (o *ClaudeOAuth) storePending(state, verifier string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.evictExpiredLocked()
	o.pending[state] = pendingAuth{verifier: verifier, createdAt: time.Now()}
}

func (o *ClaudeOAuth) take(state string) (string, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.evictExpiredLocked()
	p, ok := o.pending[state]
	if !ok {
		return "", false
	}
	delete(o.pending, state)
	return p.verifier, true
}

func (o *ClaudeOAuth) evictExpiredLocked() {
	for k, p := range o.pending {
		if time.Since(p.createdAt) > pendingAuthTTL {
			delete(o.pending, k)
		}
	}
}

// memoryStore is a minimal in-memory TokenStore for the SDK.
type memoryStore struct {
	mu    sync.Mutex
	token *storage.Token
}

func (s *memoryStore) Save(_ context.Context, token *storage.Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
	return nil
}

func (s *memoryStore) Load(_ context.Context) (*storage.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token, nil
}

func (s *memoryStore) Delete(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = nil
	return nil
}
