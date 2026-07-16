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

func FuzzFingerprint(f *testing.F) {
	f.Add("yaml", []byte("schema_version: 1\nid: policy.example\n"))
	f.Add("markdown", []byte("# Heading\r\n\r\nBody  \r\n"))
	f.Add("json", []byte(`{"id":"policy.example","revision":1}`))
	f.Fuzz(func(t *testing.T, kind string, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		first, firstErr := contextpkg.Fingerprint(kind, data)
		second, secondErr := contextpkg.Fingerprint(kind, data)
		if (firstErr == nil) != (secondErr == nil) {
			t.Fatalf("same input returned inconsistent errors")
		}
		if firstErr == nil && first != second {
			t.Fatalf("same input returned different fingerprints")
		}
	})
}
