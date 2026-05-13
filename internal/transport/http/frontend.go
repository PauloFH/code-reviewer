package http

import (
	"embed"
	"io/fs"
	"net/http"
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

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the exact file; fall back to index.html for SPA routing.
		f, err := sub.Open(r.URL.Path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// Serve index.html for unknown paths (SPA catch-all).
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}
