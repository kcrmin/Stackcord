package command_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"fullstack-orchestrator/cli/internal/command"
	"github.com/stretchr/testify/require"
)

func TestGovernanceCheckReportsGeneratedProjectAsDisabled(t *testing.T) {
	root := filepath.Join(t.TempDir(), "product")
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "init", "--root", root, "--id", "project.governance", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())

	var output bytes.Buffer
	check := command.New("1.0.0", &output, &bytes.Buffer{})
	check.SetArgs([]string{"governance", "check", "--root", root, "--json"})

	require.NoError(t, check.Execute())
	require.Equal(t, 0, command.ExitCode(check), output.String())
	require.Contains(t, output.String(), `"command":"governance.check"`)
	require.Contains(t, output.String(), `"governance.status","message":"disabled"`)
}

func TestProtectedMeaningWithoutAuthorityApprovalBlocksIntegrationAndRelease(t *testing.T) {
	root := filepath.Join(t.TempDir(), "product")
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "init", "--root", root, "--id", "project.protected", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())
	policy := "schema_version: 1\nenabled: true\nprovider: github\nrepository: example/service\nproduct_authorities: [user:product-owner]\nprotected_kinds: [product, policy, business, contract]\napproval:\n  minimum: 1\n  authority_self_approval: true\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "governance.yaml"), []byte(policy), 0o600))
	commandGit(t, root, "init", "--initial-branch=main")
	commandGit(t, root, "config", "user.name", "Ordinary Member")
	commandGit(t, root, "config", "user.email", "product-owner@example.invalid")
	commandGit(t, root, "add", ".")
	commandGit(t, root, "commit", "-m", "chore: initialize protected project")

	var integrationOutput bytes.Buffer
	integrate := command.New("1.0.0", &integrationOutput, &bytes.Buffer{})
	integrate.SetArgs([]string{"integrate", "plan", "--root", root, "--json"})
	require.NoError(t, integrate.Execute())
	require.Contains(t, integrationOutput.String(), "integrate.governance.approval-unknown")

	var releaseOutput bytes.Buffer
	prepare := command.New("1.0.0", &releaseOutput, &bytes.Buffer{})
	prepare.SetArgs([]string{"release", "prepare", "--root", root, "--release-version", "1.0.0", "--json"})
	require.NoError(t, prepare.Execute())
	require.Contains(t, releaseOutput.String(), "release.governance.approval-unknown")
}
