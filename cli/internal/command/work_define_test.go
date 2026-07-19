package command_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/command"
	"github.com/stretchr/testify/require"
)

func TestWorkDefineWritesFingerprintButNoLiveOwnerOrStatus(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte("schema_version: 1\nproject_id: project.example\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n"), 0o600))
	input := filepath.Join(t.TempDir(), "work.yaml")
	require.NoError(t, os.WriteFile(input, []byte(`schema_version: 1
id: work.health-check
readiness: ready
title: Add a health check
outcome: Operators can distinguish healthy and unavailable service states.
acceptance:
  - id: scenario.health-check
    given: a running service
    when: the health endpoint is requested
    then: dependency health is reported
    failure: unavailable dependencies produce a non-healthy result
refs: []
workspaces: [workspace.root]
scope:
  repositories: [repository.root]
  paths: [health]
  policy_ids: []
  scenario_ids: []
  contract_ids: []
  db_entities: []
  migration_slots: []
  ui_flows: []
  dependency_majors: []
  root_pointers: []
dependencies: []
merge_order: [workspace.root]
first_failing_test: test.health-check
evidence:
  kinds: [unit]
  integration_required: false
  user_validation: false
  migration_required: false
  rollback_required: false
`), 0o600))

	var stdout bytes.Buffer
	cmd := command.New("1.0.0", &stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"work", "define", "--root", root, "--input", input, "--apply", "--json"})
	require.NoError(t, cmd.Execute())

	definitionPath := filepath.Join(root, ".harness", "work", "definitions", "work.health-check.yaml")
	data, err := os.ReadFile(definitionPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "fingerprint: sha256:")
	require.NotContains(t, string(data), "owner:")
	require.NotContains(t, string(data), "status:")
}
