package schema_test

import (
	"os"
	"path/filepath"
	"testing"

	"fullstack-orchestrator/cli/internal/schema"
	"github.com/stretchr/testify/require"
)

type sampleDocument struct {
	SchemaVersion int      `yaml:"schema_version"`
	ID            string   `yaml:"id"`
	Status        string   `yaml:"status"`
	Revision      int      `yaml:"revision"`
	Refs          []string `yaml:"refs"`
}

func TestLoadYAMLRejectsUnknownAndDuplicateFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "document.yaml")
	require.NoError(t, os.WriteFile(path, []byte("schema_version: 1\nid: policy.account\nstatus: approved\nrevision: 1\nrefs: []\nunexpected: true\n"), 0o600))
	_, err := schema.LoadYAML[sampleDocument](path)
	require.ErrorContains(t, err, "field unexpected not found")

	require.NoError(t, os.WriteFile(path, []byte("schema_version: 1\nid: policy.account\nid: policy.other\nstatus: approved\nrevision: 1\nrefs: []\n"), 0o600))
	_, err = schema.LoadYAML[sampleDocument](path)
	require.ErrorContains(t, err, "duplicate key")
}

func TestValidateReturnsStableIssues(t *testing.T) {
	valid := map[string]any{
		"schema_version": 1,
		"id":             "policy.account.rate-limit",
		"kind":           "policy",
		"status":         "approved",
		"revision":       1,
		"owners":         []any{"workspace.identity"},
		"refs":           []any{"scenario.account.rate-limited"},
	}
	require.Empty(t, schema.Validate("spec", valid))

	invalid := map[string]any{
		"schema_version": 1,
		"id":             "INVALID",
		"kind":           "policy",
		"status":         "approved",
		"revision":       0,
		"refs":           []any{"policy.one", "policy.one"},
		"api_token":      "must-never-be-stored",
	}
	issues := schema.Validate("spec", invalid)
	require.NotEmpty(t, issues)
	for _, issue := range issues {
		require.Equal(t, "schema.invalid", issue.Code)
	}
}

func TestValidateRejectsUnknownSchemaKind(t *testing.T) {
	issues := schema.Validate("not-registered", map[string]any{})
	require.Len(t, issues, 1)
	require.Equal(t, "schema.unknown-kind", issues[0].Code)
}

func TestDecodeJSONRejectsDuplicateAndUnknownFields(t *testing.T) {
	_, err := schema.DecodeJSON[sampleDocument]([]byte(`{"schema_version":1,"id":"policy.account","id":"policy.other","status":"approved","revision":1,"refs":[]}`))
	require.ErrorContains(t, err, "duplicate key")

	_, err = schema.DecodeJSON[sampleDocument]([]byte(`{"schema_version":1,"id":"policy.account","status":"approved","revision":1,"refs":[],"unexpected":true}`))
	require.ErrorContains(t, err, "unknown field")
}

func TestValidateDiscoveryCheckpointRequiresNormalizedSections(t *testing.T) {
	valid := map[string]any{
		"schema_version": 1, "summary": "Account recovery", "current_focus": "Recovery proof",
		"roles": []any{}, "journeys": []any{}, "capabilities": []any{}, "policies": []any{}, "scenarios": []any{},
		"quality": []any{}, "ui_coverage": []any{}, "technology_needs": []any{}, "decisions": []any{}, "assumptions": []any{}, "open_questions": []any{},
	}
	require.Empty(t, schema.Validate("discovery", valid))
	delete(valid, "open_questions")
	require.NotEmpty(t, schema.Validate("discovery", valid))
}

func TestValidateFocusedProjectProfile(t *testing.T) {
	valid := map[string]any{"schema_version": 1, "tdd": "default", "git": map[string]any{"collaboration": "strongly_recommended", "release": "required"}, "task_source": "git-local", "release": "core"}
	require.Empty(t, schema.Validate("profile", valid))
	valid["release"] = "enterprise-everywhere"
	require.NotEmpty(t, schema.Validate("profile", valid))
}
