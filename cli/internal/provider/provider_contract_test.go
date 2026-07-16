package provider_test

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/policy"
	"fullstack-orchestrator/cli/internal/provider"
	dbadapter "fullstack-orchestrator/cli/internal/provider/dbdiagram"
	gitadapter "fullstack-orchestrator/cli/internal/provider/gitgeneric"
	ghadapter "fullstack-orchestrator/cli/internal/provider/github"
	"github.com/stretchr/testify/require"
)

func TestProviderContract(t *testing.T) {
	adapters := []provider.Adapter{
		gitadapter.New(gitadapter.Config{}),
		ghadapter.New(ghadapter.Config{}),
		dbadapter.New(dbadapter.Config{}),
	}
	for _, adapter := range adapters {
		t.Run(adapter.Descriptor().ID, func(t *testing.T) {
			health := adapter.Health(context.Background())
			require.NotEmpty(t, health.Status)
			capabilities := adapter.Capabilities(context.Background())
			require.NotNil(t, capabilities)

			unsupported := adapter.Plan(context.Background(), provider.Request{OperationID: "unsupported", Capability: provider.CapabilityHierarchy})
			if !provider.HasCapability(capabilities, provider.CapabilityHierarchy) {
				require.NotEmpty(t, unsupported.Blockers)
				require.Equal(t, "provider.capability-unsupported", unsupported.Blockers[0].Code)
			}

			write := provider.Request{OperationID: "write-1", Objective: "update task", Repository: "/project", Target: "task-1", Capability: provider.CapabilityWrite}
			result := adapter.Execute(context.Background(), write, policy.Consent{})
			if provider.HasCapability(capabilities, provider.CapabilityWrite) {
				require.Equal(t, domain.StatusApprovalRequired, result.Status)
			}

			items, err := adapter.Normalize([]byte(`{"message":"token=super-secret-value password=also-secret"}`))
			require.NoError(t, err)
			for _, item := range items {
				require.NotContains(t, item.Message, "super-secret-value")
				require.NotContains(t, item.Message, "also-secret")
			}
		})
	}
}

func TestProviderReceiptSurvivesAdapterRestart(t *testing.T) {
	store, err := provider.NewFileReceiptStore(t.TempDir())
	require.NoError(t, err)
	var executions int32
	config := provider.GuardedConfig{
		Descriptor: provider.Descriptor{ID: "fixture", Name: "Fixture", Version: "1"}, Capabilities: []provider.Capability{provider.CapabilityWrite},
		Health: provider.Health{Status: "ready"}, ReceiptStore: store,
		Execute: func(context.Context, provider.Request) error { atomic.AddInt32(&executions, 1); return nil },
	}
	request := provider.Request{OperationID: "01JPERSIST", Objective: "update issue", Repository: "/project", Target: "GH-12", Capability: provider.CapabilityWrite, Payload: map[string]any{"title": "safe"}}
	consent := policy.Consent{Approved: true, Action: policy.PushBranch, Objective: request.Objective, Repository: request.Repository, Target: request.Target, ExpiresAt: time.Now().UTC().Add(time.Hour)}

	require.Equal(t, domain.StatusPassed, provider.NewGuarded(config).Execute(context.Background(), request, consent).Status)
	restarted := provider.NewGuarded(config).Execute(context.Background(), request, consent)

	require.Equal(t, domain.StatusPassed, restarted.Status)
	require.Contains(t, restarted.Summary, "already")
	require.Equal(t, int32(1), atomic.LoadInt32(&executions))
}

func TestFileReceiptStoreRejectsSymlinkReceipt(t *testing.T) {
	root := t.TempDir()
	store, err := provider.NewFileReceiptStore(root)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "fixture"), 0o700))
	target := filepath.Join(t.TempDir(), "outside.json")
	require.NoError(t, os.WriteFile(target, []byte(`{"operation_id":"01JLINK","provider":"fixture","target":"x","fingerprint":"sha256:abc","completed_at":"2026-07-16T00:00:00Z"}`), 0o600))
	if err := os.Symlink(target, filepath.Join(root, "fixture", "01JLINK.json")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	_, _, err = store.Load("fixture", "01JLINK")
	require.ErrorContains(t, err, "symlink")
}

func TestProviderExecutionIsIdempotentAndScoped(t *testing.T) {
	adapter := ghadapter.New(ghadapter.Config{Authenticated: true, Execute: func(context.Context, provider.Request) error { return nil }})
	request := provider.Request{OperationID: "01JWRITE", Objective: "update issue", Repository: "/project", Target: "GH-12", Capability: provider.CapabilityWrite}
	now := time.Now().UTC()
	consent := policy.Consent{Approved: true, Action: policy.PushBranch, Objective: request.Objective, Repository: request.Repository, Target: request.Target, ExpiresAt: now.Add(time.Hour)}

	first := adapter.Execute(context.Background(), request, consent)
	require.Equal(t, domain.StatusPassed, first.Status)
	second := adapter.Execute(context.Background(), request, consent)
	require.Equal(t, domain.StatusPassed, second.Status)
	require.Contains(t, second.Summary, "already")
	_, exists := adapter.Receipt(request.OperationID)
	require.True(t, exists)

	request.Target = "GH-13"
	require.Equal(t, domain.StatusApprovalRequired, adapter.Execute(context.Background(), request, consent).Status)
}

func TestProviderRejectsOperationIDReuseForDifferentPayload(t *testing.T) {
	adapter := ghadapter.New(ghadapter.Config{Authenticated: true, Execute: func(context.Context, provider.Request) error { return nil }})
	request := provider.Request{OperationID: "01JREUSE", Objective: "update issue", Repository: "/project", Target: "GH-12", Capability: provider.CapabilityWrite, Payload: map[string]any{"title": "first"}}
	consent := policy.Consent{Approved: true, Action: policy.PushBranch, Objective: request.Objective, Repository: request.Repository, Target: request.Target, ExpiresAt: time.Now().UTC().Add(time.Hour)}
	require.Equal(t, domain.StatusPassed, adapter.Execute(context.Background(), request, consent).Status)

	request.Payload["title"] = "different"
	result := adapter.Execute(context.Background(), request, consent)

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Equal(t, domain.ExitBlocked, result.ExitCode)
	require.Equal(t, "provider.operation-id-reused", result.Blockers[0].Code)
}

func TestRegistryUsesExplicitThenDetectedThenLocal(t *testing.T) {
	local := gitadapter.New(gitadapter.Config{})
	github := ghadapter.New(ghadapter.Config{Authenticated: true})
	registry := provider.NewRegistry(local, github)
	require.Equal(t, "github", registry.Select("github", "").Descriptor().ID)
	require.Equal(t, "github", registry.Select("", "github").Descriptor().ID)
	require.Equal(t, "git-generic", registry.Select("", "").Descriptor().ID)
}
