package command

import (
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/workspace"
	"github.com/spf13/cobra"
)

func newWorkspaceCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "workspace", Short: "Register recoverable service workspace boundaries"}
	var request workspace.RegistrationRequest
	var apply bool
	register := &cobra.Command{Use: "register", RunE: func(cmd *cobra.Command, _ []string) error {
		plan, err := workspace.PlanRegistration(cmd.Context(), request)
		if err != nil {
			return err
		}
		if !apply {
			return writeResult(cmd, *jsonOutput, planResult(version, "workspace.register.plan", plan, "Workspace registration plan is ready; existing project files are unchanged."))
		}
		if len(plan.Blockers) > 0 {
			return writeResult(cmd, *jsonOutput, planResult(version, "workspace.register", plan, "Workspace registration is blocked by current project state."))
		}
		result := operation.Apply(cmd.Context(), plan)
		result.ToolVersion, result.Command = version, "workspace.register"
		if result.Status == domain.StatusPassed {
			result.Summary = "Workspace identity and optional framework-neutral starter were written; no commit or push was performed."
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	register.Flags().StringVar(&request.Root, "root", ".", "exact orchestration root")
	register.Flags().StringVar(&request.ID, "id", "", "workspace stable ID")
	register.Flags().StringVar(&request.Kind, "kind", "directory", "directory or submodule")
	register.Flags().StringVar(&request.Path, "path", "", "workspace path")
	register.Flags().StringVar(&request.Repository, "repository", "", "repository stable ID")
	register.Flags().StringVar(&request.Remote, "remote", "", "child repository remote")
	register.Flags().StringVar(&request.RootRemote, "root-remote", "", "orchestration repository remote for child recovery")
	register.Flags().StringSliceVar(&request.Responsibilities, "responsibility", nil, "workspace responsibility")
	register.Flags().StringSliceVar(&request.Dependencies, "dependency", nil, "workspace dependency")
	register.Flags().StringSliceVar(&request.Consumers, "consumer", nil, "workspace that depends on this boundary")
	register.Flags().StringVar(&request.Initialize, "initialize", "", "optional framework-neutral initializer: ui")
	register.Flags().BoolVar(&apply, "apply", false, "write the reviewed workspace registration")
	_ = register.MarkFlagRequired("id")
	_ = register.MarkFlagRequired("path")
	_ = register.MarkFlagRequired("responsibility")
	parent.AddCommand(register)
	return parent
}
