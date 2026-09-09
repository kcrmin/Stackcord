package channel

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Store, *Store) {
	t.Helper()
	remote := filepath.Join(t.TempDir(), "mail.git")
	if out, e := exec.Command("git", "init", "--bare", remote).CombinedOutput(); e != nil {
		t.Fatalf("%s %v", out, e)
	}
	a, _ := Open(t.TempDir())
	b, _ := Open(t.TempDir())
	sa, e := a.Setup(Config{Channel: "test-channel", Remote: remote, Peer: "alice"})
	if e != nil {
		t.Fatal(e)
	}
	sb, e := b.Setup(Config{Channel: "test-channel", Remote: remote, Peer: "bob"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.Trust("bob", sb.PublicKey); e != nil {
		t.Fatal(e)
	}
	if _, e = b.Trust("alice", sa.PublicKey); e != nil {
		t.Fatal(e)
	}
	return a, b
}

func TestProjectAliasResolvesButChannelSymlinksRemainBlocked(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(t.TempDir(), "project-alias")
	if e := os.Symlink(root, alias); e != nil {
		t.Skipf("symlink privilege unavailable: %v", e)
	}
	s, e := Open(alias)
	if e != nil {
		t.Fatal(e)
	}
	state, e := s.State(context.Background(), false)
	if e != nil || state.Configured {
		t.Fatalf("ordinary project alias: %+v %v", state, e)
	}
	target := t.TempDir()
	if e = os.Symlink(target, filepath.Join(root, ".harness")); e != nil {
		t.Fatal(e)
	}
	if _, e = s.State(context.Background(), false); e == nil {
		t.Fatal("channel storage symlink accepted")
	}
}
func TestSignedRoundTripAndDependencies(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	first, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "first", Body: "hello"})
	if e != nil {
		t.Fatal(e)
	}
	_, e = a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "second", Dependencies: []string{first.ID}})
	if e != nil {
		t.Fatal(e)
	}
	s, e := b.State(ctx, true)
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Requests) != 2 || s.Requests[1].Status != "blocked" {
		t.Fatalf("%+v", s)
	}
	if _, e = a.Respond(ctx, ResultInput{RequestID: first.ID, Status: "success"}); e == nil {
		t.Fatal("nonrecipient response accepted")
	}
	if _, e = b.Respond(ctx, ResultInput{RequestID: first.ID, Status: "success", Body: "done"}); e != nil {
		t.Fatal(e)
	}
	s, e = a.State(ctx, true)
	if e != nil || s.Requests[1].Status != "ready" {
		t.Fatalf("%+v %v", s, e)
	}
}
func TestMainWorktreeUntouchedAndSecretNotInState(t *testing.T) {
	a, _ := fixture(t)
	if _, e := os.Stat(filepath.Join(a.root, ".git")); !os.IsNotExist(e) {
		t.Fatal("main git modified")
	}
	s, e := a.State(context.Background(), false)
	if e != nil || s.PublicKey == "" {
		t.Fatalf("%+v %v", s, e)
	}
}

func TestRetryRejectsInvalidIDBeforeStorageAccess(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "../outside", `..\outside`, "/absolute", "a/b", "a:b"} {
		if _, err := s.Retry(context.Background(), id); err == nil || err.Error() != "invalid request identifier" {
			t.Fatalf("Retry(%q): %v", id, err)
		}
	}
	if _, err := os.Stat(s.dir); !os.IsNotExist(err) {
		t.Fatalf("invalid retry touched storage: %v", err)
	}
}

