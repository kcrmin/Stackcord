package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/schema"
)

// Kind separates service promise, business rule, observable behavior, technical interface, and data obligation.
type Kind string

const (
	Product   Kind = "product"
	Business  Kind = "business"
	Behavior  Kind = "behavior"
	Interface Kind = "interface"
	Data      Kind = "data"
)

// Status is authored contract approval state.
type Status string

const (
	Draft      Status = "draft"
	Proposed   Status = "proposed"
	Approved   Status = "approved"
	Deprecated Status = "deprecated"
	Stale      Status = "stale"
	Unknown    Status = "unknown"
)

// Compatibility states the coordination promise for future change.
type Compatibility string

const (
	Additive    Compatibility = "additive"
	Coordinated Compatibility = "coordinated"
	Breaking    Compatibility = "breaking"
)

// Entry connects one canonical obligation source to every known dependent meaning.
type Entry struct {
	ID            string        `json:"id" yaml:"id"`
	Kind          Kind          `json:"kind" yaml:"kind"`
	Status        Status        `json:"status" yaml:"status"`
	Revision      int           `json:"revision" yaml:"revision"`
	Source        string        `json:"source" yaml:"source"`
	Compatibility Compatibility `json:"compatibility" yaml:"compatibility"`
	Providers     []string      `json:"providers" yaml:"providers"`
	Consumers     []string      `json:"consumers" yaml:"consumers"`
	ProductIDs    []string      `json:"product_ids" yaml:"product_ids"`
	ScenarioIDs   []string      `json:"scenario_ids" yaml:"scenario_ids"`
	DataIDs       []string      `json:"data_ids" yaml:"data_ids"`
	UIIDs         []string      `json:"ui_ids" yaml:"ui_ids"`
	MigrationIDs  []string      `json:"migration_ids" yaml:"migration_ids"`
	WorkIDs       []string      `json:"work_ids" yaml:"work_ids"`
	TestIDs       []string      `json:"test_ids" yaml:"test_ids"`
	Refs          []string      `json:"refs" yaml:"refs"`
	Fingerprint   string        `json:"fingerprint" yaml:"fingerprint"`
}

// Registry is the committed service-obligation graph input.
type Registry struct {
	SchemaVersion int     `json:"schema_version" yaml:"schema_version"`
	Contracts     []Entry `json:"contracts" yaml:"contracts"`
	BasePath      string  `json:"-" yaml:"-"`
}

// Drift reports a source whose current bytes no longer match the approved registry identity.
type Drift struct {
	ID       string
	Expected string
	Actual   string
}

// Impact is the deterministic transitive dependent set for one contract.
type ImpactResult struct {
	ContractID string   `json:"contract_id"`
	Dependents []string `json:"dependents"`
	Unknown    []string `json:"unknown"`
}

// LoadRegistry reads strict registry data and rejects stale approved source identities.
func LoadRegistry(root string) (Registry, error) {
	registry, drift, err := LoadRegistryWithDrift(root)
	if err != nil {
		return Registry{}, err
	}
	if len(drift) > 0 {
		return Registry{}, fmt.Errorf("contract source fingerprint differs for %s", drift[0].ID)
	}
	return registry, nil
}

// LoadRegistryWithDrift preserves the validated graph so context recovery can mark dependents stale.
func LoadRegistryWithDrift(root string) (Registry, []Drift, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Registry{}, nil, err
	}
	base, err := contractBasePath(root)
	if err != nil {
		return Registry{}, nil, err
	}
	path := filepath.Join(base, "registry.yaml")
	if err := regularContractFile(path); err != nil {
		return Registry{}, nil, err
	}
	registry, err := schema.LoadYAML[Registry](path)
	if err != nil {
		return Registry{}, nil, err
	}
	if issues := schema.Validate("contract-registry", registry); len(issues) > 0 {
		return Registry{}, nil, fmt.Errorf("validate contract registry: %s", issues[0].Message)
	}
	registry.BasePath = base
	seen := map[string]bool{}
	drift := []Drift{}
	for index := range registry.Contracts {
		entry := &registry.Contracts[index]
		if seen[entry.ID] {
			return Registry{}, nil, fmt.Errorf("duplicate contract ID %s", entry.ID)
		}
		seen[entry.ID] = true
		source, err := safeContractSource(base, entry.Source)
		if err != nil {
			return Registry{}, nil, fmt.Errorf("contract %s source: %w", entry.ID, err)
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return Registry{}, nil, err
		}
		actual := contractDigest(data)
		if actual != entry.Fingerprint {
			drift = append(drift, Drift{ID: entry.ID, Expected: entry.Fingerprint, Actual: actual})
		}
		normalizeEntry(entry)
	}
	sort.Slice(registry.Contracts, func(left, right int) bool { return registry.Contracts[left].ID < registry.Contracts[right].ID })
	sort.Slice(drift, func(left, right int) bool { return drift[left].ID < drift[right].ID })
	return registry, drift, nil
}

