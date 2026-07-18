package context

import (
	"bytes"
	stdcontext "context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/contract"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/schema"
	"go.yaml.in/yaml/v3"
)

// RefreshMode controls whether rebuilt generated checkpoints are persisted.
type RefreshMode int

const (
	// ReadOnly guarantees project files are not changed.
	ReadOnly RefreshMode = iota
	// WriteCheckpoint atomically replaces ignored local generated index and graph files.
	WriteCheckpoint
)

// Refresh rebuilds project context from authored repository files.
func Refresh(ctx stdcontext.Context, root string, mode RefreshMode) (Snapshot, []domain.Item) {
	snapshot := Snapshot{SchemaVersion: 1, Index: map[string]IndexEntry{}, Impact: map[string][]string{}, Stale: []string{}, Unknown: []string{}}
	var issues []domain.Item

	resolved, err := FindRoot(root)
	if err != nil {
		return snapshot, []domain.Item{{Code: "context.error.root", Message: err.Error()}}
	}
	root = resolved
	manifest, err := schema.LoadYAML[map[string]any](filepath.Join(root, ".harness", "manifest.yaml"))
	if err != nil {
		return snapshot, []domain.Item{{Code: "context.error.manifest", Message: err.Error(), Refs: []string{".harness/manifest.yaml"}}}
	}
	if schemaIssues := schema.Validate("manifest", manifest); len(schemaIssues) > 0 {
		for _, issue := range schemaIssues {
			issues = append(issues, domain.Item{Code: "context.error.manifest", Message: issue.Message, Refs: append([]string{".harness/manifest.yaml"}, issue.Refs...)})
		}
		return snapshot, issues
	}
	authoredRoots, err := manifestAuthoredRoots(manifest)
	if err != nil {
		return snapshot, []domain.Item{{Code: "context.error.manifest", Message: err.Error(), Refs: []string{".harness/manifest.yaml"}}}
	}
	for _, authoredRoot := range authoredRoots {
		base := filepath.Join(root, authoredRoot)
		if _, err := os.Stat(base); os.IsNotExist(err) {
			continue
		}
		walkErr := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink is not allowed in authored root: %s", path)
			}
			if entry.IsDir() || !isAuthoredDocument(path) {
				return nil
			}
			metadata, fingerprint, parseErr := parseDocument(path)
			if parseErr != nil {
				issues = append(issues, domain.Item{Code: "context.error.document", Message: parseErr.Error(), Refs: []string{slashRelative(root, path)}})
				return nil
			}
			if metadata.ID == "" {
				return nil
			}
			if existing, duplicate := snapshot.Index[metadata.ID]; duplicate {
				issues = append(issues, domain.Item{Code: "context.error.duplicate-id", Message: fmt.Sprintf("stable ID %s exists in more than one authored document", metadata.ID), Refs: []string{existing.Path, slashRelative(root, path)}})
				return nil
			}
			refs := append([]string(nil), metadata.Refs...)
			sort.Strings(refs)
			snapshot.Index[metadata.ID] = IndexEntry{ID: metadata.ID, Path: slashRelative(root, path), Kind: metadata.Kind, Status: metadata.Status, Revision: metadata.Revision, Fingerprint: fingerprint, Refs: refs, Sources: metadata.Sources}
			return nil
		})
		if walkErr != nil {
			issues = append(issues, domain.Item{Code: "context.error.walk", Message: walkErr.Error(), Refs: []string{authoredRoot}})
		}
	}

	registryEdges, registryStale, registryUnknown, registryIssues := contractRegistryContext(root, snapshot.Index)
	issues = append(issues, registryIssues...)
	snapshot.Unknown = append(snapshot.Unknown, registryUnknown...)
	snapshot.Impact = buildImpact(snapshot.Index, registryEdges)
	staleSeeds := registryStale
	for _, entry := range snapshot.Index {
		for _, source := range entry.Sources {
			current, exists := snapshot.Index[source.Source]
			if !exists {
				snapshot.Unknown = append(snapshot.Unknown, entry.ID+".source."+source.Source)
				continue
			}
			if source.Fingerprint != "" && source.Fingerprint != current.Fingerprint {
				staleSeeds[source.Source] = struct{}{}
			}
		}
	}
	snapshot.Stale = propagateStale(staleSeeds, snapshot.Impact)
	snapshot.Unknown = append(snapshot.Unknown, semanticConflicts(root, snapshot.Index)...)
	sort.Strings(snapshot.Unknown)
	snapshot.Unknown = unique(snapshot.Unknown)

	if mode == WriteCheckpoint && len(errorsByPrefix(issues, "context.error")) == 0 {
		if err := writeSnapshot(root, snapshot); err != nil {
			issues = append(issues, domain.Item{Code: "context.error.write", Message: err.Error()})
		}
	}
	return snapshot, issues
}

