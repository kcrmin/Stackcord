package diagnostic_test

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"fullstack-orchestrator/cli/internal/diagnostic"
	"github.com/stretchr/testify/require"
)

func TestExportOmitsPrivateContentAndRedactsPathsAndSecrets(t *testing.T) {
	input := diagnostic.Input{
		Versions: map[string]string{"cli": "1.0.0", "os": "darwin-arm64"},
		Root:     "/Users/alex/private/product", Home: "/Users/alex",
		Errors:   []string{"provider unavailable at /Users/alex/private/product"},
		State:    map[string]string{"branch": "feature/recovery", "remote": "https://alice:password@github.com/private/repo.git", "provider": "token=super-secret-token"},
		Receipts: []string{"operation-01", "/Users/alex/private/product/.harness/local/receipt-token=receipt-secret-value"}, ProviderOutput: "prompt=private product source; api_token=another-secret-value",
	}
	var archive bytes.Buffer
	require.NoError(t, diagnostic.Export(&archive, input))
	reader, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	require.NoError(t, err)
	require.Len(t, reader.File, 1)
	entry, err := reader.File[0].Open()
	require.NoError(t, err)
	data, err := io.ReadAll(entry)
	require.NoError(t, err)
	require.NoError(t, entry.Close())
	text := string(data)
	for _, private := range []string{"/Users/alex", "private product source", "password@", "super-secret-token", "another-secret-value", "receipt-secret-value", "alice:"} {
		require.NotContains(t, text, private)
	}
	require.Contains(t, text, "<PROJECT_ROOT>")
	require.Contains(t, text, "darwin-arm64")
	require.Contains(t, text, "operation-01")
}
