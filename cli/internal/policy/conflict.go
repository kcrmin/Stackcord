package policy

import (
	"path"
	"sort"
	"strings"
	"time"

	contextpkg "github.com/kcrmin/Stackcord/cli/internal/context"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
)

// ConflictLevel is the pre-implementation coordination decision.
type ConflictLevel string

const (
	ConflictClear      ConflictLevel = "clear"
	ConflictCoordinate ConflictLevel = "coordinate"
	ConflictBlock      ConflictLevel = "block"
	ConflictUnknown    ConflictLevel = "unknown"
)

// Candidate is the normalized scope of proposed work.
type Candidate struct {
	Repository       string    `json:"repository" yaml:"repository"`
	Workspace        string    `json:"workspace,omitempty" yaml:"workspace,omitempty"`
	Paths            []string  `json:"paths" yaml:"paths"`
	PolicyIDs        []string  `json:"policy_ids" yaml:"policy_ids"`
	ScenarioIDs      []string  `json:"scenario_ids" yaml:"scenario_ids"`
	ContractIDs      []string  `json:"contract_ids" yaml:"contract_ids"`
	DBEntities       []string  `json:"db_entities" yaml:"db_entities"`
	MigrationSlots   []string  `json:"migration_slots" yaml:"migration_slots"`
	UIFlows          []string  `json:"ui_flows" yaml:"ui_flows"`
	DependencyMajors []string  `json:"dependency_majors" yaml:"dependency_majors"`
	StableIDs        []string  `json:"stable_ids,omitempty" yaml:"stable_ids,omitempty"`
	RootPointer      bool      `json:"root_pointer" yaml:"root_pointer"`
	Now              time.Time `json:"now" yaml:"now"`
}

// Claim declares a collaborator's intended scope. It is not a distributed lock.
type Claim struct {
	SchemaVersion    int       `json:"schema_version" yaml:"schema_version"`
	ID               string    `json:"id" yaml:"id"`
	WorkID           string    `json:"work_id" yaml:"work_id"`
	Repository       string    `json:"repository" yaml:"repository"`
	Workspace        string    `json:"workspace,omitempty" yaml:"workspace,omitempty"`
	Owner            string    `json:"owner" yaml:"owner"`
	Branch           string    `json:"branch" yaml:"branch"`
	Paths            []string  `json:"paths" yaml:"paths"`
	PolicyIDs        []string  `json:"policy_ids" yaml:"policy_ids"`
	ScenarioIDs      []string  `json:"scenario_ids" yaml:"scenario_ids"`
	ContractIDs      []string  `json:"contract_ids" yaml:"contract_ids"`
	DBEntities       []string  `json:"db_entities" yaml:"db_entities"`
	MigrationSlots   []string  `json:"migration_slots" yaml:"migration_slots"`
	UIFlows          []string  `json:"ui_flows" yaml:"ui_flows"`
	DependencyMajors []string  `json:"dependency_majors" yaml:"dependency_majors"`
	StableIDs        []string  `json:"stable_ids,omitempty" yaml:"stable_ids,omitempty"`
	RootPointer      bool      `json:"root_pointer" yaml:"root_pointer"`
	StartsAt         time.Time `json:"starts_at" yaml:"starts_at"`
	ExpiresAt        time.Time `json:"expires_at" yaml:"expires_at"`
	Observable       bool      `json:"observable,omitempty" yaml:"observable,omitempty"`
}

// ConflictReport contains stable reasons and one concrete safe next action.
type ConflictReport struct {
	Level      ConflictLevel `json:"level"`
	Reasons    []domain.Item `json:"reasons"`
	NextAction string        `json:"next_action"`
}