func manifestAuthoredRoots(manifest map[string]any) ([]string, error) {
	paths := map[string]string{"specs": "specs", "contracts": "contracts", "docs": "docs"}
	if configured, ok := manifest["paths"].(map[string]any); ok {
		for _, name := range []string{"specs", "contracts", "docs"} {
			if value, exists := configured[name]; exists {
				pathValue, ok := value.(string)
				if !ok {
					return nil, fmt.Errorf("manifest path %s must be a string", name)
				}
				paths[name] = pathValue
			}
		}
	}
	result := []string{paths["specs"], paths["contracts"], filepath.Join(paths["docs"], "generated")}
	for _, value := range result {
		clean := filepath.Clean(value)
		if value == "" || filepath.IsAbs(value) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("manifest authored path %q escapes project root", value)
		}
	}
	return result, nil
}

func parseDocument(path string) (documentMetadata, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return documentMetadata{}, "", fmt.Errorf("read authored document: %w", err)
	}
	kind := "yaml"
	metadataBytes := data
	if strings.EqualFold(filepath.Ext(path), ".md") {
		kind = "markdown"
		metadataBytes, err = frontmatter(data)
		if err != nil {
			return documentMetadata{}, "", fmt.Errorf("parse frontmatter in %s: %w", path, err)
		}
	}
	raw, err := schema.DecodeYAML[map[string]any](metadataBytes)
	if err != nil {
		return documentMetadata{}, "", fmt.Errorf("decode metadata in %s: %w", path, err)
	}
	var metadata documentMetadata
	if err := yaml.Unmarshal(metadataBytes, &metadata); err != nil {
		return documentMetadata{}, "", fmt.Errorf("decode metadata in %s: %w", path, err)
	}
	if metadata.ID != "" {
		validationValue := raw
		if contractKind(metadata.Kind) {
			validationValue = map[string]any{
				"schema_version": metadata.SchemaVersion, "id": metadata.ID, "kind": metadata.Kind,
				"status": metadata.Status, "revision": metadata.Revision, "refs": metadata.Refs,
			}
			if len(metadata.Sources) > 0 {
				validationValue["sources"] = metadata.Sources
			}
		}
		if issues := schema.Validate("spec", validationValue); len(issues) > 0 {
			return documentMetadata{}, "", fmt.Errorf("schema validation failed for %s: %s", path, issues[0].Message)
		}
	}
	fingerprint, err := Fingerprint(kind, data)
	if err != nil {
		return documentMetadata{}, "", fmt.Errorf("fingerprint %s: %w", path, err)
	}
	return metadata, fingerprint, nil
}

func contractRegistryContext(root string, index map[string]IndexEntry) (map[string][]string, map[string]struct{}, []string, []domain.Item) {
	edges := map[string][]string{}
	stale := map[string]struct{}{}
	unknown := []string{}
	manifest, manifestErr := schema.LoadYAML[map[string]any](filepath.Join(root, ".harness", "manifest.yaml"))
	contractsRoot := "contracts"
	if manifestErr == nil {
		if paths, ok := manifest["paths"].(map[string]any); ok {
			if configured, ok := paths["contracts"].(string); ok && configured != "" {
				contractsRoot = filepath.ToSlash(filepath.Clean(filepath.FromSlash(configured)))
			}
		}
	}
	path := filepath.Join(root, filepath.FromSlash(contractsRoot), "registry.yaml")
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return edges, stale, unknown, nil
	} else if err != nil {
		return edges, stale, unknown, []domain.Item{{Code: "context.error.contract-registry", Message: err.Error(), Refs: []string{"contracts/registry.yaml"}}}
	}
	registry, drift, err := contract.LoadRegistryWithDrift(root)
	if err != nil {
		return edges, stale, unknown, []domain.Item{{Code: "context.error.contract-registry", Message: err.Error(), Refs: []string{"contracts/registry.yaml"}}}
	}
	drifted := map[string]bool{}
	for _, item := range drift {
		drifted[item.ID] = true
		stale[item.ID] = struct{}{}
	}
	for _, entry := range registry.Contracts {
		indexed, exists := index[entry.ID]
		expectedPath := filepath.ToSlash(filepath.Join(contractsRoot, filepath.FromSlash(entry.Source)))
		if !exists || indexed.Path != expectedPath {
			unknown = append(unknown, entry.ID+".source-missing")
			continue
		}
		if indexed.Kind != string(entry.Kind) || indexed.Status != string(entry.Status) || indexed.Revision != entry.Revision {
			unknown = append(unknown, entry.ID+".metadata-mismatch")
		}
		indexed.Kind, indexed.Status, indexed.Revision, indexed.Fingerprint, indexed.ContractRegistered = string(entry.Kind), string(entry.Status), entry.Revision, entry.Fingerprint, true
		index[entry.ID] = indexed
		impact := contract.Impact(registry, entry.ID)
		edges[entry.ID] = append(edges[entry.ID], impact.Dependents...)
		if entry.Status == contract.Stale || entry.Status == contract.Unknown {
			unknown = append(unknown, entry.ID+".registry-status")
		}
		if drifted[entry.ID] {
			unknown = append(unknown, entry.ID+".fingerprint-drift")
		}
	}
	for source := range edges {
		sort.Strings(edges[source])
		edges[source] = unique(edges[source])
	}
	sort.Strings(unknown)
	return edges, stale, unique(unknown), nil
}

