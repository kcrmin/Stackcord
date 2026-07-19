package governance

import (
	"context"
	"errors"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/gitx"
)

const MaxObservationAge = 15 * time.Minute

// Check evaluates committed policy against actual Git and one fresh normalized provider review.
func Check(ctx context.Context, root, observationPath string, now time.Time) Report {
	report := Report{Status: Blocked, Authorities: []string{}, Approvers: []string{}, Issues: []domain.Item{}}
	policy, err := LoadPolicy(root)
	if err != nil {
		report.Issues = append(report.Issues, issue("governance.policy-invalid", err.Error()))
		return report
	}
	report.Enabled = policy.Enabled
	report.Authorities = append([]string(nil), policy.ProductAuthorities...)
	report.ProtectedFingerprint, err = ProtectedFingerprint(root)
	if err != nil {
		report.Issues = append(report.Issues, issue("governance.fingerprint-invalid", err.Error()))
		return report
	}
	if !policy.Enabled {
		report.Status = Disabled
		report.ApprovalRevision = "disabled"
		return report
	}
	state, err := gitx.Inspect(ctx, root)
	if err != nil {
		report.Status = Unknown
		report.Issues = append(report.Issues, issue("governance.git-unknown", "Actual Git identity is unavailable."))
		return report
	}
	observation, err := loadObservation(root, observationPath)
	if err != nil {
		report.Status = Unknown
		code := "governance.approval-unknown"
		if !errors.Is(err, os.ErrNotExist) {
			code = "governance.observation-invalid"
		}
		report.Issues = append(report.Issues, issue(code, err.Error()))
		return report
	}
	report.ApprovalRevision = observation.ReviewID + ":" + observation.ReviewRevision
	identityMismatch := false
	checkIdentity := func(ok bool, code, message string, refs ...string) {
		if !ok {
			identityMismatch = true
			report.Issues = append(report.Issues, issue(code, message, refs...))
		}
	}
	checkIdentity(observation.Provider == policy.Provider, "governance.provider-mismatch", "Approval provider differs from the configured Git review provider.", policy.Provider, observation.Provider)
	checkIdentity(observation.Repository == policy.Repository, "governance.repository-mismatch", "Approval belongs to a different repository.", policy.Repository, observation.Repository)
	checkIdentity(observation.HeadCommit == state.Head, "governance.commit-stale", "Approval belongs to a different commit.", state.Head, observation.HeadCommit)
	checkIdentity(observation.ProtectedFingerprint == report.ProtectedFingerprint, "governance.meaning-stale", "Protected product meaning changed after approval.", report.ProtectedFingerprint, observation.ProtectedFingerprint)
	checkIdentity(observation.Status == "approved" || observation.Status == "merged", "governance.review-incomplete", "The provider review has not approved the protected change.", observation.Status)
	if identityMismatch {
		report.Status = Blocked
		report.Issues = normalizeIssues(report.Issues)
		return report
	}
	if observation.Source != "connector-live" || observation.FetchedAt.IsZero() || now.Sub(observation.FetchedAt) > MaxObservationAge || observation.FetchedAt.After(now.Add(2*time.Minute)) {
		report.Status = Unknown
		report.Issues = append(report.Issues, issue("governance.approval-unknown", "A fresh live Git review observation is required."))
		report.Issues = normalizeIssues(report.Issues)
		return report
	}
	authorities := map[string]bool{}
	for _, subject := range policy.ProductAuthorities {
		authorities[subject] = true
	}
	approvers := map[string]bool{}
	for _, decision := range observation.Decisions {
		if decision.State != "approved" || (decision.Kind != "review" && decision.Kind != "merge") || !authorities[decision.Subject] {
			continue
		}
		if !policy.Approval.AuthoritySelfApproval && decision.Subject == observation.AuthorSubject {
			continue
		}
		approvers[decision.Subject] = true
	}
	for subject := range approvers {
		report.Approvers = append(report.Approvers, subject)
	}
	sort.Strings(report.Approvers)
	if len(report.Approvers) < policy.Approval.Minimum {
		report.Status = Proposed
		report.Issues = append(report.Issues, issue("governance.approval-insufficient", "Protected product meaning still needs approval from a configured product authority."))
		report.Issues = normalizeIssues(report.Issues)
		return report
	}
	report.Status = Approved
	report.Issues = normalizeIssues(report.Issues)
	return report
}

func issue(code, message string, refs ...string) domain.Item {
	return domain.Item{Code: code, Message: message, Refs: uniqueSorted(refs)}
}

func normalizeIssues(items []domain.Item) []domain.Item {
	for index := range items {
		items[index].Refs = uniqueSorted(items[index].Refs)
	}
	sort.Slice(items, func(left, right int) bool {
		if items[left].Code == items[right].Code {
			return strings.Join(items[left].Refs, "\x00") < strings.Join(items[right].Refs, "\x00")
		}
		return items[left].Code < items[right].Code
	})
	return items
}
