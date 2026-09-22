package coordination

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func apply(t *testing.T, s *Store, m Mutation) Task {
	t.Helper()
	task, err := s.Apply(m)
	if err != nil {
		t.Fatal(err)
	}
	return task
}
func create(t *testing.T, s *Store, id string, deps ...string) Task {
	return apply(t, s, Mutation{ID: "create-" + id, Project: "p", Task: id, Kind: "create", Definition: &Definition{Goal: "Implement " + id, Constraints: []string{"Preserve API compatibility"}, Dependencies: deps}})
}
func claim(t *testing.T, s *Store, id, owner, session string) Task {
	return apply(t, s, Mutation{ID: "claim-" + id + "-" + session, Project: "p", Task: id, Kind: "claim", Owner: owner, Session: session, LeaseSeconds: 60})
}
func update(id, kind string, task Task) Mutation {
	return Mutation{ID: id, Project: task.Project, Task: task.ID, Kind: kind, Owner: task.Owner, Session: task.Session, Epoch: task.Epoch}
}
func TestQuotaHandoffAndStaleSession(t *testing.T) {
	s := openTest(t)
	create(t, s, "api")
	a := claim(t, s, "api", "alice", "a1")
	pause := update("pause-a", "pause", a)
	pause.Reason = "quota"
	pause.Checkpoint = &Checkpoint{Summary: "Schema complete", Next: "Implement handler", Artifacts: []Reference{{URI: "git:abc", Digest: "abc"}}}
	pause.Usage = &Usage{InputTokens: 100, OutputTokens: 30}
	apply(t, s, pause)
	b := claim(t, s, "api", "bob", "b1")
	if b.Checkpoint.Next != "Implement handler" || b.Epoch <= a.Epoch {
		t.Fatalf("handoff lost: %+v", b)
	}
	if _, err := s.Apply(update("late-a", "complete", a)); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale writer accepted: %v", err)
	}
	done := apply(t, s, update("finish-b", "complete", b))
	if done.State != "completed" {
		t.Fatal(done)
	}
	brief, err := s.Context("p", "api", 16384)
	if err != nil || brief.Task.State != "completed" || brief.Task.Usage.InputTokens != 100 {
		t.Fatalf("resume: %+v %v", brief, err)
	}
	if _, err := s.Apply(Mutation{ID: "reclaim", Project: "p", Task: "api", Kind: "claim", Owner: "alice", Session: "a2", LeaseSeconds: 60}); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}
func TestRestartIdempotencyAndBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	create(t, s, "api")
	a := claim(t, s, "api", "alice", "one")
	m := update("saved", "checkpoint", a)
	m.Checkpoint = &Checkpoint{Summary: "Saved", Next: "Tests"}
	m.Usage = &Usage{InputTokens: 42, OutputTokens: 9}
	m.LeaseSeconds = 60
	want := apply(t, s, m)
	s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got := apply(t, s, m)
	if got.Revision != want.Revision || got.Usage.InputTokens != 42 {
		t.Fatal(got)
	}
	m.Checkpoint.Next = "Changed"
	if _, err = s.Apply(m); !errors.Is(err, ErrConflict) {
		t.Fatalf("key reused: %v", err)
	}
	page, err := s.Events("p", 0, 100)
	if err != nil || len(page.Events) != 3 {
		t.Fatalf("events: %+v %v", page, err)
	}
	var backup bytes.Buffer
	if err = s.Backup(&backup); err != nil {
		t.Fatal(err)
	}
	restored := filepath.Join(t.TempDir(), "restored.db")
	if err = os.WriteFile(restored, backup.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	copy, err := Open(restored)
	if err != nil {
		t.Fatal(err)
	}
	defer copy.Close()
	tasks, err := copy.Tasks("p")
	if err != nil || len(tasks) != 1 || tasks[0].Checkpoint.Next != "Tests" {
		t.Fatalf("backup: %+v %v", tasks, err)
	}
}
func TestExpiredOwnershipAndConcurrentClaims(t *testing.T) {
	s := openTest(t)
	now := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	create(t, s, "api")
	a := claim(t, s, "api", "alice", "one")
	now = now.Add(61 * time.Second)
	if _, err := s.Apply(update("expired-write", "complete", a)); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wins := make(chan Task, 2)
	for _, session := range []string{"two", "three"} {
		wg.Add(1)
		go func(session string) {
			defer wg.Done()
			task, err := s.Apply(Mutation{ID: "claim-" + session, Project: "p", Task: "api", Kind: "claim", Owner: "alice", Session: session, LeaseSeconds: 60})
			if err == nil {
				wins <- task
			} else if !errors.Is(err, ErrConflict) {
				t.Error(err)
			}
		}(session)
	}
	wg.Wait()
	close(wins)
	if len(wins) != 1 {
		t.Fatalf("owners: %d", len(wins))
	}
	b := <-wins
	if b.Epoch != 2 {
		t.Fatal(b)
	}
	if _, err := s.Apply(update("stale-session", "complete", a)); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}
