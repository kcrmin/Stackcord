package command_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fullstack-orchestrator/cli/internal/command"
	"fullstack-orchestrator/cli/internal/provider"
	"github.com/stretchr/testify/require"
)

func TestWorkStartPublishesObservableGitLocalClaimBeforeBranchWork(t *testing.T) {
	root, remote := commandSharedRemote(t)
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "adopt", "--root", root, "--id", "project.claim-race", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())
	defineCommandWork(t, root, "work.account-recovery", "services/identity/**")
	commandGit(t, root, "add", ".")
	commandGit(t, root, "commit", "-m", "chore: initialize project harness")
	commandGit(t, root, "push")

	var output bytes.Buffer
	start := command.New("1.0.0", &output, &bytes.Buffer{})
	start.SetArgs([]string{
		"work", "start", "--root", root,
		"--work-id", "work.account-recovery", "--claim-id", "claim.account-recovery", "--owner", "alex",
		"--branch", "feature/account-recovery", "--path", "services/identity/**",
		"--apply", "--json",
	})
	require.NoError(t, start.Execute())
	require.Equal(t, 0, command.ExitCode(start), output.String())
	require.Contains(t, output.String(), "provider.git-local-claim")

	observed, err := provider.NewGitLocalStore(root, remote, "coordination").Read(context.Background())
	require.NoError(t, err)
	require.Len(t, observed.Claims, 1)
	require.Equal(t, "alex", observed.Claims[0].Owner)
	require.FileExists(t, filepath.Join(root, ".harness", "work", "claims", "claim.account-recovery.yaml"))

	candidate := filepath.Join(t.TempDir(), "candidate.yaml")
	require.NoError(t, os.WriteFile(candidate, []byte("repository: repository.root\npaths: [services/identity/handler/**]\npolicy_ids: []\nscenario_ids: []\ncontract_ids: []\ndb_entities: []\nmigration_slots: []\nui_flows: []\ndependency_majors: []\nstable_ids: []\nroot_pointer: false\n"), 0o600))
	var conflictOutput bytes.Buffer
	conflict := command.New("1.0.0", &conflictOutput, &bytes.Buffer{})
	conflict.SetArgs([]string{"work", "conflict", "--root", root, "--candidate", candidate, "--json"})
	require.NoError(t, conflict.Execute())
	require.Contains(t, conflictOutput.String(), `"status":"warning"`)
	require.Contains(t, conflictOutput.String(), "conflict.path-overlap")

	expired := observed
	expired.Claims[0].StartsAt = time.Now().UTC().Add(-2 * time.Hour)
	expired.Claims[0].ExpiresAt = time.Now().UTC().Add(-time.Hour)
	expiredRevision, err := provider.NewGitLocalStore(root, remote, "coordination").CompareAndSwap(context.Background(), observed.Revision, expired)
	require.NoError(t, err)
	require.NotEmpty(t, expiredRevision)
	var reclaimOutput bytes.Buffer
	reclaim := command.New("1.0.0", &reclaimOutput, &bytes.Buffer{})
	reclaim.SetArgs([]string{
		"work", "start", "--root", root,
		"--work-id", "work.account-recovery", "--claim-id", "claim.account-recovery-retry", "--owner", "sam",
		"--branch", "feature/account-recovery-retry", "--path", "services/identity/**", "--apply", "--json",
	})
	require.NoError(t, reclaim.Execute())
	require.Equal(t, 0, command.ExitCode(reclaim), reclaimOutput.String())
	reclaimed, err := provider.NewGitLocalStore(root, remote, "coordination").Read(context.Background())
	require.NoError(t, err)
	require.Len(t, reclaimed.Claims, 1)
	require.Equal(t, "sam", reclaimed.Claims[0].Owner)
}

func TestGitWorktreeCommandCreatesVerifiedConventionalBranch(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	commandGit(t, "", "init", "--initial-branch=main", root)
	commandGit(t, root, "config", "user.email", "fixture@example.invalid")
	commandGit(t, root, "config", "user.name", "Fixture User")
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o600))
	commandGit(t, root, "add", "README.md")
	commandGit(t, root, "commit", "-m", "chore: initialize fixture")
	target := filepath.Join(t.TempDir(), "account-recovery")
	var output bytes.Buffer
	cmd := command.New("1.0.0", &output, &bytes.Buffer{})
	cmd.SetArgs([]string{"git", "worktree", "--root", root, "--branch", "feature/account-recovery", "--base", "main", "--target", target, "--apply", "--json"})

	require.NoError(t, cmd.Execute())
	require.Equal(t, 0, command.ExitCode(cmd), output.String())
	require.Contains(t, output.String(), "git.worktree-verified")
	require.Equal(t, "feature/account-recovery", commandGit(t, target, "branch", "--show-current"))
	require.Equal(t, commandGit(t, root, "rev-parse", "main"), commandGit(t, target, "rev-parse", "HEAD"))
}

func defineCommandWork(t *testing.T, root, id, path string) {
	t.Helper()
	definition := filepath.Join(t.TempDir(), "work.yaml")
	content := "schema_version: 1\nid: " + id + "\nreadiness: ready\ntitle: Account recovery\noutcome: A user can recover access safely.\nacceptance:\n  - id: scenario.account-recovery\n    given: A recoverable account\n    when: Recovery is requested\n    then: Access is restored after verification\n    failure: Unsafe recovery is rejected\nrefs: []\nworkspaces: [workspace.root]\nscope:\n  repositories: [repository.root]\n  paths: [\"" + path + "\"]\n  policy_ids: []\n  scenario_ids: []\n  contract_ids: []\n  db_entities: []\n  migration_slots: []\n  ui_flows: []\n  dependency_majors: []\n  root_pointers: []\ndependencies: []\nmerge_order: [workspace.root]\nfirst_failing_test: test.account-recovery\nevidence:\n  kinds: [test]\n  integration_required: false\n  user_validation: false\n  migration_required: false\n  rollback_required: false\n"
	require.NoError(t, os.WriteFile(definition, []byte(content), 0o600))
	define := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	define.SetArgs([]string{"work", "define", "--root", root, "--input", definition, "--apply", "--json"})
	require.NoError(t, define.Execute())
	require.Equal(t, 0, command.ExitCode(define))
}

func commandSharedRemote(t *testing.T) (string, string) {
	t.Helper()
	base := t.TempDir()
	remote, seed, root := filepath.Join(base, "remote.git"), filepath.Join(base, "seed"), filepath.Join(base, "work")
	commandGit(t, "", "init", "--bare", "--initial-branch=main", remote)
	commandGit(t, "", "init", "--initial-branch=main", seed)
	commandGit(t, seed, "config", "user.email", "fixture@example.invalid")
	commandGit(t, seed, "config", "user.name", "Fixture User")
	require.NoError(t, os.WriteFile(filepath.Join(seed, "README.md"), []byte("fixture\n"), 0o600))
	commandGit(t, seed, "add", "README.md")
	commandGit(t, seed, "commit", "-m", "chore: initialize fixture")
	commandGit(t, seed, "remote", "add", "origin", remote)
	commandGit(t, seed, "push", "-u", "origin", "main")
	commandGit(t, "", "clone", remote, root)
	commandGit(t, root, "config", "user.email", "fixture@example.invalid")
	commandGit(t, root, "config", "user.name", "Fixture User")
	return root, remote
}

func commandGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	process := exec.Command("git", args...)
	process.Dir = root
	process.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := process.CombinedOutput()
	require.NoError(t, err, "git %s failed: %s", strings.Join(args, " "), output)
	return strings.TrimSpace(string(output))
}
