package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectArtifactDigestsRejectsInvalidNamesAndUnsafePaths(t *testing.T) {
	workspace := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workspace, "artifact.bin"), []byte("verified\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(workspace), "outside.bin"), []byte("outside\n"), 0o600))

	for name, values := range map[string][]string{
		"invalid name": {"bad name=artifact.bin"},
		"escape":       {"service=../outside.bin"},
		"absolute":     {"service=" + filepath.Join(workspace, "artifact.bin")},
		"duplicate":    {"service=artifact.bin", "service=artifact.bin"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := collectArtifactDigests(workspace, values)
			require.Error(t, err)
		})
	}
}

func TestCollectArtifactDigestsRejectsSymlink(t *testing.T) {
	workspace := t.TempDir()
	target := filepath.Join(workspace, "artifact.bin")
	require.NoError(t, os.WriteFile(target, []byte("verified\n"), 0o600))
	link := filepath.Join(workspace, "artifact-link.bin")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	_, err := collectArtifactDigests(workspace, []string{"service=artifact-link.bin"})

	require.Error(t, err)
}