// Impact follows contract-to-contract relationships and includes typed external dependents.
func Impact(registry Registry, id string) ImpactResult {
	entries := make(map[string]Entry, len(registry.Contracts))
	edges := make(map[string][]string, len(registry.Contracts))
	for _, entry := range registry.Contracts {
		entries[entry.ID] = entry
		edges[entry.ID] = append(edges[entry.ID], entryDependents(entry)...)
		for _, dependency := range entry.Refs {
			edges[dependency] = append(edges[dependency], entry.ID)
		}
	}
	result := ImpactResult{ContractID: id, Dependents: []string{}, Unknown: []string{}}
	if _, exists := entries[id]; !exists {
		result.Unknown = []string{id}
		return result
	}
	seen, dependents := map[string]bool{id: true}, map[string]bool{}
	queue := []string{id}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, dependent := range normalizedContractSet(edges[current]) {
			if dependent == "" || dependent == id {
				continue
			}
			dependents[dependent] = true
			if _, contractDependent := entries[dependent]; contractDependent && !seen[dependent] {
				seen[dependent] = true
				queue = append(queue, dependent)
			}
		}
		sort.Strings(queue)
	}
	for dependent := range dependents {
		result.Dependents = append(result.Dependents, dependent)
	}
	sort.Strings(result.Dependents)
	return result
}

func entryDependents(entry Entry) []string {
	values := []string{}
	for _, set := range [][]string{entry.Providers, entry.Consumers, entry.ProductIDs, entry.ScenarioIDs, entry.DataIDs, entry.UIIDs, entry.MigrationIDs, entry.WorkIDs, entry.TestIDs} {
		values = append(values, set...)
	}
	return normalizedContractSet(values)
}

func normalizeEntry(entry *Entry) {
	entry.Providers = normalizedContractSet(entry.Providers)
	entry.Consumers = normalizedContractSet(entry.Consumers)
	entry.ProductIDs = normalizedContractSet(entry.ProductIDs)
	entry.ScenarioIDs = normalizedContractSet(entry.ScenarioIDs)
	entry.DataIDs = normalizedContractSet(entry.DataIDs)
	entry.UIIDs = normalizedContractSet(entry.UIIDs)
	entry.MigrationIDs = normalizedContractSet(entry.MigrationIDs)
	entry.WorkIDs = normalizedContractSet(entry.WorkIDs)
	entry.TestIDs = normalizedContractSet(entry.TestIDs)
	entry.Refs = normalizedContractSet(entry.Refs)
}

func normalizedContractSet(values []string) []string {
	result, seen := []string{}, map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func safeContractSource(base, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) {
		return "", fmt.Errorf("must be relative to contracts/")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.ToSlash(clean) == "registry.yaml" {
		return "", fmt.Errorf("escapes authored contract sources")
	}
	path := filepath.Join(base, clean)
	if err := regularContractFile(path); err != nil {
		return "", err
	}
	return path, nil
}

func contractBasePath(root string) (string, error) {
	manifest, err := schema.LoadYAML[map[string]any](filepath.Join(root, ".harness", "manifest.yaml"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return filepath.Join(root, "contracts"), nil
		}
		return "", err
	}
	relative := "contracts"
	if paths, ok := manifest["paths"].(map[string]any); ok {
		if configured, ok := paths["contracts"].(string); ok && configured != "" {
			relative = configured
		}
	}
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("contracts path must be repository-relative")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("contracts path escapes repository")
	}
	return filepath.Join(root, clean), nil
}

func regularContractFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("canonical contract file must be a regular non-symlink file")
	}
	return nil
}

func contractDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}
