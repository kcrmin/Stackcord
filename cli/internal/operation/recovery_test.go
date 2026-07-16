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

func TestApplyRejectsUnsafeOperationIDBeforeCreatingJournal(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "project")
	require.NoError(t, os.MkdirAll(root, 0o700))
	plan := operation.Plan{ID: "../../../escape", Root: root, Files: []operation.FileChange{{Path: "safe.txt", Content: []byte("safe\n"), Mode: 0o600}}}
	plan.InitialStateFingerprint, _ = operation.StateFingerprint(plan)

	result := operation.Apply(context.Background(), plan)

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Equal(t, domain.ExitInvalid, result.ExitCode)
	require.NoDirExists(t, filepath.Join(root, "escape"))
	require.NoFileExists(t, filepath.Join(root, "safe.txt"))
}

func TestCompletedOperationIDCannotBeReusedForDifferentPlan(t *testing.T) {
	root := t.TempDir()
	first := operation.Plan{ID: "01JIMMUTABLE", Root: root, Files: []operation.FileChange{{Path: "one.txt", Content: []byte("one\n"), Mode: 0o600}}}
	first.InitialStateFingerprint, _ = operation.StateFingerprint(first)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), first).Status)

	changed := operation.Plan{ID: first.ID, Root: root, Files: []operation.FileChange{{Path: "two.txt", Content: []byte("two\n"), Mode: 0o600}}}
	changed.InitialStateFingerprint, _ = operation.StateFingerprint(changed)
	result := operation.Apply(context.Background(), changed)

	require.Equal(t, domain.StatusFailed, result.Status)
	require.Equal(t, "operation.plan-conflict", result.Blockers[0].Code)
	require.NoFileExists(t, filepath.Join(root, "two.txt"))
}

func TestCompletedOperationDoesNotHideLaterTargetDrift(t *testing.T) {
	root := t.TempDir()
	plan := operation.Plan{ID: "01JDRIFT", Root: root, Files: []operation.FileChange{{Path: "target.txt", Content: []byte("approved\n"), Mode: 0o600}}}
	plan.InitialStateFingerprint, _ = operation.StateFingerprint(plan)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)
	require.NoError(t, os.WriteFile(filepath.Join(root, "target.txt"), []byte("changed later\n"), 0o600))

	result := operation.Apply(context.Background(), plan)

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Equal(t, domain.ExitBlocked, result.ExitCode)
	require.Equal(t, "operation.completed-state-changed", result.Blockers[0].Code)
}

func TestApplyRefusesConcurrentOperationLock(t *testing.T) {
	root := t.TempDir()
	plan := operation.Plan{ID: "01JLOCKED", Root: root, Files: []operation.FileChange{{Path: "target.txt", Content: []byte("safe\n"), Mode: 0o600}}}
	plan.InitialStateFingerprint, _ = operation.StateFingerprint(plan)
	lockDirectory := filepath.Join(root, ".harness", "local", "operations", plan.ID)
	require.NoError(t, os.MkdirAll(lockDirectory, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(lockDirectory, "apply.lock"), []byte("active\n"), 0o600))

	result := operation.Apply(context.Background(), plan)

	require.Equal(t, domain.StatusUnknown, result.Status)
	require.Equal(t, domain.ExitUnavailable, result.ExitCode)
	require.Equal(t, "operation.concurrent", result.Blockers[0].Code)
	require.NoFileExists(t, filepath.Join(root, "target.txt"))
}

func TestStateFingerprintRejectsSymlinkParentEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	_, err := operation.StateFingerprint(operation.Plan{ID: "01JLINK", Root: root, Files: []operation.FileChange{{Path: "linked/escape.txt", Content: []byte("unsafe")}}})

	require.ErrorContains(t, err, "symlink")
}
