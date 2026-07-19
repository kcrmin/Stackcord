package policy_test

import (
	"testing"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/policy"
	"github.com/stretchr/testify/require"
)

func TestApprovalClasses(t *testing.T) {
	cases := []struct {
		action policy.Action
		want   string
		always bool
	}{
		{policy.ReadStatus, "A", false},
		{policy.WriteRequestedCode, "B", false},
		{policy.AddSubmodule, "C", false},
		{policy.PushBranch, "C", false},
		{policy.ForcePush, "D", true},
		{policy.PublishProduction, "D", true},
		{policy.SendSecretExternal, "D", true},
	}
	for _, test := range cases {
		got := policy.Classify(test.action, policy.Consent{})
		require.Equal(t, test.want, got.Class)
		require.Equal(t, test.always, got.AlwaysConfirm)
	}
}

func TestConsentMustMatchCurrentScope(t *testing.T) {
	now := time.Now().UTC()
	consent := policy.Consent{Objective: "add workspace", Repository: "/project", Action: policy.AddSubmodule, Target: "services/identity", ExpiresAt: now.Add(time.Hour), Approved: true}
	scope := policy.Scope{Objective: "add workspace", Repository: "/project", Target: "services/identity", Now: now}
	require.False(t, policy.Classify(policy.AddSubmodule, consent, scope).Required)

	scope.Target = "services/payments"
	require.True(t, policy.Classify(policy.AddSubmodule, consent, scope).Required)

	consent.Action = policy.ForcePush
	decision := policy.Classify(policy.ForcePush, consent, policy.Scope{Objective: consent.Objective, Repository: consent.Repository, Target: consent.Target, Now: now})
	require.True(t, decision.Required, "class D requires a separate exact-target receipt")
}
