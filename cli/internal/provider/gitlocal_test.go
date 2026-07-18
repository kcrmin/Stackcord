package provider

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGitLocalCompareAndSwapAllowsOneConcurrentClaim(t *testing.T) {
	remote, left, right := newSharedRemote(t)
	leftStore := NewGitLocalStore(left, remote, "coordination")
	rightStore := NewGitLocalStore(right, remote, "coordination")
	base, err := leftStore.CompareAndSwap(context.Background(), "", SnapshotSet{SchemaVersion: 1, Claims: []GitLocalClaim{}})
	require.NoError(t, err)

	leftHead, leftStatus := gitLocalRun(t, left, "rev-parse", "HEAD"), gitLocalRun(t, left, "status", "--porcelain=v2", "--untracked-files=all")
	rightHead, rightStatus := gitLocalRun(t, right, "rev-parse", "HEAD"), gitLocalRun(t, right, "status", "--porcelain=v2", "--untracked-files=all")
	type outcome struct {
		revision string
		err      error
	}
	results := make(chan outcome, 2)
	go func() {
		revision, compareErr := leftStore.CompareAndSwap(context.Background(), base, claimedBy("left"))
		results <- outcome{revision: revision, err: compareErr}
	}()
	go func() {
		revision, compareErr := rightStore.CompareAndSwap(context.Background(), base, claimedBy("right"))
		results <- outcome{revision: revision, err: compareErr}
	}()

	first, second := <-results, <-results
	outcomes := []outcome{first, second}
	successes, conflicts := 0, 0
	for _, result := range outcomes {
		if result.err == nil {
			successes++
			require.NotEmpty(t, result.revision)
		} else if errors.Is(result.err, ErrCASConflict) {
			conflicts++
		} else {
			var typed *GitLocalError
			if errors.As(result.err, &typed) {
				t.Logf("unexpected Git-local failure: kind=%v operation=%s", typed.Kind, typed.Operation)
			} else {
				t.Logf("unexpected Git-local failure: %T", result.err)
			}
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)

	observed, err := leftStore.Read(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, observed.Revision)
	require.Len(t, observed.Claims, 1)
	require.Contains(t, []string{"left", "right"}, observed.Claims[0].Owner)
	require.Equal(t, leftHead, gitLocalRun(t, left, "rev-parse", "HEAD"))
	require.Equal(t, leftStatus, gitLocalRun(t, left, "status", "--porcelain=v2", "--untracked-files=all"))
	require.Equal(t, rightHead, gitLocalRun(t, right, "rev-parse", "HEAD"))
	require.Equal(t, rightStatus, gitLocalRun(t, right, "status", "--porcelain=v2", "--untracked-files=all"))
}

func TestGitLocalPushPolicyRejectionIsNotReportedAsClaimRace(t *testing.T) {
	remote, left, _ := newSharedRemote(t)
	store := NewGitLocalStore(left, remote, "coordination")
	base, err := store.CompareAndSwap(context.Background(), "", SnapshotSet{SchemaVersion: 1, Claims: []GitLocalClaim{}})
	require.NoError(t, err)
	hook := filepath.Join(remote, "hooks", "pre-receive")
	require.NoError(t, os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o700))

	_, err = store.CompareAndSwap(context.Background(), base, claimedBy("left"))

	require.ErrorIs(t, err, ErrPushRejected)
	require.False(t, errors.Is(err, ErrCASConflict), "server policy rejection is not a concurrent writer")
}

func claimedBy(owner string) SnapshotSet {
	return SnapshotSet{SchemaVersion: 1, Claims: []GitLocalClaim{{
		ID: "claim.account-recovery", WorkID: "work.account-recovery", DefinitionFingerprint: "sha256:" + strings.Repeat("a", 64),
		Owner: owner, Branch: "feature/account-recovery", Repository: "repository.root", Paths: []string{"services/account"},
		PolicyIDs: []string{"policy.account-recovery"}, ScenarioIDs: []string{"scenario.account-recovery"}, ContractIDs: []string{"contract.account-recovery"},
		DBEntities: []string{"account_recovery"}, MigrationSlots: []string{"20260718-account-recovery"}, UIFlows: []string{"flow.account-recovery"},
		DependencyMajors: []string{}, StableIDs: []string{"feature.account-recovery"}, RootPointer: false,
		StartsAt: time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC), ExpiresAt: time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC),
	}}}
}

func newSharedRemote(t *testing.T) (string, string, string) {
	t.Helper()
	base := t.TempDir()
	remote, seed := filepath.Join(base, "remote.git"), filepath.Join(base, "seed")
	gitLocalRun(t, "", "init", "--bare", "--initial-branch=main", remote)
	gitLocalRun(t, "", "init", "--initial-branch=main", seed)
	configureGitLocal(t, seed)
	require.NoError(t, os.WriteFile(filepath.Join(seed, "README.md"), []byte("fixture\n"), 0o600))
	gitLocalRun(t, seed, "add", "README.md")
	gitLocalRun(t, seed, "commit", "-m", "chore: initialize fixture")
	gitLocalRun(t, seed, "remote", "add", "origin", remote)
	gitLocalRun(t, seed, "push", "-u", "origin", "main")
	left, right := filepath.Join(base, "left"), filepath.Join(base, "right")
	gitLocalRun(t, "", "clone", remote, left)
	gitLocalRun(t, "", "clone", remote, right)
	configureGitLocal(t, left)
	configureGitLocal(t, right)
	return remote, left, right
}

func configureGitLocal(t *testing.T, root string) {
	t.Helper()
	gitLocalRun(t, root, "config", "user.email", "fixture@example.invalid")
	gitLocalRun(t, root, "config", "user.name", "Fixture User")
}

func gitLocalRun(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "git %s failed: %s", strings.Join(args, " "), output)
	return strings.TrimSpace(string(output))
}
