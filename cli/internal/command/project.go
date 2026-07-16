package command

import (
	"fmt"
	"strconv"
	"strings"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/project"
	"github.com/spf13/cobra"
)

func newProjectCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "project", Short: "Create or adopt a durable framework-neutral project harness"}
	parent.AddCommand(newProjectDraft(version, jsonOutput))
	parent.AddCommand(newProjectMutation("init", version, jsonOutput, false))
	parent.AddCommand(newProjectMutation("adopt", version, jsonOutput, true))
	return parent
}

func newProjectDraft(version string, jsonOutput *bool) *cobra.Command {
	var request project.DraftRequest
	var apply bool
	command := &cobra.Command{Use: "draft", Short: "Checkpoint normalized service discovery before naming the repository", RunE: func(cmd *cobra.Command, _ []string) error {
		plan, err := project.CreateDraft(request)
		if err != nil {
			return err
		}
		if apply {
			result := operation.Apply(cmd.Context(), plan)
			result.ToolVersion, result.Command = version, "project.draft"
			return writeResult(cmd, *jsonOutput, result)
		}
		result := planResult(version, "project.draft.plan", plan, "Discovery draft plan is ready; no files were changed.")
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&request.Parent, "parent", "", "parent directory for .harness-drafts")
	command.Flags().StringVar(&request.DraftID, "id", "", "sortable draft ID")
	command.Flags().StringVar(&request.Locale, "locale", "en", "discovery language: en or ko")
	command.Flags().StringVar(&request.Summary, "summary", "", "normalized product summary, never raw conversation")
	command.Flags().StringSliceVar(&request.Decisions, "decision", nil, "approved normalized decision (repeatable)")
	command.Flags().StringSliceVar(&request.OpenQuestions, "open-question", nil, "material unresolved question (repeatable)")
	command.Flags().BoolVar(&apply, "apply", false, "write the reviewed discovery checkpoint")
	_ = command.MarkFlagRequired("parent")
	_ = command.MarkFlagRequired("id")
	return command
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
