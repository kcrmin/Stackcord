package database_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/database"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/stretchr/testify/require"
)

func TestDBDiagramPullCannotOverwriteCanonicalDBML(t *testing.T) {
	root := t.TempDir()
	canonical := []byte("Table accounts {\n id int [pk]\n recovery_state varchar\n}\n")
	candidate := []byte("Table accounts {\n id int [pk]\n recovery_state int [not null]\n}\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "schema.dbml"), canonical, 0o600))
	request := database.ProposalRequest{
		Root: root, OperationID: "refund-review", Entry: "schema.dbml", Candidate: candidate,
		Tool: "dbdiagram", ToolVersion: "1.4.2", ProjectID: "diagram-1", Action: "pull", FetchedAt: time.Date(2026, 7, 18, 3, 0, 0, 0, time.UTC),
		ContractIDs: []string{"contract.data.accounts"}, MigrationIDs: []string{"migration.accounts.recovery-state"}, TestIDs: []string{"test.accounts.migration"}, RollbackIDs: []string{"rollback.accounts.recovery-state"},
	}

	proposal, plan, err := database.PrepareProposal(request)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)
	require.FileExists(t, proposal.CandidatePath)
	actual, err := os.ReadFile(filepath.Join(root, "schema.dbml"))
	require.NoError(t, err)
	require.Equal(t, canonical, actual)
	require.Contains(t, proposal.Diff.ChangedColumns, "accounts.recovery_state")
	require.Contains(t, proposal.Impact.Entities, "accounts")
	require.Contains(t, proposal.Impact.Contracts, "contract.data.accounts")
}

func TestReconcileProposalRejectsStaleCanonicalBase(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "schema.dbml"), []byte("Table accounts { id int [pk] }\n"), 0o600))
	proposal, plan, err := database.PrepareProposal(database.ProposalRequest{Root: root, OperationID: "stale-base", Entry: "schema.dbml", Candidate: []byte("Table accounts { id bigint [pk] }\n"), Tool: "dbdiagram", ToolVersion: "1.4.2", ProjectID: "diagram-1", Action: "pull", FetchedAt: time.Now().UTC()})
	require.NoError(t, err)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)
	require.NoError(t, os.WriteFile(filepath.Join(root, "schema.dbml"), []byte("Table accounts { id uuid [pk] }\n"), 0o600))

	_, reconcile, issues, err := database.ReconcileProposal(database.ReconcileRequest{Root: root, ProposalPath: proposal.RecordPath})
	require.NoError(t, err)
	require.Empty(t, reconcile.Files)
	require.Contains(t, databaseIssueCodes(issues), "db.proposal-stale-base")
}

func TestReconcileProposalRequiresImpactEvidenceBeforeCanonicalApply(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "schema.dbml"), []byte("Table accounts { id int [pk] }\n"), 0o600))
	proposal, plan, err := database.PrepareProposal(database.ProposalRequest{
		Root: root, OperationID: "missing-impact", Entry: "schema.dbml", Candidate: []byte("Table accounts { id bigint [pk] }\n"),
		Tool: "dbdiagram", ToolVersion: "1.4.2", ProjectID: "diagram-1", Action: "pull", FetchedAt: time.Now().UTC(),
	})
	require.NoError(t, err)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)

	_, reconcile, issues, err := database.ReconcileProposal(database.ReconcileRequest{Root: root, ProposalPath: proposal.RecordPath})
	require.NoError(t, err)
	require.Empty(t, reconcile.Files)
	codes := databaseIssueCodes(issues)
	require.Contains(t, codes, "db.impact-contract-missing")
	require.Contains(t, codes, "db.impact-migration-missing")
	require.Contains(t, codes, "db.impact-test-missing")
	require.Contains(t, codes, "db.impact-rollback-missing")
}

func databaseIssueCodes(items []domain.Item) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Code)
	}
	return result
}
