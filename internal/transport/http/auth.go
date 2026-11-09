package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// jwtHeader is the fixed base64url-encoded header for HS256 JWTs.
var jwtHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

const tokenTTL = 24 * time.Hour

type authHandler struct {
	users  domain.UserRepository
	secret []byte
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	user, err := h.users.GetByUsername(req.Username)
	if errors.Is(err, domain.ErrNotFound) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": issueToken(user, h.secret)})
}

func (h *authHandler) me(w http.ResponseWriter, r *http.Request) {
	claims, _ := userClaimsFromCtx(r)
	user, err := h.users.GetByID(claims.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// issueToken creates a signed HS256 JWT with id, sub, role, and exp claims.
func issueToken(user *domain.User, secret []byte) string {
	now := time.Now().UTC()
	payload := fmt.Sprintf(
		`{"id":%d,"sub":%q,"role":%q,"iat":%d,"exp":%d}`,
		user.ID, user.Username, user.Role, now.Unix(), now.Add(tokenTTL).Unix(),
	)
	signingInput := jwtHeader + "." + base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// UserClaims is the authenticated identity carried by the JWT.
type UserClaims struct {
	ID       int64
	Username string
	Role     domain.Role
}

type userClaimsKey struct{}

func userClaimsFromCtx(r *http.Request) (UserClaims, bool) {
	v, ok := r.Context().Value(userClaimsKey{}).(UserClaims)
	return v, ok
}

// jwtAuth requires a valid Bearer token (or, with allowQuery, a ?token= one)
// and puts its claims in the request context.
func jwtAuth(secret []byte, allowQuery bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if allowQuery && token == "" {
				token = r.URL.Query().Get("token")
			}
			claims, err := validateToken(token, secret)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userClaimsKey{}, *claims)))
		})
	}
}

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

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
