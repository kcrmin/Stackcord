package github

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type protectionTransport struct {
	base                          *fakeTransport
	classic, rules                string
	classicErr, rulesErr, repoErr error
}

func readinessClient(t *testing.T, f *fakeTransport) *Client {
	t.Helper()
	c, e := New("owner/repo", &protectionTransport{base: f, classicErr: ErrNotFound, rules: `[[]]`})
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func (f *protectionTransport) Run(ctx context.Context, args ...string) ([]byte, error) {
	switch args[3] {
	case "repos/owner/repo":
		return []byte(`{"full_name":"owner/repo","default_branch":"main"}`), f.repoErr
	case "repos/owner/repo/branches/main/protection/required_status_checks":
		return []byte(f.classic), f.classicErr
	case "repos/owner/repo/rules/branches/main?per_page=100":
		return []byte(f.rules), f.rulesErr
	}
	return f.base.Run(ctx, args...)
}

func TestRequiredChecksDiscoversClassicAndEffectiveRules(t *testing.T) {
	f := &protectionTransport{classic: `{"contexts":["ci"],"checks":[{"context":"ci","app_id":42}]}`, rules: `[[{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"security","integration_id":99}]}}],[]]`}
	c, _ := New("owner/repo", f)
	checks, e := c.RequiredChecks(context.Background(), "main")
	if e != nil || len(checks) != 2 || checks[0].Context != "ci" || checks[0].AppID != 42 || checks[1].Context != "security" || checks[1].AppID != 99 {
		t.Fatal(checks, e)
	}
}
func TestRequiredChecksAllowsProvenUnprotectedBranch(t *testing.T) {
	f := &protectionTransport{classicErr: ErrNotFound, rules: `[[]]`}
	c, _ := New("owner/repo", f)
	checks, e := c.RequiredChecks(context.Background(), "main")
	if e != nil || checks == nil || len(checks) != 0 {
		t.Fatal(checks, e)
	}
}
func TestRequiredChecksFailsClosedOnInaccessibleOrInvalidData(t *testing.T) {
	for _, f := range []*protectionTransport{
		{repoErr: ErrNotFound}, {classicErr: errors.New("403 secret")}, {classicErr: ErrNotFound, rulesErr: errors.New("503 secret")},
		{classic: `null`, rules: `[[]]`}, {classic: `{}`, rules: `[[]]`}, {classic: `{"contexts":[],"checks":[]}`, rules: `[[{"type":"required_status_checks","parameters":{}}]]`},
	} {
		c, _ := New("owner/repo", f)
		if _, e := c.RequiredChecks(context.Background(), "main"); e == nil || strings.Contains(e.Error(), "secret") {
			t.Fatal("unknown requirement accepted or leaked", e)
		}
	}
}

func TestReadinessRequiresConfiguredContextsAndApps(t *testing.T) {
	for _, tc := range []struct {
		name, check string
		want        bool
	}{
		{"present", `{"name":"ci","status":"completed","conclusion":"success","app":{"id":42}}`, true},
		{"missing", `{"name":"optional","status":"completed","conclusion":"success"}`, false},
		{"wrong app", `{"name":"ci","status":"completed","conclusion":"success","app":{"id":7}}`, false},
		{"pending", `{"name":"ci","status":"in_progress","app":{"id":42}}`, false},
	} {
		pr := `{"state":"open","base":{"ref":"main"},"head":{"sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"mergeable":true,"mergeable_state":"blocked"}`
		f := &protectionTransport{classic: `{"contexts":[],"checks":[{"context":"ci","app_id":42}]}`, rules: `[[]]`, base: &fakeTransport{outputs: []string{pr, `[{"check_runs":[` + tc.check + `]}]`, `[{"statuses":[]}]`, pr}}}
		c, _ := New("owner/repo", f)
		r, e := c.Readiness(context.Background(), 1, strings.Repeat("a", 40))
		if e != nil || r.Ready != tc.want {
			t.Fatal(tc.name, r, e)
		}
	}
}
