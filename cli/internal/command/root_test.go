package command_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"fullstack-orchestrator/cli/internal/command"
	"fullstack-orchestrator/cli/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestDoctorWritesStableJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &stderr)
	cmd.SetArgs([]string{"doctor", "--json"})

	require.NoError(t, cmd.Execute())
	require.Empty(t, stderr.String())
	var result domain.Result
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Equal(t, domain.StatusPassed, result.Status)
	require.Equal(t, runtime.GOOS, factMessage(result.Facts, "environment.os"))
	require.Equal(t, runtime.GOARCH, factMessage(result.Facts, "environment.arch"))
	require.Equal(t, runtime.Version(), factMessage(result.Facts, "environment.go"))
	require.NotEmpty(t, factMessage(result.Facts, "environment.cli-path"))
	require.NotEmpty(t, factMessage(result.Facts, "environment.git-version"))
	require.Contains(t, []string{"true", "false"}, factMessage(result.Facts, "environment.dbdiagram-available"))
}

func factMessage(items []domain.Item, code string) string {
	for _, item := range items {
		if item.Code == code {
			return item.Message
		}
	}
	return ""
}

func TestContextAuditInspectsProjectWithoutWriting(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "state"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "specs", "policies"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))
	policy := "---\nschema_version: 1\nid: policy.example.ready\nkind: policy\nstatus: approved\nrevision: 1\nrefs: []\n---\nReady.\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "specs", "policies", "ready.md"), []byte(policy), 0o600))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &stderr)
	cmd.SetArgs([]string{"context", "audit", "--root", root, "--json"})
	require.NoError(t, cmd.Execute())
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), `"context.documents"`)
	_, err := os.Stat(filepath.Join(root, ".harness", "state", "context-index.json"))
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(root, ".harness", "local", "context", "context-index.json"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestCommandExposesRenderedDomainExitCode(t *testing.T) {
	var stdout bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"context", "audit", "--root", filepath.Join(t.TempDir(), "missing"), "--json"})

	require.NoError(t, cmd.Execute(), "domain outcomes are rendered, not returned as Cobra errors")
	require.Equal(t, 4, command.ExitCode(cmd))
	require.Contains(t, stdout.String(), `"exit_code":4`)
}

func TestProjectInitPlansThenAppliesNeutralHarness(t *testing.T) {
	root := filepath.Join(t.TempDir(), "product")
	var stdout bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"project", "init", "--root", root, "--id", "project.command-example", "--name", "Command Example", "--locale", "en", "--json"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "project.init.plan")
	_, err := os.Stat(filepath.Join(root, ".harness", "manifest.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)

	stdout.Reset()
	cmd = command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"project", "init", "--root", root, "--id", "project.command-example", "--name", "Command Example", "--locale", "en", "--apply", "--json"})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, filepath.Join(root, ".harness", "manifest.yaml"))
	require.Contains(t, stdout.String(), "project.init")
}

func TestGitInspectCommandReportsActualState(t *testing.T) {
	root := t.TempDir()
	run := func(args ...string) {
		process := exec.Command("git", args...)
		process.Dir = root
		output, err := process.CombinedOutput()
		require.NoError(t, err, string(output))
	}
	run("init", "--initial-branch=main")
	run("config", "user.email", "fixture@example.invalid")
	run("config", "user.name", "Fixture")
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o600))
	run("add", "README.md")
	run("commit", "-m", "chore: initialize")

	var stdout bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"git", "inspect", "--root", root, "--json"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), `"git.branch"`)
	require.Contains(t, stdout.String(), `"main"`)
}

