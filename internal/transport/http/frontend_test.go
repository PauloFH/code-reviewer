package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestFrontendHandler_ServesEmbeddedAsset(t *testing.T) {
	handler := frontendHandlerFromFS(fstest.MapFS{
		"index.html": {
			Data: []byte(`<!DOCTYPE html><script type="module" src="/assets/app.js"></script>`),
		},
		"assets/app.js": {
			Data: []byte(`console.log("ok")`),
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want non-html asset type", ct)
	}
	if strings.Contains(rec.Body.String(), "<!DOCTYPE html>") {
		t.Fatal("asset response unexpectedly returned index.html")
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q, want immutable for hashed asset", cc)
	}
}

func TestFrontendHandler_IndexIsRevalidated(t *testing.T) {
	handler := frontendHandlerFromFS(fstest.MapFS{
		"index.html": {Data: []byte(`<!DOCTYPE html>`)},
	})
	for _, path := range []string{"/", "/dashboard"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
			t.Fatalf("%s: Cache-Control = %q, want no-cache", path, cc)
		}
	}
}

func TestFrontendHandler_FallsBackToIndexForSPARoute(t *testing.T) {
	handler := frontendHandlerFromFS(fstest.MapFS{
		"index.html": {
			Data: []byte(`<!DOCTYPE html><div id="app"></div>`),
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "<!DOCTYPE html>") {
		t.Fatal("SPA route did not return index.html")
	}
}
