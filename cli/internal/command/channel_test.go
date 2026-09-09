package command

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/channel"
	"github.com/stretchr/testify/require"
)

func TestChannelCodeConfigurationPreview(t *testing.T) {
	cmd := newChannelCommand()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetArgs([]string{"send", "--to", "peer", "--kind", "implementation", "--title", "work", "--scope", "src", "--code-repository", "project", "--base-commit", strings.Repeat("a", 40), "--resource", "contract.api"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "base_commit")
	require.Contains(t, out.String(), "contract.api")
	cmd = newChannelCommand()
	out.Reset()
	cmd.SetOut(out)
	exe, _ := os.Executable()
	argv, _ := json.Marshal([]string{exe})
	cmd.SetArgs([]string{"runner", "--argv", string(argv), "--kind", "implementation", "--code-repository", "project", "--code-remote", "https://example.invalid/code.git", "--verify-argv", string(argv)})
	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "code_repository")
}

func TestChannelActiveRunnerCannotBlockWaitingOrNestWorker(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(t.TempDir(), "mail.git")
	data, err := exec.Command("git", "init", "--bare", remote).CombinedOutput()
	require.NoError(t, err, string(data))
	s, err := channel.Open(root)
	require.NoError(t, err)
	_, err = s.Setup(channel.Config{Channel: "test", Peer: "self", Remote: remote})
	require.NoError(t, err)
	r, err := s.Send(context.Background(), channel.RequestInput{To: "self", Kind: "review", Title: "pending"})
	require.NoError(t, err)
	t.Setenv("STACKCORD_ACTIVE_REQUEST", r.ID)
	t.Setenv("STACKCORD_CHANNEL_ROOT", root)
	// An active runner must reject pending waits even while the remote is offline.
	require.NoError(t, os.Rename(remote, remote+".offline"))
	cmd := newChannelCommand()
	cmd.SetArgs([]string{"wait", "--root", root, "--request", r.ID, "--timeout", "30s", "--interval", "1s"})
	err = cmd.Execute()
	require.ErrorContains(t, err, "active runner cannot wait")
	cmd = newChannelCommand()
	cmd.SetArgs([]string{"worker", "--apply", "--once"})
	require.ErrorContains(t, cmd.Execute(), "active runner cannot start another worker")
	require.NoError(t, os.Rename(remote+".offline", remote))
	_, err = s.Respond(context.Background(), channel.ResultInput{RequestID: r.ID, Status: "success", Body: "completed"})
	require.NoError(t, err)
	require.NoError(t, os.Rename(remote, remote+".offline"))
	cmd = newChannelCommand()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetArgs([]string{"wait", "--request", r.ID, "--timeout", "30s"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "completed")
}

func TestChannelSetupPreviewDoesNotEnrollOrContactRemote(t *testing.T) {
	root := t.TempDir()
	out := &bytes.Buffer{}
	cmd := newChannelCommand()
	cmd.SetOut(out)
	cmd.SetArgs([]string{"setup", "--root", root, "--channel", "demo", "--peer", "alice", "--remote", "https://example.invalid/project.git"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "preview")
	_, err := os.Stat(filepath.Join(root, ".harness"))
	require.True(t, os.IsNotExist(err), "a preview must not create enrollment state")
}

func TestChannelCLIProcessesDependenciesAcrossIndependentPeers(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "shared.git")
	output, err := exec.Command("git", "init", "--bare", remote).CombinedOutput()
	require.NoError(t, err, string(output))
	alice, bob := filepath.Join(base, "alice"), filepath.Join(base, "bob")
	require.NoError(t, os.MkdirAll(alice, 0700))
	require.NoError(t, os.MkdirAll(bob, 0700))
	call := func(root string, args ...string) map[string]any {
		t.Helper()
		cmd := newChannelCommand()
		out := &bytes.Buffer{}
		cmd.SetOut(out)
		cmd.SetArgs(append([]string{"--root", root}, args...))
		require.NoError(t, cmd.Execute(), out.String())
		var v map[string]any
		require.NoError(t, json.Unmarshal(out.Bytes(), &v), out.String())
		return v
	}
	a := call(alice, "setup", "--channel", "service", "--peer", "alice", "--remote", remote, "--apply")
	b := call(bob, "setup", "--channel", "service", "--peer", "bob", "--remote", remote, "--apply")
	call(alice, "trust", "--peer", "bob", "--public-key", b["public_key"].(string), "--apply")
	call(bob, "trust", "--peer", "alice", "--public-key", a["public_key"].(string), "--apply")
	binary, err := os.Executable()
	require.NoError(t, err)
	argv, _ := json.Marshal([]string{binary, "-test.run=^TestChannelRunnerHelper$", "--", "channel-helper"})
	call(bob, "runner", "--argv", string(argv), "--kind", "implementation", "--apply")
	first := call(alice, "send", "--to", "bob", "--kind", "implementation", "--title", "Prepare contract", "--body", "First task", "--apply")
	second := call(alice, "send", "--to", "bob", "--kind", "implementation", "--title", "Use contract", "--depends-on", first["id"].(string), "--apply")
	call(bob, "worker", "--once", "--apply")
	call(bob, "worker", "--once", "--apply")
	completed := call(alice, "wait", "--request", second["id"].(string), "--timeout", "5s", "--interval", "1s")
	result := completed["result"].(map[string]any)
	require.Equal(t, "success", result["status"])
	require.Contains(t, result["body"], "Use contract")
	state := call(bob, "status")
	require.Len(t, state["requests"], 2)
	require.NotContains(t, fmt.Sprint(state), binary, "runner arguments must not leak into public state")
}

func TestChannelRunnerHelper(t *testing.T) {
	if len(os.Args) == 0 || os.Args[len(os.Args)-1] != "channel-helper" {
		return
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(3)
	}
	var request map[string]any
	if json.Unmarshal(data, &request) != nil {
		os.Exit(4)
	}
	fmt.Printf("Completed: %s", request["title"])
	os.Exit(0)
}

func TestChannelWorkerLoopStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	require.NoError(t, runChannelWorker(ctx, time.Millisecond, false, func(context.Context) error { calls++; cancel(); return nil }))
	require.Equal(t, 1, calls)
}

