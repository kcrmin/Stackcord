package console

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// localSession provides an opt-in browser session for `stackcord start` only.
// The one-use launch code is not the durable API credential. No durable
// API credential is returned to JavaScript or stored in browser storage.
type localSession struct {
	mu                         sync.Mutex
	token, code, file, session string
	expires, sessionExpires    time.Time
}

func newLocalSession(token, code, file string) *localSession {
	return &localSession{token: token, code: code, file: file, expires: time.Now().Add(5 * time.Minute)}
}
func same(a, b string) bool {
	return a != "" && b != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func (s *localSession) exchange(code string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !time.Now().Before(s.expires) || !same(code, s.code) {
		return "", false
	}
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", false
	}
	s.code = ""
	s.session = hex.EncodeToString(raw[:])
	s.sessionExpires = time.Now().Add(12 * time.Hour)
	return s.session, true
}
func (s *localSession) valid(value string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Now().Before(s.sessionExpires) && same(value, s.session)
}
func NewLocal(api http.Handler, host, token, code, file string) http.Handler {
	s := newLocalSession(token, code, file)
	executable, _ := os.Executable()
	assets := New(api, host)
	cookieName := "stackcord_" + strings.ReplaceAll(host, ":", "_")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Host != host || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+host) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(w, "same-origin request required", 403)
			return
		}
		cookie, _ := r.Cookie(cookieName)
		authenticated := cookie != nil && s.valid(cookie.Value)
		if r.URL.Path == "/console/session" {
			// Bootstrap and logout require a browser Origin, not only an absent one.
			if r.Header.Get("Origin") != "http://"+host {
				http.Error(w, "same-origin request required", 403)
				return
			}
			if r.Method == "DELETE" {
				if authenticated {
					s.mu.Lock()
					s.session = ""
					s.mu.Unlock()
				}
				http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
				w.WriteHeader(204)
				return
			}
			if r.Method != "POST" {
				w.WriteHeader(405)
				return
			}
			if r.Header.Get("Content-Type") != "application/json" {
				w.WriteHeader(415)
				return
			}
			var input struct {
				Code string `json:"code"`
			}
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&input); err != nil {
				w.WriteHeader(400)
				return
			}
			value, ok := s.exchange(input.Code)
			if !ok {
				w.WriteHeader(401)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: cookieName, Value: value, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 43200})
			w.WriteHeader(204)
			return
		}
		if r.URL.Path == "/console/info" {
			if r.Method != "GET" {
				w.WriteHeader(405)
				return
			}
			if !authenticated && !same(r.Header.Get("Authorization"), "Bearer "+token) {
				w.WriteHeader(401)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"token_file": file, "endpoint": "http://" + host, "cli": executable})
			return
		}
		if authenticated && strings.HasPrefix(r.URL.Path, "/v1/") && r.Header.Get("Authorization") == "" {
			r = r.Clone(r.Context())
			r.Header.Set("Authorization", "Bearer "+token)
		}
		assets.ServeHTTP(w, r)
	})
}
