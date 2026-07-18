package work

import (
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/evidence"
)

// State is normalized live work status owned by the selected provider.
type State string

const (
	Proposed   State = "proposed"
	ReadyState State = "ready"
	InProgress State = "in_progress"
	Blocked    State = "blocked"
	Review     State = "review"
	Integrated State = "integrated"
	Done       State = "done"
)

// LiveState is a provider-neutral, freshly reconciled lifecycle observation.
type LiveState struct {
	Status                State
	Owner                 string
	Revision              string
	Confirmed             bool
	Children              map[string]State
	RootPointersConfirmed bool
}

// Transition verifies evidence and lifecycle invariants without mutating a provider.
func Transition(definition Definition, live LiveState, records []evidence.Record, target State) domain.Result {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: "work.transition", OperationID: "work-transition-" + strings.ReplaceAll(definition.ID, ".", "-"), Status: domain.StatusBlocked, ExitCode: domain.ExitVerification, Summary: "Work lifecycle transition is not yet proven."}
	blockers := []domain.Item{}
	add := func(code, message string, refs ...string) {
		blockers = append(blockers, domain.Item{Code: code, Message: message, Refs: refs})
	}
	if len(ValidateDefinition(definition)) > 0 || definition.Fingerprint == "" || definition.Fingerprint != Fingerprint(definition) {
		add("work.definition-invalid", "Current executable work definition is invalid or stale.", definition.ID)
	}
	if target != ReadyState && target != Blocked && (!live.Confirmed || strings.TrimSpace(live.Owner) == "" || strings.TrimSpace(live.Revision) == "") {
		add("work.live-state-unconfirmed", "Current provider owner and revision must be freshly confirmed.", definition.ID)
	}
	if !allowedTransition(live.Status, target) {
		add("work.transition-invalid", "Requested lifecycle transition is not allowed from the current live state.", string(live.Status), string(target))
	}
	validRecords := map[string][]evidence.Record{}
	for _, record := range records {
		if record.WorkID != definition.ID || record.DefinitionFingerprint != definition.Fingerprint || record.ExitCode != 0 {
			continue
		}
		validRecords[record.Kind] = append(validRecords[record.Kind], record)
	}
	if target == Review && len(validRecords["test"]) == 0 {
		add("work.implementation-evidence-required", "Review requires passing implementation evidence bound to the current definition.", definition.ID)
	}
	if target == Integrated {
		if len(validRecords["integration"]) == 0 && len(validRecords["child-merge"]) == 0 {
			add("work.integration-evidence-required", "Integrated state requires child merge or integration evidence.", definition.ID)
		}
		for _, workspaceID := range definition.Workspaces {
			if !workspaceProven(validRecords, workspaceID) {
				add("work.workspace-not-integrated", "Affected workspace has no current integration evidence.", workspaceID)
			}
		}
	}
	if target == Done {
		for childID, state := range live.Children {
			if state != Integrated && state != Done {
				add("work.children-not-integrated", "Child work must be integrated before the parent is done.", childID)
			}
		}
		for _, kind := range definition.Evidence.Kinds {
			if len(validRecords[kind]) == 0 {
				add("work.required-evidence-missing", "Required current evidence is missing.", kind)
			}
		}
		if definition.Evidence.IntegrationRequired && len(validRecords["integration"])+len(validRecords["child-merge"]) == 0 {
			add("work.integration-evidence-required", "Done state requires current integration evidence.")
		}
		if definition.Evidence.UserValidation && len(validRecords["user"]) == 0 {
			add("work.user-validation-required", "Done state requires user validation of the same implementation.")
		}
		if definition.Evidence.MigrationRequired && len(validRecords["migration"]) == 0 {
			add("work.migration-evidence-required", "Done state requires migration evidence.")
		}
		if definition.Evidence.RollbackRequired && len(validRecords["rollback"]) == 0 {
			add("work.rollback-evidence-required", "Done state requires rollback evidence.")
		}
		if len(definition.Scope.RootPointers) > 0 && !live.RootPointersConfirmed && len(validRecords["root-pointer"]) == 0 {
			add("work.root-pointer-evidence-required", "Done state requires exact root pointer evidence.")
		}
	}
	result.Blockers = normalizeLifecycleItems(blockers)
	if len(result.Blockers) > 0 {
		return result
	}
	result.Status, result.ExitCode, result.Summary = domain.StatusPassed, domain.ExitSuccess, "Lifecycle transition is supported by current provider state and commit-bound evidence."
	result.Facts = []domain.Item{{Code: "work.transition", Message: string(target), Refs: []string{definition.ID, live.Revision}}}
	return result
}

func allowedTransition(current, target State) bool {
	if current == target {
		return true
	}
	switch current {
	case Proposed:
		return target == ReadyState
	case ReadyState:
		return target == InProgress || target == Blocked
	case InProgress:
		return target == Review || target == Blocked
	case Blocked:
		return target == InProgress
	case Review:
		return target == InProgress || target == Integrated || target == Blocked
	case Integrated:
		return target == Review || target == Done || target == Blocked
	default:
		return false
	}
}

func workspaceProven(records map[string][]evidence.Record, workspaceID string) bool {
	for _, kind := range []string{"integration", "child-merge"} {
		for _, record := range records[kind] {
			if record.WorkspaceID == workspaceID {
				return true
			}
		}
	}
	return false
}

func normalizeLifecycleItems(items []domain.Item) []domain.Item {
	for index := range items {
		sort.Strings(items[index].Refs)
	}
	sort.Slice(items, func(left, right int) bool {
		if items[left].Code == items[right].Code {
			return strings.Join(items[left].Refs, "\x00") < strings.Join(items[right].Refs, "\x00")
		}
		return items[left].Code < items[right].Code
	})
	return items
}
