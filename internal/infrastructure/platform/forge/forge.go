// Package forge holds what every Git hosting client shares: identity, the
// authenticated REST round-trip and webhook signature checks.
package forge

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// Identity implements the identity part of domain.PlatformClient.
type Identity struct{ Client domain.Client }

func (i Identity) Platform() domain.PlatformType { return i.Client.Platform }
func (i Identity) ClientName() string            { return i.Client.Name }
func (i Identity) WatchedUsername() string       { return i.Client.Username }

// API is an authenticated REST client rooted at a base URL.
type API struct {
	BaseURL string
	auth    func(*http.Request)
	http    *http.Client
}

func NewAPI(baseURL string, timeout time.Duration, auth func(*http.Request)) *API {
	return &API{BaseURL: strings.TrimRight(baseURL, "/"), auth: auth, http: &http.Client{Timeout: timeout}}
}

// GetJSON decodes the JSON response of GET BaseURL+path into out.
func (a *API) GetJSON(path string, out any) error {
	body, err := a.do(http.MethodGet, path, "", nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func (a *API) GetText(path string) (string, error) {
	body, err := a.do(http.MethodGet, path, "", nil)
	return string(body), err
}

func (a *API) PostJSON(path string, in any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	_, err = a.do(http.MethodPost, path, "application/json", b)
	return err
}

func (a *API) PostForm(path string, form url.Values) error {
	_, err := a.do(http.MethodPost, path, "application/x-www-form-urlencoded", []byte(form.Encode()))
	return err
}

func (a *API) do(method, path, contentType string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(method, a.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	a.auth(req)
	resp, err := a.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s: status %d: %s", method, path, resp.StatusCode, data)
	}
	return data, nil
}

// ValidHMAC reports whether signature equals prefix + hex(HMAC-SHA256(secret, body)).
func ValidHMAC(secret string, body []byte, signature, prefix string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal([]byte(signature), []byte(prefix+hex.EncodeToString(mac.Sum(nil))))
}

// SplitOwnerRepo splits an "owner/repo" project ref.
func SplitOwnerRepo(ref string) (owner, repo string, err error) {
	owner, repo, ok := strings.Cut(ref, "/")
	if !ok || owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid project ref %q (expected owner/repo)", ref)
	}
	return owner, repo, nil
}

// FileDiff renders one file's patch with the ---/+++ header the review expects.
func FileDiff(oldPath, newPath, patch string) string {
	return fmt.Sprintf("--- a/%s\n+++ b/%s\n%s\n", oldPath, newPath, patch)
}
