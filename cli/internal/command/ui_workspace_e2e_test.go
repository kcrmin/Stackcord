package command_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	uiimport "fullstack-orchestrator/cli/internal/ui"
	"github.com/stretchr/testify/require"
)

func TestEditableUIWorkspaceDrivesAndInvalidatesFrontendWork(t *testing.T) {
	parent := t.TempDir()
	uiRemote := filepath.Join(parent, "ui.git")
	uiSeed := filepath.Join(parent, "ui-seed")
	focusedGit(t, "", "init", "--bare", "--initial-branch=main", uiRemote)
	focusedGit(t, "", "init", "--initial-branch=main", uiSeed)
	focusedGit(t, uiSeed, "config", "user.email", "fixture@example.invalid")
	focusedGit(t, uiSeed, "config", "user.name", "Fixture User")
	require.NoError(t, os.WriteFile(filepath.Join(uiSeed, "README.md"), []byte("UI source\n"), 0o600))
	focusedGit(t, uiSeed, "add", ".")
	focusedGit(t, uiSeed, "commit", "-m", "chore: initialize UI workspace")
	focusedGit(t, uiSeed, "remote", "add", "origin", uiRemote)
	focusedGit(t, uiSeed, "push", "-u", "origin", "main")

	root := filepath.Join(parent, "root")
	require.Contains(t, runFocusedCommand(t, "project", "init", "--root", root, "--id", "project.ui-continuity", "--locale", "en", "--apply", "--json"), `"status":"passed"`)
	focusedGit(t, root, "init", "--initial-branch=main")
	focusedGit(t, root, "config", "user.email", "fixture@example.invalid")
	focusedGit(t, root, "config", "user.name", "Fixture User")
	focusedGit(t, root, "add", ".")
	focusedGit(t, root, "commit", "-m", "chore: initialize project")
	frontendRegistered := runFocusedCommand(t, "workspace", "register", "--root", root, "--id", "workspace.frontend", "--kind", "directory", "--path", "frontend", "--responsibility", "frontend", "--dependency", "workspace.root", "--apply", "--json")
	require.Contains(t, frontendRegistered, `"status":"passed"`)
	focusedGit(t, root, "add", ".harness/workspaces.yaml")
	focusedGit(t, root, "commit", "-m", "build: register frontend workspace")
	focusedGit(t, root, "-c", "protocol.file.allow=always", "submodule", "add", uiRemote, "ui")
	focusedGit(t, root, "config", "-f", ".gitmodules", "submodule.ui.url", "https://example.test/product-ui.git")
	focusedGit(t, root, "config", "submodule.ui.url", "https://example.test/product-ui.git")
	focusedGit(t, filepath.Join(root, "ui"), "remote", "set-url", "origin", "https://example.test/product-ui.git")
	focusedGit(t, root, "add", ".gitmodules", "ui")
	focusedGit(t, root, "commit", "-m", "build: add UI workspace")

	registered := runFocusedCommand(t, "workspace", "register", "--root", root, "--id", "workspace.ui", "--kind", "submodule", "--path", "ui", "--remote", "https://example.test/product-ui.git", "--root-remote", "https://example.test/root.git", "--responsibility", "ui-baseline", "--consumer", "workspace.frontend", "--initialize", "ui", "--apply", "--json")
	require.Contains(t, registered, `"status":"passed"`)
	uiRoot := filepath.Join(root, "ui")
	focusedGit(t, uiRoot, "config", "user.email", "fixture@example.invalid")
	focusedGit(t, uiRoot, "config", "user.name", "Fixture User")
	focusedGit(t, uiRoot, "add", ".")
	focusedGit(t, uiRoot, "commit", "-m", "docs: define UI workspace")
	focusedGit(t, uiRoot, "update-ref", "refs/remotes/origin/main", "HEAD")
	focusedGit(t, root, "add", ".harness/workspaces.yaml", "ui")
	focusedGit(t, root, "commit", "-m", "build: register UI workspace")
	require.FileExists(t, filepath.Join(uiRoot, ".harness", "bridge.yaml"))

	archive := focusedUIArchive(t)
	imported := runFocusedCommand(t, "ui", "import", "--root", root, "--archive", archive, "--id", "ui.external.recovery", "--authority", "seed", "--apply", "--json")
	require.Contains(t, imported, `"status":"passed"`)
	promoted := runFocusedCommand(t, "ui", "promote", "--root", root, "--id", "ui.external.recovery", "--workspace", "workspace.ui", "--mode", "whole", "--apply", "--json")
	require.Contains(t, promoted, `"status":"passed"`)
	require.FileExists(t, filepath.Join(uiRoot, "sources", "ui.external.recovery", "screens", "recovery.html"))
	focusedGit(t, uiRoot, "add", ".")
	focusedGit(t, uiRoot, "commit", "-m", "feat(ui): define recovery flow")
	focusedGit(t, uiRoot, "update-ref", "refs/remotes/origin/main", "HEAD")

	require.NoError(t, os.MkdirAll(filepath.Join(root, "specs", "ui"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "specs", "ui", "recovery.md"), []byte("---\nschema_version: 1\nid: ui.recovery\nkind: ui\nstatus: approved\nrevision: 1\nrefs: []\n---\n\nRecovery screen and failure states.\n"), 0o600))
	bound := runFocusedCommand(t, "ui", "baseline", "bind", "--root", root, "--id", "ui.baseline.recovery", "--workspace", "workspace.ui", "--source", "ui.external.recovery", "--ref", "ui.recovery", "--consumer", "workspace.frontend", "--apply", "--json")
	require.Contains(t, bound, `"status":"passed"`)
	baseline, err := uiimport.LoadBaseline(root, "ui.baseline.recovery")
	require.NoError(t, err)
	focusedGit(t, root, "add", ".harness", "specs", "ui")
	focusedGit(t, root, "commit", "-m", "feat(ui): approve recovery baseline")

	definition := map[string]any{
		"schema_version": 1, "id": "work.recovery-frontend", "readiness": "ready", "title": "Implement recovery UI", "outcome": "Users can complete recovery with visible failure states.",
		"acceptance": []map[string]string{{"id": "scenario.recovery-ui", "given": "an eligible account", "when": "the user follows recovery", "then": "the approved UI states are implemented", "failure": "invalid proof is rejected without disclosure"}},
		"refs":       []string{"ui.recovery"}, "workspaces": []string{"workspace.frontend"}, "dependencies": []string{}, "merge_order": []string{"workspace.frontend"}, "first_failing_test": "test.recovery-ui",
		"scope":        map[string]any{"repositories": []string{"repository.frontend"}, "paths": []string{"frontend/recovery"}, "policy_ids": []string{}, "scenario_ids": []string{}, "contract_ids": []string{}, "db_entities": []string{}, "migration_slots": []string{}, "ui_flows": []string{"ui.recovery"}, "dependency_majors": []string{}, "root_pointers": []string{"workspace.ui"}},
		"evidence":     map[string]any{"kinds": []string{"ui", "test"}, "integration_required": true, "user_validation": false, "migration_required": false, "rollback_required": false},
		"ui_baselines": map[string]string{baseline.ID: baseline.Fingerprint},
	}
	definitionPath := filepath.Join(parent, "work.json")
	data, err := json.Marshal(definition)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(definitionPath, data, 0o600))
	defined := runFocusedCommand(t, "work", "define", "--root", root, "--input", definitionPath, "--apply", "--json")
	require.Contains(t, defined, `"status":"passed"`)
	require.NotContains(t, runFocusedCommand(t, "status", "--root", root, "--json"), "work.ui-baseline-stale")

	require.NoError(t, os.WriteFile(filepath.Join(uiRoot, "states.md"), []byte("Updated failure state\n"), 0o600))
	focusedGit(t, uiRoot, "add", ".")
	focusedGit(t, uiRoot, "commit", "-m", "feat(ui): refine recovery failure state")
	focusedGit(t, uiRoot, "update-ref", "refs/remotes/origin/main", "HEAD")
	rebound := runFocusedCommand(t, "ui", "baseline", "bind", "--root", root, "--id", "ui.baseline.recovery", "--workspace", "workspace.ui", "--source", "ui.external.recovery", "--ref", "ui.recovery", "--consumer", "workspace.frontend", "--apply", "--json")
	require.Contains(t, rebound, `"status":"passed"`)
	require.Contains(t, runFocusedCommand(t, "status", "--root", root, "--json"), "work.ui-baseline-stale")
}
