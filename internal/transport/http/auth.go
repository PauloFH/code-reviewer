package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// jwtHeader is the fixed base64url-encoded header for HS256 JWTs.
var jwtHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

// tokenTTL is the JWT validity window.
const tokenTTL = 24 * time.Hour

// AuthHandler handles /api/v1/auth/login.
type AuthHandler struct {
	user      string
	password  string
	jwtSecret []byte
}

func NewAuthHandler(user, password, jwtSecret string) *AuthHandler {
	secret := []byte(jwtSecret)
	if len(secret) == 0 {
		// Generate a random secret at runtime so the server works without config,
		// but tokens won't survive a restart.
		secret = make([]byte, 32)
		_, _ = rand.Read(secret)
	}
	return &AuthHandler{user: user, password: password, jwtSecret: secret}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username != h.user || req.Password != h.password {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.issueToken(req.Username)
	if err != nil {
		http.Error(w, "failed to issue token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

// issueToken creates a signed HS256 JWT with an exp claim.
func (h *AuthHandler) issueToken(username string) (string, error) {
	now := time.Now().UTC()
	payload := fmt.Sprintf(`{"sub":%q,"iat":%d,"exp":%d}`, username, now.Unix(), now.Add(tokenTTL).Unix())
	encodedPayload := base64.RawURLEncoding.EncodeToString([]byte(payload))

	signingInput := jwtHeader + "." + encodedPayload
	mac := hmac.New(sha256.New, h.jwtSecret)
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + sig, nil
}

// jwtAuth returns a middleware that validates Bearer tokens signed with secret.
func jwtAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow token via query param for SSE clients that can't set headers.
			token := r.URL.Query().Get("token")
			if token == "" {
				token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			}

			if err := validateToken(token, secret); err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// validateToken checks the JWT signature and expiry.
func validateToken(token string, secret []byte) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("malformed token")
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[2]), []byte(expected)) {
		return fmt.Errorf("invalid signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("invalid payload encoding")
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return fmt.Errorf("invalid claims")
	}

	if claims.Exp < time.Now().UTC().Unix() {
		return fmt.Errorf("token expired")
	}

	return nil
}