func TestDependenciesApprovalAndExplicitRetry(t *testing.T) {
	s := openTest(t)
	create(t, s, "api")
	create(t, s, "ui", "api")
	if _, err := s.Apply(Mutation{ID: "blocked", Project: "p", Task: "ui", Kind: "claim", Owner: "bob", Session: "one", LeaseSeconds: 60}); !errors.Is(err, ErrBlocked) {
		t.Fatal(err)
	}
	a := claim(t, s, "api", "alice", "one")
	pause := update("approval", "pause", a)
	pause.Reason = "approval_required"
	pause.Checkpoint = &Checkpoint{Next: "Review contract"}
	apply(t, s, pause)
	if _, err := s.Apply(Mutation{ID: "steal", Project: "p", Task: "api", Kind: "claim", Owner: "bob", Session: "two", LeaseSeconds: 60}); !errors.Is(err, ErrBlocked) {
		t.Fatal(err)
	}
	apply(t, s, Mutation{ID: "requeue", Project: "p", Task: "api", Kind: "requeue", ExpectedRevision: 3})
	a = claim(t, s, "api", "alice", "new")
	if a.Checkpoint.Next != "Review contract" {
		t.Fatal(a)
	}
	apply(t, s, update("done", "complete", a))
	claim(t, s, "ui", "bob", "two")
}
func TestProjectIsolationPagingAndContextBudget(t *testing.T) {
	s := openTest(t)
	create(t, s, "one")
	create(t, s, "two")
	apply(t, s, Mutation{ID: "other", Project: "secret", Task: "one", Kind: "create", Definition: &Definition{Goal: "Private goal"}})
	first, err := s.Events("p", 0, 1)
	if err != nil || len(first.Events) != 1 || !first.More || first.Next != 1 {
		t.Fatalf("%+v %v", first, err)
	}
	second, err := s.Events("p", first.Next, 1)
	if err != nil || len(second.Events) != 1 || second.More || second.Events[0].Task != "two" {
		t.Fatalf("%+v %v", second, err)
	}
	empty, err := s.Events("p", second.Next, 1)
	if err != nil || len(empty.Events) != 0 || empty.Next != second.Next {
		t.Fatalf("%+v %v", empty, err)
	}
	raw, _ := json.Marshal(first)
	if strings.Contains(string(raw), "Private") || strings.Contains(string(raw), "Implement") {
		t.Fatal(string(raw))
	}
	if _, err := s.Context("p", "one", 20); !errors.Is(err, ErrBudget) {
		t.Fatal(err)
	}
	if _, err := s.Tasks(""); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := s.Events("p", 99, 1); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
}
func TestInvalidMutationsAreAtomic(t *testing.T) {
	s := openTest(t)
	cases := []Mutation{
		{ID: "a", Project: "p", Task: "x", Kind: "create", Definition: &Definition{Goal: ""}},
		{ID: "b", Project: "p", Task: "x", Kind: "create", Definition: &Definition{Goal: "goal", Dependencies: []string{"missing"}}},
		{ID: "c", Project: "p", Task: "x", Kind: "create", Definition: &Definition{Goal: strings.Repeat("a", 70000)}},
		{ID: "d", Project: "../p", Task: "x", Kind: "create", Definition: &Definition{Goal: "goal"}},
	}
	for _, m := range cases {
		if _, err := s.Apply(m); err == nil {
			t.Fatalf("accepted %+v", m)
		}
	}
	tasks, err := s.Tasks("p")
	if err != nil || len(tasks) != 0 {
		t.Fatalf("%+v %v", tasks, err)
	}
	create(t, s, "ok")
	a := claim(t, s, "ok", "a", "one")
	bad := update("bad", "checkpoint", a)
	bad.Usage = &Usage{InputTokens: -1}
	if _, err = s.Apply(bad); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
}

