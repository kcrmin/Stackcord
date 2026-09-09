package command_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/command"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/project"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestDiscoveryCommandRecoversDraftAndCloneWithoutAcceptingDefaults(t *testing.T) {
	parent := t.TempDir()
	c := project.ExampleDiscoveryCheckpoint()
	c.Decisions[0].QuestionID = "question.scope"
	c.Discovery = &project.DiscoveryPlan{ScopeReady: true, Sections: []project.DiscoverySection{{ID: "section.policy", Title: "Policy", Status: "active", EstimatedRemaining: 3}}, Questions: []project.DiscoveryQuestion{{QuestionID: c.OpenQuestions[0].ID, SectionID: "section.policy", Blocking: true, Options: []project.DiscoveryOption{{ID: "all", Label: "All payments"}, {ID: "card", Label: "Cards"}}, RecommendedOptionID: "card"}}}
	data, err := yaml.Marshal(c)
	require.NoError(t, err)
	input := filepath.Join(parent, "input.yaml")
	require.NoError(t, os.WriteFile(input, data, 0600))
	runFocusedCommand(t, "project", "checkpoint", "--parent", parent, "--id", "01JDISCOVERY", "--locale", "ko", "--input", input, "--apply", "--json")
	draft := filepath.Join(parent, ".harness-drafts", "01JDISCOVERY")
	before, err := os.ReadFile(filepath.Join(draft, "checkpoint.yaml"))
	require.NoError(t, err)
	result := discoveryResult(t, "--draft", draft)
	require.Equal(t, "project.discovery", result.Command)
	require.Contains(t, result.Summary, "약 1–3개")
	require.Contains(t, result.Facts, domain.Item{Code: "discovery.recommendation", Message: "Cards", Refs: []string{c.OpenQuestions[0].ID, "card"}})
	require.Contains(t, result.NextActions, domain.Item{Code: "discovery.question.batch", Message: c.OpenQuestions[0].Summary, Refs: []string{c.OpenQuestions[0].ID, "section.policy"}})
	after, err := os.ReadFile(filepath.Join(draft, "checkpoint.yaml"))
	require.NoError(t, err)
	require.Equal(t, before, after)
	root := filepath.Join(parent, "service")
	runFocusedCommand(t, "project", "init", "--root", root, "--id", "project.discovery", "--locale", "ko", "--draft", draft, "--apply", "--json")
	focusedGit(t, root, "init", "--initial-branch=main")
	focusedGit(t, root, "config", "user.email", "fixture@example.invalid")
	focusedGit(t, root, "config", "user.name", "Fixture")
	focusedGit(t, root, "add", ".")
	focusedGit(t, root, "commit", "-m", "chore: initialize fixture")
	clone := filepath.Join(parent, "clone")
	focusedGit(t, "", "clone", root, clone)
	require.Equal(t, result, discoveryResult(t, "--root", clone))
	// Read canonical question text, not a saved copy of the draft.
	qpath := filepath.Join(clone, "specs", "product", "open-questions", c.OpenQuestions[0].ID+".md")
	raw, err := os.ReadFile(qpath)
	require.NoError(t, err)
	raw = bytes.ReplaceAll(raw, []byte(c.OpenQuestions[0].Summary), []byte("Updated question?"))
	require.NoError(t, os.WriteFile(qpath, raw, 0600))
	updated := discoveryResult(t, "--root", clone)
	require.Equal(t, "Updated question?", updated.NextActions[0].Message)
}

func TestDiscoveryCommandLegacyAndInvalidInputs(t *testing.T) {
	parent := t.TempDir()
	c := project.ExampleDiscoveryCheckpoint()
	c.Discovery = nil
	data, err := yaml.Marshal(c)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(parent, "checkpoint.yaml"), data, 0600))
	r := discoveryResult(t, "--draft", parent)
	require.Contains(t, r.Summary, "unknown")
	require.NotEmpty(t, r.NextActions, "legacy questions must remain visible")
	for _, args := range [][]string{{"project", "discovery"}, {"project", "discovery", "--draft", parent, "--root", parent}, {"project", "discovery", "--draft", filepath.Join(parent, "missing")}} {
		cmd := command.New("test", &bytes.Buffer{}, &bytes.Buffer{})
		cmd.SetArgs(args)
		require.Error(t, cmd.Execute())
	}
}

func TestDiscoveryRejectsOmittedSafetyClassification(t *testing.T) {
	parent := t.TempDir()
	c := project.ExampleDiscoveryCheckpoint()
	c.Discovery = &project.DiscoveryPlan{ScopeReady: true, Sections: []project.DiscoverySection{{ID: "section.policy", Title: "Policy", Status: "active", EstimatedRemaining: 1}}, Questions: []project.DiscoveryQuestion{{QuestionID: c.OpenQuestions[0].ID, SectionID: "section.policy", Options: []project.DiscoveryOption{{ID: "on", Label: "Enabled"}, {ID: "off", Label: "Disabled"}}}}}
	data, err := yaml.Marshal(c)
	require.NoError(t, err)
	data = bytes.ReplaceAll(data, []byte("requires_explicit_answer: false"), []byte(""))
	input := filepath.Join(parent, "checkpoint.yaml")
	require.NoError(t, os.WriteFile(input, data, 0600))
	for _, args := range [][]string{
		{"project", "discovery", "--draft", parent},
		{"project", "checkpoint", "--parent", parent, "--id", "01JINVALID", "--input", input, "--apply"},
		{"project", "init", "--root", filepath.Join(parent, "service"), "--id", "project.invalid", "--draft", parent, "--apply"},
	} {
		cmd := command.New("test", &bytes.Buffer{}, &bytes.Buffer{})
		cmd.SetArgs(args)
		require.ErrorContains(t, cmd.Execute(), "requires_explicit_answer")
	}
}

func discoveryResult(t *testing.T, args ...string) domain.Result {
	t.Helper()
	out := runFocusedCommand(t, append(append([]string{"project", "discovery"}, args...), "--json")...)
	var r domain.Result
	require.NoError(t, json.Unmarshal([]byte(out), &r))
	return r
}