func TestWorkerDurableReceiptAndExplicitRetry(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	r, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "work"})
	if e != nil {
		t.Fatal(e)
	}
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = b.ConfigureRunner(RunnerConfig{Argv: []string{exe, "-test.run=TestRunnerHelper", b.root}, AllowedKinds: []string{"review"}, TimeoutSeconds: 5}); e != nil {
		t.Fatal(e)
	}
	if e = writeJSON(filepath.Join(b.dir, "receipt-"+r.ID+".json"), receipt{RequestID: r.ID, Started: true}); e != nil {
		t.Fatal(e)
	}
	s, e := b.WorkOnce(ctx)
	if e != nil || s.Requests[0].Status != "interrupted" {
		t.Fatalf("%+v %v", s, e)
	}
	if _, e = b.Retry(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	s, e = b.WorkOnce(ctx)
	if e != nil || s.Requests[0].Status != "success" {
		t.Fatalf("%+v %v", s, e)
	}
	s, e = b.WorkOnce(ctx)
	if e != nil || s.Requests[0].Status != "success" {
		t.Fatalf("%+v %v", s, e)
	}
}
func TestRunnerHelper(t *testing.T) {
	if len(os.Args) > 1 && os.Args[1] == "-test.run=TestRunnerHelper" {
		store, err := Open(os.Args[2])
		if err != nil {
			os.Exit(3)
		}
		if _, err = store.State(context.Background(), false); err != nil {
			os.Exit(4)
		}
		if len(os.Args) > 3 {
			if os.Args[3] == "spawn-child" {
				exe, _ := os.Executable()
				child := exec.Command(exe, "-test.run=TestDetachedDescendantHelper", filepath.Join(os.Args[2], "descendant-marker"))
				if child.Start() != nil {
					os.Exit(5)
				}
				time.Sleep(20 * time.Second)
			}
			if os.Args[3] == "duplicate-json" {
				os.Stdout.WriteString(`{"status":"failed","status":"success","body":"ambiguous"}`)
				os.Exit(0)
			}
			if os.Args[3] == "blocked-json" {
				os.Stdout.WriteString(`{"status":"failed","body":"blocked by host"}`)
				os.Exit(0)
			}
			if os.Args[3] == "malformed-json" {
				os.Stdout.WriteString("PRIVATE malformed")
				os.Exit(0)
			}
			if os.Args[3] == "timeout" {
				time.Sleep(5 * time.Second)
			}
			if os.Args[3] == "large" {
				os.Stdout.WriteString(strings.Repeat("\x00", 200000))
			}
		}
		os.Stderr.WriteString("PRIVATE STDERR")
		os.Stdout.WriteString("completed")
		os.Exit(0)
	}
}
func TestUntrustedTamperReplayAndRewrite(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	r, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "original"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = b.State(ctx, true); e != nil {
		t.Fatal(e)
	}
	c, _ := a.load()
	events, head, e := a.read(ctx, c, false)
	if e != nil {
		t.Fatal(e)
	}
	bad := r
	bad.Body = "tampered"
	if e = validate(c, nil, bad, true); e == nil {
		t.Fatal("tamper accepted")
	}
	if e = validate(c, events, r, true); e == nil {
		t.Fatal("replay accepted")
	}
	delete(c.Peers, "alice")
	if e = validate(c, nil, r, true); e == nil {
		t.Fatal("untrusted accepted")
	}
	c, _ = a.load()
	_, e = a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "new"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = b.State(ctx, true); e != nil {
		t.Fatal(e)
	}
	if _, e = a.git(ctx, nil, "push", "--force", c.Remote, head+":"+branch(c)); e != nil {
		t.Fatal(e)
	}
	if _, e = b.State(ctx, true); e == nil {
		t.Fatal("rollback accepted")
	}
}
func TestConcurrentAppendAndRevision(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	s, e := a.State(ctx, false)
	if e != nil {
		t.Fatal(e)
	}
	a.ExpectedRevision = s.Revision
	if _, e = a.ConfigureRunner(RunnerConfig{}); e != nil {
		t.Fatal(e)
	}
	a.ExpectedRevision = "stale"
	if _, e = a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "stale"}); e == nil {
		t.Fatal("stale revision accepted")
	}
	a.ExpectedRevision = ""
	errs := make(chan error, 2)
	go func() { _, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "a"}); errs <- e }()
	go func() { _, e := b.Send(ctx, RequestInput{To: "alice", Kind: "review", Title: "b"}); errs <- e }()
	for range 2 {
		if e := <-errs; e != nil {
			t.Fatal(e)
		}
	}
	s, e = a.State(ctx, true)
	if e != nil || len(s.Requests) != 2 {
		t.Fatalf("%+v %v", s, e)
	}
}

