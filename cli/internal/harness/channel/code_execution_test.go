package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCodeExecutionHelper(t *testing.T) {
	if os.Getenv("STACKCORD_CODE_TEST_HELPER") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) < 2 {
		return
	}
	switch args[1] {
	case "edit", "hang":
		var r Event
		data, _ := io.ReadAll(os.Stdin)
		if json.Unmarshal(data, &r) != nil {
			os.Exit(2)
		}
		var edit struct {
			Path  string
			Value string
		}
		if json.Unmarshal([]byte(r.Body), &edit) != nil {
			os.Exit(2)
		}
		if err := os.WriteFile(edit.Path, []byte(edit.Value), 0600); err != nil {
			os.Exit(2)
		}
		os.WriteFile(".git/runner-invoked", []byte("yes"), 0600)
		if args[1] == "hang" {
			time.Sleep(10 * time.Second)
		}
		fmt.Print(`{"status":"success","body":"done"}`)
	case "verify":
		a, _ := os.ReadFile("src/a.txt")
		b, _ := os.ReadFile("src/b.txt")
		if string(a) == "broken" || (string(a) == "new" && string(b) == "new") {
			os.Exit(1)
		}
		if string(a) == "new" && os.Getenv("STACKCORD_TEST_PUBLICATION_REMOTE") != "" {
			remote := os.Getenv("STACKCORD_TEST_PUBLICATION_REMOTE")
			if err := os.Rename(remote, remote+".offline"); err != nil {
				os.Exit(2)
			}
		}
	default:
		os.Exit(2)
	}
	os.Exit(0)
}

func TestCodeInterruptedRetryPreservesOldJob(t *testing.T) {
	a, b, base := codeFixture(t)
	c, _ := b.load()
	c.Runner.Argv[len(c.Runner.Argv)-1] = "hang"
	c.Runner.TimeoutSeconds = 1
	if _, err := b.ConfigureRunner(c.Runner); err != nil {
		t.Fatal(err)
	}
	r := sendCode(t, a, "bob", base, "src/a.txt", "new")
	st, err := b.WorkOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.WorkerBlockedReason == "" {
		t.Fatal("interrupted code job was not quarantined")
	}
	if _, err = b.Retry(context.Background(), r.ID); err != nil {
		t.Fatal(err)
	}
	c.Runner.Argv[len(c.Runner.Argv)-1] = "edit"
	c.Runner.TimeoutSeconds = 30
	if _, err = b.ConfigureRunner(c.Runner); err != nil {
		t.Fatal(err)
	}
	st, err = b.WorkOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Requests[0].Result == nil || st.Requests[0].Result.Status != "success" {
		t.Fatal("fresh isolated retry did not succeed")
	}
	old, _ := filepath.Glob(b.jobPath(r.ID) + "-interrupted-*")
	if len(old) != 1 {
		t.Fatal("interrupted job was not retained for inspection")
	}
}

func codeGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, data)
	}
	return strings.TrimSpace(string(data))
}
func codeFixture(t *testing.T) (*Store, *Store, string) {
	t.Helper()
	t.Setenv("STACKCORD_CODE_TEST_HELPER", "1")
	a, b := fixture(t)
	remote := filepath.Join(t.TempDir(), "code.git")
	codeGit(t, a.root, "init", "--bare", remote)
	codeGit(t, a.root, "init", "-b", "main")
	codeGit(t, a.root, "config", "user.name", "Fixture")
	codeGit(t, a.root, "config", "user.email", "fixture@example.invalid")
	os.MkdirAll(filepath.Join(a.root, "src"), 0700)
	for _, p := range []string{"src/a.txt", "src/b.txt"} {
		os.WriteFile(filepath.Join(a.root, p), []byte("old"), 0600)
	}
	os.WriteFile(filepath.Join(a.root, ".gitignore"), []byte(".harness/local/\n"), 0600)
	codeGit(t, a.root, "add", "--all")
	codeGit(t, a.root, "commit", "-m", "test: baseline")
	base := codeGit(t, a.root, "rev-parse", "HEAD")
	codeGit(t, a.root, "push", remote, "HEAD:refs/heads/main")
	codeGit(t, b.root, "init", "-b", "main")
	codeGit(t, b.root, "fetch", remote, "main")
	codeGit(t, b.root, "checkout", "-B", "main", "FETCH_HEAD")
	exe, _ := os.Executable()
	policy := &CodePolicy{Repository: "project", Remote: remote, VerifyArgv: []string{exe, "-test.run=^TestCodeExecutionHelper$", "--", "verify"}, VerifyTimeoutSeconds: 30}
	for _, s := range []*Store{a, b} {
		_, err := s.ConfigureRunner(RunnerConfig{Argv: []string{exe, "-test.run=^TestCodeExecutionHelper$", "--", "edit"}, AllowedKinds: []string{"implementation"}, ResultFormat: "json", Code: policy})
		if err != nil {
			t.Fatal(err)
		}
	}
	return a, b, base
}
func sendCode(t *testing.T, s *Store, to, base, p, value string, deps ...string) Event {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"Path": p, "Value": value})
	r, err := s.Send(context.Background(), RequestInput{To: to, Kind: "implementation", Title: "checked edit", Body: string(body), Scope: []string{p}, Dependencies: deps, Code: &CodeRequest{Repository: "project", BaseCommit: base}})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestCodeExecutionIsolatedAndCommitVerified(t *testing.T) {
	a, b, base := codeFixture(t)
	r := sendCode(t, a, "bob", base, "src/a.txt", "new")
	st, err := b.WorkOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Requests) != 1 || st.Requests[0].Result == nil || st.Requests[0].Result.Status != "success" || st.Requests[0].Result.Artifact == nil {
		t.Fatalf("not verified: %+v", st.Requests)
	}
	artifact := st.Requests[0].Result.Artifact
	if artifact.Commit == base || artifact.BaseCommit != base {
		t.Fatal("wrong artifact identity")
	}
	if got, _ := os.ReadFile(filepath.Join(b.root, "src/a.txt")); string(got) != "old" {
		t.Fatal("worker changed original checkout")
	}
	c, _ := b.load()
	if got := codeGit(t, b.root, "ls-remote", c.Runner.Code.Remote, "refs/heads/work/"+r.ID); !strings.HasPrefix(got, artifact.Commit) {
		t.Fatalf("artifact not published: %s", got)
	}
	if _, err := b.WorkOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
}
func TestCodeFalseSuccessAndScopeEscapeRejected(t *testing.T) {
	for _, mode := range []string{"broken", "scope"} {
		t.Run(mode, func(t *testing.T) {
			a, b, base := codeFixture(t)
			r := sendCode(t, a, "bob", base, "src/a.txt", "broken")
			if mode == "scope" {
				// A declared file cannot authorize writes to its neighbor.
				// Use a separate request whose body and declared scope differ.
				body, _ := json.Marshal(map[string]string{"Path": "src/b.txt", "Value": "new"})
				// The first request is failed explicitly so it does not execute in this subcase.
				if _, err := b.Respond(context.Background(), ResultInput{RequestID: r.ID, Status: "failed"}); err != nil {
					t.Fatal(err)
				}
				_, err := a.Send(context.Background(), RequestInput{To: "bob", Kind: "implementation", Title: "scope escape", Body: string(body), Scope: []string{"src/c.txt"}, Code: &CodeRequest{Repository: "project", BaseCommit: base}})
				if err != nil {
					t.Fatal(err)
				}
			}
			st, err := b.WorkOnce(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			last := st.Requests[len(st.Requests)-1]
			if last.Result == nil || last.Result.Status != "failed" || last.Result.Artifact != nil {
				t.Fatalf("false success: %+v", last)
			}
		})
	}
}

func TestCodeModeCannotBeBypassedByMessageOnlyRequest(t *testing.T) {
	a, b, _ := codeFixture(t)
	_, err := a.Send(context.Background(), RequestInput{To: "bob", Kind: "implementation", Title: "missing code metadata", Body: `{"Path":"src/a.txt","Value":"new"}`})
	if err != nil {
		t.Fatal(err)
	}
	st, err := b.WorkOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Requests[0].Status != "blocked" {
		t.Fatal("code policy bypassed by legacy request")
	}
	data, _ := os.ReadFile(filepath.Join(b.root, "src/a.txt"))
	if string(data) != "old" {
		t.Fatal("legacy request mutated code-mode checkout")
	}
}

func TestCodePublicationRetryDoesNotRerun(t *testing.T) {
	a, b, base := codeFixture(t)
	c, _ := b.load()
	t.Setenv("STACKCORD_TEST_PUBLICATION_REMOTE", c.Runner.Code.Remote)
	r := sendCode(t, a, "bob", base, "src/a.txt", "new")
	if _, err := b.WorkOnce(context.Background()); err == nil {
		t.Fatal("expected interrupted publication")
	}
	if err := os.Rename(c.Runner.Code.Remote+".offline", c.Runner.Code.Remote); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STACKCORD_TEST_PUBLICATION_REMOTE", "")
	c.Runner.Argv = []string{c.Runner.Argv[0], "-test.run=^TestCodeExecutionHelper$", "--", "reject"}
	if _, err := b.ConfigureRunner(c.Runner); err != nil {
		t.Fatal(err)
	}
	st, err := b.WorkOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Requests[0].Result == nil || st.Requests[0].Result.Status != "success" || st.Requests[0].Result.RequestID != r.ID {
		t.Fatal("durable result did not survive publication retry")
	}
}
func TestCodeCombinedDependenciesCheckedBeforeRunner(t *testing.T) {
	a, b, base := codeFixture(t)
	first := sendCode(t, a, "bob", base, "src/a.txt", "new")
	second := sendCode(t, a, "bob", base, "src/b.txt", "new")
	for i := 0; i < 2; i++ {
		if _, err := b.WorkOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	dependent := sendCode(t, a, "alice", base, "src/c.txt", "new", first.ID, second.ID)
	st, err := a.WorkOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	last := st.Requests[len(st.Requests)-1]
	if last.Result == nil || last.Result.Status != "failed" {
		t.Fatalf("incompatible integration accepted: %+v", last)
	}
	if _, err := os.Stat(filepath.Join(a.jobPath(dependent.ID), ".git", "runner-invoked")); !os.IsNotExist(err) {
		t.Fatal("downstream AI ran before integration passed")
	}
}

func TestCodeHostAccessIsLimitedToMailboxNotOtherJobs(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if rel, _ := filepath.Rel(s.dir, s.jobPath("request")); !strings.HasPrefix(rel, "..") {
		t.Fatal("mailbox permission also grants access to every code job")
	}
	args := codeRunnerArguments(RunnerConfig{Host: "codex", Argv: []string{"codex", "exec", "instruction"}}, s.dir)
	if strings.Join(args, "|") != "codex|exec|--add-dir|"+s.dir+"|instruction" {
		t.Fatalf("wrong codex mailbox permission: %v", args)
	}
	args = codeRunnerArguments(RunnerConfig{Host: "claude", Argv: []string{"claude", "--append-system-prompt", "instruction"}}, s.dir)
	if strings.Join(args, "|") != "claude|--append-system-prompt|instruction|--add-dir|"+s.dir {
		t.Fatalf("wrong claude mailbox permission: %v", args)
	}
	args = codeRunnerArguments(RunnerConfig{Argv: []string{"custom", "arg"}}, s.dir)
	if strings.Join(args, "|") != "custom|arg" {
		t.Fatal("custom runner arguments were rewritten")
	}
}

func TestCodeConcurrentConflictingClaimsHaveOneWinner(t *testing.T) {
	a, b, base := codeFixture(t)
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, worker := range []*Store{a, b} {
		go func(s *Store) {
			<-start
			_, err := s.Send(context.Background(), RequestInput{To: "bob", Kind: "implementation", Title: "shared contract", Scope: []string{"src"}, Code: &CodeRequest{Repository: "project", BaseCommit: base, Resources: []string{"contract.api"}}})
			results <- err
		}(worker)
	}
	close(start)
	accepted := 0
	for i := 0; i < 2; i++ {
		if err := <-results; err == nil {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("concurrent overlapping claims accepted: %d", accepted)
	}
	st, err := a.State(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Requests) != 1 {
		t.Fatalf("conflicting work entered shared history: %d", len(st.Requests))
	}
}

func TestCodeDependentArtifactContainsActualPrerequisite(t *testing.T) {
	a, b, base := codeFixture(t)
	first := sendCode(t, a, "bob", base, "src/a.txt", "new")
	if _, err := b.WorkOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	dependent := sendCode(t, a, "alice", base, "src/c.txt", "new", first.ID)
	st, err := a.WorkOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	last := st.Requests[len(st.Requests)-1]
	if last.Result == nil || last.Result.Status != "success" || last.Result.Artifact == nil {
		t.Fatalf("dependent failed: %+v", last)
	}
	job := a.jobPath(dependent.ID)
	if got := codeGit(t, job, "show", last.Result.Artifact.Commit+":src/a.txt"); got != "new" {
		t.Fatal("runner did not consume actual prerequisite code")
	}
	if got := codeGit(t, job, "show", last.Result.Artifact.Commit+":src/c.txt"); got != "new" {
		t.Fatal("dependent artifact missing its own change")
	}
}

func TestCodeGitIgnoresCallerRepositoryOverrides(t *testing.T) {
	a, _, base := codeFixture(t)
	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), "not-this-repository"))
	head, err := codeGitRun(context.Background(), a.root, "rev-parse", "HEAD")
	if err != nil || head != base {
		t.Fatalf("caller environment redirected isolated Git: %q %v", head, err)
	}
}

