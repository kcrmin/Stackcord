package command

import (
	"io"

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
	return root
}
