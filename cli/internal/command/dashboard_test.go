package command_test

import (
	"bytes"
	"github.com/kcrmin/Stackcord/cli/internal/command"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSetupChoiceRequiresExplicitApply(t *testing.T) {
	root := t.TempDir()
	out := &bytes.Buffer{}
	cmd := command.New("test", out, &bytes.Buffer{})
	cmd.SetArgs([]string{"setup", "--root", root, "--ui", "disable", "--json"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "preview")
	out.Reset()
	cmd = command.New("test", out, &bytes.Buffer{})
	cmd.SetArgs([]string{"setup", "--root", root, "--json"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "unset")
	out.Reset()
	cmd = command.New("test", out, &bytes.Buffer{})
	cmd.SetArgs([]string{"setup", "--root", root, "--ui", "disable", "--apply", "--json"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "declined")
}
