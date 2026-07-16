package gitgeneric

import (
	"context"
	"fullstack-orchestrator/cli/internal/provider"
)

// Config injects an explicitly authorized write executor; nil keeps writes unavailable.
type Config struct {
	Offline bool
	Execute func(context.Context, provider.Request) error
}

// New returns the generic local Git fallback adapter.
func New(config Config) provider.Adapter {
	health := provider.Health{Status: "ready", Message: "Local Git is available."}
	if config.Offline {
		health = provider.Health{Status: "offline", Message: "Remote Git state is offline; local state remains available."}
	}
	return provider.NewGuarded(provider.GuardedConfig{Descriptor: provider.Descriptor{ID: "git-generic", Name: "Generic Git", Version: "1"}, Capabilities: []provider.Capability{provider.CapabilityRead, provider.CapabilityWrite}, Health: health, Execute: config.Execute})
}