func TestCommandSurfaceCoversProjectLifecycle(t *testing.T) {
	cmd := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	paths := []string{
		"project checkpoint", "project init", "project adopt",
		"context audit", "context refresh",
		"git inspect", "git sync-plan", "git sync", "git worktree-plan", "git worktree",
		"work next", "work conflict", "work start", "work evidence", "work transition", "work finish", "work handoff",
		"change plan", "contract check", "contract impact",
		"db diff", "db diagram", "db diagram prepare", "db diagram reconcile", "ui import", "ui reconcile", "integrate plan",
		"release prepare", "release verify",
	}
	for _, path := range paths {
		found, _, err := cmd.Find(strings.Fields(path))
		require.NoError(t, err, path)
		require.Equal(t, strings.Fields(path)[len(strings.Fields(path))-1], found.Name(), path)
	}
}

func TestWorkFinishDoesNotAcceptStringOnlyEvidence(t *testing.T) {
	cmd := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	finish, _, err := cmd.Find([]string{"work", "finish"})
	require.NoError(t, err)
	require.Nil(t, finish.Flags().Lookup("evidence"))
}

func TestCommandSurfaceOmitsRemovedPlatformCommands(t *testing.T) {
	cmd := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	for _, removed := range []struct {
		parent string
		name   string
	}{
		{parent: "context", name: "pack"},
		{name: "verify"},
		{name: "rc"},
		{parent: "release", name: "publish"},
	} {
		parent := cmd
		if removed.parent != "" {
			parent = mustFindCommand(t, cmd, removed.parent)
		}
		for _, child := range parent.Commands() {
			require.NotEqual(t, removed.name, child.Name())
		}
	}
}

func mustFindCommand(t *testing.T, root *cobra.Command, path string) *cobra.Command {
	t.Helper()
	found, _, err := root.Find(strings.Fields(path))
	require.NoError(t, err)
	return found
}

func TestDoctorExportsPrivacySafeDiagnostics(t *testing.T) {
	exportPath := filepath.Join(t.TempDir(), "diagnostic.zip")
	var stdout bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "--root", t.TempDir(), "--export", exportPath, "--json"})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, exportPath)
	require.Contains(t, stdout.String(), "diagnostic.export")
}

func TestDoctorExportNeverOverwritesExistingPath(t *testing.T) {
	exportPath := filepath.Join(t.TempDir(), "diagnostic.zip")
	require.NoError(t, os.WriteFile(exportPath, []byte("keep\n"), 0o600))
	cmd := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "--root", t.TempDir(), "--export", exportPath, "--json"})

	require.Error(t, cmd.Execute())
	data, err := os.ReadFile(exportPath)
	require.NoError(t, err)
	require.Equal(t, "keep\n", string(data))
}

func TestWorkStartCreatesClaimReadableByNextConflictCheck(t *testing.T) {
	root := filepath.Join(t.TempDir(), "product")
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "init", "--root", root, "--id", "project.claim-test", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())
	defineCommandWork(t, root, "work.account-recovery", "services/identity/**")

	var startOutput bytes.Buffer
	start := command.New("1.0.0", &startOutput, &bytes.Buffer{})
	start.SetArgs([]string{"work", "start", "--root", root, "--work-id", "work.account-recovery", "--claim-id", "claim.account-recovery", "--owner", "alex", "--branch", "feature/account-recovery", "--path", "services/identity/**", "--apply", "--json"})
	require.NoError(t, start.Execute())
	require.Equal(t, 0, command.ExitCode(start), startOutput.String())
	_, err := os.ReadFile(filepath.Join(root, ".harness", "work", "claims", "claim.account-recovery.yaml"))
	require.NoError(t, err)

	candidatePath := filepath.Join(root, "candidate.yaml")
	require.NoError(t, os.WriteFile(candidatePath, []byte("repository: repository.root\npaths: [services/identity/handler/**]\npolicy_ids: []\nscenario_ids: []\ncontract_ids: []\ndb_entities: []\nmigration_slots: []\nui_flows: []\ndependency_majors: []\nstable_ids: []\nroot_pointer: false\nnow: 2026-07-16T00:00:00Z\n"), 0o600))
	var output bytes.Buffer
	conflict := command.New("1.0.0", &output, &bytes.Buffer{})
	conflict.SetArgs([]string{"work", "conflict", "--root", root, "--candidate", candidatePath, "--json"})

	require.NoError(t, conflict.Execute())
	require.Equal(t, 6, command.ExitCode(conflict), "a local-only claim cannot prove team ownership")
	require.Contains(t, output.String(), `"status":"unknown"`)
	require.Contains(t, output.String(), "conflict.claim-unobservable")
}

