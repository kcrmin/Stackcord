package project_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/project"
	"github.com/stretchr/testify/require"
)

func TestCheckpointRevisesOneDiscoveryWithoutRawConversation(t *testing.T) {
	parent := t.TempDir()
	first := validCheckpoint("Account recovery", "Which recovery proof is allowed?")
	plan, err := project.PlanCheckpoint(project.CheckpointRequest{Parent: parent, DraftID: "01JDISCOVERY", Locale: "ko", Checkpoint: first})
	require.NoError(t, err)
	require.Contains(t, plan.ID, "r1")
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)

	second := validCheckpoint("Account recovery for members", "How long are recovery attempts retained?")
	plan, err = project.PlanCheckpoint(project.CheckpointRequest{Parent: parent, DraftID: "01JDISCOVERY", Locale: "ko", Checkpoint: second})
	require.NoError(t, err)
	require.Contains(t, plan.ID, "r2")
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), plan).Status)

	root := filepath.Join(parent, ".harness-drafts", "01JDISCOVERY")
	require.Contains(t, mustReadCheckpoint(t, filepath.Join(root, "state.yaml")), "revision: 2")
	require.Contains(t, mustReadCheckpoint(t, filepath.Join(root, "specs", "product", "summary.md")), "Account recovery for members")
	for _, path := range []string{"checkpoint.yaml", "specs/product/summary.md", "specs/product/policies.yaml", "specs/product/scenarios.yaml", "specs/product/open-questions.yaml"} {
		data := mustReadCheckpoint(t, filepath.Join(root, filepath.FromSlash(path)))
		require.NotContains(t, data, "User said")
		require.NotContains(t, data, "반말")
	}
}

func TestCheckpointRejectsMissingStableIdentity(t *testing.T) {
	checkpoint := validCheckpoint("Example", "Question")
	checkpoint.Policies[0].ID = ""
	_, err := project.PlanCheckpoint(project.CheckpointRequest{Parent: t.TempDir(), DraftID: "01JINVALID", Locale: "en", Checkpoint: checkpoint})
	require.ErrorContains(t, err, "does not match pattern")
}

func validCheckpoint(summary, question string) project.DiscoveryCheckpoint {
	return project.DiscoveryCheckpoint{
		SchemaVersion:   1,
		Summary:         summary,
		CurrentFocus:    "Clarify recovery policy",
		Roles:           []project.DiscoveryFact{{ID: "role.member", Summary: "Registered member"}},
		Journeys:        []project.DiscoveryFact{{ID: "journey.account.recovery", Summary: "Recover access"}},
		Capabilities:    []project.DiscoveryFact{{ID: "capability.account.recovery", Summary: "Recover account access"}},
		Policies:        []project.DiscoveryFact{{ID: "policy.account.recovery-proof", Summary: "Require verified proof"}},
		Scenarios:       []project.DiscoveryScenario{{ID: "scenario.account.recovery-success", Actor: "role.member", Trigger: "valid proof", Outcome: "access restored", Failure: "invalid proof is rejected"}},
		Quality:         []project.DiscoveryFact{{ID: "quality.account.accessibility", Summary: "Keyboard accessible"}},
		UICoverage:      []project.UICoverage{{ID: "ui.account.recovery", RoleID: "role.member", JourneyID: "journey.account.recovery", States: []string{"ready", "submitting", "success", "error"}}},
		TechnologyNeeds: []project.DiscoveryFact{{ID: "technology.need.secure-token", Summary: "Secure expiring recovery token; implementation not selected"}},
		Decisions:       []project.DiscoveryDecision{{ID: "decision.account.recovery-channel", Choice: "email link", Rationale: "Available to current users"}},
		Assumptions:     []project.DiscoveryFact{{ID: "assumption.account.email-verified", Summary: "Member emails are verified"}},
		OpenQuestions:   []project.DiscoveryFact{{ID: "question.account.recovery", Summary: question}},
	}
}

func mustReadCheckpoint(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}
