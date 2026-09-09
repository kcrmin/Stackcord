package command

import (
	"bytes"
	"context"
	"errors"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	gh "github.com/kcrmin/Stackcord/cli/internal/github"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/provider"
	"github.com/kcrmin/Stackcord/cli/internal/work"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIssueMigrationPreservesDefinitionsAndRejectsStaleApply(t *testing.T) {
	root := t.TempDir()
	definitionPath := filepath.Join(root, "specs", "meaning.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(definitionPath), 0700))
	require.NoError(t, os.WriteFile(definitionPath, []byte("approved meaning"), 0600))
	def := work.Definition{ID: "work.test", Readiness: work.Ready, Fingerprint: "sha256:" + strings.Repeat("a", 64)}
	replies := []string{`{"login":"alice","id":1}`, `{"full_name":"org/repo"}`, `{"number":7,"state":"open","html_url":"https://github.com/org/repo/issues/7"}`, `[[]]`}
	plan, err := githubMappingPlan(context.Background(), root, "org/repo", []work.Definition{def}, map[string]int{def.ID: 7}, &issueTransport{replies: replies})
	require.NoError(t, err)
	require.Len(t, plan.Files, 3)
	require.NoFileExists(t, filepath.Join(root, ".harness", "work", "provider.yaml"))
	result := operation.Apply(context.Background(), plan)
	require.Equal(t, domain.StatusPassed, result.Status, result)
	data, err := os.ReadFile(definitionPath)
	require.NoError(t, err)
	require.Equal(t, "approved meaning", string(data))
	selected, err := loadTaskProvider(root)
	require.NoError(t, err)
	require.Equal(t, "github", selected.LiveStatusSource)
	for _, file := range plan.Files {
		require.NotContains(t, file.Path, "definitions")
	}
	plan.ID += "-stale"
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "work", "provider.yaml"), []byte("changed"), 0600))
	result = operation.Apply(context.Background(), plan)
	require.Equal(t, domain.StatusBlocked, result.Status)
}

func TestGitHubLifecycleDoesNotTrustCachedConnectorMarker(t *testing.T) {
	root := t.TempDir()
	def := work.Definition{ID: "work.test", Fingerprint: "sha256:" + strings.Repeat("a", 64)}
	mapping := provider.Mapping{SchemaVersion: 1, WorkID: def.ID, DefinitionFingerprint: def.Fingerprint, Provider: "github", ItemID: "42", DependencyItems: map[string]string{}}
	snapshot := provider.Snapshot{SchemaVersion: 1, Provider: "github", ItemID: "42", DefinitionFingerprint: def.Fingerprint, Status: "done", Source: "connector-live", FetchedAt: time.Now(), RawHash: "sha256:" + strings.Repeat("b", 64), Capabilities: provider.Capabilities{Claim: "none"}}
	for path, value := range map[string]any{".harness/work/mappings/work.test.yaml": mapping, ".harness/local/providers/github/work.test.yaml": snapshot} {
		absolute := filepath.Join(root, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(absolute), 0700))
		data, err := yaml.Marshal(value)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(absolute, data, 0600))
	}
	_, err := loadExternalProviderObservation(root, taskProviderConfig{Provider: "github", LiveStatusSource: "github"}, def, time.Now())
	require.Error(t, err, "a local connector-live string must not establish GitHub status")
}

func TestIssueCreatePreviewDoesNotContactGitHubAndApplyUsesStableMarker(t *testing.T) {
	body := filepath.Join(t.TempDir(), "body.md")
	require.NoError(t, os.WriteFile(body, []byte("literal body"), 0600))
	f := &issueTransport{}
	output := &bytes.Buffer{}
	jsonOutput := true
	cmd := newGitHubIssuesWithTransport("test", &jsonOutput, f)
	cmd.SetOut(output)
	cmd.SetArgs([]string{"create", "--repo", "org/repo", "--id", "work.test", "--title", "New issue", "--body-file", body})
	require.NoError(t, cmd.Execute())
	require.Empty(t, f.calls)
	require.Contains(t, output.String(), "literal body")
	f.replies = []string{`{"login":"alice","id":1}`, `{"full_name":"org/repo"}`, `[[{"number":7,"state":"closed","body":"<!-- stackcord:issue:work.test -->","html_url":"https://github.com/org/repo/issues/7"}]]`}
	cmd = newGitHubIssuesWithTransport("test", &jsonOutput, f)
	cmd.SetOut(output)
	cmd.SetArgs([]string{"create", "--repo", "org/repo", "--id", "work.test", "--title", "New issue", "--body-file", body, "--apply"})
	require.NoError(t, cmd.Execute())
	require.Len(t, f.calls, 3)
	require.Contains(t, output.String(), "https://github.com/org/repo/issues/7")
}

type issueTransport struct {
	replies []string
	calls   [][]string
}

func (f *issueTransport) Run(_ context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, args)
	if len(f.replies) == 0 {
		return nil, errors.New("API unavailable")
	}
	result := f.replies[0]
	f.replies = f.replies[1:]
	return []byte(result), nil
}
func TestGitHubObservationUsesLiveStatusAndRejectsDependencyDrift(t *testing.T) {
	def := work.Definition{ID: "work.test", Fingerprint: "sha256:" + strings.Repeat("a", 64), Dependencies: []string{"work.dep"}}
	mapping := provider.Mapping{SchemaVersion: 1, WorkID: def.ID, DefinitionFingerprint: def.Fingerprint, Provider: "github", ItemID: "https://github.com/org/repo/issues/7", DependencyItems: map[string]string{"work.dep": "https://github.com/org/repo/issues/2"}}
	for _, dependency := range []string{"2", "3"} {
		f := &issueTransport{replies: []string{`{"login":"alice","id":1}`, `{"full_name":"org/repo"}`, `{"number":7,"state":"closed","html_url":"https://github.com/org/repo/issues/7"}`, `[[{"number":` + dependency + `,"html_url":"https://github.com/org/repo/issues/` + dependency + `"}]]`}}
		got, err := readGitHubObservation(context.Background(), mapping, def, time.Now().UTC(), f)
		require.NoError(t, err)
		require.Equal(t, "done", got.Snapshot.Status)
		if dependency == "2" {
			require.Equal(t, provider.Confirmed, got.State.Confidence)
		} else {
			require.Equal(t, provider.Unknown, got.State.Confidence)
		}
	}
	_, err := readGitHubObservation(context.Background(), mapping, def, time.Now(), &issueTransport{})
	require.Error(t, err)
}
func TestIssueURLRejectsOtherHostsAndPullRequests(t *testing.T) {
	for _, url := range []string{"42", "https://evil.test/org/repo/issues/7", "https://github.com/org/repo/pull/7", "https://github.com/org/repo/issues/7?x=1"} {
		_, _, err := parseGitHubIssueURL(url)
		require.Error(t, err)
	}
	var _ gh.Transport = &issueTransport{}
}
