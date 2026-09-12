package command_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/command"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestGovernanceCheckReportsGeneratedProjectAsDisabled(t *testing.T) {
	root := filepath.Join(t.TempDir(), "product")
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "init", "--root", root, "--id", "project.governance", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())

	var output bytes.Buffer
	check := command.New("1.0.0", &output, &bytes.Buffer{})
	check.SetArgs([]string{"governance", "check", "--root", root, "--json"})

	require.NoError(t, check.Execute())
	require.Equal(t, 0, command.ExitCode(check), output.String())
	require.Contains(t, output.String(), `"command":"governance.check"`)
	require.Contains(t, output.String(), `"governance.status","message":"disabled"`)
}

func TestProtectedMeaningWithoutAuthorityApprovalBlocksIntegrationAndRelease(t *testing.T) {
	root := filepath.Join(t.TempDir(), "product")
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "init", "--root", root, "--id", "project.protected", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())
	policy := "schema_version: 1\nenabled: true\nprovider: github\nrepository: example/service\nproduct_authorities: [user:product-owner]\nprotected_kinds: [product, policy, business, contract]\napproval:\n  minimum: 1\n  authority_self_approval: true\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "governance.yaml"), []byte(policy), 0o600))
	commandGit(t, root, "init", "--initial-branch=main")
	commandGit(t, root, "config", "user.name", "Ordinary Member")
	commandGit(t, root, "config", "user.email", "product-owner@example.invalid")
	commandGit(t, root, "add", ".")
	commandGit(t, root, "commit", "-m", "chore: initialize protected project")

	var integrationOutput bytes.Buffer
	integrate := command.New("1.0.0", &integrationOutput, &bytes.Buffer{})
	integrate.SetArgs([]string{"integrate", "plan", "--root", root, "--json"})
	require.NoError(t, integrate.Execute())
	require.Contains(t, integrationOutput.String(), "integrate.governance.approval-unknown")

	var releaseOutput bytes.Buffer
	prepare := command.New("1.0.0", &releaseOutput, &bytes.Buffer{})
	prepare.SetArgs([]string{"release", "prepare", "--root", root, "--release-version", "1.0.0", "--json"})
	require.NoError(t, prepare.Execute())
	require.Contains(t, releaseOutput.String(), "release.governance.approval-unknown")
}

func TestGovernanceAuthorityAddPlansThenAppliesAgainstReviewedPolicy(t *testing.T) {
	root := configuredGovernanceProject(t, []string{"user:product-owner"}, 1)
	missingExpected := runGovernanceAuthorityCommand(t, root, "add", "user:second-owner", "--apply")
	require.Equal(t, domain.StatusBlocked, missingExpected.Status)
	require.Equal(t, "governance.expected-policy-required", missingExpected.Blockers[0].Code)

	planned := runGovernanceAuthorityCommand(t, root, "add", "user:second-owner")
	require.Equal(t, domain.StatusPassed, planned.Status)
	require.True(t, planned.Approval.Required)
	policyFingerprint := factMessage(planned.Facts, "governance.policy-fingerprint")
	require.Regexp(t, `^sha256:[0-9a-f]{64}$`, policyFingerprint)
	require.NotContains(t, readGovernancePolicy(t, root), "user:second-owner")

	applied := runGovernanceAuthorityCommand(t, root, "add", "user:second-owner", "--expected-policy", policyFingerprint, "--apply")
	require.Equal(t, domain.StatusPassed, applied.Status)
	updated := readGovernancePolicy(t, root)
	require.Contains(t, updated, "user:second-owner")
	require.Contains(t, updated, "# Product direction reviewers.")
	require.Contains(t, updated, "# Primary authority note.")
	require.Equal(t, "governance.request-review", applied.NextActions[0].Code)
}

