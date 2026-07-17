package database_test

import (
	"os"
	"path/filepath"
	"testing"

	"fullstack-orchestrator/cli/internal/database"
	"github.com/stretchr/testify/require"
)

func TestSyncPlanUsesOfficialCLIWithIsolatedCanonicalCopyAndEnvironmentSecret(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "schema.dbml")
	require.NoError(t, os.WriteFile(entry, []byte("Table users { id int [pk] }\n"), 0o600))
	plan, err := database.SyncPlan(database.DBDiagramConfig{Root: root, OperationID: "01JDB", Action: "push", Entry: "schema.dbml", ProjectID: "project-123", TokenEnvironment: "DBDIAGRAM_TOKEN"})
	require.NoError(t, err)
	require.Len(t, plan.Files, 1)
	require.Equal(t, filepath.ToSlash(filepath.Join(".harness", "local", "dbdiagram", "01JDB", "candidate.dbml")), plan.Files[0].Path)
	require.Equal(t, []byte("Table users { id int [pk] }\n"), plan.Files[0].Content)
	require.Len(t, plan.Commands, 2)
	require.Equal(t, filepath.Join(root, ".harness", "local", "dbdiagram", "01JDB"), plan.Commands[0].Directory)
	require.Equal(t, "dbdiagram", plan.Commands[0].Program)
	require.Equal(t, []string{"init", "--entry", "candidate.dbml", "--diagram-id", "project-123"}, plan.Commands[0].Args)
	require.Equal(t, []string{"push"}, plan.Commands[1].Args)
	for _, command := range plan.Commands {
		for _, argument := range command.Args {
			require.NotContains(t, argument, "DBDIAGRAM_TOKEN")
			require.NotContains(t, argument, "token")
		}
	}
}

func TestSyncPlanRejectsEntryOutsideProjectAndUnknownAction(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "schema.dbml")
	require.NoError(t, os.WriteFile(outside, []byte("Table users { id int [pk] }\n"), 0o600))

	_, err := database.SyncPlan(database.DBDiagramConfig{Root: root, OperationID: "01JDB", Action: "pull", Entry: outside, ProjectID: "project-123", TokenEnvironment: "DBDIAGRAM_TOKEN"})
	require.ErrorContains(t, err, "inside the project root")
	_, err = database.SyncPlan(database.DBDiagramConfig{Root: root, OperationID: "01JDB", Action: "delete", Entry: "schema.dbml", ProjectID: "project-123", TokenEnvironment: "DBDIAGRAM_TOKEN"})
	require.ErrorContains(t, err, "push or pull")
}

func TestSemanticDiffExplainsDatabaseChanges(t *testing.T) {
	before := "Table users {\n id int [pk]\n}\n"
	after := "Table users {\n id int [pk]\n email varchar [not null]\n}\nTable sessions {\n id int [pk]\n user_id int\n}\nRef: sessions.user_id > users.id\n"
	diff, err := database.SemanticDiff([]byte(before), []byte(after))
	require.NoError(t, err)
	require.Contains(t, diff.AddedTables, "sessions")
	require.Contains(t, diff.AddedColumns, "users.email")
	require.NotEmpty(t, diff.AddedRelations)
}

func TestSemanticDiffIncludesIndexesAndNotesWithoutTreatingThemAsColumns(t *testing.T) {
	before := "Table users {\n id int [pk]\n indexes {\n  (id) [name: 'idx_users_id']\n }\n Note: 'original'\n}\n"
	after := "Table users {\n id int [pk]\n email varchar\n indexes {\n  (email) [name: 'idx_users_email']\n }\n Note: 'updated'\n}\n"

	diff, err := database.SemanticDiff([]byte(before), []byte(after))
	require.NoError(t, err)
	require.Equal(t, []string{"users.(email) [name: 'idx_users_email']"}, diff.AddedIndexes)
	require.Equal(t, []string{"users.(id) [name: 'idx_users_id']"}, diff.RemovedIndexes)
	require.Equal(t, []string{"users.'updated'"}, diff.AddedNotes)
	require.Equal(t, []string{"users.'original'"}, diff.RemovedNotes)
	require.NotContains(t, diff.AddedColumns, "users.indexes")
	require.NotContains(t, diff.RemovedColumns, "users.indexes")
}

func TestSemanticDiffDetectsColumnSemanticsAndInlineRelations(t *testing.T) {
	before := "Table sessions {\n user_id int [ref: > users.id]\n expires_at timestamp\n}\nTable users {\n id int [pk]\n}\n"
	after := "Table sessions {\n user_id bigint [ref: > accounts.id]\n expires_at timestamp [not null]\n}\nTable accounts {\n id bigint [pk]\n}\n"

	diff, err := database.SemanticDiff([]byte(before), []byte(after))
	require.NoError(t, err)
	require.Equal(t, []string{"sessions.expires_at", "sessions.user_id"}, diff.ChangedColumns)
	require.Contains(t, diff.RemovedRelations, "sessions.user_id > users.id")
	require.Contains(t, diff.AddedRelations, "sessions.user_id > accounts.id")
}
