package provider

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
)

const MaxLiveSnapshotAge = 15 * time.Minute

var providerDigest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// Reconcile compares canonical expectation, stable mapping, and one freshly observed snapshot.
func Reconcile(expectation Expectation, mapping Mapping, snapshot Snapshot, now time.Time) State {
	state := State{Confidence: Confirmed, Provider: snapshot.Provider, ItemID: snapshot.ItemID, Revision: snapshot.Revision, Status: snapshot.Status, Owner: snapshot.Owner, Issues: []domain.Item{}}
	add := func(code, message string, refs ...string) {
		state.Issues = append(state.Issues, domain.Item{Code: code, Message: message, Refs: refs})
	}
	if mapping.SchemaVersion != 1 || snapshot.SchemaVersion != 1 {
		add("provider.schema-invalid", "Provider mapping and snapshot must use schema_version 1.")
	}
	if expectation.WorkID == "" || mapping.WorkID != expectation.WorkID {
		add("provider.work-mismatch", "Provider mapping does not belong to the expected work definition.", expectation.WorkID, mapping.WorkID)
	}
	if mapping.Provider == "" || mapping.Provider != snapshot.Provider {
		add("provider.identity-mismatch", "Snapshot provider differs from the selected mapping.", mapping.Provider, snapshot.Provider)
	}
	if mapping.ItemID == "" || mapping.ItemID != snapshot.ItemID {
		add("provider.item-mismatch", "Snapshot item differs from the selected mapping.", mapping.ItemID, snapshot.ItemID)
	}
	if expectation.DefinitionFingerprint == "" || mapping.DefinitionFingerprint != expectation.DefinitionFingerprint || snapshot.DefinitionFingerprint != expectation.DefinitionFingerprint {
		add("provider.definition-drift", "Provider state belongs to a different work-definition fingerprint.")
	}
	if !sameSet(expectation.Dependencies, mapKeys(mapping.DependencyItems)) || !sameSet(snapshot.Dependencies, mapValues(mapping.DependencyItems)) {
		add("provider.dependency-drift", "Provider dependencies differ from canonical work dependencies.")
	}
	if len(expectation.Dependencies) > 0 && !snapshot.Capabilities.Dependencies {
		add("provider.capability-missing", "Selected provider cannot represent required dependencies.", "dependencies")
	}
	if snapshot.Capabilities.Revision && strings.TrimSpace(snapshot.Revision) == "" {
		add("provider.revision-missing", "Provider claims revision support but supplied no concurrency revision.")
	}
	switch snapshot.Capabilities.Claim {
	case "atomic", "verified", "advisory", "none":
	default:
		add("provider.capability-invalid", "Provider claim capability is invalid.", snapshot.Capabilities.Claim)
	}
	if activeStatus(snapshot.Status) && snapshot.Capabilities.Claim != "atomic" && snapshot.Capabilities.Claim != "verified" {
		add("provider.claim-unverified", "The provider cannot confirm exclusive active ownership.", snapshot.Capabilities.Claim)
	}
	if activeStatus(snapshot.Status) && snapshot.Capabilities.Claim != "none" && strings.TrimSpace(snapshot.Owner) == "" {
		add("provider.owner-missing", "Active provider work has no observable owner.")
	}
	if !knownStatus(snapshot.Status) {
		add("provider.status-unknown", "Provider status is not mapped to the normalized lifecycle.", snapshot.Status)
	}
	if snapshot.FetchedAt.IsZero() || now.Sub(snapshot.FetchedAt) > MaxLiveSnapshotAge || snapshot.FetchedAt.After(now.Add(2*time.Minute)) {
		add("provider.snapshot-stale", "Provider snapshot is too old or from the future to coordinate work.")
	}
	if snapshot.Source != "connector-live" && snapshot.Source != "git-local-remote" {
		add("provider.not-live", "A cached or manually copied snapshot cannot confirm live provider state.", snapshot.Source)
	}
	if !providerDigest.MatchString(snapshot.RawHash) {
		add("provider.raw-hash-invalid", "Provider connector output hash is missing or invalid.")
	}
	state.Issues = normalizeProviderIssues(state.Issues)
	if len(state.Issues) > 0 {
		state.Confidence = Unknown
	}
	return state
}

func knownStatus(value string) bool {
	switch value {
	case "proposed", "ready", "in_progress", "blocked", "review", "integrated", "done", "closed":
		return true
	default:
		return false
	}
}

func activeStatus(value string) bool {
	return value == "in_progress" || value == "review" || value == "integrated"
}

func mapKeys(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	return result
}

func mapValues(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func sameSet(left, right []string) bool {
	left = normalizedProviderSet(left)
	right = normalizedProviderSet(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func normalizedProviderSet(values []string) []string {
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

func normalizeProviderIssues(items []domain.Item) []domain.Item {
	for index := range items {
		items[index].Refs = normalizedProviderSet(items[index].Refs)
	}
	sort.Slice(items, func(left, right int) bool {
		if items[left].Code == items[right].Code {
			return strings.Join(items[left].Refs, "\x00") < strings.Join(items[right].Refs, "\x00")
		}
		return items[left].Code < items[right].Code
	})
	return items
}
