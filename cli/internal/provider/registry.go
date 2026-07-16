package provider

// Registry selects one live task-status source by explicit config, detection, then local fallback.
type Registry struct {
	adapters map[string]Adapter
	local    Adapter
}

// NewRegistry creates a registry; the first adapter is the local fallback.
func NewRegistry(adapters ...Adapter) *Registry {
	registry := &Registry{adapters: map[string]Adapter{}}
	for index, adapter := range adapters {
		registry.adapters[adapter.Descriptor().ID] = adapter
		if index == 0 {
			registry.local = adapter
		}
	}
	return registry
}

// Select honors explicit project configuration before detected provider.
func (registry *Registry) Select(explicit, detected string) Adapter {
	if adapter := registry.adapters[explicit]; adapter != nil {
		return adapter
	}
	if adapter := registry.adapters[detected]; adapter != nil {
		return adapter
	}
	return registry.local
}
