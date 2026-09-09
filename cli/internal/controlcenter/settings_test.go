package controlcenter

import (
	"context"
	"github.com/kcrmin/Stackcord/cli/internal/governance"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestSetupRejectsStaleHostWrite(t *testing.T) {
	root := t.TempDir()
	s, rev, e := ReadSettings(root)
	require.NoError(t, e)
	s.UIPreference = "enabled"
	_, e = SavePersonal(context.Background(), root, rev, s)
	require.NoError(t, e)
	s.UIPreference = "declined"
	_, e = SavePersonal(context.Background(), root, rev, s)
	require.Error(t, e)
	current, _, e := ReadSettings(root)
	require.NoError(t, e)
	require.Equal(t, "enabled", current.UIPreference)
}
func TestSettingsRejectUnsafeDefaults(t *testing.T) {
	s := Settings{Language: "en", SecurityMode: "weak", Repository: "acme/app", BranchPattern: "codex/{description}", CommitConvention: "{type}: {description}", UIPreference: "enabled"}
	require.Error(t, validateSettings(s))
	s.BranchPattern = "{type}/{description}"
	s.SecurityMode = "medium"
	s.Admins = []string{}
	require.Error(t, validateSettings(s))
}

func TestProposalPreservesExistingQuorum(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness/governance.yaml"), []byte("schema_version: 2\nmode: strong\nenabled: true\nprovider: github\nrepository: acme/app\nproduct_authorities: [user:alice, user:bob]\nprotected_kinds: [policy]\napproval: {minimum: 2, authority_self_approval: false}\n"), 0600))
	s, rev, e := ReadSettings(root)
	require.NoError(t, e)
	s.Language = "ko"
	_, e = SaveProposal(context.Background(), root, rev, s)
	require.NoError(t, e)
	p, e := governance.LoadPolicy(root)
	require.NoError(t, e)
	require.Equal(t, 2, p.Approval.Minimum)
	require.Equal(t, []string{"policy"}, p.ProtectedKinds)
}

func TestConcurrentHostsCannotOverwriteEachOther(t *testing.T) {
	root := t.TempDir()
	first, rev, e := ReadSettings(root)
	require.NoError(t, e)
	second := first
	first.UIPreference = "enabled"
	second.UIPreference = "declined"
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, s := range []Settings{first, second} {
		go func(s Settings) { <-start; _, err := SavePersonal(context.Background(), root, rev, s); results <- err }(s)
	}
	close(start)
	failures := 0
	for i := 0; i < 2; i++ {
		if <-results != nil {
			failures++
		}
	}
	require.Equal(t, 1, failures)
	saved, _, e := ReadSettings(root)
	require.NoError(t, e)
	require.Contains(t, []string{"enabled", "declined"}, saved.UIPreference)
}
