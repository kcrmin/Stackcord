package work

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/schema"
	"fullstack-orchestrator/cli/internal/workspace"
	"go.yaml.in/yaml/v3"
)

var (
	workIDPattern   = regexp.MustCompile(`^work\.[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
	stableIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z][a-z0-9-]*)+$`)
	digestPattern   = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// ValidateDefinition checks intrinsic executable-work invariants.
func ValidateDefinition(definition Definition) []domain.Item {
	issues := []domain.Item{}
	add := func(code, message string, refs ...string) {
		issues = append(issues, domain.Item{Code: code, Message: message, Refs: refs})
	}
	if definition.SchemaVersion != 1 {
		add("work.schema-invalid", "Work definition schema_version must be 1.")
	}
	if !workIDPattern.MatchString(definition.ID) {
		add("work.id-invalid", "Work definition ID must be a conventional stable work ID.", definition.ID)
	}
	if definition.ParentID != "" && !workIDPattern.MatchString(definition.ParentID) {
		add("work.parent-invalid", "Parent work ID is invalid.", definition.ParentID)
	}
	if definition.Readiness != Draft && definition.Readiness != Ready {
		add("work.readiness-invalid", "Readiness must be draft or ready.")
	}
	if strings.TrimSpace(definition.Title) == "" {
		add("work.title-required", "Work title is required.")
	}
	if strings.TrimSpace(definition.Outcome) == "" {
		add("work.outcome-required", "An observable product outcome is required.")
	}
	for _, duplicate := range duplicateValues(definition.Refs, definition.Workspaces, definition.Dependencies, definition.Scope.Repositories, definition.Scope.Paths, definition.Scope.PolicyIDs, definition.Scope.ScenarioIDs, definition.Scope.ContractIDs, definition.Scope.DBEntities, definition.Scope.MigrationSlots, definition.Scope.UIFlows, definition.Scope.DependencyMajors, definition.Scope.RootPointers, definition.Evidence.Kinds) {
		add("work.duplicate-scope", "Work definition contains a duplicate set value.", duplicate)
	}
	acceptanceIDs := make([]string, 0, len(definition.Acceptance))
	for _, acceptance := range definition.Acceptance {
		acceptanceIDs = append(acceptanceIDs, acceptance.ID)
	}
	for _, duplicate := range duplicateValues(acceptanceIDs) {
		add("work.acceptance-duplicate", "Acceptance scenario IDs must be unique.", duplicate)
	}
	for _, path := range definition.Scope.Paths {
		if !safeRelativePath(path) {
			add("work.path-invalid", "Work scope paths must remain repository-relative.", path)
		}
	}
	if definition.Readiness == Ready {
		if len(definition.Acceptance) == 0 {
			add("work.acceptance-required", "Ready work needs at least one acceptance scenario.")
		}
		for _, acceptance := range definition.Acceptance {
			if !stableIDPattern.MatchString(acceptance.ID) || strings.TrimSpace(acceptance.Given) == "" || strings.TrimSpace(acceptance.When) == "" || strings.TrimSpace(acceptance.Then) == "" || strings.TrimSpace(acceptance.Failure) == "" {
				add("work.acceptance-invalid", "Acceptance scenarios need a stable ID and observable success and failure behavior.", acceptance.ID)
			}
		}
		if scopeEmpty(definition.Scope) {
			add("work.scope-required", "Ready work needs path or semantic scope.")
		}
		if len(definition.Workspaces) == 0 {
			add("work.workspace-required", "Ready work needs at least one declared workspace.")
		}
		if len(definition.MergeOrder) == 0 {
			add("work.merge-order-required", "Ready multi-workspace work needs an explicit merge order.")
		} else if !sameStringSet(definition.Workspaces, definition.MergeOrder) {
			add("work.merge-order-invalid", "Merge order must contain each affected workspace exactly once.", definition.MergeOrder...)
		}
		if strings.TrimSpace(definition.FirstFailingTest) == "" {
			add("work.first-test-required", "Ready work needs the first failing TDD test or approved test key.")
		}
		if len(definition.Evidence.Kinds) == 0 {
			add("work.evidence-required", "Ready work needs explicit evidence kinds.")
		}
	}
	want := Fingerprint(definition)
	if definition.Fingerprint != "" && (!digestPattern.MatchString(definition.Fingerprint) || definition.Fingerprint != want) {
		add("work.fingerprint-mismatch", "Supplied definition fingerprint differs from normalized content.", definition.Fingerprint, want)
	}
	sort.Slice(issues, func(left, right int) bool {
		if issues[left].Code == issues[right].Code {
			return strings.Join(issues[left].Refs, "\x00") < strings.Join(issues[right].Refs, "\x00")
		}
		return issues[left].Code < issues[right].Code
	})
	return issues
}

// Fingerprint hashes normalized work meaning and excludes the fingerprint field itself.
func Fingerprint(definition Definition) string {
	normalized := normalize(definition)
	normalized.Fingerprint = ""
	data, _ := json.Marshal(normalized)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

// PlanDefinition validates repository references and prepares one atomic canonical write.
func PlanDefinition(ctx context.Context, root string, definition Definition) (operation.Plan, error) {
	plan := operation.Plan{ID: "work-define-" + strings.ReplaceAll(definition.ID, ".", "-"), Root: root}
	plan.Blockers = append(plan.Blockers, ValidateDefinition(definition)...)
	definition = normalize(definition)
	if schemaIssues := schema.Validate("work-item", definition); len(schemaIssues) > 0 {
		plan.Blockers = append(plan.Blockers, domain.Item{Code: "work.schema-invalid", Message: schemaIssues[0].Message})
	}
	if len(plan.Blockers) > 0 {
		return plan, nil
	}
	rootInfo, err := workspace.FindRoot(ctx, root)
	if err != nil {
		return operation.Plan{}, err
	}
	root = rootInfo.Path
	plan.Root = root
	contextSnapshot, contextIssues := contextpkg.Refresh(ctx, root, contextpkg.ReadOnly)
	for _, issue := range contextIssues {
		if strings.HasPrefix(issue.Code, "context.error") {
			plan.Blockers = append(plan.Blockers, issue)
		}
	}
	for _, ref := range canonicalRefs(definition) {
		if _, exists := contextSnapshot.Index[ref]; !exists {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "work.ref-missing", Message: "Work scope references missing canonical context.", Refs: []string{ref}})
		}
	}
	workspaceIDs := map[string]bool{}
	for _, entry := range rootInfo.Manifest.Workspaces {
		workspaceIDs[entry.ID] = true
	}
	for _, id := range append(append([]string(nil), definition.Workspaces...), definition.Scope.RootPointers...) {
		if !workspaceIDs[id] {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "work.workspace-missing", Message: "Work references an undeclared workspace.", Refs: []string{id}})
		}
	}
	existing, loadErr := loadDefinitions(root)
	if loadErr != nil {
		return operation.Plan{}, loadErr
	}
	definitions := make(map[string]Definition, len(existing)+1)
	for _, current := range existing {
		definitions[current.ID] = current
	}
	definition.Fingerprint = Fingerprint(definition)
	definitions[definition.ID] = definition
	for _, dependency := range append(append([]string(nil), definition.Dependencies...), definition.ParentID) {
		if dependency != "" {
			if _, exists := definitions[dependency]; !exists {
				plan.Blockers = append(plan.Blockers, domain.Item{Code: "work.dependency-missing", Message: "Work dependency has no canonical definition.", Refs: []string{dependency}})
			}
		}
	}
	if cycle := dependencyCycle(definitions); len(cycle) > 0 {
		plan.Blockers = append(plan.Blockers, domain.Item{Code: "work.dependency-cycle", Message: "Work dependencies contain a cycle.", Refs: cycle})
	}
	plan.Blockers = normalizeItems(plan.Blockers)
	if len(plan.Blockers) > 0 {
		return plan, nil
	}
	data, err := yaml.Marshal(definition)
	if err != nil {
		return operation.Plan{}, err
	}
	plan.Files = []operation.FileChange{{Path: filepath.ToSlash(filepath.Join(".harness", "work", "definitions", definition.ID+".yaml")), Content: data, Mode: 0o644}}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return plan, err
}