func TestGovernanceAuthorityAddPlansAndAppliesMultipleSubjectsTogether(t *testing.T) {
	root := configuredGovernanceProject(t, []string{"user:product-owner"}, 1)
	subjects := []string{"user:second-owner", "team:platform"}

	planned := runGovernanceAuthoritySubjectsCommand(t, root, "add", subjects)
	require.Equal(t, domain.StatusPassed, planned.Status)
	require.ElementsMatch(t, subjects, factRefs(planned.Facts, "governance.authority-subjects"))
	require.ElementsMatch(t, []string{"user:product-owner", "user:second-owner", "team:platform"}, factRefs(planned.Facts, "governance.authorities-after"))
	require.NotContains(t, readGovernancePolicy(t, root), "user:second-owner")
	require.NotContains(t, readGovernancePolicy(t, root), "team:platform")

	policyFingerprint := factMessage(planned.Facts, "governance.policy-fingerprint")
	applied := runGovernanceAuthoritySubjectsCommand(t, root, "add", subjects, "--expected-policy", policyFingerprint, "--apply")
	require.Equal(t, domain.StatusPassed, applied.Status)
	updated := readGovernancePolicy(t, root)
	require.Contains(t, updated, "user:second-owner")
	require.Contains(t, updated, "team:platform")
}

func TestGovernanceAuthorityRemoveAppliesWithoutLockingOutTheProject(t *testing.T) {
	root := configuredGovernanceProject(t, []string{"user:first-owner", "user:second-owner"}, 1)

	planned := runGovernanceAuthorityCommand(t, root, "remove", "user:second-owner")
	policyFingerprint := factMessage(planned.Facts, "governance.policy-fingerprint")
	applied := runGovernanceAuthorityCommand(t, root, "remove", "user:second-owner", "--expected-policy", policyFingerprint, "--apply")

	require.Equal(t, domain.StatusPassed, applied.Status)
	policy := readGovernancePolicy(t, root)
	require.Contains(t, policy, "user:first-owner")
	require.NotContains(t, policy, "user:second-owner")
	require.Contains(t, policy, "# Primary authority note.")
}

func TestGovernanceAuthorityRemoveAppliesMultipleSubjectsTogether(t *testing.T) {
	root := configuredGovernanceProject(t, []string{"user:first-owner", "user:second-owner", "team:platform"}, 1)
	subjects := []string{"user:second-owner", "team:platform"}

	planned := runGovernanceAuthoritySubjectsCommand(t, root, "remove", subjects)
	policyFingerprint := factMessage(planned.Facts, "governance.policy-fingerprint")
	applied := runGovernanceAuthoritySubjectsCommand(t, root, "remove", subjects, "--expected-policy", policyFingerprint, "--apply")

	require.Equal(t, domain.StatusPassed, applied.Status)
	policy := readGovernancePolicy(t, root)
	require.Contains(t, policy, "user:first-owner")
	require.NotContains(t, policy, "user:second-owner")
	require.NotContains(t, policy, "team:platform")
}

func TestGovernanceAuthorityRemoveRejectsLockoutAndStalePlans(t *testing.T) {
	root := configuredGovernanceProject(t, []string{"user:only-owner"}, 1)
	before := readGovernancePolicy(t, root)

	lockout := runGovernanceAuthorityCommand(t, root, "remove", "user:only-owner")
	require.Equal(t, domain.StatusBlocked, lockout.Status)
	require.Equal(t, "governance.authority-lockout", lockout.Blockers[0].Code)
	require.Equal(t, before, readGovernancePolicy(t, root))

	root = configuredGovernanceProject(t, []string{"user:first-owner", "user:second-owner"}, 1)
	planned := runGovernanceAuthorityCommand(t, root, "remove", "user:second-owner")
	policyFingerprint := factMessage(planned.Facts, "governance.policy-fingerprint")
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "governance.yaml"), append([]byte(readGovernancePolicy(t, root)), []byte("# concurrent policy edit\n")...), 0o600))

	stale := runGovernanceAuthorityCommand(t, root, "remove", "user:second-owner", "--expected-policy", policyFingerprint, "--apply")
	require.Equal(t, domain.StatusBlocked, stale.Status)
	require.Equal(t, "governance.policy-stale", stale.Blockers[0].Code)
	require.Contains(t, readGovernancePolicy(t, root), "user:second-owner")
}

