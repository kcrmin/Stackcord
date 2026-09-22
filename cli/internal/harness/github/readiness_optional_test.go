package github

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestOptionalChecksAllowPendingReviewAndRequireRealSuccess(t *testing.T) {
	for _, tc := range []struct {
		conclusion, mergeState string
		success, ready         bool
	}{
		{"skipped", "clean", true, true}, {"neutral", "clean", true, true},
		{"skipped", "clean", false, false}, {"neutral", "blocked", true, true}, {"skipped", "blocked", true, true},
		{"skipped", "unknown", true, false}, {"failure", "clean", true, false},
	} {
		pr := fmt.Sprintf(`{"state":"open","base":{"ref":"main"},"head":{"sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"mergeable":true,"mergeable_state":%q}`, tc.mergeState)
		checks := fmt.Sprintf(`{"name":"scheduled-only","status":"completed","conclusion":%q}`, tc.conclusion)
		if tc.success {
			checks += `,{"name":"CI gate","status":"completed","conclusion":"success"}`
		}
		f := &fakeTransport{outputs: []string{pr, `[{"check_runs":[` + checks + `]}]`, `[{"statuses":[]}]`, pr}}
		r, e := readinessClient(t, f).Readiness(context.Background(), 1, strings.Repeat("a", 40))
		if e != nil || r.Ready != tc.ready {
			t.Fatalf("%+v: %+v %v", tc, r, e)
		}
	}
}

func TestReadinessRejectsMergeabilityBecomingUnknown(t *testing.T) {
	pr := `{"state":"open","base":{"ref":"main"},"head":{"sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"mergeable":true,"mergeable_state":"clean"}`
	f := &fakeTransport{outputs: []string{pr, `[{"check_runs":[{"name":"ci","status":"completed","conclusion":"success"}]}]`, `[{"statuses":[]}]`, strings.Replace(pr, `"clean"`, `"unknown"`, 1)}}
	r, err := readinessClient(t, f).Readiness(context.Background(), 1, strings.Repeat("a", 40))
	if err != nil || r.Ready {
		t.Fatal("unknown final mergeability accepted", r, err)
	}
}
