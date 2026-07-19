package command

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/continuity"
	hookpkg "github.com/kcrmin/Stackcord/cli/internal/hook"
	"github.com/spf13/cobra"
)

type hookInput struct {
	CWD           string `json:"cwd"`
	HookEventName string `json:"hook_event_name"`
}

func newHookCommand() *cobra.Command {
	var rootOverride string
	command := &cobra.Command{
		Use:          "hook session-start|post-compact",
		Short:        "Render read-only Codex lifecycle context",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			event := args[0]
			if event != "session-start" && event != "post-compact" {
				return fmt.Errorf("unsupported hook event %q", event)
			}
			start := rootOverride
			if start == "" {
				var input hookInput
				decoder := json.NewDecoder(io.LimitReader(cmd.InOrStdin(), 64*1024))
				if err := decoder.Decode(&input); err != nil {
					return fmt.Errorf("decode hook input: %w", err)
				}
				expected := map[string]string{"session-start": "SessionStart", "post-compact": "PostCompact"}[event]
				if input.HookEventName != expected {
					return fmt.Errorf("hook input event %q does not match %s", input.HookEventName, expected)
				}
				start = input.CWD
			}
			if strings.TrimSpace(start) == "" {
				return fmt.Errorf("hook input has no working directory")
			}
			snapshot := continuity.Collect(cmd.Context(), start, continuity.Options{})
			if snapshot.ProjectID == "" && len(snapshot.Issues) == 1 && snapshot.Issues[0].Code == "project.not-found" {
				return nil
			}
			data, err := hookpkg.Render(event, snapshot)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return err
		},
	}
	command.Flags().StringVar(&rootOverride, "root", "", "explicit project path for diagnostics and tests")
	return command
}
