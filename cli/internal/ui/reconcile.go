package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/schema"
)

// ReconcileState describes authority-aware effects without changing canonical UI code.
type ReconcileState struct {
	Changed          bool     `json:"changed"`
	RequiresApproval bool     `json:"requires_approval"`
	StaleRefs        []string `json:"stale_refs"`
	Blockers         []string `json:"blockers"`
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
