package github

import (
	"context"
	"fullstack-orchestrator/cli/internal/provider"
)

// Config carries capability state without credentials themselves.
type Config struct {
	Authenticated, Offline, RateLimited bool
	Execute                             func(context.Context, provider.Request) error
}

// New returns a GitHub Issues/Projects/PR adapter.
func New(config Config) provider.Adapter {
	health := provider.Health{Status: "ready", Message: "GitHub adapter is ready."}
	if !config.Authenticated {
		health = provider.Health{Status: "unavailable", Message: "GitHub authentication is not configured."}
	}
	if config.Offline {
		health = provider.Health{Status: "offline", Message: "GitHub is offline."}
	}
	if config.RateLimited {
		health = provider.Health{Status: "rate_limited", Message: "GitHub rate limit prevents current-state verification."}
	}
	return provider.NewGuarded(provider.GuardedConfig{Descriptor: provider.Descriptor{ID: "github", Name: "GitHub", Version: "1"}, Capabilities: []provider.Capability{provider.CapabilityRead, provider.CapabilityWrite, provider.CapabilityHierarchy, provider.CapabilityDependency, provider.CapabilityDraftReview, provider.CapabilityRelease}, Health: health, Execute: config.Execute})
}
