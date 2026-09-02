package adminui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// static contains the production React bundle. Docker and release builds regenerate it
// from admin-ui/ before compiling the self-contained gateway binary.
//
//go:embed static/* static/assets/*
var static embed.FS

func Handler() http.Handler {
	content, err := fs.Sub(static, "static")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(content))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setHeaders(w)
		path := strings.TrimPrefix(r.URL.Path, "/admin")
		if path == "" || path == "/" {
			serveIndex(w, content)
			return
		}
		assetPath := strings.TrimPrefix(path, "/")
		if _, err := fs.Stat(content, assetPath); err != nil {
			serveIndex(w, content)
			return
		}
		if strings.HasPrefix(assetPath, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		request := r.Clone(r.Context())
		urlCopy := *r.URL
		urlCopy.Path = path
		request.URL = &urlCopy
		files.ServeHTTP(w, request)
	})
}

func serveIndex(w http.ResponseWriter, content fs.FS) {
	body, err := fs.ReadFile(content, "index.html")
	if err != nil {
		http.Error(w, "Music Room admin UI is unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(body)
}

func setHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self' ws: wss:; img-src 'self' data:; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
}
