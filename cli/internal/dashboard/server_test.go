package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testBackend struct {
	calls int
	fail  bool
}

func (b *testBackend) Snapshot(context.Context) (any, error) {
	b.calls++
	if b.fail {
		return nil, errors.New("provider unavailable")
	}
	return map[string]any{"revision": "r1"}, nil
}
func (b *testBackend) Action(_ context.Context, kind string, payload json.RawMessage) (any, error) {
	b.calls++
	return map[string]any{"kind": kind, "payload": payload}, nil
}
func TestBoundary(t *testing.T) {
	tests := []struct {
		name, method, path, host, token, origin, content, body string
		want                                                   int
	}{
		{"state", "GET", "/api/state", "127.0.0.1:8123", "secret", "", "", "", 200},
		{"missing token", "GET", "/api/state", "127.0.0.1:8123", "", "", "", "", 401},
		{"wrong token", "GET", "/api/state", "127.0.0.1:8123", "other", "", "", "", 401},
		{"host rebinding", "GET", "/api/state", "attacker.test", "secret", "", "", "", 403},
		{"write", "POST", "/api/action", "127.0.0.1:8123", "secret", "http://127.0.0.1:8123", "application/json", `{"kind":"settings.preview","payload":{"revision":"r1"}}`, 200},
		{"missing origin", "POST", "/api/action", "127.0.0.1:8123", "secret", "", "application/json", `{"kind":"x"}`, 403},
		{"foreign origin", "POST", "/api/action", "127.0.0.1:8123", "secret", "https://evil.test", "application/json", `{"kind":"x"}`, 403},
		{"wrong method", "GET", "/api/action", "127.0.0.1:8123", "secret", "", "", "", 405},
		{"form write", "POST", "/api/action", "127.0.0.1:8123", "secret", "http://127.0.0.1:8123", "text/plain", `{"kind":"x"}`, 415},
		{"unknown fields", "POST", "/api/action", "127.0.0.1:8123", "secret", "http://127.0.0.1:8123", "application/json", `{"kind":"x","shell":"bad"}`, 400},
		{"trailing JSON", "POST", "/api/action", "127.0.0.1:8123", "secret", "http://127.0.0.1:8123", "application/json", `{"kind":"x"}{}`, 400},
		{"empty kind", "POST", "/api/action", "127.0.0.1:8123", "secret", "http://127.0.0.1:8123", "application/json", `{"kind":""}`, 400},
		{"oversized", "POST", "/api/action", "127.0.0.1:8123", "secret", "http://127.0.0.1:8123", "application/json", `{"kind":"x","payload":"` + strings.Repeat("a", 1<<20) + `"}`, 413},
		{"unknown API protected", "GET", "/api/unknown", "127.0.0.1:8123", "", "", "", "", 401},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &testBackend{}
			h := New(b, "secret", "127.0.0.1:8123")
			req := httptest.NewRequest(tt.method, "http://127.0.0.1:8123"+tt.path, strings.NewReader(tt.body))
			req.Host = tt.host
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Content-Type", tt.content)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Fatalf("status %d want %d: %s", w.Code, tt.want, w.Body.String())
			}
			if tt.want != 200 && b.calls != 0 {
				t.Fatal("rejected request reached backend")
			}
		})
	}
}
func TestAssetsAndFailures(t *testing.T) {
	for _, path := range []string{"/", "/app.js", "/styles.css"} {
		w := httptest.NewRecorder()
		New(&testBackend{}, "secret", "127.0.0.1:8123").ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:8123"+path, nil))
		if w.Code != 200 {
			t.Fatalf("asset %s: %d", path, w.Code)
		}
		for _, key := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy", "Cache-Control"} {
			if w.Header().Get(key) == "" {
				t.Errorf("missing %s", key)
			}
		}
		if strings.Contains(w.Header().Get("Content-Security-Policy"), "unsafe-") {
			t.Error("unsafe CSP")
		}
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://127.0.0.1:8123/api/state", nil)
	r.Header.Set("Authorization", "Bearer secret")
	New(&testBackend{fail: true}, "secret", "127.0.0.1:8123").ServeHTTP(w, r)
	if w.Code != http.StatusBadGateway || !strings.Contains(w.Body.String(), "provider unavailable") {
		t.Fatalf("failure hidden: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	New(&testBackend{}, "", "127.0.0.1:8123").ServeHTTP(w, r)
	if w.Code == 200 {
		t.Fatal("empty session token accepted")
	}
}

func TestStaticRoutesDoNotExposeFilesOrAcceptWrites(t *testing.T) {
	for _, test := range []struct {
		method, path, host string
		status             int
	}{
		{"GET", "/server.go", "127.0.0.1:8123", 404},
		{"GET", "/assets/../server.go", "127.0.0.1:8123", 404},
		{"POST", "/", "127.0.0.1:8123", 405},
		{"GET", "/", "evil.test", 403},
		{"HEAD", "/", "127.0.0.1:8123", 200},
	} {
		r := httptest.NewRequest(test.method, "http://127.0.0.1:8123"+test.path, nil)
		r.Host = test.host
		w := httptest.NewRecorder()
		New(&testBackend{}, "secret", "127.0.0.1:8123").ServeHTTP(w, r)
		if w.Code != test.status {
			t.Errorf("%s %s: got %d want %d", test.method, test.path, w.Code, test.status)
		}
		if test.method == "HEAD" && w.Body.Len() != 0 {
			t.Error("HEAD returned asset body")
		}
	}
}
