package command

import (
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/governance"
	"github.com/kcrmin/Stackcord/cli/internal/workspace"
	"github.com/spf13/cobra"
)

func newGovernanceCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "governance", Short: "Verify product-authority approval for protected service meaning"}
	var root, observation string
	check := &cobra.Command{Use: "check", RunE: func(cmd *cobra.Command, _ []string) error {
		located, err := workspace.FindRoot(cmd.Context(), root)
		if err != nil {
			return err
		}
		report := governance.Check(cmd.Context(), located.Path, observation, time.Now().UTC())
		result := governanceResult(version, report)
		return writeResult(cmd, *jsonOutput, result)
	}}
	check.Flags().StringVar(&root, "root", ".", "project path or any path inside the orchestration root")
	check.Flags().StringVar(&observation, "observation", "", "fresh normalized review observation; defaults to ignored local governance state")
	parent.AddCommand(check)
	return parent
}

func governanceResult(version string, report governance.Report) domain.Result {
	result := domain.Result{
		SchemaVersion: "1.0", ToolVersion: version, Command: "governance.check", OperationID: "governance-check-read-only",
		Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Product governance is not enabled for this project.",
		Facts: []domain.Item{
			{Code: "governance.status", Message: string(report.Status)},
			{Code: "governance.protected-fingerprint", Message: report.ProtectedFingerprint},
			{Code: "governance.approval-revision", Message: report.ApprovalRevision},
			{Code: "governance.authorities", Message: "Configured product authorities.", Refs: report.Authorities},
			{Code: "governance.approvers", Message: "Verified product authorities.", Refs: report.Approvers},
		},
	}
	switch report.Status {
	case governance.Approved:
		result.Summary = "Protected product meaning has fresh approval from a configured product authority."
	case governance.Unknown:
		result.Status, result.ExitCode, result.Summary = domain.StatusUnknown, domain.ExitUnavailable, "Product-authority approval cannot be verified from fresh provider evidence."
		result.Blockers = report.Issues
	case governance.Proposed:
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitVerification, "The protected change remains a proposal until a configured product authority approves it."
		result.Blockers = report.Issues
	case governance.Blocked:
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitVerification, "Product governance evidence does not match the protected change."
		result.Blockers = report.Issues
	}
	return result
}
