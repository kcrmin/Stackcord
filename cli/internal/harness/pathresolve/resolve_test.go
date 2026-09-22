package pathresolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveCanonicalExistingPathAndMissingTarget(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	require.NoError(t, os.Mkdir(nested, 0700))
	got, err := Resolve(filepath.Join(nested, "..", "nested"))
	require.NoError(t, err)
	want, err := filepath.EvalSymlinks(nested)
	require.NoError(t, err)
	require.Equal(t, want, got)
	_, err = Resolve(filepath.Join(root, "missing"))
	require.Error(t, err)
}

func TestResolveFollowsLinkToActualTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	require.NoError(t, os.Mkdir(target, 0700))
	link := filepath.Join(root, "alias")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink privilege unavailable: %v", err)
	}
	got, err := Resolve(link)
	require.NoError(t, err)
	want, err := filepath.EvalSymlinks(target)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
