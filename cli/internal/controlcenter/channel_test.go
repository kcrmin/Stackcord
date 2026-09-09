package controlcenter

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/channel"
	"github.com/stretchr/testify/require"
)

func TestChannelPreviewDoesNotEnrollAndStaleApplyCannotTrustPeer(t *testing.T) {
	root := t.TempDir()
	b := &Backend{Root: root}
	ctx := context.Background()
	s, err := channel.Open(root)
	require.NoError(t, err)
	state, err := s.State(ctx, false)
	require.NoError(t, err)
	payload, _ := json.Marshal(map[string]any{"channel": "team", "remote": "https://example.invalid/project.git", "peer": "alice", "expected_revision": state.Revision, "apply": false})
	result, err := b.Action(ctx, "channel.setup", payload)
	require.NoError(t, err)
	require.NotNil(t, result)
	_, err = os.Stat(filepath.Join(root, ".harness"))
	require.True(t, os.IsNotExist(err))
	payload, _ = json.Marshal(map[string]any{"peer": "mallory", "public_key": "invalid", "expected_revision": "stale", "apply": true})
	_, err = b.Action(ctx, "channel.trust", payload)
	require.ErrorContains(t, err, "changed")
}

func TestChannelUIRejectsRunnerExecutionEndpoint(t *testing.T) {
	b := &Backend{Root: t.TempDir()}
	_, err := b.Action(context.Background(), "channel.worker", json.RawMessage(`{"apply":true,"argv":["untrusted"]}`))
	require.Error(t, err)
}
