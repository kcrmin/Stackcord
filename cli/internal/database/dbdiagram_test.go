package database_test

import (
	"path/filepath"
	"testing"

	"fullstack-orchestrator/cli/internal/database"
	"github.com/stretchr/testify/require"
)

func TestPullPlanUsesIsolatedScratchAndEnvironmentSecret(t *testing.T) {
	root := t.TempDir()
	plan, err := database.PullPlan(database.DBDiagramConfig{Root: root, OperationID: "01JDB", Executable: "db2", ProjectID: "project-123", TokenEnvironment: "DBDIAGRAM_TOKEN"})
	require.NoError(t, err)
	require.Len(t, plan.Commands, 1)
	require.Equal(t, filepath.Join(root, ".harness", "local", "dbdiagram", "01JDB"), plan.Commands[0].Directory)
	require.NotContains(t, plan.Commands[0].Args, "DBDIAGRAM_TOKEN")
	for _, argument := range plan.Commands[0].Args {
		require.NotContains(t, argument, "token")
	}
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
