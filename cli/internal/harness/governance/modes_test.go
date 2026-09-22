package governance

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestDecodeRejectsDuplicateAccountCasing(t *testing.T) {
	body := strings.Replace(livePolicy("strong"), `["user:admin"]`, `["user:admin", "user:ADMIN"]`, 1)
	if _, err := DecodePolicy([]byte(body)); err == nil {
		t.Fatal("duplicate GitHub identity accepted")
	}
}

func TestDecodeGitHubAccountGrammar(t *testing.T) {
	for _, login := range []string{"bad/name", "bad.name", "bad_name", "bad-", "bad--name", strings.Repeat("a", 40)} {
		t.Run(login, func(t *testing.T) {
			body := strings.Replace(livePolicy("strong"), "user:admin", "user:"+login, 1)
			if _, err := DecodePolicy([]byte(body)); err == nil {
				t.Fatalf("invalid GitHub account %q accepted", login)
			}
		})
	}
	for _, login := range []string{"a", "A-b", strings.Repeat("a", 39)} {
		body := strings.Replace(livePolicy("strong"), "user:admin", "user:"+login, 1)
		if _, err := DecodePolicy([]byte(body)); err != nil {
			t.Fatalf("valid account %q rejected: %v", login, err)
		}
	}
}

func TestDecodeDelegateUsesSameAccountGrammar(t *testing.T) {
	body := livePolicy("medium") + "delegates:\n  - subject: user:bad--name\n    repository: owner/repo\n    kinds: [contract]\n    expires_at: 2027-01-01T00:00:00Z\n"
	if _, err := DecodePolicy([]byte(body)); err == nil {
		t.Fatal("invalid delegate login accepted")
	}
}

func TestModeApprovalUsesTrustedPolicy(t *testing.T) {
	now := time.Now().UTC()
	trusted := Policy{SchemaVersion: 2, Enabled: true, Mode: "strong", ProductAuthorities: []string{"user:admin"}, Approval: ApprovalPolicy{Minimum: 1}}
	require.False(t, Eligible(trusted, "user:attacker", []string{"policy"}, false, now))
	require.True(t, Eligible(trusted, "user:admin", []string{"policy"}, true, now))
	trusted.Mode = "medium"
	trusted.Delegates = []Delegate{{Subject: "user:temp", Repository: "acme/service", Kinds: []string{"policy"}, ExpiresAt: now.Add(time.Hour)}}
	trusted.Repository = "acme/service"
	require.True(t, Eligible(trusted, "user:temp", []string{"policy"}, false, now))
	require.False(t, Eligible(trusted, "user:temp", []string{"contract"}, false, now))
	require.False(t, Eligible(trusted, "user:temp", []string{"policy"}, true, now))
	require.False(t, Eligible(trusted, "user:temp", []string{"policy"}, false, now.Add(2*time.Hour)))
	trusted.Mode = "strong"
	require.False(t, Eligible(trusted, "user:temp", []string{"policy"}, false, now))
}
func TestDecodeModesRequiresConsistentExplicitSecurity(t *testing.T) {
	for _, body := range []string{
		`schema_version: 2
mode: weak
enabled: true`,
		`schema_version: 2
mode: strong
enabled: false`,
	} {
		_, err := DecodePolicy([]byte(body))
		require.Error(t, err)
	}
}