func TestAcceptedDefinitionRemainsClaimableWithFullIdentity(t *testing.T) {
	s := openTest(t)
	_, err := s.Apply(Mutation{ID: "large", Project: "p", Task: "large", Kind: "create", Definition: &Definition{Goal: strings.Repeat("g", 65330)}})
	if !errors.Is(err, ErrInvalid) {
		t.Fatal("must reject definition before it creates unclaimable work", err)
	}
	create(t, s, "normal")
	task := apply(t, s, Mutation{ID: "near-limit", Project: "p", Task: "bounded", Kind: "create", Definition: &Definition{Goal: strings.Repeat("g", 32000)}})
	task = apply(t, s, Mutation{ID: "claim-bounded", Project: "p", Task: task.ID, Kind: "claim", Owner: strings.Repeat("a", 128), Session: strings.Repeat("s", 128), LeaseSeconds: 60})
	if task.State != "running" {
		t.Fatal(task)
	}
}
func TestAggregateDependencyContextHasARecoverableBudget(t *testing.T) {
	s := openTest(t)
	for _, id := range []string{"a", "b", "c", "d"} {
		create(t, s, id)
		task := claim(t, s, id, "worker", id)
		done := update("complete-"+id, "complete", task)
		done.Checkpoint = &Checkpoint{Artifacts: []Reference{{URI: "artifact:" + strings.Repeat("x", 20000)}}}
		apply(t, s, done)
	}
	create(t, s, "consumer", "a", "b", "c", "d")
	if _, err := s.Context("p", "consumer", 65536); !errors.Is(err, ErrBudget) {
		t.Fatal(err)
	}
	value, err := s.Context("p", "consumer", 100000)
	if err != nil || len(value.Dependencies) != 4 {
		t.Fatalf("valid context unrecoverable: %v", err)
	}
}

func TestResumeSizeDoesNotGrowWithCheckpointHistory(t *testing.T) {
	s := openTest(t)
	create(t, s, "api")
	task := claim(t, s, "api", "alice", "one")
	checkpoint := &Checkpoint{Summary: strings.Repeat("completed step; ", 100), Next: "Verify integration", Artifacts: []Reference{{URI: "git:commit:file"}}}
	var firstSize int
	for i := 0; i < 50; i++ {
		m := update(fmt.Sprintf("checkpoint-%d", i), "checkpoint", task)
		m.Checkpoint = checkpoint
		m.LeaseSeconds = 60
		task = apply(t, s, m)
		brief, err := s.Context("p", "api", 16384)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(brief)
		if i == 0 {
			firstSize = len(raw)
		}
		if len(raw) > firstSize+16 {
			t.Fatalf("resume grew with history: %d -> %d", firstSize, len(raw))
		}
	}
	page, err := s.Events("p", 2, 100)
	if err != nil || len(page.Events) != 50 {
		t.Fatal(page, err)
	}
	raw, _ := json.Marshal(page)
	if bytes.Contains(raw, []byte("completed step")) {
		t.Fatal("checkpoint bodies leaked into event feed")
	}
	t.Logf("50 checkpoints; initial resume %d bytes; compact event feed %d bytes (not token counts)", firstSize, len(raw))
}
