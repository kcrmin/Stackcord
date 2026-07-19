package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/project"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

func newProjectCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "project", Short: "Create or adopt a durable framework-neutral project harness"}
	parent.AddCommand(newProjectCheckpoint(version, jsonOutput))
	parent.AddCommand(newProjectMutation("init", version, jsonOutput, false))
	parent.AddCommand(newProjectMutation("adopt", version, jsonOutput, true))
	return parent
}

func newProjectCheckpoint(version string, jsonOutput *bool) *cobra.Command {
	var request project.CheckpointRequest
	var inputPath string
	var apply bool
	example, _ := yaml.Marshal(project.ExampleDiscoveryCheckpoint())
	command := &cobra.Command{
		Use:   "checkpoint",
		Short: "Save the next normalized service-discovery revision",
		Long:  "Save a complete normalized service-discovery snapshot. Replace the example values with current product meaning; never copy raw dialogue or tone.",
		Example: "  # checkpoint.yaml\n" + indentLines(string(example), "  ") +
			"\n  stackcord project checkpoint --parent . --id 01JDISCOVERY --input checkpoint.yaml --apply --json",
		RunE: func(cmd *cobra.Command, _ []string) error {
			checkpoint, err := schema.LoadYAML[project.DiscoveryCheckpoint](inputPath)
			if err != nil {
				return err
			}
			request.Checkpoint = checkpoint
			plan, err := project.PlanCheckpoint(request)
			if err != nil {
				return err
			}
			if apply {
				result := operation.Apply(cmd.Context(), plan)
				result.ToolVersion, result.Command = version, "project.checkpoint"
				return writeResult(cmd, *jsonOutput, result)
			}
			result := planResult(version, "project.checkpoint.plan", plan, "Normalized discovery checkpoint is ready; no files were changed.")
			return writeResult(cmd, *jsonOutput, result)
		}}
	command.Flags().StringVar(&request.Parent, "parent", "", "parent directory for .harness-drafts")
	command.Flags().StringVar(&request.DraftID, "id", "", "sortable draft ID")
	command.Flags().StringVar(&request.Locale, "locale", "en", "discovery language: en or ko")
	command.Flags().StringVar(&inputPath, "input", "", "strict normalized discovery YAML or JSON")
	command.Flags().BoolVar(&apply, "apply", false, "write the reviewed discovery checkpoint")
	_ = command.MarkFlagRequired("parent")
	_ = command.MarkFlagRequired("id")
	_ = command.MarkFlagRequired("input")
	return command
}

func indentLines(value, prefix string) string {
	return prefix + strings.ReplaceAll(strings.TrimRight(value, "\n"), "\n", "\n"+prefix)
}

func newProjectMutation(name, version string, jsonOutput *bool, adopt bool) *cobra.Command {
	var request project.InitRequest
	var apply bool
	command := &cobra.Command{
		Use: name,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var plan operation.Plan
			var err error
			if adopt {
				plan, err = project.PlanAdopt(request)
			} else {
				plan, err = project.PlanInit(request)
			}
			if err != nil {
				return err
			}
			if len(plan.Blockers) > 0 {
				return writeResult(cmd, *jsonOutput, domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "project." + name + ".plan", OperationID: plan.ID, Status: domain.StatusBlocked, ExitCode: domain.ExitBlocked, Summary: "Project plan is blocked by existing state.", Blockers: plan.Blockers})
			}
			if apply {
				result := operation.Apply(cmd.Context(), plan)
				result.ToolVersion = version
				result.Command = "project." + name
				return writeResult(cmd, *jsonOutput, result)
			}
			result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "project." + name + ".plan", OperationID: plan.ID, Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: fmt.Sprintf("Project %s plan is ready; no files were changed.", name), Facts: []domain.Item{{Code: "project.plan-files", Message: strconv.Itoa(len(plan.Files))}}, Approval: domain.Approval{Required: false, Class: "B", Reason: "Use --apply to execute the reviewed local write plan."}}
			for _, file := range plan.Files {
				result.Changes = append(result.Changes, domain.Item{Code: "project.file-planned", Message: file.Path})
			}
			return writeResult(cmd, *jsonOutput, result)
		},
	}
	command.Flags().StringVar(&request.Root, "root", "", "target project root")
	command.Flags().StringVar(&request.ProjectID, "id", "", "stable project ID")
	command.Flags().StringVar(&request.Name, "name", "", "human project name")
	command.Flags().StringVar(&request.Locale, "locale", "en", "project language: en or ko")
	command.Flags().StringVar(&request.DraftRoot, "draft", "", "approved discovery draft root")
	command.Flags().BoolVar(&apply, "apply", false, "apply the reviewed local plan")
	_ = command.MarkFlagRequired("root")
	_ = command.MarkFlagRequired("id")
	return command
}

func planResult(version, command string, plan operation.Plan, summary string) domain.Result {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: command, OperationID: plan.ID, Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: summary, Blockers: plan.Blockers}
	for _, file := range plan.Files {
		result.Changes = append(result.Changes, domain.Item{Code: "operation.file-planned", Message: file.Path})
	}
	for _, step := range plan.Commands {
		result.NextActions = append(result.NextActions, domain.Item{Code: "operation.command-planned", Message: step.Program + " " + strings.Join(step.Args, " "), Refs: []string{step.Directory, step.ApprovalClass}})
	}
	if len(plan.Blockers) > 0 {
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitBlocked, "Plan is blocked by actual state."
	}
	return result
}