func TestUnconfiguredStateDoesNotCreateFiles(t *testing.T) {
	root := t.TempDir()
	s, _ := Open(root)
	st, e := s.State(context.Background(), false)
	if e != nil || st.Configured {
		t.Fatalf("%+v %v", st, e)
	}
	if _, e = os.Stat(filepath.Join(root, ".harness")); !os.IsNotExist(e) {
		t.Fatal("read initialized state")
	}
}
func TestCompletedReceiptPublishesWithoutRunner(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	r, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "work"})
	if e != nil {
		t.Fatal(e)
	}
	c, _ := b.load()
	ev := newEvent(c, "result")
	ev.RequestID = r.ID
	ev.Status = "success"
	ev.Body = "already completed"
	if e = writeJSON(filepath.Join(b.dir, "receipt-"+r.ID+".json"), receipt{RequestID: r.ID, Started: true, Result: &ev}); e != nil {
		t.Fatal(e)
	}
	exe, _ := os.Executable()
	if _, e = b.ConfigureRunner(RunnerConfig{Argv: []string{exe, "-invalid-option"}, AllowedKinds: []string{"review"}}); e != nil {
		t.Fatal(e)
	}
	s, e := b.WorkOnce(ctx)
	if e != nil || s.Requests[0].Status != "success" || s.Requests[0].Result.Body != "already completed" {
		t.Fatalf("%+v %v", s, e)
	}
}
func TestRemoteTamperFailsClosed(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	ev, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "authentic"})
	if e != nil {
		t.Fatal(e)
	}
	c, _ := a.load()
	_, head, e := a.read(ctx, c, false)
	if e != nil {
		t.Fatal(e)
	}
	ev.Body = "tampered"
	raw, _ := json.Marshal(ev)
	blob, e := a.git(ctx, raw, "hash-object", "-w", "--stdin")
	if e != nil {
		t.Fatal(e)
	}
	tree, e := a.git(ctx, []byte("100644 blob "+blob+"\tevent.json\n"), "mktree")
	if e != nil {
		t.Fatal(e)
	}
	commit, e := a.git(ctx, nil, "commit-tree", tree, "-p", head, "-m", "tamper")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.git(ctx, nil, "push", c.Remote, commit+":"+branch(c)); e != nil {
		t.Fatal(e)
	}
	if _, e = b.State(ctx, true); e == nil {
		t.Fatal("remote tamper accepted")
	}
}
func TestRunnerStderrNeverPublished(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	_, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "work"})
	if e != nil {
		t.Fatal(e)
	}
	exe, _ := os.Executable()
	_, e = b.ConfigureRunner(RunnerConfig{Argv: []string{exe, "-test.run=TestRunnerHelper", b.root}, AllowedKinds: []string{"review"}})
	if e != nil {
		t.Fatal(e)
	}
	s, e := b.WorkOnce(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(s.Requests[0].Result.Body, "PRIVATE STDERR") {
		t.Fatal("stderr published")
	}
}

func TestWorkerTimeoutAllowedKindsAndOutputBound(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	_, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "work"})
	if e != nil {
		t.Fatal(e)
	}
	exe, _ := os.Executable()
	_, e = b.ConfigureRunner(RunnerConfig{Argv: []string{exe, "-test.run=TestRunnerHelper", b.root, "timeout"}, AllowedKinds: []string{"different"}, TimeoutSeconds: 1})
	if e != nil {
		t.Fatal(e)
	}
	st, e := b.WorkOnce(ctx)
	if e != nil || st.Requests[0].Result != nil {
		t.Fatalf("unallowed executed %+v %v", st, e)
	}
	_, e = b.ConfigureRunner(RunnerConfig{Argv: []string{exe, "-test.run=TestRunnerHelper", b.root, "timeout"}, AllowedKinds: []string{"review"}, TimeoutSeconds: 1})
	if e != nil {
		t.Fatal(e)
	}
	st, e = b.WorkOnce(ctx)
	if e != nil || st.Requests[0].Status != "interrupted" || st.Requests[0].Result != nil {
		t.Fatalf("timeout %+v %v", st, e)
	}
	if _, e = b.Retry(ctx, st.Requests[0].Request.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = b.Respond(ctx, ResultInput{RequestID: st.Requests[0].Request.ID, Status: "failed", Body: "cleanup acknowledged"}); e != nil {
		t.Fatal(e)
	}
	_, e = a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "large"})
	if e != nil {
		t.Fatal(e)
	}
	_, e = b.ConfigureRunner(RunnerConfig{Argv: []string{exe, "-test.run=TestRunnerHelper", b.root, "large"}, AllowedKinds: []string{"review"}, TimeoutSeconds: 5})
	if e != nil {
		t.Fatal(e)
	}
	st, e = b.WorkOnce(ctx)
	if e != nil || st.Requests[1].Status != "success" {
		t.Fatalf("bounded output %+v %v", st, e)
	}
}
func TestSetupRejectsSecretRemoteAndUnsafeRunner(t *testing.T) {
	for _, remote := range []string{"https://token@example.com/mail", "https://example.com/mail?token=secret", "ext::command", "--upload-pack=evil"} {
		s, _ := Open(t.TempDir())
		if _, e := s.Setup(Config{Channel: "test", Peer: "me", Remote: remote}); e == nil {
			t.Fatalf("accepted %s", remote)
		}
	}
	if e := validateRunner(RunnerConfig{Argv: []string{"sh", "-c", "echo hi"}, AllowedKinds: []string{"review"}}); e == nil {
		t.Fatal("relative runner accepted")
	}
}

