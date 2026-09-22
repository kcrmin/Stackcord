package runtimecmd

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kcrmin/Stackcord/core/coordination"
	"github.com/kcrmin/Stackcord/core/httpapi"
)

func TestCLIUsesHTTPWithoutGitAndPreservesCheckpoint(t *testing.T) {
	s, e := coordination.Open(filepath.Join(t.TempDir(), "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	server := httptest.NewServer(httpapi.New(s, "secret", ""))
	defer server.Close()
	t.Setenv("STACKCORD_TOKEN", "secret")
	t.Setenv("PATH", t.TempDir())
	run := func(input string, args ...string) string {
		t.Helper()
		var out, errs bytes.Buffer
		cmd := New(&out, &errs)
		cmd.SetIn(strings.NewReader(input))
		cmd.SetArgs(append([]string{"--endpoint", server.URL}, args...))
		if e := cmd.Execute(); e != nil {
			t.Fatal(e, errs.String())
		}
		return out.String()
	}
	run(`{"id":"create","project":"p","task":"api","kind":"create","definition":{"goal":"Build API"}}`, "task", "apply", "--input", "-")
	run(`{"id":"claim","project":"p","task":"api","kind":"claim","owner":"alice","session":"a1","lease_seconds":60}`, "task", "apply", "--input", "-")
	run(`{"id":"pause","project":"p","task":"api","kind":"pause","owner":"alice","session":"a1","epoch":1,"reason":"quota","checkpoint":{"summary":"Schema done","next":"Tests"}}`, "task", "apply", "--input", "-")
	var brief coordination.Context
	if e = json.Unmarshal([]byte(run("", "resume", "--project", "p", "--task", "api")), &brief); e != nil {
		t.Fatal(e)
	}
	if brief.Task.Checkpoint.Next != "Tests" || brief.Task.State != "paused" {
		t.Fatal(brief)
	}
	var page coordination.Page
	if e = json.Unmarshal([]byte(run("", "events", "--project", "p", "--after", "1")), &page); e != nil {
		t.Fatal(e)
	}
	if len(page.Events) != 2 {
		t.Fatal(page)
	}
	dest := filepath.Join(t.TempDir(), "backup.db")
	run("", "backup", "--output", dest)
	if info, e := os.Stat(dest); e != nil || info.Size() == 0 {
		t.Fatal(e)
	}
	cmd := New(&bytes.Buffer{}, &bytes.Buffer{})
	cmd.SetArgs([]string{"--endpoint", server.URL, "backup", "--output", dest})
	if e = cmd.Execute(); e == nil {
		t.Fatal("overwrote backup")
	}
}
func TestServeRejectsUnsafeConfigWithoutCreatingDatabase(t *testing.T) {
	for _, args := range [][]string{{"serve", "--port", "-1"}, {"serve", "--port", "65536"}, {"serve"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("STACKCORD_TOKEN", "")
			path := filepath.Join(t.TempDir(), "state.db")
			cmd := New(&bytes.Buffer{}, &bytes.Buffer{})
			cmd.SetArgs(append(args, "--db", path))
			if e := cmd.Execute(); e == nil {
				t.Fatal("expected configuration error")
			}
			if _, e := os.Stat(path); !os.IsNotExist(e) {
				t.Fatal("invalid config created database")
			}
		})
	}
}