func contractKind(value string) bool {
	switch value {
	case string(contract.Product), string(contract.Business), string(contract.Behavior), string(contract.Interface), string(contract.Data):
		return true
	default:
		return false
	}
}

func frontmatter(data []byte) ([]byte, error) {
	normalized := bytes.ReplaceAll(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")), []byte("\r"), []byte("\n"))
	if !bytes.HasPrefix(normalized, []byte("---\n")) {
		return nil, fmt.Errorf("markdown document has no YAML frontmatter")
	}
	end := bytes.Index(normalized[4:], []byte("\n---\n"))
	if end < 0 {
		return nil, fmt.Errorf("markdown frontmatter is not closed")
	}
	return normalized[4 : 4+end], nil
}

func semanticConflicts(root string, index map[string]IndexEntry) []string {
	itemsRoot := filepath.Join(root, ".harness", "work", "items")
	entries, err := os.ReadDir(itemsRoot)
	if err != nil {
		return nil
	}
	var unknown []string
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(itemsRoot, entry.Name()))
		if readErr != nil {
			continue
		}
		var item struct {
			ID                string   `yaml:"id"`
			SemanticOverrides []string `yaml:"semantic_overrides"`
		}
		if yaml.Unmarshal(data, &item) != nil {
			continue
		}
		for _, target := range item.SemanticOverrides {
			if canonical, exists := index[target]; exists && canonical.Status == "approved" {
				unknown = append(unknown, item.ID+".semantic-conflict")
				break
			}
		}
	}
	return unknown
}

func writeSnapshot(root string, snapshot Snapshot) error {
	stateDir := filepath.Join(root, ".harness", "local", "context")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return fmt.Errorf("create context state directory: %w", err)
	}
	index := struct {
		SchemaVersion int                   `json:"schema_version"`
		Index         map[string]IndexEntry `json:"index"`
	}{SchemaVersion: 1, Index: snapshot.Index}
	graph := struct {
		SchemaVersion int                 `json:"schema_version"`
		Impact        map[string][]string `json:"impact"`
	}{SchemaVersion: 1, Impact: snapshot.Impact}
	for name, value := range map[string]any{"context-index.json": index, "impact-graph.json": graph} {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		path := filepath.Join(stateDir, name)
		if err := replaceCheckpoint(path, append(data, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func replaceCheckpoint(path string, data []byte) error {
	temporary := path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create checkpoint temporary file: %w", err)
	}
	cleanup := true
	defer func() {
		_ = file.Close()
		if cleanup {
			_ = os.Remove(temporary)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	cleanup = false
	if directory, err := os.Open(filepath.Dir(path)); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return nil
}

func isAuthoredDocument(path string) bool {
	if strings.EqualFold(filepath.Base(path), "index.md") {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func slashRelative(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}

func unique(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func errorsByPrefix(items []domain.Item, prefix string) []domain.Item {
	var result []domain.Item
	for _, item := range items {
		if strings.HasPrefix(item.Code, prefix) {
			result = append(result, item)
		}
	}
	return result
}
