// Package command composes the focused runtime with the optional project harness.
package command

import (
	harness "github.com/kcrmin/Stackcord/cli/internal/harness/command"
	"github.com/kcrmin/Stackcord/cli/internal/runtimecmd"
	"github.com/spf13/cobra"
	"io"
)

func New(version string, stdout, stderr io.Writer) *cobra.Command {
	root := harness.New(version, stdout, stderr)
	root.Short = "Coordinate AI work across sessions with durable, bounded context"
	for _, cmd := range root.Commands() {
		cmd.Hidden = true
	}
	extension := harness.New(version, stdout, stderr)
	extension.Use = "harness"
	extension.Short = "Optional Git, project discovery, contracts and release workflows"
	root.AddCommand(extension)
	runtimecmd.Register(root)
	return root
}
func ExitCode(cmd *cobra.Command) int { return harness.ExitCode(cmd) }