func TestWorkNextUsesUnavailableExitWhenNothingIsReady(t *testing.T) {
	root := filepath.Join(t.TempDir(), "product")
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "init", "--root", root, "--id", "project.empty-work", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())

	cmd := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	cmd.SetArgs([]string{"work", "next", "--root", root, "--json"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, 6, command.ExitCode(cmd))
}

func TestWorkNextSkipsUnknownAndActivelyClaimedItems(t *testing.T) {
	root := filepath.Join(t.TempDir(), "product")
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "init", "--root", root, "--id", "project.work-selection", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())
	items := map[string]string{
		"work.A.yaml": "schema_version: 1\nid: work.A\ntitle: Unknown meaning\nstatus: ready\nrefs: [policy.missing]\ndependencies: []\nupdated_at: 2026-07-16T00:00:00Z\n",
		"work.B.yaml": "schema_version: 1\nid: work.B\ntitle: Already claimed\nstatus: ready\nrefs: []\ndependencies: []\nupdated_at: 2026-07-16T00:00:00Z\n",
		"work.C.yaml": "schema_version: 1\nid: work.C\ntitle: Safe next work\nstatus: ready\nrefs: []\ndependencies: []\nupdated_at: 2026-07-16T00:00:00Z\n",
	}
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "work", "items"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "work", "claims"), 0o700))
	for name, value := range items {
		require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "items", name), []byte(value), 0o600))
	}
	claim := "schema_version: 1\nid: claim.B\nwork_id: work.B\nowner: alex\nbranch: feature/claimed\nrepository: root\npaths: []\npolicy_ids: []\nscenario_ids: []\ncontract_ids: []\ndb_entities: []\nmigration_slots: []\nui_flows: []\ndependency_majors: []\nroot_pointer: false\nstarts_at: 2026-07-16T00:00:00Z\nexpires_at: 2099-07-17T00:00:00Z\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "claims", "claim.B.yaml"), []byte(claim), 0o600))

	var output bytes.Buffer
	cmd := command.New("1.0.0", &output, &bytes.Buffer{})
	cmd.SetArgs([]string{"work", "next", "--root", root, "--json"})
	require.NoError(t, cmd.Execute())

	require.Equal(t, 0, command.ExitCode(cmd))
	require.Contains(t, output.String(), "Safe next work")
	require.NotContains(t, output.String(), "Unknown meaning")
	require.NotContains(t, output.String(), "Already claimed")
}

func TestWorkNextDoesNotUseLocalItemsWhenExternalProviderIsCanonical(t *testing.T) {
	root := filepath.Join(t.TempDir(), "product")
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "init", "--root", root, "--id", "project.external-work", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "work", "items"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "provider.yaml"), []byte("schema_version: 1\nprovider: github\nlive_status_source: github\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "items", "work.local.yaml"), []byte("schema_version: 1\nid: work.local\ntitle: Stale local copy\nstatus: ready\nrefs: []\ndependencies: []\nupdated_at: 2026-07-16T00:00:00Z\n"), 0o600))

	var output bytes.Buffer
	cmd := command.New("1.0.0", &output, &bytes.Buffer{})
	cmd.SetArgs([]string{"work", "next", "--root", root, "--json"})
	require.NoError(t, cmd.Execute())

	require.Equal(t, 6, command.ExitCode(cmd))
	require.Contains(t, output.String(), `"status":"unknown"`)
	require.NotContains(t, output.String(), "Stale local copy")
}
