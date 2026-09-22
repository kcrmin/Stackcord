package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kcrmin/Stackcord/core/coordination"
)

func fixture(t *testing.T) (*coordination.Store, *httptest.Server, *Client) {
	t.Helper()
	s, e := coordination.Open(filepath.Join(t.TempDir(), "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	server := httptest.NewServer(New(s, "test-secret", ""))
	t.Cleanup(server.Close)
	c, e := NewClient(server.URL, "test-secret")
	if e != nil {
		t.Fatal(e)
	}
	return s, server, c
}
func TestHTTPRecoveryAndContext(t *testing.T) {
	_, _, c := fixture(t)
	ctx := context.Background()
	call := func(m coordination.Mutation) coordination.Task {
		t.Helper()
		task, e := c.Apply(ctx, m)
		if e != nil {
			t.Fatal(e)
		}
		return task
	}
	call(coordination.Mutation{ID: "create", Project: "demo", Task: "api", Kind: "create", Definition: &coordination.Definition{Goal: "Build API"}})
	a := call(coordination.Mutation{ID: "claim", Project: "demo", Task: "api", Kind: "claim", Owner: "a", Session: "a1", LeaseSeconds: 60})
	pause := coordination.Mutation{ID: "pause", Project: "demo", Task: "api", Kind: "pause", Owner: "a", Session: "a1", Epoch: a.Epoch, Reason: "quota", Checkpoint: &coordination.Checkpoint{Next: "Implement tests"}}
	call(pause)
	call(pause)
	b := call(coordination.Mutation{ID: "takeover", Project: "demo", Task: "api", Kind: "claim", Owner: "b", Session: "b1", LeaseSeconds: 60})
	if b.Checkpoint.Next != "Implement tests" {
		t.Fatal(b)
	}
	if _, e := c.Apply(ctx, coordination.Mutation{ID: "late", Project: "demo", Task: "api", Kind: "complete", Owner: "a", Session: "a1", Epoch: a.Epoch}); e == nil {
		t.Fatal("accepted stale owner")
	}
	brief, e := c.Context(ctx, "demo", "api", 16384)
	if e != nil || brief.Task.Owner != "b" {
		t.Fatalf("%+v %v", brief, e)
	}
	page, e := c.Events(ctx, "demo", 0, 100)
	if e != nil || len(page.Events) != 4 {
		t.Fatalf("%+v %v", page, e)
	}
	projects, e := c.Projects(ctx)
	if e != nil || len(projects) != 1 || projects[0] != "demo" {
		t.Fatalf("%v %v", projects, e)
	}
	var backup bytes.Buffer
	if e = c.Backup(ctx, &backup); e != nil || backup.Len() == 0 {
		t.Fatal(e)
	}
}
func TestHTTPRejectsUntrustedAndMalformedWrites(t *testing.T) {
	s, server, _ := fixture(t)
	body := `{"id":"create","project":"p","task":"t","kind":"create","definition":{"goal":"test"}}`
	for _, tc := range []struct {
		name, token, origin, body, content string
		want                               int
	}{
		{"no auth", "", "", body, "application/json", 401},
		{"wrong auth", "wrong", "", body, "application/json", 401},
		{"cross origin", "test-secret", "https://evil.example", body, "application/json", 403},
		{"unknown field", "test-secret", "", strings.TrimSuffix(body, "}") + `,"shell":"rm"}`, "application/json", 400},
		{"trailing value", "test-secret", "", body + ` {}`, "application/json", 400},
		{"too large", "test-secret", "", strings.Repeat(" ", coordination.MaxMutationBytes) + body, "application/json", 413},
		{"not json", "test-secret", "", body, "text/plain", 415},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", server.URL+"/v1/apply", strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+tc.token)
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Content-Type", tc.content)
			resp, e := http.DefaultClient.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Fatalf("got %d, want %d", resp.StatusCode, tc.want)
			}
		})
	}
	tasks, e := s.Tasks("p")
	if e != nil || len(tasks) != 0 {
		t.Fatal(tasks, e)
	}
	req := httptest.NewRequest("GET", "http://evil/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer test-secret")
	w := httptest.NewRecorder()
	New(s, "test-secret", "127.0.0.1:1").ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	w = httptest.NewRecorder()
	New(s, "", "").ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
}
func TestReadValidationAndBudget(t *testing.T) {
	_, server, c := fixture(t)
	_, e := c.Apply(context.Background(), coordination.Mutation{ID: "create", Project: "p", Task: "t", Kind: "create", Definition: &coordination.Definition{Goal: "Must never truncate obligations"}})
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{"/v1/tasks", "/v1/events?project=p&after=oops", "/v1/events?project=p&after=999", "/v1/context?project=p&task=t&max_bytes=1", "/v1/events?project=p&limit=0"} {
		req, _ := http.NewRequest("GET", server.URL+path, nil)
		req.Header.Set("Authorization", "Bearer test-secret")
		resp, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		var v map[string]any
		json.NewDecoder(resp.Body).Decode(&v)
		resp.Body.Close()
		if resp.StatusCode < 400 {
			t.Fatalf("%s accepted: %v", path, v)
		}
	}
}
func TestClientRejectsCredentialLeakingURLs(t *testing.T) {
	for _, url := range []string{"http://example.com", "ftp://localhost", "http://user:pass@localhost", "https://example.com?secret=x", "https://example.com/#fragment"} {
		if _, e := NewClient(url, "token"); e == nil {
			t.Fatal(url)
		}
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("redirect followed") }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer redirect.Close()
	c, e := NewClient(redirect.URL, "token")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Projects(context.Background()); e == nil {
		t.Fatal(fmt.Sprint("redirect accepted"))
	}
}