func TestMailboxNeverChangesExistingGitIndex(t *testing.T) {
	a, _ := fixture(t)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", a.root}, args...)...)
		if out, e := cmd.CombinedOutput(); e != nil {
			t.Fatalf("%s %v", out, e)
		}
	}
	run("init")
	if e := os.WriteFile(filepath.Join(a.root, "tracked.txt"), []byte("staged work"), 0600); e != nil {
		t.Fatal(e)
	}
	run("add", "tracked.txt")
	before, e := os.ReadFile(filepath.Join(a.root, ".git", "index"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.Send(context.Background(), RequestInput{To: "bob", Kind: "review", Title: "isolated"}); e != nil {
		t.Fatal(e)
	}
	after, e := os.ReadFile(filepath.Join(a.root, ".git", "index"))
	if e != nil || !bytes.Equal(before, after) {
		t.Fatal("main Git index changed")
	}
}

func TestUnsignedTrailingDataRejected(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	c, _ := a.load()
	ev := newEvent(c, "request")
	ev.To = "bob"
	ev.Kind = "review"
	ev.Title = "signed"
	key, _ := base64.StdEncoding.DecodeString(c.PrivateKey)
	ev.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(key, signingBytes(ev)))
	raw, _ := json.Marshal(ev)
	raw = append(raw, []byte("\n{\"unsigned\":true}")...)
	blob, e := a.git(ctx, raw, "hash-object", "-w", "--stdin")
	if e != nil {
		t.Fatal(e)
	}
	tree, e := a.git(ctx, []byte("100644 blob "+blob+"\tevent.json\n"), "mktree")
	if e != nil {
		t.Fatal(e)
	}
	commit, e := a.git(ctx, nil, "commit-tree", tree, "-m", "trailing data")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.git(ctx, nil, "push", c.Remote, commit+":"+branch(c)); e != nil {
		t.Fatal(e)
	}
	if _, e = b.State(ctx, true); e == nil {
		t.Fatal("unsigned trailing data accepted")
	}
}

func TestSuccessCannotBypassDependencies(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	first, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "first"})
	if e != nil {
		t.Fatal(e)
	}
	second, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "second", Dependencies: []string{first.ID}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = b.Respond(ctx, ResultInput{RequestID: second.ID, Status: "success"}); e == nil {
		t.Fatal("success bypassed unresolved dependency")
	}
}

func TestHundredEventHistoryReadIsBatched(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	c, _ := a.load()
	key, _ := base64.StdEncoding.DecodeString(c.PrivateKey)
	var stream bytes.Buffer
	for i := 0; i < 100; i++ {
		ev := newEvent(c, "request")
		ev.To = "bob"
		ev.Kind = "review"
		ev.Title = "batch"
		ev.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(key, signingBytes(ev)))
		raw, _ := json.Marshal(ev)
		fmt.Fprintf(&stream, "commit refs/heads/fixture\ncommitter Channel <channel@localhost> 1000000000 +0000\ndata 0\nM 100644 inline event.json\ndata %d\n%s\n\n", len(raw), raw)
	}
	if _, e := a.git(ctx, stream.Bytes(), "fast-import", "--quiet"); e != nil {
		t.Fatal(e)
	}
	if _, e := a.git(ctx, nil, "push", c.Remote, "refs/heads/fixture:"+branch(c)); e != nil {
		t.Fatal(e)
	}
	start := time.Now()
	s, e := b.State(ctx, true)
	elapsed := time.Since(start)
	if e != nil || len(s.Requests) != 100 {
		t.Fatalf("%+v %v", s, e)
	}
	t.Logf("100 signed events read in %s", elapsed)
	if elapsed > 10*time.Second {
		t.Fatalf("history read is not batched: %s", elapsed)
	}
}

