package command

import (
	"encoding/json"
	"fmt"

	"fullstack-orchestrator/cli/internal/continuity"
	"fullstack-orchestrator/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newStatusCommand(jsonOutput *bool) *cobra.Command {
	var rootPath string
	command := &cobra.Command{
		Use:   "status",
		Short: "Report one combined service, Git, workspace, work, and release status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapshot := continuity.Collect(cmd.Context(), rootPath, continuity.Options{})
			setRenderedExitCode(cmd, continuityExitCode(snapshot.Overall))
			if *jsonOutput {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetEscapeHTML(false)
				return encoder.Encode(snapshot)
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Project continuity: %s\n", snapshot.Overall); err != nil {
				return err
			}
			if len(snapshot.NextActions) > 0 {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", snapshot.NextActions[0].Message)
				return err
			}
			return nil
		},
	}
	command.Flags().StringVar(&rootPath, "root", ".", "project path or any path inside a root or child workspace")
	return command
}

func continuityExitCode(confidence continuity.Confidence) int {
	switch confidence {
	case continuity.Blocked:
		return domain.ExitBlocked
	case continuity.Unknown:
		return domain.ExitUnavailable
	default:
		return domain.ExitSuccess
	}
}

func setRenderedExitCode(command *cobra.Command, code int) {
	root := command.Root()
	if root.Annotations == nil {
		root.Annotations = map[string]string{}
	}
	root.Annotations[exitCodeAnnotation] = fmt.Sprintf("%d", code)
}
