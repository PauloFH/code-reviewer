package ai

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Claude Code public OAuth client parameters. These replicate the headless
// (manual code-paste) flow used by the official CLI; Anthropic does not offer a
// general-purpose third-party "Login with Claude", so we reuse the CLI's public
// client. Validate these against the current CLI if the flow stops working.
const (
	claudeOAuthAuthorizeURL = "https://claude.ai/oauth/authorize"
	claudeOAuthTokenURL     = "https://console.anthropic.com/v1/oauth/token"
	claudeOAuthClientID     = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	claudeOAuthRedirectURI  = "https://console.anthropic.com/oauth/code/callback"
	claudeOAuthScopes       = "org:create_api_key user:profile user:inference"
	pendingAuthTTL          = 15 * time.Minute
)

// OAuthTokens is the result of an authorization-code exchange or refresh.
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// ClaudeOAuth drives the Claude subscription OAuth flow (PKCE S256).
// Pending verifiers are held in memory keyed by state; they are short-lived and
// safe to lose on restart since the flow completes within minutes.
type ClaudeOAuth struct {
	http    *http.Client
	mu      sync.Mutex
	pending map[string]pendingAuth
}

type pendingAuth struct {
	verifier  string
	createdAt time.Time
}

func NewClaudeOAuth() *ClaudeOAuth {
	return &ClaudeOAuth{
		http:    &http.Client{Timeout: 30 * time.Second},
		pending: make(map[string]pendingAuth),
	}
}

// BuildAuthorizeURL generates a PKCE challenge, stores the verifier keyed by the
// returned state, and returns the URL the user must open to authorize.
func (o *ClaudeOAuth) BuildAuthorizeURL() (authURL, state string, err error) {
	verifier, err := randomURLSafe(32)
	if err != nil {
		return "", "", err
	}
	state, err = randomURLSafe(24)
	if err != nil {
		return "", "", err
	}

	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	o.store(state, verifier)

	q := url.Values{}
	q.Set("code", "true")
	q.Set("client_id", claudeOAuthClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", claudeOAuthRedirectURI)
	q.Set("scope", claudeOAuthScopes)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")

	return claudeOAuthAuthorizeURL + "?" + q.Encode(), state, nil
}

// Exchange swaps an authorization code for tokens. The pasted code may carry the
// state as "code#state"; only the code portion is sent. The verifier is looked
// up by the state issued in BuildAuthorizeURL.
func (o *ClaudeOAuth) Exchange(state, pastedCode string) (*OAuthTokens, error) {
	verifier, ok := o.take(state)
	if !ok {
		return nil, fmt.Errorf("ai: unknown or expired oauth state")
	}

	code := pastedCode
	if i := strings.Index(pastedCode, "#"); i >= 0 {
		code = pastedCode[:i]
	}

	payload := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  claudeOAuthRedirectURI,
		"client_id":     claudeOAuthClientID,
		"code_verifier": verifier,
		"state":         state,
	}
	return o.postToken(payload)
}

// Refresh obtains a new access token from a refresh token.
func (o *ClaudeOAuth) Refresh(refreshToken string) (*OAuthTokens, error) {
	payload := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     claudeOAuthClientID,
	}
	return o.postToken(payload)
}

func (o *ClaudeOAuth) postToken(payload map[string]string) (*OAuthTokens, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("ai: marshalling token request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, claudeOAuthTokenURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("ai: building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai: calling token endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai: token endpoint returned status %d: %s", resp.StatusCode, rb)
	}

	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ai: decoding token response: %w", err)
	}
	if out.AccessToken == "" {
		return nil, fmt.Errorf("ai: token endpoint returned no access_token")
	}

	expiresAt := time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
	return &OAuthTokens{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func (o *ClaudeOAuth) store(state, verifier string) {
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

func randomURLSafe(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("ai: generating random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
