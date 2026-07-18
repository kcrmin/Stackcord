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
		{"windows path escape", map[string]string{"..\\escape.txt": "x", "LICENSE": "MIT"}},
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

func TestImportRejectsDuplicateNormalizedPaths(t *testing.T) {
	archive := zipFixtureEntries(t, [][2]string{{"LICENSE", "MIT"}, {"screens/Recovery.html", "one"}, {"screens/recovery.html", "two"}})
	plan, err := uiimport.ImportPlan(uiimport.Source{Root: t.TempDir(), Archive: archive, ID: "ui.external.duplicate", Kind: "mockup", Authority: "reference"})
	require.ErrorContains(t, err, "duplicate")
	require.Empty(t, plan.Files)
}

func TestImportPlanQuarantinesValidReference(t *testing.T) {
	archive := zipFixture(t, map[string]string{"LICENSE": "MIT", "screens/recovery.html": "<main>Recover</main>"})
	root := t.TempDir()
	plan, err := uiimport.ImportPlan(uiimport.Source{Root: root, Archive: archive, ID: "ui.external.recovery", Kind: "mockup", Authority: "reference"})
	require.NoError(t, err)
	require.NotEmpty(t, plan.Files)
	registrationFound := false
	for _, file := range plan.Files {
		if file.Path == ".harness/sources/ui/ui.external.recovery.yaml" {
			registrationFound = true
			continue
		}
		require.Contains(t, file.Path, ".harness/local/imports/")
	}
	require.True(t, registrationFound)
}

func zipFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	entries := make([][2]string, 0, len(files))
	for name, content := range files {
		entries = append(entries, [2]string{name, content})
	}
	return zipFixtureEntries(t, entries)
}

func zipFixtureEntries(t *testing.T, entries [][2]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.zip")
	file, err := os.Create(path)
	require.NoError(t, err)
	writer := zip.NewWriter(file)
	for _, item := range entries {
		entry, createErr := writer.Create(item[0])
		require.NoError(t, createErr)
		_, createErr = entry.Write([]byte(item[1]))
		require.NoError(t, createErr)
	}
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())
	return path
}
