package integration

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/work"
)

// Plan topologically orders product contracts, providers, consumers, UI, migrations, and root pointers.
func Plan(definitions []work.Definition, providerStates []ProviderState, workspaceStates []WorkspaceState) MergePlan {
	plan := MergePlan{SchemaVersion: 1, Steps: []Step{}, WorkspaceCommits: map[string]string{}, Blockers: []domain.Item{}}
	workspaces := map[string]WorkspaceState{}
	rootID := ""
	for _, state := range workspaceStates {
		if _, duplicate := workspaces[state.ID]; duplicate {
			plan.Blockers = append(plan.Blockers, integrationItem("integrate.workspace-duplicate", "Workspace state is duplicated.", state.ID))
			continue
		}
		workspaces[state.ID] = state
		plan.WorkspaceCommits[state.ID] = state.Commit
		if state.Kind == "root" {
			rootID = state.ID
		}
		if state.Commit == "" || state.Remote == "" || !state.Clean || !state.Published {
			plan.Blockers = append(plan.Blockers, integrationItem("integrate.workspace-unready", "Workspace must be clean, published, and have exact commit and remote identity.", state.ID))
		}
		if state.Kind == "submodule" && (state.ExpectedPointer == "" || state.ExpectedPointer != state.ActualPointer || state.Commit != state.ActualPointer) {
			plan.Blockers = append(plan.Blockers, integrationItem("integrate.pointer-mismatch", "Submodule commit differs from the root pointer.", state.ID, state.ExpectedPointer, state.ActualPointer))
		}
	}
	if rootID == "" {
		plan.Blockers = append(plan.Blockers, integrationItem("integrate.root-missing", "Integration requires one orchestration root workspace."))
	}
	providers := map[string]ProviderState{}
	for _, state := range providerStates {
		if _, duplicate := providers[state.WorkID]; duplicate {
			plan.Blockers = append(plan.Blockers, integrationItem("integrate.provider-duplicate", "Provider state is duplicated.", state.WorkID))
			continue
		}
		providers[state.WorkID] = state
	}
	ordered, orderIssues := orderDefinitions(definitions)
	plan.Blockers = append(plan.Blockers, orderIssues...)
	pointerOwners := map[string]string{}
	lastWorkSteps := map[string][]string{}
	for _, definition := range ordered {
		providerState, found := providers[definition.ID]
		if !found || !providerState.Confirmed || providerState.Revision == "" {
			plan.Blockers = append(plan.Blockers, integrationItem("integrate.provider-unknown", "Work status must be freshly confirmed from the selected live provider.", definition.ID))
		} else {
			if providerState.DefinitionFingerprint != definition.Fingerprint {
				plan.Blockers = append(plan.Blockers, integrationItem("integrate.provider-stale", "Provider state references an older work definition.", definition.ID))
			}
			if providerState.Status != "review" && providerState.Status != "integrated" && providerState.Status != "done" {
				plan.Blockers = append(plan.Blockers, integrationItem("integrate.work-not-reviewable", "Work must reach review before service integration.", definition.ID, providerState.Status))
			}
		}
		dependencySteps := []string{}
		for _, dependency := range definition.Dependencies {
			dependencySteps = append(dependencySteps, lastWorkSteps[dependency]...)
		}
		previous := uniqueIntegrationStrings(dependencySteps)
		appendStep := func(kind StepKind, ref, workspaceID, evidenceKind string) {
			state, exists := workspaces[workspaceID]
			if !exists {
				plan.Blockers = append(plan.Blockers, integrationItem("integrate.workspace-missing", "Integration step references an unavailable workspace.", definition.ID, workspaceID))
				return
			}
			step := Step{
				ID: integrationStepID(definition.ID, kind, ref), Kind: kind, Ref: ref, WorkID: definition.ID, WorkspaceID: workspaceID,
				DefinitionFingerprint: definition.Fingerprint, ProviderRevision: providerState.Revision, Commit: state.Commit,
				RequiredEvidence: evidenceKind, DependsOn: append([]string(nil), previous...),
			}
			plan.Steps = append(plan.Steps, step)
			previous = []string{step.ID}
		}
		for _, contractID := range sortedIntegrationStrings(definition.Scope.ContractIDs) {
			appendStep(ContractStep, contractID, rootID, "review")
		}
		for _, workspaceID := range integrationWorkspaceOrder(definition) {
			kind := "integration"
			if state, exists := workspaces[workspaceID]; exists && (state.Kind == "submodule" || state.Kind == "external") {
				kind = "child-merge"
			}
			appendStep(WorkspaceStep, workspaceID, workspaceID, kind)
		}
		for _, uiID := range sortedIntegrationStrings(definition.Scope.UIFlows) {
			appendStep(UIConnectionStep, uiID, rootID, "integration")
		}
		if definition.Evidence.MigrationRequired {
			for _, slot := range sortedIntegrationStrings(definition.Scope.MigrationSlots) {
				appendStep(MigrationStep, slot, rootID, "migration")
			}
		}
		for _, pointer := range sortedIntegrationStrings(definition.Scope.RootPointers) {
			if owner, exists := pointerOwners[pointer]; exists && owner != definition.ID {
				plan.Blockers = append(plan.Blockers, integrationItem("integrate.pointer-overlap", "Two work items reserve the same root pointer.", pointer, owner, definition.ID))
			} else {
				pointerOwners[pointer] = definition.ID
			}
			appendStep(RootPointerStep, pointer, rootID, "root-pointer")
		}
		lastWorkSteps[definition.ID] = append([]string(nil), previous...)
	}
	plan.Blockers = normalizeIntegrationItems(plan.Blockers)
	return plan
}

