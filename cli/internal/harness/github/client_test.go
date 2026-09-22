package github

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestFilesRejectProviderTruncation(t *testing.T) {
	fs := make([]File, 3000)
	b, _ := json.Marshal([][]File{fs})
	f := &fakeTransport{outputs: []string{string(b)}}
	if _, e := client(t, f).Files(context.Background(), 1); e == nil {
		t.Fatal("provider file cap silently accepted")
	}
}
func TestApproveRejectsUnknownAuthor(t *testing.T) {
	f := &fakeTransport{outputs: []string{`{"login":"reviewer"}`, `{"number":3,"state":"open","head":{"sha":"head"}}`}}
	if _, e := client(t, f).Approve(context.Background(), 3, "head", "reviewer", "ok"); e == nil || len(f.calls) != 2 {
		t.Fatal("missing author accepted")
	}
}
func TestIssueCreationUsesStableMarkerAndLiteralFields(t *testing.T) {
	f := &fakeTransport{outputs: []string{`[[]]`, `{"number":7,"title":"literal"}`}}
	i, e := client(t, f).EnsureIssue(context.Background(), "work-2", "$(not-shell)", "body")
	if e != nil || i.Number != 7 {
		t.Fatal(i, e)
	}
	args := f.calls[1]
	foundTitle, foundBody := false, false
	for _, a := range args {
		foundTitle = foundTitle || a == "title=$(not-shell)"
		foundBody = foundBody || a == "body=body\n\n<!-- stackcord:issue:work-2 -->"
	}
	if !foundTitle || !foundBody {
		t.Fatal("unsafe request fields", args)
	}
}
func TestReadinessRechecksHead(t *testing.T) {
	f := &fakeTransport{outputs: []string{`{"state":"open","base":{"ref":"main"},"mergeable_state":"clean","head":{"sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"mergeable":true}`, `[{"check_runs":[{"name":"ci","status":"completed","conclusion":"success"}]}]`, `[{"statuses":[]}]`, `{"head":{"sha":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}`}}
	if r, e := readinessClient(t, f).Readiness(context.Background(), 1, strings.Repeat("a", 40)); e == nil || r.Ready {
		t.Fatal("moved head ready")
	}
}
func TestChecksPaginationIncludesLaterFailure(t *testing.T) {
	f := &fakeTransport{outputs: []string{`[{"check_runs":[{"name":"one","status":"completed","conclusion":"success"}]},{"check_runs":[{"name":"two","status":"in_progress"}]}]`, `[{"statuses":[{"id":2,"context":"deploy","state":"failure"}]},{"statuses":[{"id":1,"context":"deploy","state":"success"}]}]`}}
	cs, e := client(t, f).Checks(context.Background(), strings.Repeat("a", 40))
	if e != nil || len(cs) != 3 || cs[2].Conclusion != "failure" {
		t.Fatal(cs, e)
	}
}
func TestAccountMismatchPreventsWrite(t *testing.T) {
	f := &fakeTransport{outputs: []string{`{"login":"other"}`}}
	if _, e := client(t, f).Approve(context.Background(), 3, "head", "reviewer", "ok"); e == nil || len(f.calls) != 1 {
		t.Fatal("account mismatch accepted")
	}
}

type fakeTransport struct {
	outputs []string
	calls   [][]string
	err     error
}

