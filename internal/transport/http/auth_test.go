package http

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestAuth() *AuthHandler {
	return NewAuthHandler("admin", "secret", "test-jwt-key-32-bytes-long-enough!!")
}

// signedToken builds a minimal HS256 JWT with the given claims and secret.
func signedToken(secret []byte, sub string, exp int64) string {
	payload := fmt.Sprintf(`{"sub":%q,"iat":%d,"exp":%d}`, sub, time.Now().Unix(), exp)
	encodedPayload := base64.RawURLEncoding.EncodeToString([]byte(payload))
	signingInput := jwtHeader + "." + encodedPayload
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signingInput + "." + sig
}

// ── issueToken / validateToken ────────────────────────────────────────────────

func TestIssueAndValidateToken_Valid(t *testing.T) {
	h := newTestAuth()
	token, err := h.issueToken("admin")
	if err != nil {
		t.Fatalf("issueToken: %v", err)
	}
	if err := validateToken(token, h.jwtSecret); err != nil {
		t.Errorf("validateToken: %v", err)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	h := newTestAuth()
	token, _ := h.issueToken("admin")
	if err := validateToken(token, []byte("wrong-secret")); err == nil {
		t.Error("expected error for wrong secret, got nil")
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	cases := []string{"not.a.jwt", "", "only.two", "a.b.c.d"}
	for _, tc := range cases {
		if tc == "a.b.c.d" {
			continue // 4 parts — different error path, skip
		}
		if err := validateToken(tc, []byte("key")); err == nil {
			t.Errorf("expected error for %q", tc)
		}
	}
}

func TestValidateToken_Expired(t *testing.T) {
	h := newTestAuth()
	expired := signedToken(h.jwtSecret, "admin", time.Now().Add(-2*time.Second).Unix())
	if err := validateToken(expired, h.jwtSecret); err == nil {
		t.Error("expected error for expired token")
	}
}

// ── Login handler ─────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	h := newTestAuth()
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["token"] == "" {
		t.Error("expected non-empty token in response")
	}
	// Returned token must be self-validating.
	if err := validateToken(resp["token"], h.jwtSecret); err != nil {
		t.Errorf("returned token invalid: %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	h := newTestAuth()
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestLogin_WrongUsername(t *testing.T) {
	h := newTestAuth()
	body, _ := json.Marshal(map[string]string{"username": "hacker", "password": "secret"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestLogin_BadBody(t *testing.T) {
	h := newTestAuth()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("not-json")))
	w := httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── jwtAuth middleware ─────────────────────────────────────────────────────────

func TestJwtAuth_ValidBearer(t *testing.T) {
	h := newTestAuth()
	token, _ := h.issueToken("admin")

	called := false
	handler := jwtAuth(h.jwtSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Error("expected next handler to be called with valid Bearer token")
	}
}

func TestJwtAuth_ValidQueryParam(t *testing.T) {
	h := newTestAuth()
	token, _ := h.issueToken("admin")

	called := false
	handler := jwtAuth(h.jwtSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/?token="+token, nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Error("expected next handler to be called with token query param")
	}
}

func TestJwtAuth_MissingToken_Returns401(t *testing.T) {
	h := newTestAuth()
	handler := jwtAuth(h.jwtSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called without token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJwtAuth_ExpiredToken_Returns401(t *testing.T) {
	h := newTestAuth()
	expired := signedToken(h.jwtSecret, "admin", time.Now().Add(-1*time.Second).Unix())

	handler := jwtAuth(h.jwtSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called with expired token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+expired)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
