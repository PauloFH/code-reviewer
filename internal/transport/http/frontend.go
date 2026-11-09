package http

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed frontend/dist
var frontendFS embed.FS

// frontendHandler returns an http.Handler that serves the Vue SPA.
// Unknown routes fall through to index.html so the Vue router can handle them.
func frontendHandler() http.Handler {
	sub, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		panic("frontend/dist not found in embed.FS: " + err.Error())
	}

	return frontendHandlerFromFS(sub)
}

func frontendHandlerFromFS(sub fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the exact file; fall back to index.html for SPA routing.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := sub.Open(path)
		if err == nil {
			f.Close()
			// Vite hashes every file under assets/, so its content never changes; index.html must
			// be revalidated so a deploy is picked up immediately (embed.FS gives no ETag/Last-Modified).
			if strings.HasPrefix(path, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		// Serve index.html for unknown paths (SPA catch-all).
		w.Header().Set("Cache-Control", "no-cache")
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}
