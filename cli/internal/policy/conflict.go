package policy

import (
	"path"
	"sort"
	"strings"
	"time"

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"fullstack-orchestrator/cli/internal/domain"
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
	Repository       string
	Workspace        string
	Paths            []string
	PolicyIDs        []string
	ScenarioIDs      []string
	ContractIDs      []string
	DBEntities       []string
	MigrationSlots   []string
	UIFlows          []string
	DependencyMajors []string
	StableIDs        []string
	RootPointer      bool
	Now              time.Time
}

// Claim declares a collaborator's intended scope. It is not a distributed lock.
type Claim struct {
	ID               string
	Repository       string
	Workspace        string
	Owner            string
	Branch           string
	Paths            []string
	PolicyIDs        []string
	ScenarioIDs      []string
	ContractIDs      []string
	DBEntities       []string
	MigrationSlots   []string
	UIFlows          []string
	DependencyMajors []string
	RootPointer      bool
	StartsAt         time.Time
	ExpiresAt        time.Time
	Observable       bool
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
			promote(&report, ConflictBlock, reason("conflict.contract", "The same shared contract is being changed.", claim))
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
	priority := map[ConflictLevel]int{ConflictClear: 0, ConflictUnknown: 1, ConflictCoordinate: 2, ConflictBlock: 3}
	if priority[level] > priority[report.Level] {
		report.Level = level
	}
	report.Reasons = append(report.Reasons, item)
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
