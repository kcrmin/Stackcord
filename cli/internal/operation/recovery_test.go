package operation_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"github.com/stretchr/testify/require"
)

func TestApplyRecoversAfterPartialFilesystemFailure(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "local", "operations"), 0o700))
	blockedPath := filepath.Join(root, "generated", "two.txt")
	require.NoError(t, os.MkdirAll(blockedPath, 0o700))

	plan := operation.Plan{
		ID: "01JRECOVERY", Root: root,
		Files: []operation.FileChange{
			{Path: "generated/one.txt", Content: []byte("one\n"), Mode: 0o644},
			{Path: "generated/two.txt", Content: []byte("two\n"), Mode: 0o644},
			{Path: "generated/three.txt", Content: []byte("three\n"), Mode: 0o644},
		},
	}
	fingerprint, err := operation.StateFingerprint(plan)
	require.NoError(t, err)
	plan.InitialStateFingerprint = fingerprint

	failed := operation.Apply(context.Background(), plan)
	require.Equal(t, domain.StatusFailed, failed.Status)
	require.FileExists(t, filepath.Join(root, "generated", "one.txt"))
	require.NoFileExists(t, filepath.Join(root, "generated", "three.txt"))
	require.NoFileExists(t, filepath.Join(root, "generated", ".01JRECOVERY.tmp"))

	require.NoError(t, os.Remove(blockedPath))
	completed := operation.Apply(context.Background(), plan)
	require.Equal(t, domain.StatusPassed, completed.Status)
	for _, name := range []string{"one.txt", "two.txt", "three.txt"} {
		data, readErr := os.ReadFile(filepath.Join(root, "generated", name))
		require.NoError(t, readErr)
		require.Equal(t, name[:len(name)-4]+"\n", string(data))
	}
	require.FileExists(t, filepath.Join(root, ".harness", "local", "operations", plan.ID, "receipt.json"))

	again := operation.Apply(context.Background(), plan)
	require.Equal(t, domain.StatusPassed, again.Status)
	require.Contains(t, again.Summary, "already completed")
}

func TestStateFingerprintRejectsPathEscape(t *testing.T) {
	_, err := operation.StateFingerprint(operation.Plan{ID: "unsafe", Root: t.TempDir(), Files: []operation.FileChange{{Path: "../escape", Content: []byte("x")}}})
	require.ErrorContains(t, err, "escapes project root")
}
