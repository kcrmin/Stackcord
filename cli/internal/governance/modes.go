package governance

import (
	"fmt"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
	"regexp"
	"strings"
	"time"
)

type Delegate struct {
	Subject    string    `json:"subject" yaml:"subject"`
	Repository string    `json:"repository" yaml:"repository"`
	Kinds      []string  `json:"kinds" yaml:"kinds"`
	ExpiresAt  time.Time `json:"expires_at" yaml:"expires_at"`
}

var githubLogin = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)

func validGitHubSubject(subject string) bool {
	if !strings.HasPrefix(subject, "user:") {
		return false
	}
	login := strings.TrimPrefix(subject, "user:")
	return githubLogin.MatchString(login) && !strings.Contains(login, "--")
}

func DecodePolicy(data []byte) (Policy, error) {
	raw, err := schema.DecodeYAML[map[string]any](data)
	if err != nil {
		return Policy{}, err
	}
	if issues := schema.Validate("governance", raw); len(issues) > 0 {
		return Policy{}, fmt.Errorf("invalid policy: %s", issues[0].Message)
	}
	p, err := schema.DecodeYAML[Policy](data)
	if err != nil {
		return p, err
	}
	if p.SchemaVersion == 2 {
		if (p.Mode == "weak") == p.Enabled {
			return p, fmt.Errorf("security mode and enabled flag conflict")
		}
		if p.Provider != "github" {
			return p, fmt.Errorf("version 2 currently requires the GitHub review adapter")
		}
		if p.Approval.AuthoritySelfApproval {
			return p, fmt.Errorf("GitHub does not support PR author self-approval")
		}
		admins := map[string]bool{}
		for _, a := range p.ProductAuthorities {
			if !validGitHubSubject(a) {
				return p, fmt.Errorf("register valid individual GitHub accounts; team expansion is not configured")
			}
			key := strings.ToLower(a)
			if admins[key] {
				return p, fmt.Errorf("duplicate GitHub administrator identity")
			}
			admins[key] = true
		}
	} else if p.Mode != "" || len(p.Delegates) > 0 {
		return p, fmt.Errorf("mode and delegates require an explicit version 2 migration")
	}
	if p.Enabled && p.Approval.Minimum > len(p.ProductAuthorities) {
		return p, fmt.Errorf("approval minimum exceeds registered admins")
	}
	seen := map[string]bool{}
	for _, d := range p.Delegates {
		if !validGitHubSubject(d.Subject) {
			return p, fmt.Errorf("invalid GitHub delegate account")
		}
		subject := strings.ToLower(d.Subject)
		if seen[subject] || d.Repository != p.Repository || d.ExpiresAt.IsZero() {
			return p, fmt.Errorf("invalid duplicate or cross-repository delegate")
		}
		seen[subject] = true
	}
	return p, nil
}

func SecurityMode(p Policy) string {
	if p.Mode != "" {
		return p.Mode
	}
	if p.Enabled {
		return "strong"
	}
	return "weak"
}

// Eligible uses only the trusted target policy. Delegates cannot change governance.
func Eligible(p Policy, subject string, kinds []string, policyChange bool, now time.Time) bool {
	for _, a := range p.ProductAuthorities {
		if strings.EqualFold(a, subject) {
			return true
		}
	}
	if SecurityMode(p) != "medium" || policyChange || len(kinds) == 0 {
		return false
	}
	for _, d := range p.Delegates {
		if !strings.EqualFold(d.Subject, subject) || d.Repository != p.Repository || !now.Before(d.ExpiresAt) {
			continue
		}
		allowed := true
		for _, kind := range kinds {
			found := false
			for _, k := range d.Kinds {
				if k == kind {
					found = true
				}
			}
			if !found {
				allowed = false
			}
		}
		if allowed {
			return true
		}
	}
	return false
}
