// Package dashboard serves the session-authenticated local control center.
package dashboard

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

// Backend owns project authorization, provider credentials, and revision checks.
type Backend interface {
	Snapshot(context.Context) (any, error)
	Action(context.Context, string, json.RawMessage) (any, error)
}

//go:embed assets/*
var assets embed.FS

// New creates a handler for one exact loopback host and ephemeral credential.
// The caller must bind a loopback listener and use an unpredictable session token.
func New(backend Backend, token, host string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Frame-Options", "DENY")
		if r.Host != host || host == "" {
			writeError(w, 403, "Unexpected host")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			supplied := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if token == "" || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) != 1 {
				writeError(w, 401, "Session credential missing or expired; reopen the dashboard from the CLI.")
				return
			}
			switch r.URL.Path {
			case "/api/state":
				if r.Method != http.MethodGet {
					w.Header().Set("Allow", "GET")
					writeError(w, 405, "Method not allowed")
					return
				}
				result, err := backend.Snapshot(r.Context())
				respond(w, result, err)
			case "/api/action":
				if r.Method != http.MethodPost {
					w.Header().Set("Allow", "POST")
					writeError(w, 405, "Method not allowed")
					return
				}
				if r.Header.Get("Origin") != "http://"+host {
					writeError(w, 403, "Same-origin request required")
					return
				}
				media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if err != nil || media != "application/json" {
					writeError(w, 415, "JSON content type required")
					return
				}
				var action struct {
					Kind    string          `json:"kind"`
					Payload json.RawMessage `json:"payload"`
				}
				decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
				decoder.DisallowUnknownFields()
				if err := decoder.Decode(&action); err != nil {
					var large *http.MaxBytesError
					if errors.As(err, &large) {
						writeError(w, 413, "Request exceeds 1 MiB")
					} else {
						writeError(w, 400, "Invalid action JSON")
					}
					return
				}
				if strings.TrimSpace(action.Kind) == "" {
					writeError(w, 400, "Action kind required")
					return
				}
				if err := decoder.Decode(new(any)); err != io.EOF {
					writeError(w, 400, "Exactly one JSON value required")
					return
				}
				result, err := backend.Action(r.Context(), action.Kind, action.Payload)
				respond(w, result, err)
			default:
				writeError(w, 404, "API route not found")
			}
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			writeError(w, 405, "Method not allowed")
			return
		}
		files := map[string]string{"/": "index.html", "/app.js": "app.js", "/styles.css": "styles.css"}
		name, ok := files[r.URL.Path]
		if !ok {
			writeError(w, 404, "Not found")
			return
		}
		data, err := assets.ReadFile("assets/" + name)
		if err != nil {
			writeError(w, 500, "Embedded asset unavailable")
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
		if r.Method != http.MethodHead {
			_, _ = w.Write(data)
		}
	})
}
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
func respond(w http.ResponseWriter, result any, err error) {
	if err != nil {
		writeError(w, 502, err.Error())
		return
	}
	data, err := json.Marshal(result)
	if err != nil {
		writeError(w, 500, "Unable to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(data)
}
