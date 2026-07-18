package project_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/project"
	"github.com/stretchr/testify/require"
)

func TestNewProjectCreatesNeutralHarness(t *testing.T) {
	parent := t.TempDir()
	draftPlan, err := project.PlanCheckpoint(project.CheckpointRequest{Parent: parent, DraftID: "01JPROJECT", Locale: "ko", Checkpoint: validCheckpoint("팀이 승인한 제품 요약", "공개 제품 이름")})
	require.NoError(t, err)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), draftPlan).Status)

	root := filepath.Join(parent, "service-product")
	plan, err := project.PlanInit(project.InitRequest{Root: root, ProjectID: "project.service-product", Name: "Service Product", Locale: "ko", DraftRoot: filepath.Join(parent, ".harness-drafts", "01JPROJECT")})
	require.NoError(t, err)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)

	for _, path := range []string{
		"AGENTS.md", ".agents/skills/use-project-harness/SKILL.md", ".agents/skills/use-project-harness/references/fallback.md",
		".harness/manifest.yaml", ".harness/entry.md", ".harness/profile.yaml", ".harness/workspaces.yaml",
		"specs/index.md", "specs/product/summary.md", "specs/policies/policy.account.recovery-proof.md", "specs/scenarios/scenario.account.recovery-success.md",
		"contracts/registry.yaml", "contracts/product/index.md", "contracts/business/index.md", "contracts/behaviors/index.md", "contracts/interfaces/index.md", "contracts/data/index.md", "docs/index.md",
	} {
		require.FileExists(t, filepath.Join(root, filepath.FromSlash(path)))
	}
	for _, path := range []string{"frontend", "backend", "src", "app"} {
		_, statErr := os.Stat(filepath.Join(root, path))
		require.ErrorIs(t, statErr, os.ErrNotExist)
	}
	for _, path := range []string{
		".harness/policies", ".harness/templates", ".harness/integrations", ".harness/state/lifecycle.yaml", ".harness/state/baselines.yaml",
		".harness/state/release-candidate.yaml", ".harness/work/links.yaml", "contracts/errors.yaml",
	} {
		require.NoDirExists(t, filepath.Join(root, filepath.FromSlash(path)))
		require.NoFileExists(t, filepath.Join(root, filepath.FromSlash(path)))
	}
	require.NoFileExists(t, filepath.Join(root, ".harness", "state", "context-index.json"))
	require.NoFileExists(t, filepath.Join(root, ".harness", "state", "impact-graph.json"))
	require.NoFileExists(t, filepath.Join(root, ".harness", "local", "context", "context-index.json"))
	require.Contains(t, mustRead(t, filepath.Join(root, ".gitignore")), ".harness/local/")
	manifest, err := os.ReadFile(filepath.Join(root, ".harness", "manifest.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(manifest), "project.service-product")
	workspaces := mustRead(t, filepath.Join(root, ".harness", "workspaces.yaml"))
	require.Contains(t, workspaces, "project_id: project.service-product")
	_, issues := contextpkg.Refresh(context.Background(), root, contextpkg.ReadOnly)
	require.Empty(t, issues, "a newly generated project must be immediately resumable")
	snapshot, _ := contextpkg.Refresh(context.Background(), root, contextpkg.ReadOnly)
	require.Contains(t, snapshot.Index, "policy.account.recovery-proof")
	require.Contains(t, snapshot.Index, "scenario.account.recovery-success")
}

func TestGeneratedRepoLocalGuidanceIsFlexibleAndMatchesPluginTemplate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "service-product")
	plan, err := project.PlanInit(project.InitRequest{Root: root, ProjectID: "project.service-product", Name: "Service Product", Locale: "en"})
	require.NoError(t, err)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)

	skill := mustRead(t, filepath.Join(root, ".agents", "skills", "use-project-harness", "SKILL.md"))
	fallback := mustRead(t, filepath.Join(root, ".agents", "skills", "use-project-harness", "references", "fallback.md"))
	for _, required := range []string{
		"natural-language", "initial product request", "A/B/C", "free-form", "small private local edit", "selected task source", "Git work reservation",
		"service purpose", "business rules", "context audit", "strict release",
	} {
		require.Contains(t, skill+fallback, required)
	}
	for _, internal := range []string{"active claim", "live claim", "remote-claim", "claiming service-wide context"} {
		require.NotContains(t, skill+fallback, internal)
	}

	_, source, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", ".."))
	require.Equal(t,
		mustRead(t, filepath.Join(repositoryRoot, "templates", "project", ".agents", "skills", "use-project-harness", "SKILL.md")),
		skill,
	)
	require.Equal(t,
		mustRead(t, filepath.Join(repositoryRoot, "templates", "project", ".agents", "skills", "use-project-harness", "references", "fallback.md")),
		fallback,
	)
}