func TestChannelWorkerDrainsReadyQueueWithoutPollingDelay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	calls := 0
	err := runChannelWorker(ctx, time.Hour, false, func(context.Context) error {
		calls++
		if calls == 1 {
			return errChannelMoreReady
		}
		cancel()
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, calls, "ready work must not wait for the idle poll interval")
}

func TestChannelBodyAllowsEmptyFileAndRejectsOversize(t *testing.T) {
	p := filepath.Join(t.TempDir(), "body.txt")
	require.NoError(t, os.WriteFile(p, nil, 0600))
	body, err := readChannelBody(p)
	require.NoError(t, err)
	require.Empty(t, body)
	require.NoError(t, os.WriteFile(p, make([]byte, 64*1024+1), 0600))
	_, err = readChannelBody(p)
	require.Error(t, err)
}

func TestChannelWorkerRequiresExplicitApply(t *testing.T) {
	cmd := newChannelCommand()
	cmd.SetArgs([]string{"worker", "--root", t.TempDir(), "--once"})
	require.ErrorContains(t, cmd.Execute(), "--apply")
}

func TestChannelRunnerAcceptsArgumentArrayNotShellString(t *testing.T) {
	cmd := newChannelCommand()
	cmd.SetArgs([]string{"runner", "--root", t.TempDir(), "--argv", "echo unsafe", "--kind", "implementation"})
	require.ErrorContains(t, cmd.Execute(), "JSON")
}

func TestChannelHostPresetsKeepLocalPermissionBoundaries(t *testing.T) {
	codex, err := channelHostArgs("codex")
	require.NoError(t, err)
	require.Contains(t, codex, "workspace-write")
	require.Contains(t, codex, "never")
	require.NotContains(t, codex, "--dangerously-bypass-approvals-and-sandbox")
	claude, err := channelHostArgs("claude")
	require.NoError(t, err)
	require.Contains(t, claude, "dontAsk")
	require.NotContains(t, claude, "--dangerously-skip-permissions")
	_, err = channelHostArgs("remote-command")
	require.Error(t, err)
}

func TestChannelHostPresetApplyResolvesInstalledExecutable(t *testing.T) {
	root := t.TempDir()
	bin := t.TempDir()
	name := "codex"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	source, err := os.Executable()
	require.NoError(t, err)
	data, err := os.ReadFile(source)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(bin, name), data, 0700))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	s, err := channel.Open(root)
	require.NoError(t, err)
	_, err = s.Setup(channel.Config{Channel: "preset", Peer: "alice", Remote: filepath.Join(root, "remote.git")})
	require.NoError(t, err)
	cmd := newChannelCommand()
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"runner", "--root", root, "--host", "codex", "--kind", "implementation", "--apply"})
	require.NoError(t, cmd.Execute())
	state, err := s.State(context.Background(), false)
	require.NoError(t, err)
	require.True(t, state.Runner.Enabled)
	require.Equal(t, "json", state.Runner.ResultFormat)
}

func TestChannelWorkerRejectsUnboundedPollingConfiguration(t *testing.T) {
	cmd := newChannelCommand()
	cmd.SetArgs([]string{"worker", "--root", t.TempDir(), "--apply", "--interval", "0s"})
	require.ErrorContains(t, cmd.Execute(), "interval")
}
