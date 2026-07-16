package command

import (
	"io"
	"strconv"
	"strings"

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/output"
	"github.com/spf13/cobra"
)

// New creates the command tree with explicit output streams for testability.
func New(version string, stdout, stderr io.Writer) *cobra.Command {
	var jsonOutput bool

	root := &cobra.Command{
		Use:           "orchestrator",
		Short:         "Coordinate full-stack projects from discovery to release",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "write the stable machine-readable result")

	doctor := &cobra.Command{
		Use:   "doctor",
		Short: "Inspect the local environment",
		RunE: func(cmd *cobra.Command, _ []string) error {
			result := domain.Result{
				SchemaVersion: "1.0",
				ToolVersion:   version,
				Command:       "doctor",
				OperationID:   "doctor-read-only",
				Status:        domain.StatusPassed,
				ExitCode:      domain.ExitSuccess,
				Summary:       "Environment inspection completed.",
			}
			if jsonOutput {
				return output.WriteJSON(cmd.OutOrStdout(), result)
			}
			return output.WriteHuman(cmd.OutOrStdout(), result)
		},
	}
	root.AddCommand(doctor)
	root.AddCommand(newContextCommand(version, &jsonOutput))
	return root
}

func newContextCommand(version string, jsonOutput *bool) *cobra.Command {
	contextCommand := &cobra.Command{Use: "context", Short: "Rebuild project understanding from canonical repository files"}
	for _, name := range []string{"audit", "pack", "refresh"} {
		name := name
		var rootPath string
		var write bool
		child := &cobra.Command{
			Use:   name,
			Short: "Inspect canonical project context",
			RunE: func(cmd *cobra.Command, _ []string) error {
				mode := contextpkg.ReadOnly
				if name == "refresh" && write {
					mode = contextpkg.WriteCheckpoint
				}
				snapshot, issues := contextpkg.Refresh(cmd.Context(), rootPath, mode)
				result := contextResult(version, name, rootPath, snapshot, issues, mode)
				if *jsonOutput {
					return output.WriteJSON(cmd.OutOrStdout(), result)
				}
				return output.WriteHuman(cmd.OutOrStdout(), result)
			},
		}
		child.Flags().StringVar(&rootPath, "root", ".", "project path or any path inside it")
		if name == "refresh" {
			child.Flags().BoolVar(&write, "write", false, "replace tracked generated context checkpoints")
		}
		contextCommand.AddCommand(child)
	}
	return contextCommand
}

func contextResult(version, commandName, root string, snapshot contextpkg.Snapshot, issues []domain.Item, mode contextpkg.RefreshMode) domain.Result {
	result := domain.Result{
		SchemaVersion: "1.0", ToolVersion: version, Command: "context." + commandName,
		OperationID: "context-" + commandName + "-read-only", Status: domain.StatusPassed,
		ExitCode: domain.ExitSuccess, Summary: "Project context rebuilt from canonical sources.",
		Project: &domain.Project{Root: root},
		Facts: []domain.Item{
			{Code: "context.documents", Message: strconv.Itoa(len(snapshot.Index))},
			{Code: "context.stale", Message: strconv.Itoa(len(snapshot.Stale)), Refs: snapshot.Stale},
			{Code: "context.unknown", Message: strconv.Itoa(len(snapshot.Unknown)), Refs: snapshot.Unknown},
		},
	}
	if mode == contextpkg.WriteCheckpoint {
		result.OperationID = "context-refresh-checkpoint"
		result.Changes = []domain.Item{{Code: "context.checkpoint.updated", Message: "Generated context index and impact graph were replaced atomically."}}
	}
	for _, issue := range issues {
		if strings.HasPrefix(issue.Code, "context.error") {
			result.Blockers = append(result.Blockers, issue)
		} else {
			result.Warnings = append(result.Warnings, issue)
		}
	}
	if len(result.Blockers) > 0 {
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitBlocked, "Project context could not be rebuilt safely."
	} else if len(snapshot.Unknown) > 0 {
		result.Status, result.ExitCode, result.Summary = domain.StatusUnknown, domain.ExitUnavailable, "Project context was rebuilt with unknown external or semantic state."
	} else if len(snapshot.Stale) > 0 {
		result.Status, result.Summary = domain.StatusWarning, "Project context was rebuilt and stale dependents were found."
	}
	return result
}