func TestReceiptStateDistinguishesActiveAndAwaitingPublication(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	r, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "work"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = b.State(ctx, true); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(b.dir, "receipt-"+r.ID+".json")
	if e = writeJSON(p, receipt{RequestID: r.ID, Started: true}); e != nil {
		t.Fatal(e)
	}
	u, e := fileLock(filepath.Join(b.dir, "active-"+r.ID+".lock"))
	if e != nil {
		t.Fatal(e)
	}
	st, e := b.State(ctx, false)
	u()
	if e != nil || st.Requests[0].Status != "running" {
		t.Fatalf("active %+v %v", st, e)
	}
	ev := Event{Status: "success"}
	if e = writeJSON(p, receipt{RequestID: r.ID, Started: true, Result: &ev}); e != nil {
		t.Fatal(e)
	}
	st, e = b.State(ctx, false)
	if e != nil || st.Requests[0].Status != "awaiting_publication" {
		t.Fatalf("publication %+v %v", st, e)
	}
}
func TestStructuredRunnerBlockedAndMalformedFailClosed(t *testing.T) {
	for _, mode := range []string{"blocked-json", "malformed-json"} {
		t.Run(mode, func(t *testing.T) {
			a, b := fixture(t)
			ctx := context.Background()
			_, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "work"})
			if e != nil {
				t.Fatal(e)
			}
			exe, _ := os.Executable()
			_, e = b.ConfigureRunner(RunnerConfig{Argv: []string{exe, "-test.run=TestRunnerHelper", b.root, mode}, AllowedKinds: []string{"review"}, ResultFormat: "json"})
			if e != nil {
				t.Fatal(e)
			}
			st, e := b.WorkOnce(ctx)
			if e != nil || st.Requests[0].Status != "failed" {
				t.Fatalf("%+v %v", st, e)
			}
		})
	}
}

func TestMalformedPrivateConfigurationFailsClosed(t *testing.T) {
	a, _ := fixture(t)
	c, _ := a.load()
	c.PrivateKey = "bad"
	if e := writeJSON(filepath.Join(a.dir, "config.json"), c); e != nil {
		t.Fatal(e)
	}
	if _, e := a.State(context.Background(), false); e == nil {
		t.Fatal("malformed private key accepted")
	}
}

func TestStructuredRunnerRejectsDuplicateFields(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	_, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "work"})
	if e != nil {
		t.Fatal(e)
	}
	exe, _ := os.Executable()
	_, e = b.ConfigureRunner(RunnerConfig{Argv: []string{exe, "-test.run=TestRunnerHelper", b.root, "duplicate-json"}, AllowedKinds: []string{"review"}, ResultFormat: "json"})
	if e != nil {
		t.Fatal(e)
	}
	st, e := b.WorkOnce(ctx)
	if e != nil || st.Requests[0].Status != "failed" {
		t.Fatalf("%+v %v", st, e)
	}
}

func TestStructuredResultSuccess(t *testing.T) {
	status, body, e := parseResult(" { \"body\": \"done\", \"status\": \"success\" } \n")
	if e != nil || status != "success" || body != "done" {
		t.Fatalf("%s %s %v", status, body, e)
	}
}

func TestStructuredResultRequiresStringBody(t *testing.T) {
	if _, _, e := parseResult(`{"status":"success","body":null}`); e == nil {
		t.Fatal("null body accepted as a string")
	}
}

