package command

import (
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/gitx"
	"github.com/spf13/cobra"
)

func newIntegrateCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "integrate", Short: "Plan compatibility-first workspace and pointer integration"}
	var root string
	planCommand := &cobra.Command{Use: "plan", RunE: func(cmd *cobra.Command, _ []string) error {
		state, err := gitx.Inspect(cmd.Context(), root)
		if err != nil {
			return err
		}
		plan := gitx.PlanWorkspaceSync(state)
		if state.Dirty {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "integrate.root-dirty", Message: "Root repository has local changes."})
		}
		if state.Diverged {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "integrate.root-diverged", Message: "Root branch diverged from its upstream."})
		}
		result := planResult(version, "integrate.plan", plan, "Compatibility-first integration plan is ready.")
		result.NextActions = append(result.NextActions, domain.Item{Code: "integrate.order", Message: "Merge additive contract, providers, consumers, UI connection, then exact root pointers; verify after every boundary."})
		return writeResult(cmd, *jsonOutput, result)
	}}
	planCommand.Flags().StringVar(&root, "root", ".", "root orchestration repository")
	parent.AddCommand(planCommand)
	return parent
}