func integrationWorkspaceOrder(definition work.Definition) []string {
	declared := map[string]bool{}
	for _, id := range definition.Workspaces {
		declared[id] = true
	}
	result, seen := []string{}, map[string]bool{}
	for _, id := range definition.MergeOrder {
		if declared[id] && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	remaining := []string{}
	for _, id := range definition.Workspaces {
		if !seen[id] {
			remaining = append(remaining, id)
		}
	}
	sort.Strings(remaining)
	return append(result, remaining...)
}

func orderDefinitions(definitions []work.Definition) ([]work.Definition, []domain.Item) {
	byID := map[string]work.Definition{}
	issues := []domain.Item{}
	for _, definition := range definitions {
		if _, duplicate := byID[definition.ID]; duplicate {
			issues = append(issues, integrationItem("integrate.work-duplicate", "Work definition is duplicated.", definition.ID))
		}
		byID[definition.ID] = definition
	}
	state := map[string]int{}
	result := []work.Definition{}
	var visit func(string)
	visit = func(id string) {
		if state[id] == 2 {
			return
		}
		if state[id] == 1 {
			issues = append(issues, integrationItem("integrate.dependency-cycle", "Work dependencies contain a cycle.", id))
			return
		}
		definition, exists := byID[id]
		if !exists {
			issues = append(issues, integrationItem("integrate.dependency-missing", "Work dependency is missing.", id))
			return
		}
		state[id] = 1
		dependencies := append([]string(nil), definition.Dependencies...)
		sort.Strings(dependencies)
		for _, dependency := range dependencies {
			visit(dependency)
		}
		state[id] = 2
		result = append(result, definition)
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		visit(id)
	}
	return result, issues
}

func integrationStepID(workID string, kind StepKind, ref string) string {
	value := strings.ToLower(string(kind) + "." + ref)
	replacer := strings.NewReplacer("_", "-", "/", "-", " ", "-", ":", "-")
	value = replacer.Replace(value)
	return "integration." + strings.TrimPrefix(workID, "work.") + "." + value
}

func integrationItem(code, message string, refs ...string) domain.Item {
	return domain.Item{Code: code, Message: message, Refs: refs}
}

func sortedIntegrationStrings(values []string) []string {
	result := uniqueIntegrationStrings(values)
	sort.Strings(result)
	return result
}

func uniqueIntegrationStrings(values []string) []string {
	result, seen := []string{}, map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func normalizeIntegrationItems(items []domain.Item) []domain.Item {
	sort.Slice(items, func(left, right int) bool {
		if items[left].Code == items[right].Code {
			return fmt.Sprint(items[left].Refs) < fmt.Sprint(items[right].Refs)
		}
		return items[left].Code < items[right].Code
	})
	return items
}
