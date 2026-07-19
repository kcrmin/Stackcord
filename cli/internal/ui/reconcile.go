package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
	"github.com/kcrmin/Stackcord/cli/internal/work"
	"go.yaml.in/yaml/v3"
)

// ReconcileState describes authority-aware effects without changing canonical UI code.
type ReconcileState struct {
	Changed          bool     `json:"changed"`
	RequiresApproval bool     `json:"requires_approval"`
	StaleRefs        []string `json:"stale_refs"`
	Blockers         []string `json:"blockers"`
}

// AcceptIntegratedBaseline plans the tracked acknowledgement included with implemented UI work.
func AcceptIntegratedBaseline(root, id string, definition work.Definition) (Registration, operation.Plan, error) {
	registration, err := LoadRegistration(root, id)
	if err != nil {
		return Registration{}, operation.Plan{}, err
	}
	if registration.Authority != "seed" && registration.Authority != "canonical" {
		return Registration{}, operation.Plan{}, fmt.Errorf("reference-only UI sources have no canonical baseline to accept")
	}
	if definition.Readiness != work.Ready || !definition.Evidence.IntegrationRequired || !strings.HasPrefix(definition.ID, "work.") {
		return Registration{}, operation.Plan{}, fmt.Errorf("UI baseline acceptance requires ready work with integration evidence")
	}
	uiScope := append(append([]string(nil), definition.Scope.UIFlows...), definition.Refs...)
	consumerScope := append(append([]string(nil), definition.Workspaces...), definition.Refs...)
	for _, ref := range registration.MappedRefs {
		if !containsUIValue(uiScope, ref) {
			return Registration{}, operation.Plan{}, fmt.Errorf("UI mapping %s is outside work scope", ref)
		}
	}
	for _, consumer := range registration.Consumers {
		if !containsUIValue(consumerScope, consumer) {
			return Registration{}, operation.Plan{}, fmt.Errorf("UI consumer %s is outside work scope", consumer)
		}
	}
	if registration.BaselineFingerprint == registration.ContentHash {
		return Registration{}, operation.Plan{}, fmt.Errorf("UI source baseline is already current")
	}
	registration.BaselineFingerprint = registration.ContentHash
	data, err := yaml.Marshal(registration)
	if err != nil {
		return Registration{}, operation.Plan{}, err
	}
	plan := operation.Plan{ID: "ui-integrate-" + strings.ReplaceAll(id, ".", "-"), Root: root, Files: []operation.FileChange{{Path: filepath.ToSlash(filepath.Join(".harness", "sources", "ui", id+".yaml")), Content: data, Mode: 0o644}}}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return registration, plan, err
}

func containsUIValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// Reconcile compares stable registrations. Authority can only change as a separate product decision.
func Reconcile(current, next Registration) ReconcileState {
	state := ReconcileState{StaleRefs: []string{}, Blockers: []string{}}
	if current.ID != next.ID || current.Kind != next.Kind {
		state.Blockers = append(state.Blockers, "ui.source-identity")
	}
	if current.Authority != next.Authority {
		state.Blockers = append(state.Blockers, "ui.authority-change")
	}
	if current.ContentHash == "" || next.ContentHash == "" || !strings.HasPrefix(current.ContentHash, "sha256:") || !strings.HasPrefix(next.ContentHash, "sha256:") {
		state.Blockers = append(state.Blockers, "ui.content-identity")
	}
	sort.Strings(state.Blockers)
	if len(state.Blockers) > 0 {
		return state
	}
	state.Changed = current.ContentHash != next.ContentHash || current.SourceVersion != next.SourceVersion || current.License != next.License
	state.RequiresApproval = state.Changed
	if state.Changed && !next.FetchedAt.After(current.FetchedAt) {
		state.Blockers = append(state.Blockers, "ui.source-not-newer")
		sort.Strings(state.Blockers)
		return state
	}
	if state.Changed && (current.Authority == "seed" || current.Authority == "canonical") {
		state.StaleRefs = normalizedUIRefs(append(append([]string(nil), current.MappedRefs...), current.Consumers...))
	}
	return state
}

// LoadRegistration reads one committed registration without following links.
func LoadRegistration(root, id string) (Registration, error) {
	if !sourceIDPattern.MatchString(id) {
		return Registration{}, fmt.Errorf("UI source ID is invalid")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return Registration{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return Registration{}, err
	}
	path := filepath.Join(root, ".harness", "sources", "ui", id+".yaml")
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Registration{}, fmt.Errorf("UI registration must be a regular non-symlink file")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(path) {
		return Registration{}, fmt.Errorf("UI registration cannot use symlinked storage")
	}
	registration, err := schema.LoadYAML[Registration](resolved)
	if err != nil {
		return Registration{}, err
	}
	if registration.BaselineFingerprint == "" {
		registration.BaselineFingerprint = registration.ContentHash
	}
	if issues := schema.Validate("external-source", registration); len(issues) > 0 {
		return Registration{}, fmt.Errorf("validate UI registration: %s", issues[0].Message)
	}
	return registration, nil
}
