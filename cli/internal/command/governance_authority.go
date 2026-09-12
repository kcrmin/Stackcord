package command

import (
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/governance"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/spf13/cobra"
)

func newGovernanceAuthorityCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "authority", Short: "Plan and apply protected product-authority changes"}
	parent.AddCommand(newGovernanceAuthorityMutation(governance.AuthorityAdd, version, jsonOutput))
	parent.AddCommand(newGovernanceAuthorityMutation(governance.AuthorityRemove, version, jsonOutput))
	return parent
}

func newGovernanceAuthorityMutation(action governance.AuthorityAction, version string, jsonOutput *bool) *cobra.Command {
	var request governance.AuthorityChangeRequest
	var apply bool
	request.Action = action
	command := &cobra.Command{
		Use:   string(action),
		Short: string(action) + " a product authority through a reviewed policy plan",
		RunE: func(cmd *cobra.Command, _ []string) error {
			change, plan, err := governance.PlanAuthorityChange(cmd.Context(), request)
			if err != nil {
				return err
			}
			facts := authorityChangeFacts(change)
			if apply && request.ExpectedPolicyFingerprint == "" {
				plan.Blockers = append(plan.Blockers, domain.Item{Code: "governance.expected-policy-required", Message: "Applying an authority change requires the exact policy fingerprint from the reviewed plan."})
			}
			if len(plan.Blockers) > 0 {
				return writeResult(cmd, *jsonOutput, domain.Result{
					SchemaVersion: "1.0", ToolVersion: version, Command: "governance.authority." + string(action), OperationID: plan.ID,
					Status: domain.StatusBlocked, ExitCode: domain.ExitBlocked, Summary: "Product-authority change is blocked by the current policy.", Facts: facts, Blockers: plan.Blockers,
				})
			}
			if apply {
				result := operation.Apply(cmd.Context(), plan)
				result.ToolVersion, result.Command = version, "governance.authority."+string(action)
				result.Facts = facts
				if result.Status == domain.StatusPassed {
					result.Summary = "Product-authority policy proposal was written; it is not approved yet."
					result.NextActions = []domain.Item{{Code: "governance.request-review", Message: "Commit the proposal and complete the configured provider review before integration or release."}}
				}
				return writeResult(cmd, *jsonOutput, result)
			}
			return writeResult(cmd, *jsonOutput, domain.Result{
				SchemaVersion: "1.0", ToolVersion: version, Command: "governance.authority." + string(action) + ".plan", OperationID: plan.ID,
				Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Product-authority change plan is ready; no files were changed.", Facts: facts,
				Changes:     []domain.Item{{Code: "governance.policy-planned", Message: ".harness/governance.yaml", Refs: change.After}},
				NextActions: []domain.Item{{Code: "governance.apply-reviewed-plan", Message: "Rerun with --expected-policy and --apply after reviewing the exact authority list.", Refs: []string{change.PolicyFingerprint}}},
				Approval:    domain.Approval{Required: true, Class: "B", Reason: "This writes a protected governance proposal that still requires provider review."},
			})
		},
	}
	command.Flags().StringVar(&request.Root, "root", ".", "project path or any path inside the orchestration root")
	command.Flags().StringArrayVar(&request.Subjects, "subject", nil, "normalized product authority; repeat for multiple user: or team: subjects")
	command.Flags().StringVar(&request.ExpectedPolicyFingerprint, "expected-policy", "", "exact governance policy fingerprint from the reviewed plan")
	command.Flags().BoolVar(&apply, "apply", false, "write the reviewed authority proposal")
	_ = command.MarkFlagRequired("subject")
	return command
}

func authorityChangeFacts(change governance.AuthorityChange) []domain.Item {
	return []domain.Item{
		{Code: "governance.authority-action", Message: string(change.Action)},
		{Code: "governance.authority-subjects", Message: "Product authorities included in this change.", Refs: change.Subjects},
		{Code: "governance.policy-fingerprint", Message: change.PolicyFingerprint},
		{Code: "governance.authorities-before", Message: "Configured product authorities before the proposal.", Refs: change.Before},
		{Code: "governance.authorities-after", Message: "Configured product authorities after the proposal.", Refs: change.After},
	}
}
