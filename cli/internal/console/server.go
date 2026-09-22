// Package console is an optional view of the runtime API. It holds no task state
// and is not a dependency of the public coordination library.
package console

import (
	"embed"
	"net/http"
	"strings"
)

//go:embed assets/*
var assets embed.FS

func New(api http.Handler, host string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != host {
			http.Error(w, "unexpected host", 403)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			api.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "GET required", 405)
			return
		}
		files := map[string]string{"/": "index.html", "/app.js": "app.js", "/style.css": "style.css"}
		name, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		data, e := assets.ReadFile("assets/" + name)
		if e != nil {
			http.Error(w, "asset unavailable", 500)
			return
		}
		typ := "text/html; charset=utf-8"
		if strings.HasSuffix(name, ".js") {
			typ = "text/javascript; charset=utf-8"
		}
		if strings.HasSuffix(name, ".css") {
			typ = "text/css; charset=utf-8"
		}
		w.Header().Set("Content-Type", typ)
		if r.Method != "HEAD" {
			_, _ = w.Write(data)
		}
	})
}
