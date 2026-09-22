// Package httpapi exposes coordination over authenticated HTTP. The handler does
// not bind a socket or serve a GUI. Operators own TLS and network access policy.
package httpapi

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/kcrmin/Stackcord/core/coordination"
)

// New serves one trust domain. The bearer credential authorizes every project
// and explicit operator action, including requeue and backup. Use separate
// runtimes for mutually untrusted teams. Empty tokens fail closed.
func New(store *coordination.Store, token, host string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if host != "" && r.Host != host {
			writeError(w, 403, "unexpected host")
			return
		}
		auth := r.Header.Get("Authorization")
		if token == "" || !strings.HasPrefix(auth, "Bearer ") || subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(auth, "Bearer ")), []byte(token)) != 1 {
			writeError(w, 401, "bearer credential required")
			return
		}
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != scheme+"://"+r.Host {
			writeError(w, 403, "same-origin request required")
			return
		}
		if r.URL.Path == "/v1/apply" {
			if r.Method != "POST" {
				w.Header().Set("Allow", "POST")
				writeError(w, 405, "POST required")
				return
			}
			media, _, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if e != nil || media != "application/json" {
				writeError(w, 415, "application/json required")
				return
			}
			raw, e := io.ReadAll(io.LimitReader(r.Body, coordination.MaxMutationBytes+1))
			if e != nil {
				writeError(w, 400, "cannot read request")
				return
			}
			if len(raw) > coordination.MaxMutationBytes {
				writeError(w, 413, "mutation exceeds 64 KiB")
				return
			}
			var m coordination.Mutation
			d := json.NewDecoder(bytes.NewReader(raw))
			d.DisallowUnknownFields()
			if e = d.Decode(&m); e != nil {
				writeError(w, 400, "invalid mutation JSON")
				return
			}
			if e = d.Decode(new(any)); e != io.EOF {
				writeError(w, 400, "exactly one JSON value required")
				return
			}
			result, e := store.Apply(m)
			respond(w, result, e)
			return
		}
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			writeError(w, 405, "GET required")
			return
		}
		q := r.URL.Query()
		project := q.Get("project")
		switch r.URL.Path {
		case "/v1/projects":
			v, e := store.Projects()
			respond(w, v, e)
		case "/v1/tasks":
			v, e := store.Tasks(project)
			respond(w, v, e)
		case "/v1/context":
			n, e := integer(q.Get("max_bytes"), coordination.DefaultContextBytes)
			if e != nil {
				respond(w, nil, coordination.ErrInvalid)
				return
			}
			v, e := store.Context(project, q.Get("task"), n)
			respond(w, v, e)
		case "/v1/events":
			after := uint64(0)
			var e error
			if q.Get("after") != "" {
				after, e = strconv.ParseUint(q.Get("after"), 10, 64)
			}
			limit, le := integer(q.Get("limit"), 100)
			if e != nil || le != nil {
				respond(w, nil, coordination.ErrInvalid)
				return
			}
			v, e := store.Events(project, after, limit)
			respond(w, v, e)
		case "/v1/backup":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", `attachment; filename="stackcord.db"`)
			// WriteTo keeps a consistent read transaction. On a partial write the client
			// must discard its output; never append a JSON error to a database stream.
			if e := store.Backup(w); e != nil {
				panic(http.ErrAbortHandler)
			}
		default:
			writeError(w, 404, "route not found")
		}
	})
}
func integer(raw string, defaultValue int) (int, error) {
	if raw == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(raw)
}
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
func respond(w http.ResponseWriter, v any, e error) {
	if e != nil {
		status := 500
		message := "runtime operation failed"
		switch {
		case errors.Is(e, coordination.ErrInvalid):
			status = 400
			message = e.Error()
		case errors.Is(e, coordination.ErrNotFound):
			status = 404
			message = e.Error()
		case errors.Is(e, coordination.ErrConflict), errors.Is(e, coordination.ErrBlocked):
			status = 409
			message = e.Error()
		case errors.Is(e, coordination.ErrBudget):
			status = 422
			message = e.Error()
		}
		writeError(w, status, message)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
