package command_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"fullstack-orchestrator/cli/internal/command"
	"fullstack-orchestrator/cli/internal/evidence"
	"fullstack-orchestrator/cli/internal/provider"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestWorkEvidenceAndTransitionUseApprovedCommandAndLiveRevision(t *testing.T) {
	root, remote := commandSharedRemote(t)
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "adopt", "--root", root, "--id", "project.lifecycle", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte("schema_version: 1\nproject_id: project.lifecycle\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n    commands_path: .harness/commands.yaml\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "commands.yaml"), []byte("schema_version: 1\nworkspace_id: workspace.root\ncommands:\n  - id: command.test\n    kind: test\n    argv: [git, status, --porcelain=v2]\n    timeout_seconds: 30\n"), 0o600))
	artifact := []byte("verified service artifact\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "service.bin"), artifact, 0o600))
	defineCommandWork(t, root, "work.account-recovery", "services/identity/**")
	commandGit(t, root, "add", ".")
	commandGit(t, root, "commit", "-m", "chore: initialize lifecycle harness")
	commandGit(t, root, "push")

	start := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	start.SetArgs([]string{"work", "start", "--root", root, "--work-id", "work.account-recovery", "--claim-id", "claim.account-recovery", "--owner", "alex", "--branch", "feature/account-recovery", "--apply", "--json"})
	require.NoError(t, start.Execute())
	require.Equal(t, 0, command.ExitCode(start))
	commandGit(t, root, "add", ".harness/work")
	commandGit(t, root, "commit", "-m", "chore: record work checkpoint")
	worktree := filepath.Join(t.TempDir(), "account-recovery")
	commandGit(t, root, "worktree", "add", "-b", "feature/account-recovery", worktree, "HEAD")
	commandGit(t, worktree, "push", "-u", "origin", "feature/account-recovery")

	var evidenceOutput bytes.Buffer
	record := command.New("1.0.0", &evidenceOutput, &bytes.Buffer{})
	record.SetArgs([]string{"work", "evidence", "--root", worktree, "--work-id", "work.account-recovery", "--workspace", "workspace.root", "--command-id", "command.test", "--artifact", "service=service.bin", "--apply", "--json"})
	require.NoError(t, record.Execute())
	require.Equal(t, 0, command.ExitCode(record), evidenceOutput.String())
	require.Contains(t, evidenceOutput.String(), "evidence.verified")
	entries, err := os.ReadDir(filepath.Join(worktree, ".harness", "local", "evidence", "work.account-recovery"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	recordData, err := os.ReadFile(filepath.Join(worktree, ".harness", "local", "evidence", "work.account-recovery", entries[0].Name()))
	require.NoError(t, err)
	var stored evidence.Record
	require.NoError(t, yaml.Unmarshal(recordData, &stored))
	digest := sha256.Sum256(artifact)
	require.Equal(t, "sha256:"+hex.EncodeToString(digest[:]), stored.ArtifactDigests["service"])
	require.Empty(t, commandGit(t, worktree, "status", "--porcelain=v2", "--untracked-files=all"))

	var transitionOutput bytes.Buffer
	transition := command.New("1.0.0", &transitionOutput, &bytes.Buffer{})
	transition.SetArgs([]string{"work", "transition", "--root", worktree, "--work-id", "work.account-recovery", "--target", "review", "--apply", "--json"})
	require.NoError(t, transition.Execute())
	require.Equal(t, 0, command.ExitCode(transition), transitionOutput.String())
	observed, err := provider.NewGitLocalStore(root, remote, "coordination").Read(context.Background())
	require.NoError(t, err)
	require.Equal(t, "review", observed.Claims[0].Status)

	var handoffOutput bytes.Buffer
	handoff := command.New("1.0.0", &handoffOutput, &bytes.Buffer{})
	handoff.SetArgs([]string{"work", "handoff", "--root", worktree, "--work-id", "work.account-recovery", "--workspace", "workspace.root", "--owner", "sam", "--next-action", "Run the approved integration test", "--apply", "--json"})
	require.NoError(t, handoff.Execute())
	require.Equal(t, 0, command.ExitCode(handoff), handoffOutput.String())
	observed, err = provider.NewGitLocalStore(root, remote, "coordination").Read(context.Background())
	require.NoError(t, err)
	require.Equal(t, "sam", observed.Claims[0].Owner)
	require.NotNil(t, observed.Claims[0].Handoff)
	require.Equal(t, "feature/account-recovery", observed.Claims[0].Handoff.Branch)
	require.Equal(t, commandGit(t, worktree, "rev-parse", "HEAD"), observed.Claims[0].Handoff.Commit)
	require.False(t, observed.Claims[0].Handoff.LocalOnly)

	terminal := observed
	terminal.Claims[0].Status = "done"
	_, err = provider.NewGitLocalStore(root, remote, "coordination").CompareAndSwap(context.Background(), observed.Revision, terminal)
	require.NoError(t, err)
	var nextOutput bytes.Buffer
	next := command.New("1.0.0", &nextOutput, &bytes.Buffer{})
	next.SetArgs([]string{"work", "next", "--root", worktree, "--json"})
	require.NoError(t, next.Execute())
	require.NotEqual(t, 0, command.ExitCode(next), nextOutput.String())
	require.NotContains(t, nextOutput.String(), "work.recommended")

	var restartOutput bytes.Buffer
	restart := command.New("1.0.0", &restartOutput, &bytes.Buffer{})
	restart.SetArgs([]string{"work", "start", "--root", worktree, "--work-id", "work.account-recovery", "--claim-id", "claim.account-recovery-again", "--owner", "lee", "--branch", "feature/account-recovery-again", "--apply", "--json"})
	require.NoError(t, restart.Execute())
	require.NotEqual(t, 0, command.ExitCode(restart), restartOutput.String())
	require.Contains(t, restartOutput.String(), "work.already-terminal")
}

func TestWorkHandoffRejectsUnpublishedCommit(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	commandGit(t, "", "init", "--initial-branch=main", root)
	commandGit(t, root, "config", "user.email", "fixture@example.invalid")
	commandGit(t, root, "config", "user.name", "Fixture User")
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o600))
	commandGit(t, root, "add", "README.md")
	commandGit(t, root, "commit", "-m", "chore: initialize fixture")

	cmd := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	cmd.SetArgs([]string{"work", "handoff", "--root", root, "--work-id", "work.missing", "--workspace", "workspace.root", "--owner", "sam", "--next-action", "Continue", "--apply", "--json"})
	require.Error(t, cmd.Execute(), "a repository without the harness cannot claim a safe handoff")
}

func TestWorkEvidenceRejectsUnlistedCommand(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "init", "--root", root, "--id", "project.unlisted-command", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())

	var output bytes.Buffer
	record := command.New("1.0.0", &output, &bytes.Buffer{})
	record.SetArgs([]string{"work", "evidence", "--root", root, "--work-id", "work.missing", "--workspace", "workspace.root", "--command-id", "command.shell", "--apply", "--json"})
	require.NoError(t, record.Execute())
	require.NotEqual(t, 0, command.ExitCode(record))
	require.NotContains(t, output.String(), "passed")
}
