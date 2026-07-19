package continuity

import (
	"strings"

	"fullstack-orchestrator/cli/internal/domain"
)

func overallConfidence(issues []domain.Item) Confidence {
	result := Confirmed
	for _, item := range issues {
		result = maxConfidence(result, confidenceForCode(item.Code))
	}
	return result
}

func confidenceFromIssues(issues []domain.Item) Confidence {
	result := Confirmed
	for _, item := range issues {
		result = maxConfidence(result, confidenceForCode(item.Code))
	}
	return result
}

func confidenceForCode(code string) Confidence {
	switch {
	case code == "workspace.pointer-mismatch", code == "workspace.diverged", code == "project.root-unavailable", strings.HasPrefix(code, "context.error"), strings.HasSuffix(code, "-invalid"), code == "governance.commit-stale", code == "governance.meaning-stale", code == "governance.provider-mismatch", code == "governance.repository-mismatch":
		return Blocked
	case code == "provider.live-unknown", code == "workspace.git-unknown", code == "workspace.missing", code == "context.unknown", code == "project.not-found", code == "governance.approval-unknown", code == "governance.git-unknown":
		return Unknown
	case code == "context.stale":
		return Stale
	case code == "workspace.local-only":
		return LocalOnly
	default:
		return Warning
	}
}

func maxConfidence(left, right Confidence) Confidence {
	rank := map[Confidence]int{Confirmed: 0, Warning: 1, LocalOnly: 2, Stale: 3, Unknown: 4, Blocked: 5}
	if rank[right] > rank[left] {
		return right
	}
	return left
}

func nextActions(snapshot Snapshot) []domain.Item {
	for _, priority := range []struct {
		code   string
		action domain.Item
	}{
		{"project.root-unavailable", domain.Item{Code: "project.root-locate", Message: "Locate or clone the orchestration root before service-wide work."}},
		{"project.not-found", domain.Item{Code: "project.start-or-adopt", Message: "Start a new harness or adopt the existing project."}},
		{"workspace.pointer-mismatch", domain.Item{Code: "workspace.pointer-review", Message: "Review the child commit and root gitlink before changing either side."}},
		{"workspace.diverged", domain.Item{Code: "workspace.divergence-review", Message: "Choose an explicit branch reconciliation strategy before shared mutation."}},
		{"workspace.missing", domain.Item{Code: "workspace.initialize", Message: "Initialize the declared submodule at the root-pinned commit."}},
		{"context.stale", domain.Item{Code: "context.reconcile", Message: "Reconcile stale canonical dependents before implementation."}},
		{"context.unknown", domain.Item{Code: "context.resolve", Message: "Resolve the highest-impact unknown product or contract reference."}},
		{"governance.commit-stale", domain.Item{Code: "governance.refresh-review", Message: "Request product-authority review for the current protected commit."}},
		{"governance.meaning-stale", domain.Item{Code: "governance.refresh-review", Message: "Request product-authority review again because protected service meaning changed."}},
		{"governance.approval-unknown", domain.Item{Code: "governance.refresh", Message: "Refresh the selected Git review provider before treating the protected change as approved."}},
		{"governance.approval-insufficient", domain.Item{Code: "governance.request-review", Message: "Keep the change as a proposal and request approval from a configured product authority."}},
		{"provider.live-unknown", domain.Item{Code: "provider.reconcile", Message: "Refresh the selected task provider before claiming or starting work."}},
		{"workspace.local-only", domain.Item{Code: "workspace.share-plan", Message: "Choose how the current work becomes recoverable before another person depends on it."}},
		{"workspace.dirty", domain.Item{Code: "workspace.scope-review", Message: "Confirm the current dirty paths belong to the intended change."}},
	} {
		for _, issue := range snapshot.Issues {
			if issue.Code == priority.code {
				priority.action.Refs = issue.Refs
				return []domain.Item{priority.action}
			}
		}
	}
	if len(snapshot.ActiveWork) == 0 {
		return []domain.Item{{Code: "work.define-next", Message: "Define the next smallest end-to-end product slice and its acceptance evidence."}}
	}
	return []domain.Item{{Code: "work.continue", Message: "Continue the highest-priority active work after its scope and evidence are confirmed.", Refs: []string{snapshot.ActiveWork[0].ID}}}
}
