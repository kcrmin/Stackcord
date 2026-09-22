package console

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLocalBrowserSession(t *testing.T) {
	const host = "127.0.0.1:7331"
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer team-secret" {
			w.WriteHeader(401)
			return
		}
		w.WriteHeader(204)
	})
	handler := NewLocal(api, host, "team-secret", "launch-code", "private/token")
	request := func(path, method, body, origin string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://"+host+path, strings.NewReader(body))
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := request("/v1/projects", "GET", "", "", nil); w.Code != 401 {
		t.Fatal("anonymous access", w.Code)
	}
	if w := request("/console/session", "POST", `{"code":"launch-code"}`, "https://foreign.example", nil); w.Code != 403 {
		t.Fatal("cross-origin bootstrap", w.Code)
	}
	if w := request("/console/session", "POST", `{"code":"wrong"}`, "http://"+host, nil); w.Code != 401 {
		t.Fatal("wrong code", w.Code)
	}
	w := request("/console/session", "POST", `{"code":"launch-code"}`, "http://"+host, nil)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal(cookies)
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || strings.Contains(w.Body.String(), "team-secret") {
		t.Fatal("unsafe bootstrap", cookie)
	}
	if w := request("/console/session", "POST", `{"code":"launch-code"}`, "http://"+host, nil); w.Code != 401 {
		t.Fatal("code reused", w.Code)
	}
	if w := request("/v1/projects", "GET", "", "", cookie); w.Code != 204 {
		t.Fatal("reload session", w.Code)
	}
	if w := request("/v1/projects", "GET", "", "https://foreign.example", cookie); w.Code != 403 {
		t.Fatal("cross origin read", w.Code)
	}
	if w := request("/console/info", "GET", "", "", cookie); w.Code != 200 || !strings.Contains(w.Body.String(), "private/token") {
		t.Fatal("connection instructions", w.Code, w.Body.String())
	}
	r := httptest.NewRequest("GET", "http://evil.example/v1/projects", nil)
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("host bypass", w.Code)
	}
	if w := request("/console/session", "DELETE", "", "http://"+host, cookie); w.Code != 204 {
		t.Fatal("logout", w.Code)
	}
	if w := request("/v1/projects", "GET", "", "", cookie); w.Code != 401 {
		t.Fatal("revoked session", w.Code)
	}
}

func TestLaunchCodeExpires(t *testing.T) {
	s := newLocalSession("token", "code", "file")
	s.expires = time.Now().Add(-time.Second)
	if _, ok := s.exchange("code"); ok {
		t.Fatal("expired launch accepted")
	}
}