func TestCodeNewBaselineMustContainCompletedOverlappingWork(t *testing.T) {
	a, b, base := codeFixture(t)
	sendCode(t, a, "bob", base, "src/a.txt", "new")
	if _, err := b.WorkOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(a.root, "baseline-next.txt"), []byte("unrelated change"), 0600)
	codeGit(t, a.root, "add", "baseline-next.txt")
	codeGit(t, a.root, "commit", "-m", "test: another baseline")
	newBase := codeGit(t, a.root, "rev-parse", "HEAD")
	c, _ := a.load()
	codeGit(t, a.root, "push", c.Runner.Code.Remote, "HEAD:refs/heads/main")
	r := sendCode(t, a, "alice", newBase, "src/a.txt", "third")
	st, err := a.WorkOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	last := st.Requests[len(st.Requests)-1]
	if last.Result == nil || last.Result.Status != "failed" {
		t.Fatal("new baseline silently discarded completed overlapping work")
	}
	if _, err := os.Stat(filepath.Join(a.jobPath(r.ID), ".git", "runner-invoked")); !os.IsNotExist(err) {
		t.Fatal("runner started on stale input")
	}
}

func TestCodeSequentialAndParallelWorkersProduceWorkingIntegration(t *testing.T) {
	for _, parallel := range []bool{false, true} {
		t.Run(fmt.Sprintf("parallel=%t", parallel), func(t *testing.T) {
			a, b, base := codeFixture(t)
			first := sendCode(t, a, "bob", base, "src/a.txt", "new")
			second := sendCode(t, a, "alice", base, "src/b.txt", "other")
			started := time.Now()
			if parallel {
				errs := make(chan error, 2)
				for _, worker := range []*Store{a, b} {
					go func(s *Store) { _, err := s.WorkOnce(context.Background()); errs <- err }(worker)
				}
				for i := 0; i < 2; i++ {
					if err := <-errs; err != nil {
						t.Fatal(err)
					}
				}
			} else {
				for _, worker := range []*Store{a, b} {
					if _, err := worker.WorkOnce(context.Background()); err != nil {
						t.Fatal(err)
					}
				}
			}
			joined := sendCode(t, a, "alice", base, "src/c.txt", "new", first.ID, second.ID)
			st, err := a.WorkOnce(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			last := st.Requests[len(st.Requests)-1]
			if last.Result == nil || last.Result.Artifact == nil || last.Result.Status != "success" {
				t.Fatalf("integration failed: %+v", last)
			}
			for p, want := range map[string]string{"src/a.txt": "new", "src/b.txt": "other", "src/c.txt": "new"} {
				if got := codeGit(t, a.jobPath(joined.ID), "show", last.Result.Artifact.Commit+":"+p); got != want {
					t.Fatalf("%s: %q", p, got)
				}
			}
			t.Logf("verified end-to-end native helper workload: parallel=%t elapsed=%s (not a model benchmark)", parallel, time.Since(started))
		})
	}
}