func TestAdoptExistingProjectPreservesCustomFiles(t *testing.T) {
	root := t.TempDir()
	customReadme := "# Existing Product\n\nCustom instructions.\n"
	customAgents := "# Existing agent rules\n\nKeep this.\n"
	customGitignore := "*.log\n!important.log\n*.log\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte(customReadme), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(customAgents), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte(customGitignore), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "user-dirty.txt"), []byte("do not touch\n"), 0o600))

	plan, err := project.PlanAdopt(project.InitRequest{Root: root, ProjectID: "project.existing", Name: "Existing Product", Locale: "en"})
	require.NoError(t, err)
	require.Equal(t, customReadme, mustRead(t, filepath.Join(root, "README.md")), "planning is read-only")
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)
	require.Contains(t, mustRead(t, filepath.Join(root, "README.md")), customReadme)
	require.Contains(t, mustRead(t, filepath.Join(root, "README.md")), "orchestrator:begin")
	require.Contains(t, mustRead(t, filepath.Join(root, "AGENTS.md")), customAgents)
	require.True(t, strings.HasPrefix(mustRead(t, filepath.Join(root, ".gitignore")), customGitignore), "ordered ignore rules must remain byte-for-byte at the start")
	require.Equal(t, "do not touch\n", mustRead(t, filepath.Join(root, "user-dirty.txt")))
}

func TestAdoptBlocksSemanticToolingConflict(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("* -text\n"), 0o600))
	plan, err := project.PlanAdopt(project.InitRequest{Root: root, ProjectID: "project.conflict", Name: "Conflict", Locale: "en"})
	require.NoError(t, err)
	require.NotEmpty(t, plan.Blockers)
	require.Empty(t, plan.Files)
}

func TestAdoptUpgradesLegacyWorkspaceIdentityWithoutReplacingEntries(t *testing.T) {
	root := t.TempDir()
	legacy := `schema_version: 1
workspaces:
  - id: workspace.root
    kind: root
    path: .
    responsibilities: [orchestration, product-contracts]
    dependencies: []
`
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte(legacy), 0o600))

	plan, err := project.PlanAdopt(project.InitRequest{Root: root, ProjectID: "project.existing", Name: "Existing", Locale: "en"})
	require.NoError(t, err)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)

	got := mustRead(t, filepath.Join(root, ".harness", "workspaces.yaml"))
	require.Contains(t, got, "project_id: project.existing")
	require.Contains(t, got, "responsibilities: [orchestration, product-contracts]")
}

func TestProjectAndDraftRejectPathLikeStableIDs(t *testing.T) {
	_, err := project.CreateDraft(project.DraftRequest{Parent: t.TempDir(), DraftID: "../../escape", Locale: "en"})
	require.ErrorContains(t, err, "draft ID")

	_, err = project.PlanInit(project.InitRequest{Root: filepath.Join(t.TempDir(), "product"), ProjectID: "../project", Locale: "en"})
	require.ErrorContains(t, err, "project ID")
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}
