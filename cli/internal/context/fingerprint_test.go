package context_test

import (
	"testing"

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"github.com/stretchr/testify/require"
)

func TestFingerprintNormalizesLineEndingsAndYAMLKeyOrder(t *testing.T) {
	a, err := contextpkg.Fingerprint("yaml", []byte("id: policy.account\r\nstatus: approved\r\n"))
	require.NoError(t, err)
	b, err := contextpkg.Fingerprint("yaml", []byte("status: approved\nid: policy.account\n"))
	require.NoError(t, err)
	require.Equal(t, a, b)
}

func TestFingerprintRejectsDuplicateYAMLKeys(t *testing.T) {
	_, err := contextpkg.Fingerprint("yaml", []byte("id: one\nid: two\n"))
	require.ErrorContains(t, err, "duplicate key")
}

func TestFingerprintNormalizesMarkdown(t *testing.T) {
	a, err := contextpkg.Fingerprint("markdown", []byte("heading  \r\nbody\t\r\n\r\n"))
	require.NoError(t, err)
	b, err := contextpkg.Fingerprint("markdown", []byte("heading\nbody\n"))
	require.NoError(t, err)
	require.Equal(t, a, b)
}
