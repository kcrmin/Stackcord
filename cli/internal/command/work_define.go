package command

import (
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
	workpkg "github.com/kcrmin/Stackcord/cli/internal/work"
	"github.com/spf13/cobra"
)

func newWorkDefine(version string, jsonOutput *bool) *cobra.Command {
	var root, inputPath string
	var apply bool
	command := &cobra.Command{
		Use:   "define",
		Short: "Validate and save an executable canonical work checklist",
		RunE: func(cmd *cobra.Command, _ []string) error {
			definition, err := schema.LoadYAML[workpkg.Definition](inputPath)
			if err != nil {
				return err
			}
			plan, err := workpkg.PlanDefinition(cmd.Context(), root, definition)
			if err != nil {
				return err
			}
			if len(plan.Blockers) > 0 {
				return writeResult(cmd, *jsonOutput, domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "work.define.plan", OperationID: plan.ID, Status: domain.StatusBlocked, ExitCode: domain.ExitBlocked, Summary: "Work definition is incomplete or conflicts with canonical project state.", Blockers: plan.Blockers})
			}
			if apply {
				result := operation.Apply(cmd.Context(), plan)
				result.ToolVersion, result.Command = version, "work.define"
				return writeResult(cmd, *jsonOutput, result)
			}
			return writeResult(cmd, *jsonOutput, planResult(version, "work.define.plan", plan, "Executable work definition is ready; no files were changed."))
		},
	}
	command.Flags().StringVar(&root, "root", ".", "project root or child path")
	command.Flags().StringVar(&inputPath, "input", "", "strict normalized work definition YAML or JSON")
	command.Flags().BoolVar(&apply, "apply", false, "write the reviewed canonical definition")
	_ = command.MarkFlagRequired("input")
	return command
}
