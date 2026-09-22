package controlcenter

import (
	"context"
	"errors"
	"testing"
)

type noIssueWrite struct{ calls int }

func (f *noIssueWrite) Run(context.Context, ...string) ([]byte, error) {
	f.calls++
	return nil, errors.New("unexpected remote write")
}

func TestIssueMutationRevalidatesSettingsUnderHostLock(t *testing.T) {
	root := t.TempDir()
	s, rev, e := ReadSettings(root)
	if e != nil {
		t.Fatal(e)
	}
	s.Language = "ko"
	if _, e = SavePersonal(context.Background(), root, rev, s); e != nil {
		t.Fatal(e)
	}
	f := &noIssueWrite{}
	b := &Backend{Root: root, Transport: f}
	if _, e = b.createIssue(context.Background(), rev, "stable-id", "title", "body"); e == nil {
		t.Fatal("stale external write accepted")
	}
	if f.calls != 0 {
		t.Fatal("stale request reached provider")
	}
	unlock, e := lockRoot(root)
	if e != nil {
		t.Fatal("failed mutation leaked lock", e)
	}
	unlock()
}
