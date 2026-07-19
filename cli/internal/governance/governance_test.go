package governance_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/governance"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestDisabledGovernanceIsNonBlockingAndSchemaValid(t *testing.T) {
	root := governanceProject(t, false)

	report := governance.Check(context.Background(), root, "", time.Now().UTC())

	require.Equal(t, governance.Disabled, report.Status)
	require.False(t, report.Enabled)
	require.Empty(t, report.Issues)
	require.Regexp(t, `^sha256:[0-9a-f]{64}$`, report.ProtectedFingerprint)
	raw, err := schema.LoadYAML[map[string]any](filepath.Join(root, ".harness", "governance.yaml"))
	require.NoError(t, err)
	require.Empty(t, schema.Validate("governance", raw))
}

func TestConfiguredAuthorityApprovesExactProtectedState(t *testing.T) {
	root := governanceProject(t, true)
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	fingerprint, err := governance.ProtectedFingerprint(root)
	require.NoError(t, err)
	head := gitOutput(t, root, "rev-parse", "HEAD")
	observation := validObservation(head, fingerprint, now)
	path := writeObservation(t, root, observation)

	report := governance.Check(context.Background(), root, path, now)

	require.Equal(t, governance.Approved, report.Status)
	require.Equal(t, []string{"user:product-owner"}, report.Approvers)
	require.Equal(t, "review-42:r3", report.ApprovalRevision)
	require.Empty(t, report.Issues)
}

func TestGovernanceRejectsSpoofingStaleEvidenceAndDuplicateApprovers(t *testing.T) {
	root := governanceProject(t, true)
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	fingerprint, err := governance.ProtectedFingerprint(root)
	require.NoError(t, err)
	head := gitOutput(t, root, "rev-parse", "HEAD")

	cases := map[string]func(*governance.Observation){
		"unauthorized account": func(value *governance.Observation) {
			value.Decisions[0].Subject = "user:ordinary-member"
		},
		"stale commit": func(value *governance.Observation) {
			value.HeadCommit = strings.Repeat("b", 40)
		},
		"stale fingerprint": func(value *governance.Observation) {
			value.ProtectedFingerprint = digest("b")
		},
		"wrong provider": func(value *governance.Observation) {
			value.Provider = "gitlab"
		},
		"cached source": func(value *governance.Observation) {
			value.Source = "cache"
		},
		"old observation": func(value *governance.Observation) {
			value.FetchedAt = now.Add(-16 * time.Minute)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := validObservation(head, fingerprint, now)
			mutate(&value)
			report := governance.Check(context.Background(), root, writeObservation(t, root, value), now)
			require.NotEqual(t, governance.Approved, report.Status)
			require.NotEmpty(t, report.Issues)
		})
	}

	policy := readPolicy(t, root)
	policy.Approval.Minimum = 2
	writeYAML(t, filepath.Join(root, ".harness", "governance.yaml"), policy)
	fingerprint, err = governance.ProtectedFingerprint(root)
	require.NoError(t, err)
	value := validObservation(head, fingerprint, now)
	value.Decisions = append(value.Decisions, value.Decisions[0])
	report := governance.Check(context.Background(), root, writeObservation(t, root, value), now)
	require.Equal(t, governance.Proposed, report.Status)
	require.Contains(t, issueCodes(report), "governance.approval-insufficient")
}

func TestSelfApprovalPolicyAndGovernanceChangesAreProtected(t *testing.T) {
	root := governanceProject(t, true)
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	before, err := governance.ProtectedFingerprint(root)
	require.NoError(t, err)
	policy := readPolicy(t, root)
	policy.Approval.AuthoritySelfApproval = false
	writeYAML(t, filepath.Join(root, ".harness", "governance.yaml"), policy)
	after, err := governance.ProtectedFingerprint(root)
	require.NoError(t, err)
	require.NotEqual(t, before, after)

	value := validObservation(gitOutput(t, root, "rev-parse", "HEAD"), after, now)
	value.AuthorSubject = "user:product-owner"
	report := governance.Check(context.Background(), root, writeObservation(t, root, value), now)
	require.Equal(t, governance.Proposed, report.Status)
	require.Contains(t, issueCodes(report), "governance.approval-insufficient")

	value.Decisions = append(value.Decisions, governance.Decision{Subject: "team:product", Kind: "review", State: "approved", Revision: "decision-r2", SubmittedAt: now.Add(-time.Minute)})
	report = governance.Check(context.Background(), root, writeObservation(t, root, value), now)
	require.Equal(t, governance.Approved, report.Status)
}

func governanceProject(t *testing.T, enabled bool) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness", "local", "governance"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "specs", "policies"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "contracts"), 0o700))
	policy := governance.Policy{
		SchemaVersion: 1,
		Enabled:       enabled,
		Provider:      "github",
		Repository:    "example/service",
		ProductAuthorities: []string{
			"team:product",
			"user:product-owner",
		},
		ProtectedKinds: []string{"product", "policy", "business", "contract"},
		Approval: governance.ApprovalPolicy{
			Minimum:               1,
			AuthoritySelfApproval: true,
		},
	}
	if !enabled {
		policy.Provider, policy.Repository = "", ""
		policy.ProductAuthorities = []string{}
	}
	writeYAML(t, filepath.Join(root, ".harness", "governance.yaml"), policy)
	require.NoError(t, os.WriteFile(filepath.Join(root, "specs", "policies", "refund.md"), []byte("approved refund rule\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "registry.yaml"), []byte("schema_version: 1\ncontracts: []\n"), 0o600))
	gitRun(t, root, "init", "--initial-branch=main")
	gitRun(t, root, "config", "user.name", "Spoofable Display Name")
	gitRun(t, root, "config", "user.email", "product-owner@example.invalid")
	gitRun(t, root, "add", ".harness/governance.yaml", "specs", "contracts")
	gitRun(t, root, "commit", "-m", "chore: initialize fixture")
	return root
}

func validObservation(head, fingerprint string, now time.Time) governance.Observation {
	return governance.Observation{
		SchemaVersion:        1,
		Provider:             "github",
		Repository:           "example/service",
		ReviewID:             "review-42",
		ReviewRevision:       "r3",
		HeadCommit:           head,
		ProtectedFingerprint: fingerprint,
		AuthorSubject:        "user:ordinary-member",
		Status:               "merged",
		Decisions: []governance.Decision{{
			Subject: "user:product-owner", Kind: "review", State: "approved", Revision: "decision-r1", SubmittedAt: now.Add(-time.Minute),
		}},
		FetchedAt: now.Add(-time.Minute),
		Source:    "connector-live",
		RawHash:   digest("a"),
	}
}

func readPolicy(t *testing.T, root string) governance.Policy {
	t.Helper()
	value, err := schema.LoadYAML[governance.Policy](filepath.Join(root, ".harness", "governance.yaml"))
	require.NoError(t, err)
	return value
}

func writeObservation(t *testing.T, root string, value governance.Observation) string {
	t.Helper()
	path := filepath.Join(root, ".harness", "local", "governance", "approval.yaml")
	writeYAML(t, path, value)
	return path
}

func writeYAML(t *testing.T, path string, value any) {
	t.Helper()
	data, err := yaml.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

func gitRun(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func gitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(output))
}

func issueCodes(report governance.Report) []string {
	result := make([]string, 0, len(report.Issues))
	for _, issue := range report.Issues {
		result = append(result, issue.Code)
	}
	return result
}

func digest(character string) string { return "sha256:" + strings.Repeat(character, 64) }