func TestGovernanceAuthorityRemovePreservesAFeasibleApprovalMinimum(t *testing.T) {
	root := configuredGovernanceProject(t, []string{"user:first-owner", "user:second-owner"}, 2)

	result := runGovernanceAuthorityCommand(t, root, "remove", "user:second-owner")

	require.Equal(t, domain.StatusBlocked, result.Status)
	require.Equal(t, "governance.approval-lockout", result.Blockers[0].Code)
}

func TestGovernanceAuthorityChangeRejectsInvalidAndNoopRequests(t *testing.T) {
	root := configuredGovernanceProject(t, []string{"user:first-owner", "user:second-owner"}, 1)
	cases := []struct {
		action, subject, code string
	}{
		{action: "add", subject: "first-owner", code: "governance.authority-invalid"},
		{action: "add", subject: "user:first-owner", code: "governance.authority-exists"},
		{action: "remove", subject: "user:missing-owner", code: "governance.authority-missing"},
	}
	for _, test := range cases {
		t.Run(test.code, func(t *testing.T) {
			result := runGovernanceAuthorityCommand(t, root, test.action, test.subject)
			require.Equal(t, domain.StatusBlocked, result.Status)
			require.Equal(t, test.code, result.Blockers[0].Code)
		})
	}

	duplicate := runGovernanceAuthoritySubjectsCommand(t, root, "add", []string{"user:new-owner", "user:new-owner"})
	require.Equal(t, domain.StatusBlocked, duplicate.Status)
	require.Equal(t, "governance.authority-duplicate", duplicate.Blockers[0].Code)
	require.NotContains(t, readGovernancePolicy(t, root), "user:new-owner")

	mixed := runGovernanceAuthoritySubjectsCommand(t, root, "add", []string{"user:new-owner", "user:first-owner"})
	policyFingerprint := factMessage(mixed.Facts, "governance.policy-fingerprint")
	mixedApply := runGovernanceAuthoritySubjectsCommand(t, root, "add", []string{"user:new-owner", "user:first-owner"}, "--expected-policy", policyFingerprint, "--apply")
	require.Equal(t, domain.StatusBlocked, mixedApply.Status)
	require.Equal(t, "governance.authority-exists", mixedApply.Blockers[0].Code)
	require.NotContains(t, readGovernancePolicy(t, root), "user:new-owner")
}

func configuredGovernanceProject(t *testing.T, authorities []string, minimum int) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "product")
	init := command.New("1.0.0", &bytes.Buffer{}, &bytes.Buffer{})
	init.SetArgs([]string{"project", "init", "--root", root, "--id", "project.governance-management", "--locale", "en", "--apply", "--json"})
	require.NoError(t, init.Execute())
	policy := "schema_version: 1\nenabled: true\nprovider: github\nrepository: example/service\n# Product direction reviewers.\nproduct_authorities:\n"
	for index, authority := range authorities {
		policy += "  - " + authority
		if index == 0 {
			policy += " # Primary authority note."
		}
		policy += "\n"
	}
	policy += "protected_kinds: [product, policy, business, contract]\napproval:\n  minimum: " + strconv.Itoa(minimum) + "\n  authority_self_approval: true\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness", "governance.yaml"), []byte(policy), 0o600))
	return root
}

func runGovernanceAuthorityCommand(t *testing.T, root, action, subject string, extra ...string) domain.Result {
	t.Helper()
	return runGovernanceAuthoritySubjectsCommand(t, root, action, []string{subject}, extra...)
}

func runGovernanceAuthoritySubjectsCommand(t *testing.T, root, action string, subjects []string, extra ...string) domain.Result {
	t.Helper()
	var output bytes.Buffer
	cmd := command.New("1.0.0", &output, &bytes.Buffer{})
	args := []string{"governance", "authority", action, "--root", root, "--json"}
	for _, subject := range subjects {
		args = append(args, "--subject", subject)
	}
	cmd.SetArgs(append(args, extra...))
	require.NoError(t, cmd.Execute())
	var result domain.Result
	require.NoError(t, json.Unmarshal(output.Bytes(), &result), output.String())
	return result
}

func factRefs(items []domain.Item, code string) []string {
	for _, item := range items {
		if item.Code == code {
			return item.Refs
		}
	}
	return nil
}

func readGovernancePolicy(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".harness", "governance.yaml"))
	require.NoError(t, err)
	return string(data)
}
