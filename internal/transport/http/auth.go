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

	"golang.org/x/crypto/bcrypt"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// jwtHeader is the fixed base64url-encoded header for HS256 JWTs.
var jwtHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

// tokenTTL is the JWT validity window.
const tokenTTL = 24 * time.Hour

// AuthHandler handles /api/v1/auth/login and /api/v1/auth/me.
type AuthHandler struct {
	userRepo  domain.UserRepository
	jwtSecret []byte
}

func NewAuthHandler(userRepo domain.UserRepository, jwtSecret string) *AuthHandler {
	secret := []byte(jwtSecret)
	if len(secret) == 0 {
		secret = make([]byte, 32)
		_, _ = rand.Read(secret)
	}
	return &AuthHandler{userRepo: userRepo, jwtSecret: secret}
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

	user, err := h.userRepo.GetByUsername(req.Username)
	if err != nil || user == nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.issueToken(user)
	if err != nil {
		http.Error(w, "failed to issue token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := userClaimsFromCtx(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.userRepo.GetByID(claims.ID)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// issueToken creates a signed HS256 JWT with id, sub, role, and exp claims.
func (h *AuthHandler) issueToken(user *domain.User) (string, error) {
	now := time.Now().UTC()
	payload := fmt.Sprintf(
		`{"id":%d,"sub":%q,"role":%q,"iat":%d,"exp":%d}`,
		user.ID, user.Username, user.Role, now.Unix(), now.Add(tokenTTL).Unix(),
	)
	encodedPayload := base64.RawURLEncoding.EncodeToString([]byte(payload))

	signingInput := jwtHeader + "." + encodedPayload
	mac := hmac.New(sha256.New, h.jwtSecret)
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + sig, nil
}

// jwtAuth returns a middleware that validates Bearer tokens and injects UserClaims into context.
func jwtAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get("token")
			if token == "" {
				token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			}

			claims, err := validateToken(token, secret)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			r = withUserClaims(r, *claims)
			next.ServeHTTP(w, r)
		})
	}
}

// validateToken checks the JWT signature, expiry, and extracts claims.
func validateToken(token string, secret []byte) (*UserClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token")
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[2]), []byte(expected)) {
		return nil, fmt.Errorf("invalid signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding")
	}

	var raw struct {
		ID   int64  `json:"id"`
		Sub  string `json:"sub"`
		Role string `json:"role"`
		Exp  int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &raw); err != nil {
		return nil, fmt.Errorf("invalid claims")
	}

	if raw.Exp < time.Now().UTC().Unix() {
		return nil, fmt.Errorf("token expired")
	}

	return &UserClaims{
		ID:       raw.ID,
		Username: raw.Sub,
		Role:     domain.Role(raw.Role),
	}, nil
}

// HashPassword generates a bcrypt hash for the given password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
