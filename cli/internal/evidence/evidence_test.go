package evidence

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestEvidenceBecomesStaleWhenWorkspaceHeadChanges(t *testing.T) {
	repo := evidenceRepository(t)
	fingerprint := "sha256:" + strings.Repeat("a", 64)
	contract := "sha256:" + strings.Repeat("b", 64)
	record, result := Run(context.Background(), Request{
		Workspace: repo, WorkspaceID: "workspace.root", WorkID: "work.account-recovery",
		DefinitionFingerprint: fingerprint, ContractFingerprint: contract,
		Command: ApprovedCommand{ID: "command.git-status", Kind: "test", Argv: []string{"git", "status", "--porcelain=v2"}, TimeoutSeconds: 30},
	})
	require.Equal(t, domain.StatusPassed, result.Status, result.Blockers)
	require.NotEmpty(t, record.ID)
	commitEvidenceFile(t, repo, "after.txt", "changed\n")

	issues := VerifyCurrent(record, Actual{Workspace: repo, Head: evidenceGit(t, repo, "rev-parse", "HEAD"), DefinitionFingerprint: fingerprint, ContractFingerprint: contract})

	require.Contains(t, evidenceCodes(issues), "evidence.commit-changed")
}

func TestDirtyWorkspaceCannotProduceReusableEvidence(t *testing.T) {
	repo := evidenceRepository(t)
	require.NoError(t, os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("dirty\n"), 0o600))

	_, result := Run(context.Background(), Request{
		Workspace: repo, WorkspaceID: "workspace.root", WorkID: "work.account-recovery",
		DefinitionFingerprint: "sha256:" + strings.Repeat("a", 64), ContractFingerprint: "sha256:" + strings.Repeat("b", 64),
		Command: ApprovedCommand{ID: "command.git-status", Kind: "test", Argv: []string{"git", "status", "--porcelain=v2"}, TimeoutSeconds: 30},
	})

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Contains(t, evidenceCodes(result.Blockers), "evidence.workspace-dirty")
}

func TestFingerprintTreeChangesWithContractMeaning(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "contracts"), 0o700))
	path := filepath.Join(root, "contracts", "registry.yaml")
	require.NoError(t, os.WriteFile(path, []byte("schema_version: 1\ncontracts: []\n"), 0o600))

	before, err := FingerprintTree(root, "contracts")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("schema_version: 1\ncontracts: [contract.account]\n"), 0o600))
	after, err := FingerprintTree(root, "contracts")
	require.NoError(t, err)
	require.NotEqual(t, before, after)
}

func TestFingerprintTreeRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "contracts"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "outside.yaml"), []byte("secret\n"), 0o600))
	require.NoError(t, os.Symlink(filepath.Join(root, "outside.yaml"), filepath.Join(root, "contracts", "linked.yaml")))

	_, err := FingerprintTree(root, "contracts")
	require.ErrorContains(t, err, "symlink")
}

func evidenceRepository(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	evidenceGit(t, repo, "init", "--initial-branch=main")
	evidenceGit(t, repo, "config", "user.email", "fixture@example.invalid")
	evidenceGit(t, repo, "config", "user.name", "Fixture User")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "README.md"), []byte("fixture\n"), 0o600))
	evidenceGit(t, repo, "add", "README.md")
	evidenceGit(t, repo, "commit", "-m", "chore: initialize fixture")
	return repo
}

func commitEvidenceFile(t *testing.T, repo, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(repo, name), []byte(content), 0o600))
	evidenceGit(t, repo, "add", name)
	evidenceGit(t, repo, "commit", "-m", "test: change evidence head")
}

func evidenceGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = repo
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "git %s failed: %s", strings.Join(args, " "), output)
	return strings.TrimSpace(string(output))
}

func evidenceCodes(items []domain.Item) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}
