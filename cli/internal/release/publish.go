package release

import (
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/policy"
)

// PlanPublish returns a visible no-side-effect release plan only after exact D approval.
func PlanPublish(candidate Candidate, consent policy.Consent) (operation.Plan, domain.Result) {
	plan := operation.Plan{ID: "publish-" + candidate.Input.Version, Root: "product"}
	if verification := VerifyCandidate(candidate, candidate.Input); verification.Status != domain.StatusPassed {
		verification.Command, verification.OperationID, verification.Summary = "release.publish", plan.ID, "Release candidate manifest is invalid; publication is blocked."
		verification.Approval = domain.Approval{Required: false, Class: "D", Reason: "Approval cannot authorize an invalid candidate."}
		return plan, verification
	}
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: "release.publish", OperationID: plan.ID, Status: domain.StatusApprovalRequired, ExitCode: domain.ExitApprovalRequired, Summary: "Exact production release approval is required.", Approval: domain.Approval{Required: true, Class: "D", Reason: "Publishing mutates public immutable channels."}}
	decision := policy.Classify(policy.PublishProduction, consent, policy.Scope{Objective: "publish " + candidate.Input.Version, Repository: "product", Target: candidate.Digest, Now: time.Now().UTC()})
	if decision.Required {
		return plan, result
	}
	steps := []struct {
		program string
		args    []string
	}{
		{"git", []string{"tag", "-s", "v" + candidate.Input.Version}},
		{"goreleaser", []string{"release", "--clean"}},
		{"cosign", []string{"sign-blob", "checksums.txt"}},
		{"gh", []string{"release", "create", "v" + candidate.Input.Version}},
		{"orchestrator", []string{"marketplace", "publish", candidate.Digest}},
		{"orchestrator", []string{"homebrew", "publish", candidate.Digest}},
		{"orchestrator", []string{"winget", "publish", candidate.Digest}},
		{"orchestrator", []string{"install", "smoke", candidate.Digest}},
	}
	for _, step := range steps {
		plan.Commands = append(plan.Commands, operation.CommandStep{Program: step.program, Args: step.args, Directory: "product", ApprovalClass: "D"})
	}
	result.Status, result.ExitCode, result.Summary = domain.StatusPassed, domain.ExitSuccess, "Production publish plan is bound to the approved RC digest."
	result.Approval = domain.Approval{Required: false, Class: "D", Reason: "Exact approval receipt matched."}
	return plan, result
}
