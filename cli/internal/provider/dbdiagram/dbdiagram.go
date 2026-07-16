package dbdiagram

import (
	"context"
	"fullstack-orchestrator/cli/internal/provider"
)

// Config carries dbdiagram CLI availability without token values.
type Config struct {
	Authenticated, Offline bool
	Execute                func(context.Context, provider.Request) error
}

// New returns an isolated diagram-sync adapter.
func New(config Config) provider.Adapter {
	health := provider.Health{Status: "ready", Message: "dbdiagram adapter is ready."}
	if !config.Authenticated {
		health = provider.Health{Status: "unavailable", Message: "dbdiagram authentication is not configured."}
	}
	if config.Offline {
		health = provider.Health{Status: "offline", Message: "dbdiagram is offline."}
	}
	return provider.NewGuarded(provider.GuardedConfig{Descriptor: provider.Descriptor{ID: "dbdiagram", Name: "dbdiagram", Version: "1"}, Capabilities: []provider.Capability{provider.CapabilityRead, provider.CapabilityWrite, provider.CapabilityDiagramSync}, Health: health, Execute: config.Execute})
}