func TestGitPreservesExistingCredentialConfiguration(t *testing.T) {
	a, _ := fixture(t)
	config := filepath.Join(t.TempDir(), "gitconfig")
	if e := os.WriteFile(config, []byte("[credential]\n\thelper = approved-local-helper\n"), 0600); e != nil {
		t.Fatal(e)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	out, e := a.git(context.Background(), nil, "config", "--get", "credential.helper")
	if e != nil || out != "approved-local-helper" {
		t.Fatalf("existing Git auth config suppressed: %q %v", out, e)
	}
}

// A timeout can leave detached descendants alive. The channel must quarantine
// execution instead of publishing a completed failure or starting another task.
func TestTimeoutQuarantinesWhileDetachedChildSurvives(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	first, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "spawn child"})
	if e != nil {
		t.Fatal(e)
	}
	_, e = a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "next task"})
	if e != nil {
		t.Fatal(e)
	}
	exe, _ := os.Executable()
	_, e = b.ConfigureRunner(RunnerConfig{Argv: []string{exe, "-test.run=TestRunnerHelper", b.root, "spawn-child"}, AllowedKinds: []string{"review"}, TimeoutSeconds: 2})
	if e != nil {
		t.Fatal(e)
	}
	st, e := b.WorkOnce(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if st.Requests[0].Status != "interrupted" || st.Requests[0].Result != nil {
		t.Fatalf("timeout falsely completed: %+v", st.Requests[0])
	}
	_, e = b.ConfigureRunner(RunnerConfig{Argv: []string{exe, "-test.run=TestRunnerHelper", b.root}, AllowedKinds: []string{"review"}, TimeoutSeconds: 10})
	if e != nil {
		t.Fatal(e)
	}
	st, e = b.WorkOnce(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if st.Requests[1].Result != nil || st.Requests[1].Status != "blocked" {
		t.Fatalf("work escaped quarantine: %+v", st.Requests[1])
	}
	marker := filepath.Join(b.root, "descendant-marker")
	deadline := time.Now().Add(8 * time.Second)
	for {
		if _, e = os.Stat(marker); e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("detached-child fixture never wrote marker")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, e = b.Retry(ctx, first.ID); e != nil {
		t.Fatal(e)
	}
	st, e = b.WorkOnce(ctx)
	if e != nil || st.Requests[0].Status != "success" {
		t.Fatalf("explicit recovery did not resume: %+v %v", st, e)
	}
}
func TestDetachedDescendantHelper(t *testing.T) {
	if len(os.Args) > 2 && os.Args[1] == "-test.run=TestDetachedDescendantHelper" {
		time.Sleep(5 * time.Second)
		if os.WriteFile(os.Args[2], []byte("child survived direct timeout"), 0600) != nil {
			os.Exit(2)
		}
		os.Exit(0)
	}
}

func TestManualResponseCannotBypassExecutionQuarantine(t *testing.T) {
	a, b := fixture(t)
	ctx := context.Background()
	first, e := a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "interrupted"})
	if e != nil {
		t.Fatal(e)
	}
	_, e = a.Send(ctx, RequestInput{To: "bob", Kind: "review", Title: "next"})
	if e != nil {
		t.Fatal(e)
	}
	if e = writeJSON(filepath.Join(b.dir, "receipt-"+first.ID+".json"), receipt{RequestID: first.ID, Started: true}); e != nil {
		t.Fatal(e)
	}
	if _, e = b.Respond(ctx, ResultInput{RequestID: first.ID, Status: "failed", Body: "manual result is not cleanup"}); e != nil {
		t.Fatal(e)
	}
	exe, _ := os.Executable()
	if _, e = b.ConfigureRunner(RunnerConfig{Argv: []string{exe, "-test.run=TestRunnerHelper", b.root}, AllowedKinds: []string{"review"}}); e != nil {
		t.Fatal(e)
	}
	st, e := b.WorkOnce(ctx)
	if e != nil || st.Requests[1].Status != "blocked" || st.Requests[1].Result != nil || st.WorkerBlockedReason == "" {
		t.Fatalf("manual response bypassed quarantine: %+v %v", st, e)
	}
	if _, e = b.Retry(ctx, first.ID); e != nil {
		t.Fatal(e)
	}
	st, e = b.WorkOnce(ctx)
	if e != nil || st.Requests[1].Status != "success" {
		t.Fatalf("cleanup acknowledgement did not recover: %+v %v", st, e)
	}
}
