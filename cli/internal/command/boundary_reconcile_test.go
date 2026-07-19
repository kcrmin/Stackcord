package command_test

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/command"
	contextpkg "github.com/kcrmin/Stackcord/cli/internal/context"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestDBDiagramCommandRecordsBeforeCanonicalReconcile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	runBoundaryCommand(t, "project", "init", "--root", root, "--id", "project.database-reconcile", "--locale", "en", "--apply", "--json")
	canonicalPath := filepath.Join(root, "schema.dbml")
	canonical := "Table accounts {\n id int [pk]\n recovery_state varchar\n}\n"
	candidate := "Table accounts {\n id int [pk]\n recovery_state int [not null]\n}\n"
	require.NoError(t, os.WriteFile(canonicalPath, []byte(canonical), 0o600))

	prepared := runBoundaryCommand(t, "db", "diagram", "prepare", "--root", root, "--operation", "accounts-review", "--entry", "schema.dbml", "--tool-version", "1.4.2", "--project-id", "diagram-1", "--action", "pull", "--apply", "--json")
	require.Contains(t, prepared, `"status":"passed"`)
	candidatePath := filepath.Join(root, ".harness", "local", "dbdiagram", "accounts-review", "candidate.dbml")
	require.NoError(t, os.WriteFile(candidatePath, []byte(candidate), 0o600))

	recorded := runBoundaryCommand(t, "db", "diagram", "reconcile", "--root", root, "--operation", "accounts-review", "--contract", "contract.data.accounts", "--migration", "migration.accounts.recovery-state", "--test", "test.accounts.migration", "--rollback", "rollback.accounts.recovery-state", "--record", "--json")
	require.Contains(t, recorded, "canonical DBML is unchanged")
	require.Contains(t, recorded, `"db.columns.changed"`)
	require.Equal(t, canonical, boundaryRead(t, canonicalPath))

	applied := runBoundaryCommand(t, "db", "diagram", "reconcile", "--root", root, "--operation", "accounts-review", "--apply", "--json")
	require.Contains(t, applied, `"status":"passed"`)
	require.Equal(t, candidate, boundaryRead(t, canonicalPath))
}

func TestCanonicalUIReconcilePersistsStaleMappings(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	runBoundaryCommand(t, "project", "init", "--root", root, "--id", "project.ui-reconcile", "--locale", "en", "--apply", "--json")
	first := boundaryUIArchive(t, "<main>Refund v1</main>")
	second := boundaryUIArchive(t, "<main>Refund v2</main>")
	imported := runBoundaryCommand(t, "ui", "import", "--root", root, "--archive", first, "--id", "ui.external.refund", "--authority", "canonical", "--version", "design-1", "--ref", "ui.refund", "--consumer", "workspace.frontend", "--apply", "--json")
	require.Contains(t, imported, `"status":"passed"`)

	reconciled := runBoundaryCommand(t, "ui", "reconcile", "--root", root, "--id", "ui.external.refund", "--archive", second, "--version", "design-2", "--apply", "--json")
	require.Contains(t, reconciled, "mapped canonical UI remains stale")
	snapshot, issues := contextpkg.Refresh(context.Background(), root, contextpkg.ReadOnly)
	require.Empty(t, errorsOnlyForCommand(issues))
	for _, id := range []string{"ui.external.refund", "ui.refund", "workspace.frontend"} {
		require.Contains(t, snapshot.Stale, id)
	}
}

func runBoundaryCommand(t *testing.T, args ...string) string {
	t.Helper()
	var output, errors bytes.Buffer
	cmd := command.New("1.0.0", &output, &errors)
	cmd.SetArgs(args)
	require.NoError(t, cmd.Execute(), errors.String())
	return output.String()
}

func boundaryUIArchive(t *testing.T, html string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mockup.zip")
	file, err := os.Create(path)
	require.NoError(t, err)
	writer := zip.NewWriter(file)
	for _, item := range [][2]string{{"LICENSE", "MIT"}, {"screens/refund.html", html}} {
		entry, err := writer.Create(item[0])
		require.NoError(t, err)
		_, err = entry.Write([]byte(item[1]))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())
	return path
}

func boundaryRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func errorsOnlyForCommand(items []domain.Item) []domain.Item {
	result := []domain.Item{}
	for _, item := range items {
		if len(item.Code) >= len("context.error") && item.Code[:len("context.error")] == "context.error" {
			result = append(result, item)
		}
	}
	return result
}