// CheckConflict compares normalized semantic and filesystem scopes.
func CheckConflict(candidate Candidate, claims []Claim, snapshot contextpkg.Snapshot) ConflictReport {
	if candidate.Now.IsZero() {
		candidate.Now = time.Now().UTC()
	}
	report := ConflictReport{Level: ConflictClear, Reasons: []domain.Item{}}
	uncertainContracts := map[string]bool{}
	for _, contractID := range candidate.ContractIDs {
		if contractContextUnknown(contractID, snapshot) {
			uncertainContracts[contractID] = true
			promote(&report, ConflictUnknown, domain.Item{Code: "conflict.contract-unknown", Message: "Contract registry identity is missing, stale, or unknown.", Refs: []string{contractID}})
		}
	}
	for _, stableID := range candidate.StableIDs {
		if contains(snapshot.Stale, stableID) || contains(snapshot.Unknown, stableID) {
			promote(&report, ConflictUnknown, domain.Item{Code: "conflict.context-unknown", Message: "Related product context is stale or unknown.", Refs: []string{stableID}})
		}
	}
	for _, claim := range claims {
		if !claim.ExpiresAt.IsZero() && !claim.ExpiresAt.After(candidate.Now) {
			continue
		}
		if claim.Repository != "" && candidate.Repository != "" && claim.Repository != candidate.Repository {
			continue
		}
		if !claim.Observable {
			promote(&report, ConflictUnknown, domain.Item{Code: "conflict.claim-unobservable", Message: "Active remote claim scope cannot be verified.", Refs: []string{claim.ID}})
			continue
		}
		if pathSetsOverlap(candidate.Paths, claim.Paths) {
			promote(&report, ConflictCoordinate, reason("conflict.path-overlap", "Filesystem scopes overlap and ownership or merge order must be agreed.", claim))
		}
		if intersects(candidate.PolicyIDs, claim.PolicyIDs) || intersects(candidate.ScenarioIDs, claim.ScenarioIDs) {
			promote(&report, ConflictBlock, reason("conflict.product-meaning", "The same approved product meaning is being changed.", claim))
		}
		if intersects(candidate.ContractIDs, claim.ContractIDs) {
			shared := intersection(candidate.ContractIDs, claim.ContractIDs)
			unknown := false
			level := ConflictCoordinate
			for _, contractID := range shared {
				if uncertainContracts[contractID] || contractContextUnknown(contractID, snapshot) {
					unknown = true
					continue
				}
				if contractChangeBlocks(contractID, snapshot) {
					level = ConflictBlock
				}
			}
			if unknown {
				promote(&report, ConflictUnknown, reason("conflict.contract-unknown", "Shared contract scope cannot be classified from current registry evidence.", claim))
			} else if level == ConflictBlock {
				promote(&report, ConflictBlock, reason("conflict.contract", "The same product, business, behavior, or data obligation is being changed.", claim))
			} else {
				promote(&report, ConflictCoordinate, reason("conflict.contract-consumer", "Compatible interface work overlaps and needs explicit provider-consumer order.", claim))
			}
		}
		for _, contractID := range claim.ContractIDs {
			if intersects(candidate.StableIDs, snapshot.Impact[contractID]) {
				if contractContextUnknown(contractID, snapshot) {
					promote(&report, ConflictUnknown, reason("conflict.contract-unknown", "Consumer impact cannot be trusted without current registry evidence.", claim))
				} else {
					promote(&report, ConflictCoordinate, reason("conflict.contract-consumer", "A contract and one of its consumers are changing concurrently.", claim))
				}
			}
		}
		for _, contractID := range candidate.ContractIDs {
			if intersects(claim.StableIDs, snapshot.Impact[contractID]) {
				if contractContextUnknown(contractID, snapshot) {
					promote(&report, ConflictUnknown, reason("conflict.contract-unknown", "Consumer impact cannot be trusted without current registry evidence.", claim))
				} else {
					promote(&report, ConflictCoordinate, reason("conflict.contract-consumer", "A contract and one of its consumers are changing concurrently.", claim))
				}
			}
		}
		if intersects(candidate.DBEntities, claim.DBEntities) {
			promote(&report, ConflictCoordinate, reason("conflict.database-entity", "Database entity scopes overlap.", claim))
		}
		if intersects(candidate.MigrationSlots, claim.MigrationSlots) {
			promote(&report, ConflictBlock, reason("conflict.migration-slot", "The same migration slot is reserved.", claim))
		}
		if intersects(candidate.UIFlows, claim.UIFlows) {
			promote(&report, ConflictCoordinate, reason("conflict.ui-flow", "The same UI flow baseline is being changed.", claim))
		}
		if intersects(candidate.DependencyMajors, claim.DependencyMajors) {
			promote(&report, ConflictCoordinate, reason("conflict.dependency-major", "The same dependency major transition is in progress.", claim))
		}
		if intersects(candidate.StableIDs, claim.StableIDs) {
			promote(&report, ConflictBlock, reason("conflict.stable-id", "The same stable product meaning is being changed.", claim))
		}
		if candidate.RootPointer && claim.RootPointer {
			promote(&report, ConflictCoordinate, reason("conflict.root-pointer", "Root pointer integration order overlaps.", claim))
		}
	}
	sort.SliceStable(report.Reasons, func(i, j int) bool {
		if report.Reasons[i].Code == report.Reasons[j].Code {
			return strings.Join(report.Reasons[i].Refs, "\x00") < strings.Join(report.Reasons[j].Refs, "\x00")
		}
		return report.Reasons[i].Code < report.Reasons[j].Code
	})
	switch report.Level {
	case ConflictBlock:
		report.NextAction = "Unify the shared policy, contract, or migration design and approve one merge order before implementation."
	case ConflictCoordinate:
		report.NextAction = "Assign explicit scopes and merge order, then refresh both claims before implementation."
	case ConflictUnknown:
		report.NextAction = "Restore provider visibility or refresh stale canonical context before implementation."
	}
	return report
}

