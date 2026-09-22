package convention_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/harness/convention"
	"github.com/stretchr/testify/require"
)

func TestMissingConfigurationUsesExistingBranchConvention(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, convention.ValidateBranch(root, "feature/account-recovery"))
	require.Error(t, convention.ValidateBranch(root, "feat/OPS-42-account-recovery"))
}

func TestConfiguredBranchConventionReplacesTheDefaultFormat(t *testing.T) {
	root := conventionRoot(t, `schema_version: 1
branch:
  format: "{type}/{issue}-{description}"
  types: [feat, fix]
commit:
  title_format: "{type}({scope}): {subject}"
  types: [feat, fix]
  scopes: [api, ui]
  max_length: 72
pull_request:
  title_format: "[{issue}] {title}"
  required_sections: [Summary, Test plan]
issue:
  title_format: "[{type}] {title}"
  required_sections: [Problem, Acceptance]
`)

	require.NoError(t, convention.ValidateBranch(root, "feat/OPS-42-account-recovery"))
	require.Error(t, convention.ValidateBranch(root, "feature/account-recovery"))

	config, configured, err := convention.Load(root)
	require.NoError(t, err)
	require.True(t, configured)
	require.Equal(t, "{type}({scope}): {subject}", config.Commit.TitleFormat)
	require.Equal(t, []string{"Summary", "Test plan"}, config.PullRequest.RequiredSections)
	require.Equal(t, []string{"Problem", "Acceptance"}, config.Issue.RequiredSections)
}

func TestConfiguredConventionNeverWeakensGitReferenceSafety(t *testing.T) {
	root := conventionRoot(t, `schema_version: 1
branch:
  format: "{type}/{description}"
  types: [feat, ai]
`)

	for _, branch := range []string{"feat/../escape", "feat/name.lock", "feat/generated-by-ai", "ai/account", "-feat/account"} {
		require.Error(t, convention.ValidateBranch(root, branch), branch)
	}
}

func TestConfigurationRejectsUnknownFieldsAndUnsupportedPlaceholders(t *testing.T) {
	unknown := conventionRoot(t, `schema_version: 1
branch:
  format: "{type}/{description}"
  types: [feat]
  unexpected: true
`)
	_, _, err := convention.Load(unknown)
	require.Error(t, err)

	unsupported := conventionRoot(t, `schema_version: 1
branch:
  format: "{team}/{description}"
  types: [feat]
`)
	require.Error(t, convention.ValidateBranch(unsupported, "core/account-recovery"))
}

func conventionRoot(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "git-conventions.yaml"), []byte(content), 0o600))
	return root
}
