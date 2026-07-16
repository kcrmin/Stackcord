package ui_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	uiimport "fullstack-orchestrator/cli/internal/ui"
	"github.com/stretchr/testify/require"
)

func TestImportRejectsMaliciousArchives(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
	}{
		{"path escape", map[string]string{"../escape.txt": "x", "LICENSE": "MIT"}},
		{"executable script", map[string]string{"screen.sh": "#!/bin/sh", "LICENSE": "MIT"}},
		{"missing license", map[string]string{"screen.html": "safe"}},
		{"embedded token", map[string]string{"screen.html": "api_token=abcdefghijklmnopqrstuvwxyz123456", "LICENSE": "MIT"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			archive := zipFixture(t, test.files)
			plan, err := uiimport.ImportPlan(uiimport.Source{Root: t.TempDir(), Archive: archive, ID: "ui.external.mock", Kind: "mockup", Authority: "reference"})
			require.Error(t, err)
			require.Empty(t, plan.Files)
		})
	}
}

func TestImportPlanQuarantinesValidReference(t *testing.T) {
	archive := zipFixture(t, map[string]string{"LICENSE": "MIT", "screens/recovery.html": "<main>Recover</main>"})
	root := t.TempDir()
	plan, err := uiimport.ImportPlan(uiimport.Source{Root: root, Archive: archive, ID: "ui.external.recovery", Kind: "mockup", Authority: "reference"})
	require.NoError(t, err)
	require.NotEmpty(t, plan.Files)
	for _, file := range plan.Files {
		require.Contains(t, file.Path, ".harness/local/imports/")
	}
}

func zipFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.zip")
	file, err := os.Create(path)
	require.NoError(t, err)
	writer := zip.NewWriter(file)
	for name, content := range files {
		entry, createErr := writer.Create(name)
		require.NoError(t, createErr)
		_, createErr = entry.Write([]byte(content))
		require.NoError(t, createErr)
	}
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())
	return path
}