func promote(report *ConflictReport, level ConflictLevel, item domain.Item) {
	priority := map[ConflictLevel]int{ConflictClear: 0, ConflictCoordinate: 1, ConflictUnknown: 2, ConflictBlock: 3}
	if priority[level] > priority[report.Level] {
		report.Level = level
	}
	report.Reasons = append(report.Reasons, item)
}

func contractContextUnknown(id string, snapshot contextpkg.Snapshot) bool {
	entry, exists := snapshot.Index[id]
	if !exists || !entry.ContractRegistered {
		return true
	}
	if entry.Status == "stale" || entry.Status == "unknown" || contains(snapshot.Stale, id) {
		return true
	}
	for _, unknown := range snapshot.Unknown {
		if unknown == id || strings.HasPrefix(unknown, id+".") {
			return true
		}
	}
	return false
}

func contractChangeBlocks(id string, snapshot contextpkg.Snapshot) bool {
	entry := snapshot.Index[id]
	switch entry.Kind {
	case "interface":
		return false
	case "product", "business", "behavior", "data", "contract":
		return true
	default:
		return true
	}
}

func intersection(left, right []string) []string {
	set := make(map[string]bool, len(left))
	for _, value := range left {
		set[normalizeScope(value)] = true
	}
	result := []string{}
	for _, value := range right {
		value = normalizeScope(value)
		if set[value] {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func reason(code, message string, claim Claim) domain.Item {
	refs := []string{claim.ID}
	if claim.Owner != "" {
		refs = append(refs, claim.Owner)
	}
	if claim.Branch != "" {
		refs = append(refs, claim.Branch)
	}
	return domain.Item{Code: code, Message: message, Refs: refs}
}

func intersects(left, right []string) bool {
	set := make(map[string]struct{}, len(left))
	for _, value := range left {
		set[normalizeScope(value)] = struct{}{}
	}
	for _, value := range right {
		if _, exists := set[normalizeScope(value)]; exists {
			return true
		}
	}
	return false
}

func pathSetsOverlap(left, right []string) bool {
	for _, first := range left {
		for _, second := range right {
			if pathScopeOverlap(first, second) {
				return true
			}
		}
	}
	return false
}

func pathScopeOverlap(left, right string) bool {
	left, right = normalizeScope(left), normalizeScope(right)
	leftPrefix := strings.TrimSuffix(strings.Split(left, "*")[0], "/")
	rightPrefix := strings.TrimSuffix(strings.Split(right, "*")[0], "/")
	if leftPrefix == rightPrefix || strings.HasPrefix(leftPrefix, rightPrefix+"/") || strings.HasPrefix(rightPrefix, leftPrefix+"/") {
		return true
	}
	leftMatches, _ := path.Match(left, rightPrefix)
	rightMatches, _ := path.Match(right, leftPrefix)
	return leftMatches || rightMatches
}

func normalizeScope(value string) string {
	value = filepathSlash(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "./")
	return strings.TrimSuffix(value, "/")
}

func filepathSlash(value string) string { return strings.ReplaceAll(value, "\\", "/") }

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
