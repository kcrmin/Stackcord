package ui_test

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	uiimport "github.com/kcrmin/Stackcord/cli/internal/ui"
	"github.com/stretchr/testify/require"
)

func TestPromoteReviewedUISourceIntoEditableWorkspace(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "ui"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "manifest.yaml"), []byte("schema_version: 1\nid: project.example\nlocale: en\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "workspaces.yaml"), []byte("schema_version: 1\nproject_id: project.example\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n  - id: workspace.ui\n    kind: directory\n    path: ui\n    responsibilities: [ui-baseline]\n    dependencies: [workspace.root]\n"), 0o600))
	archive := promotionArchive(t)
	_, importPlan, err := uiimport.Register(uiimport.Source{Root: root, Archive: archive, ID: "ui.external.checkout", Kind: "mockup", Authority: "seed"})
	require.NoError(t, err)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), importPlan).Status)

	promotion, err := uiimport.Promote(uiimport.PromotionRequest{Root: root, SourceID: "ui.external.checkout", WorkspaceID: "workspace.ui", Mode: "selected", Paths: []string{"screens/checkout.html"}})
	require.NoError(t, err)
	require.Empty(t, promotion.Blockers)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), promotion).Status)
	require.FileExists(t, filepath.Join(root, "ui", "sources", "ui.external.checkout", "screens", "checkout.html"))
	require.NoFileExists(t, filepath.Join(root, "ui", "sources", "ui.external.checkout", "notes.txt"))
	require.FileExists(t, filepath.Join(root, "ui", "sources", "ui.external.checkout", "promotion.yaml"))

	require.NoError(t, os.WriteFile(filepath.Join(root, "ui", "sources", "ui.external.checkout", "screens", "checkout.html"), []byte("changed locally"), 0o600))
	blocked, err := uiimport.Promote(uiimport.PromotionRequest{Root: root, SourceID: "ui.external.checkout", WorkspaceID: "workspace.ui", Mode: "selected", Paths: []string{"screens/checkout.html"}})
	require.NoError(t, err)
	require.NotEmpty(t, blocked.Blockers)
}

func promotionArchive(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.zip")
	file, err := os.Create(path)
	require.NoError(t, err)
	writer := zip.NewWriter(file)
	for name, value := range map[string]string{"LICENSE": "MIT", "screens/checkout.html": "<main>Checkout</main>", "notes.txt": "reference"} {
		entry, createErr := writer.Create(name)
		require.NoError(t, createErr)
		_, createErr = entry.Write([]byte(value))
		require.NoError(t, createErr)
	}
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())
	return path
}