// LoadDefinitions reads canonical executable definitions in stable ID order.
func LoadDefinitions(root string) ([]Definition, error) {
	return loadDefinitions(root)
}

func loadDefinitions(root string) ([]Definition, error) {
	directory := filepath.Join(root, ".harness", "work", "definitions")
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return []Definition{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := []Definition{}
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml") {
			continue
		}
		definition, loadErr := schema.LoadYAML[Definition](filepath.Join(directory, entry.Name()))
		if loadErr != nil {
			return nil, loadErr
		}
		if issues := ValidateDefinition(definition); len(issues) > 0 || definition.Fingerprint == "" {
			if len(issues) > 0 {
				return nil, fmt.Errorf("invalid work definition %s: %s", entry.Name(), issues[0].Code)
			}
			return nil, fmt.Errorf("invalid work definition %s: missing fingerprint", entry.Name())
		}
		result = append(result, definition)
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ID < result[right].ID })
	return result, nil
}

func normalize(definition Definition) Definition {
	definition.Title = strings.TrimSpace(definition.Title)
	definition.Outcome = strings.TrimSpace(definition.Outcome)
	definition.FirstFailingTest = strings.TrimSpace(definition.FirstFailingTest)
	definition.Refs = normalizedSet(definition.Refs)
	definition.Workspaces = normalizedSet(definition.Workspaces)
	definition.Dependencies = normalizedSet(definition.Dependencies)
	definition.Scope.Repositories = normalizedSet(definition.Scope.Repositories)
	definition.Scope.Paths = normalizedSet(definition.Scope.Paths)
	definition.Scope.PolicyIDs = normalizedSet(definition.Scope.PolicyIDs)
	definition.Scope.ScenarioIDs = normalizedSet(definition.Scope.ScenarioIDs)
	definition.Scope.ContractIDs = normalizedSet(definition.Scope.ContractIDs)
	definition.Scope.DBEntities = normalizedSet(definition.Scope.DBEntities)
	definition.Scope.MigrationSlots = normalizedSet(definition.Scope.MigrationSlots)
	definition.Scope.UIFlows = normalizedSet(definition.Scope.UIFlows)
	definition.Scope.DependencyMajors = normalizedSet(definition.Scope.DependencyMajors)
	definition.Scope.RootPointers = normalizedSet(definition.Scope.RootPointers)
	definition.Evidence.Kinds = normalizedSet(definition.Evidence.Kinds)
	definition.Acceptance = append([]AcceptanceScenario(nil), definition.Acceptance...)
	for index := range definition.Acceptance {
		definition.Acceptance[index].Given = strings.TrimSpace(definition.Acceptance[index].Given)
		definition.Acceptance[index].When = strings.TrimSpace(definition.Acceptance[index].When)
		definition.Acceptance[index].Then = strings.TrimSpace(definition.Acceptance[index].Then)
		definition.Acceptance[index].Failure = strings.TrimSpace(definition.Acceptance[index].Failure)
	}
	sort.Slice(definition.Acceptance, func(left, right int) bool { return definition.Acceptance[left].ID < definition.Acceptance[right].ID })
	definition.MergeOrder = trimValues(definition.MergeOrder)
	return definition
}