func (f *fakeTransport) Run(_ context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, args)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.outputs) == 0 {
		return nil, errors.New("unexpected call")
	}
	out := f.outputs[0]
	f.outputs = f.outputs[1:]
	return []byte(out), nil
}
func client(t *testing.T, f *fakeTransport) *Client {
	t.Helper()
	c, e := New("owner/repo", f)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestRejectRepositoryInjection(t *testing.T) {
	for _, r := range []string{"--help", "a/b/c", "a/..", "a/b?x=y", "https://github.com/a/b", "a b/c"} {
		if _, e := New(r, &fakeTransport{}); e == nil {
			t.Errorf("accepted %q", r)
		}
	}
}
func TestIdentityAndPagination(t *testing.T) {
	f := &fakeTransport{outputs: []string{`{"login":"Alice","id":12,"type":"User"}`, `[[{"number":1,"title":"first"}],[{"number":2,"title":"second"}]]`}}
	c := client(t, f)
	a, e := c.Identity(context.Background())
	if e != nil || a.Login != "Alice" {
		t.Fatalf("identity %v %v", a, e)
	}
	prs, e := c.OpenPullRequests(context.Background())
	if e != nil || len(prs) != 2 || prs[1].Number != 2 {
		t.Fatalf("pagination %v %v", prs, e)
	}
	if !strings.Contains(strings.Join(f.calls[1], " "), "--paginate --slurp") {
		t.Fatal("pagination not requested")
	}
}
func TestProviderErrorsDoNotLeakCredentials(t *testing.T) {
	c := client(t, &fakeTransport{err: errors.New("token ghp_SUPERSECRET")})
	_, e := c.Identity(context.Background())
	if e == nil || strings.Contains(e.Error(), "SUPERSECRET") {
		t.Fatalf("unsafe error %v", e)
	}
}
func TestEffectiveReviews(t *testing.T) {
	rs := []Review{
		{ID: 1, User: Account{Login: "Alice"}, State: "APPROVED", CommitID: "head"},
		{ID: 2, User: Account{Login: "alice"}, State: "DISMISSED", CommitID: "head"},
		{ID: 3, User: Account{Login: "Bob"}, State: "APPROVED", CommitID: "old"},
		{ID: 4, User: Account{Login: "Bot", Type: "Bot"}, State: "APPROVED", CommitID: "head"},
		{ID: 5, User: Account{Login: "Chris"}, State: "APPROVED", CommitID: "head"},
		{ID: 6, User: Account{Login: "Chris"}, State: "COMMENTED", CommitID: "head"},
		{ID: 7, User: Account{Login: "Dana"}, State: "CHANGES_REQUESTED", CommitID: "old"},
	}
	got := EffectiveReviews(rs, "head")
	if len(got) != 3 {
		t.Fatalf("effective %+v", got)
	}
	for _, r := range got {
		if r.State == "APPROVED" && r.User.Login != "Chris" {
			t.Fatalf("invalid approval %+v", r)
		}
	}
}
func TestApproveRejectsMovedHeadAndSelf(t *testing.T) {
	for _, test := range []struct{ login, sha string }{{"reviewer", "moved"}, {"author", "head"}} {
		f := &fakeTransport{outputs: []string{`{"login":"` + test.login + `"}`, `{"number":3,"state":"open","head":{"sha":"` + test.sha + `"},"user":{"login":"author"}}`}}
		_, e := client(t, f).Approve(context.Background(), 3, "head", test.login, "ok")
		if e == nil {
			t.Fatal("unsafe approval accepted")
		}
		if len(f.calls) != 2 {
			t.Fatal("unexpected write")
		}
	}
}
func TestApprovePinsCommitAndAccount(t *testing.T) {
	f := &fakeTransport{outputs: []string{`{"login":"reviewer"}`, `{"number":3,"state":"open","head":{"sha":"head"},"user":{"login":"author"}}`, `{"id":4,"state":"APPROVED","commit_id":"head"}`}}
	r, e := client(t, f).Approve(context.Background(), 3, "head", "reviewer", "ok")
	if e != nil || r.ID != 4 {
		t.Fatalf("%+v %v", r, e)
	}
	if !strings.Contains(strings.Join(f.calls[2], " "), "commit_id=head") {
		t.Fatal("approval not pinned")
	}
}
func TestIssueRetryFindsClosedIssue(t *testing.T) {
	f := &fakeTransport{outputs: []string{`[[{"number":5,"state":"closed","body":"<!-- stackcord:issue:work-1 -->"}]]`}}
	i, e := client(t, f).EnsureIssue(context.Background(), "work-1", "new", "body")
	if e != nil || i.Number != 5 || len(f.calls) != 1 {
		t.Fatalf("%+v %v", i, e)
	}
}
func TestTrustedFileRequiresSHAAndDistinguishesMissing(t *testing.T) {
	f := &fakeTransport{outputs: []string{`{"encoding":"base64","content":"cG9saWN5"}`}}
	c := client(t, f)
	if _, e := c.ReadFile(context.Background(), ".harness/governance.yaml", "main"); e == nil {
		t.Fatal("mutable ref accepted")
	}
	b, e := c.ReadFile(context.Background(), ".harness/governance.yaml", strings.Repeat("a", 40))
	if e != nil || string(b) != "policy" {
		t.Fatalf("%s %v", b, e)
	}
	f.err = ErrNotFound
	if _, e = c.ReadFile(context.Background(), "a", strings.Repeat("a", 40)); !errors.Is(e, ErrNotFound) {
		t.Fatalf("%v", e)
	}
}
func TestRepositoryBranchFiles(t *testing.T) {
	f := &fakeTransport{outputs: []string{`{"full_name":"owner/repo","default_branch":"main"}`, `{"sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`, `[[{"filename":"new","previous_filename":"old","status":"renamed"}]]`}}
	c := client(t, f)
	r, e := c.Repository(context.Background())
	if e != nil || r.DefaultBranch != "main" {
		t.Fatal(r, e)
	}
	sha, e := c.BranchHead(context.Background(), "main")
	if e != nil || sha != strings.Repeat("a", 40) {
		t.Fatal(sha, e)
	}
	fs, e := c.Files(context.Background(), 3)
	if e != nil || len(fs) != 1 || fs[0].PreviousFilename != "old" {
		t.Fatal(fs, e)
	}
}
func TestReadinessFailsClosed(t *testing.T) {
	for _, test := range []struct {
		state, conclusion string
		mergeable         bool
		ready             bool
	}{{"completed", "success", true, true}, {"in_progress", "", true, false}, {"completed", "failure", true, false}, {"completed", "success", false, false}} {
		pr := `{"number":1,"state":"open","base":{"ref":"main"},"mergeable_state":"clean","head":{"sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"mergeable":` + map[bool]string{true: "true", false: "false"}[test.mergeable] + `}`
		f := &fakeTransport{outputs: []string{pr, `[{"check_runs":[{"id":1,"name":"test","status":"` + test.state + `","conclusion":"` + test.conclusion + `"}]}]`, `[{"state":"pending","statuses":[]}]`, pr}}
		r, e := readinessClient(t, f).Readiness(context.Background(), 1, strings.Repeat("a", 40))
		if e != nil || r.Ready != test.ready {
			t.Fatalf("%+v %v", r, e)
		}
	}
}