func canonicalRefs(definition Definition) []string {
	values := append([]string(nil), definition.Refs...)
	values = append(values, definition.Scope.PolicyIDs...)
	values = append(values, definition.Scope.ScenarioIDs...)
	values = append(values, definition.Scope.ContractIDs...)
	values = append(values, definition.Scope.UIFlows...)
	return normalizedSet(values)
}

func dependencyCycle(definitions map[string]Definition) []string {
	state := map[string]int{}
	stack := []string{}
	var visit func(string) []string
	visit = func(id string) []string {
		if state[id] == 1 {
			for index, value := range stack {
				if value == id {
					return append(append([]string(nil), stack[index:]...), id)
				}
			}
			return []string{id}
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		stack = append(stack, id)
		definition := definitions[id]
		dependencies := append([]string(nil), definition.Dependencies...)
		if definition.ParentID != "" {
			dependencies = append(dependencies, definition.ParentID)
		}
		for _, dependency := range dependencies {
			if _, exists := definitions[dependency]; exists {
				if cycle := visit(dependency); len(cycle) > 0 {
					return cycle
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = 2
		return nil
	}
	ids := make([]string, 0, len(definitions))
	for id := range definitions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if cycle := visit(id); len(cycle) > 0 {
			return cycle
		}
	}
	return nil
}

func normalizeItems(items []domain.Item) []domain.Item {
	seen := map[string]bool{}
	result := []domain.Item{}
	for _, item := range items {
		sort.Strings(item.Refs)
		key := item.Code + "\x00" + strings.Join(item.Refs, "\x00")
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Code < result[right].Code })
	return result
}

func duplicateValues(groups ...[]string) []string {
	result := []string{}
	for _, values := range groups {
		seen := map[string]bool{}
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if seen[trimmed] {
				result = append(result, trimmed)
			}
			seen[trimmed] = true
		}
	}
	return normalizedSet(result)
}

func normalizedSet(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
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

func trimValues(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, strings.TrimSpace(value))
	}
	return result
}

func sameStringSet(left, right []string) bool {
	leftSet, rightSet := normalizedSet(left), normalizedSet(right)
	if len(left) != len(leftSet) || len(right) != len(rightSet) || len(leftSet) != len(rightSet) {
		return false
	}
	for index := range leftSet {
		if leftSet[index] != rightSet[index] {
			return false
		}
	}
	return true
}

func scopeEmpty(scope Scope) bool {
	return len(scope.Repositories)+len(scope.Paths)+len(scope.PolicyIDs)+len(scope.ScenarioIDs)+len(scope.ContractIDs)+len(scope.DBEntities)+len(scope.MigrationSlots)+len(scope.UIFlows)+len(scope.DependencyMajors)+len(scope.RootPointers) == 0
}

func safeRelativePath(value string) bool {
	if value == "" || filepath.IsAbs(value) {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	return clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}
