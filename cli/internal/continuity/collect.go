package continuity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	contextpkg "github.com/kcrmin/Stackcord/cli/internal/context"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/gitx"
	"github.com/kcrmin/Stackcord/cli/internal/governance"
	providerpkg "github.com/kcrmin/Stackcord/cli/internal/provider"
	"github.com/kcrmin/Stackcord/cli/internal/release"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
	workpkg "github.com/kcrmin/Stackcord/cli/internal/work"
	"github.com/kcrmin/Stackcord/cli/internal/workspace"
)

// Collect rebuilds one service continuity snapshot from actual and canonical repository state.
func Collect(ctx context.Context, start string, _ Options) Snapshot {
	snapshot := Snapshot{
		SchemaVersion: 1,
		Overall:       Blocked,
		Context: contextpkg.Snapshot{
			SchemaVersion: 1,
			Index:         map[string]contextpkg.IndexEntry{},
			Impact:        map[string][]string{},
			Stale:         []string{},
			Unknown:       []string{},
		},
		Workspaces:  []workspace.State{},
		Provider:    ProviderView{Confidence: Unknown},
		Governance:  governance.Report{Status: governance.Unknown, Authorities: []string{}, Approvers: []string{}, Issues: []domain.Item{}},
		ActiveWork:  []WorkView{},
		Release:     ReleaseView{Confidence: Unknown},
		Issues:      []domain.Item{},
		NextActions: []domain.Item{},
	}
	located, err := workspace.FindRoot(ctx, start)
	if err != nil {
		code := "project.not-found"
		var incomplete *workspace.IncompleteContextError
		if errors.As(err, &incomplete) {
			code = "project.root-unavailable"
			snapshot.ProjectID = incomplete.ProjectID
			snapshot.CurrentWorkspaceID = incomplete.WorkspaceID
		}
		snapshot.Issues = []domain.Item{{Code: code, Message: err.Error()}}
		snapshot.NextActions = nextActions(snapshot)
		return snapshot
	}
	snapshot.ProjectID = located.Manifest.ProjectID
	snapshot.Root = located.Path
	snapshot.CurrentWorkspaceID = located.CurrentWorkspaceID

	rootGit, gitErr := gitx.Inspect(ctx, located.Path)
	if gitErr != nil {
		snapshot.Issues = append(snapshot.Issues, domain.Item{Code: "workspace.git-unknown", Message: "Git state is unavailable for the orchestration root.", Refs: []string{"workspace.root"}})
	}
	states, workspaceIssues := collectWorkspaceStates(ctx, located.Path, located.Manifest, rootGit)
	snapshot.Workspaces = states
	snapshot.Issues = append(snapshot.Issues, workspaceIssues...)

	contextSnapshot, contextIssues := contextpkg.Refresh(ctx, located.Path, contextpkg.ReadOnly)
	snapshot.Context = contextSnapshot
	for _, item := range contextIssues {
		confidenceCode := "context.warning"
		if strings.HasPrefix(item.Code, "context.error") {
			confidenceCode = item.Code
		}
		snapshot.Issues = append(snapshot.Issues, domain.Item{Code: confidenceCode, Message: item.Message, Refs: item.Refs})
	}
	if len(contextSnapshot.Stale) > 0 {
		snapshot.Issues = append(snapshot.Issues, domain.Item{Code: "context.stale", Message: "Canonical dependents are stale.", Refs: contextSnapshot.Stale})
	}
	if len(contextSnapshot.Unknown) > 0 {
		snapshot.Issues = append(snapshot.Issues, domain.Item{Code: "context.unknown", Message: "Canonical context has unresolved references or semantic state.", Refs: contextSnapshot.Unknown})
	}
	snapshot.CanonicalFingerprint = canonicalFingerprint(snapshot.ProjectID, contextSnapshot)
	snapshot.Governance = governance.Check(ctx, located.Path, "", time.Now().UTC())
	snapshot.Issues = append(snapshot.Issues, snapshot.Governance.Issues...)

	snapshot.ActiveWork, err = collectActiveWork(located.Path)
	if err != nil {
		snapshot.Issues = append(snapshot.Issues, domain.Item{Code: "work.definition-invalid", Message: err.Error()})
	}
	if definitions, definitionErr := workpkg.LoadDefinitions(located.Path); definitionErr != nil {
		snapshot.Issues = append(snapshot.Issues, domain.Item{Code: "work.definition-invalid", Message: definitionErr.Error()})
	} else {
		for _, definition := range definitions {
			for id, expected := range definition.UIBaselines {
				entry, exists := snapshot.Context.Index[id]
				if !exists || entry.Kind != "ui-baseline" || entry.Fingerprint != expected {
					snapshot.Issues = append(snapshot.Issues, domain.Item{Code: "work.ui-baseline-stale", Message: "Active work references an older or missing UI baseline.", Refs: []string{definition.ID, id, expected, entry.Fingerprint}})
				}
			}
		}
	}
	var liveClaims map[string]providerpkg.GitLocalClaim
	snapshot.Provider, liveClaims, err = collectProvider(ctx, located.Path)
	if err != nil {
		snapshot.Issues = append(snapshot.Issues, domain.Item{Code: "provider.live-unknown", Message: "The selected task provider could not be read safely.", Refs: []string{snapshot.Provider.Name}})
	} else if snapshot.Provider.Confidence != Confirmed {
		snapshot.Issues = append(snapshot.Issues, domain.Item{Code: "provider.live-unknown", Message: "The selected task provider has not been freshly reconciled.", Refs: []string{snapshot.Provider.Name}})
	}
	snapshot.ActiveWork = mergeLiveWork(snapshot.ActiveWork, liveClaims)
	snapshot.Release = collectRelease(located.Path)
	snapshot.Issues = normalizeIssues(snapshot.Issues)
	snapshot.Overall = overallConfidence(snapshot.Issues)
	snapshot.NextActions = nextActions(snapshot)
	return snapshot
}

func collectWorkspaceStates(ctx context.Context, root string, manifest workspace.Manifest, rootGit gitx.State) ([]workspace.State, []domain.Item) {
	states := make([]workspace.State, 0, len(manifest.Workspaces))
	issues := []domain.Item{}
	for _, entry := range manifest.Workspaces {
		state := workspace.State{Entry: entry, Confidence: string(Confirmed), Issues: []domain.Item{}}
		workspaceRoot := filepath.Join(root, filepath.FromSlash(entry.Path))
		switch entry.Kind {
		case "root", "directory":
			state.Git = rootGit
			if rootGit.Root == "" {
				state.Confidence = string(Unknown)
			}
		case "submodule":
			submodule, found := findSubmodule(rootGit.Submodules, entry.Path)
			if !found || !submodule.Initialized {
				state.Confidence = string(Unknown)
				item := domain.Item{Code: "workspace.missing", Message: "A declared submodule is not initialized at its pinned commit.", Refs: []string{entry.ID, entry.Path}}
				issues = append(issues, item)
				state.Issues = append(state.Issues, item)
				states = append(states, state)
				continue
			}
			state.ExpectedSHA = submodule.ExpectedSHA
			if submodule.PointerDiff {
				state.Confidence = string(Blocked)
				item := domain.Item{Code: "workspace.pointer-mismatch", Message: "Child HEAD differs from the root gitlink pointer.", Refs: []string{entry.ID, entry.Path, submodule.ExpectedSHA, submodule.Head}}
				issues = append(issues, item)
				state.Issues = append(state.Issues, item)
			}
			childGit, inspectErr := gitx.Inspect(ctx, workspaceRoot)
			if inspectErr != nil {
				state.Confidence = string(maxConfidence(Confidence(state.Confidence), Unknown))
				item := domain.Item{Code: "workspace.git-unknown", Message: "Child Git state could not be inspected.", Refs: []string{entry.ID, entry.Path}}
				issues = append(issues, item)
				state.Issues = append(state.Issues, item)
			} else {
				state.Git = childGit
			}
		case "external":
			externalGit, inspectErr := gitx.Inspect(ctx, workspaceRoot)
			if inspectErr != nil {
				state.Confidence = string(Unknown)
				item := domain.Item{Code: "workspace.git-unknown", Message: "External workspace Git state could not be inspected.", Refs: []string{entry.ID, entry.Path}}
				issues = append(issues, item)
				state.Issues = append(state.Issues, item)
			} else {
				state.Git = externalGit
			}
		}
		gitIssues := evaluateGitState(entry, state.Git)
		state.Issues = append(state.Issues, gitIssues...)
		issues = append(issues, gitIssues...)
		state.Confidence = string(maxConfidence(Confidence(state.Confidence), confidenceFromIssues(gitIssues)))
		states = append(states, state)
	}
	sort.Slice(states, func(left, right int) bool { return states[left].Entry.ID < states[right].Entry.ID })
	return states, issues
}

func evaluateGitState(entry workspace.Entry, state gitx.State) []domain.Item {
	if state.Root == "" {
		return nil
	}
	refs := []string{entry.ID, entry.Path}
	issues := []domain.Item{}
	rootPinnedSubmodule := entry.Kind == "submodule" && state.Detached
	if state.Diverged {
		issues = append(issues, domain.Item{Code: "workspace.diverged", Message: "The workspace branch diverged from its upstream.", Refs: refs})
	}
	if state.Dirty {
		issues = append(issues, domain.Item{Code: "workspace.dirty", Message: "The workspace has uncommitted changes.", Refs: refs})
	}
	if (state.Upstream == "" || state.Ahead > 0) && !rootPinnedSubmodule {
		issues = append(issues, domain.Item{Code: "workspace.local-only", Message: "Some workspace state is not recoverable from an upstream branch.", Refs: refs})
	}
	if state.Detached && !rootPinnedSubmodule {
		issues = append(issues, domain.Item{Code: "workspace.detached", Message: "The workspace is on a detached HEAD.", Refs: refs})
	}
	return issues
}

func findSubmodule(submodules []gitx.Submodule, path string) (gitx.Submodule, bool) {
	wanted := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	for _, submodule := range submodules {
		if filepath.ToSlash(filepath.Clean(filepath.FromSlash(submodule.Path))) == wanted {
			return submodule, true
		}
	}
	return gitx.Submodule{}, false
}

type providerConfig struct {
	SchemaVersion      int    `yaml:"schema_version"`
	Provider           string `yaml:"provider"`
	LiveStatusSource   string `yaml:"live_status_source"`
	Remote             string `yaml:"remote,omitempty"`
	CoordinationBranch string `yaml:"coordination_branch,omitempty"`
}

func collectProvider(ctx context.Context, root string) (ProviderView, map[string]providerpkg.GitLocalClaim, error) {
	path := filepath.Join(root, ".harness", "work", "provider.yaml")
	config, err := schema.LoadYAML[providerConfig](path)
	if errors.Is(err, os.ErrNotExist) {
		return ProviderView{Name: "git-local", Confidence: Unknown}, nil, nil
	}
	if err != nil {
		return ProviderView{Confidence: Unknown}, nil, err
	}
	if config.SchemaVersion != 1 || config.Provider == "" || config.LiveStatusSource != config.Provider {
		return ProviderView{Confidence: Unknown}, nil, fmt.Errorf("task provider configuration is incomplete or has more than one live source")
	}
	view := ProviderView{Name: config.Provider, Confidence: Unknown}
	remote := config.Remote
	if remote == "" {
		remote = "origin"
	}
	branch := config.CoordinationBranch
	if branch == "" {
		branch = "coordination"
	}
	observed, readErr := providerpkg.NewGitLocalStore(root, remote, branch).Read(ctx)
	if errors.Is(readErr, providerpkg.ErrNoRemote) {
		return view, nil, nil
	}
	if readErr != nil {
		return view, nil, readErr
	}
	claims := make(map[string]providerpkg.GitLocalClaim, len(observed.Claims))
	for _, claim := range observed.Claims {
		claims[claim.WorkID] = claim
	}
	if config.Provider == "git-local" {
		view.Confidence, view.Revision = Confirmed, observed.Revision
		return view, claims, nil
	}
	definitions, definitionErr := workpkg.LoadDefinitions(root)
	if definitionErr != nil {
		return view, claims, definitionErr
	}
	if len(definitions) == 0 {
		view.Confidence, view.Revision = Confirmed, observed.Revision
		return view, claims, nil
	}
	revisions := make([]string, 0, len(definitions))
	drifted := false
	for _, definition := range definitions {
		mappingPath := filepath.Join(root, ".harness", "work", "mappings", definition.ID+".yaml")
		if _, statErr := os.Lstat(mappingPath); os.IsNotExist(statErr) {
			return view, claims, nil
		} else if statErr != nil {
			return view, claims, statErr
		}
		if locationErr := providerpkg.ValidateCanonicalMappingLocation(root, mappingPath); locationErr != nil {
			return view, claims, locationErr
		}
		mapping, loadErr := providerpkg.LoadMapping(mappingPath)
		if loadErr != nil {
			return view, claims, loadErr
		}
		if mapping.Provider != config.Provider {
			return view, claims, fmt.Errorf("provider mapping %s differs from selected provider %s", mapping.Provider, config.Provider)
		}
		if mapping.WorkID != definition.ID {
			return view, claims, fmt.Errorf("provider mapping references a different work definition %s", mapping.WorkID)
		}
		snapshotPath := filepath.Join(root, ".harness", "local", "providers", mapping.Provider, mapping.WorkID+".yaml")
		if _, statErr := os.Lstat(snapshotPath); os.IsNotExist(statErr) {
			return view, claims, nil
		} else if statErr != nil {
			return view, claims, statErr
		}
		if locationErr := providerpkg.ValidateCanonicalSnapshotLocation(root, snapshotPath); locationErr != nil {
			return view, claims, locationErr
		}
		observation, snapshotErr := providerpkg.LoadSnapshot(snapshotPath)
		if snapshotErr != nil {
			return view, claims, snapshotErr
		}
		state := providerpkg.Reconcile(providerpkg.Expectation{WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Dependencies: definition.Dependencies}, mapping, observation, time.Now().UTC())
		if state.Confidence != providerpkg.Confirmed {
			view.Revision, view.Owner, view.Status = state.Revision, state.Owner, state.Status
			view.Confidence = Stale
			return view, claims, nil
		}
		revision := state.Revision
		if revision == "" {
			revision = observation.RawHash
		}
		revisions = append(revisions, definition.ID+"\x00"+revision)
		claim, reserved := claims[definition.ID]
		if !reserved && externalStatusNeedsReservation(state.Status) {
			claim = providerpkg.GitLocalClaim{ID: definition.ID, WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Status: normalizeExternalStatus(state.Status), Owner: state.Owner, Repository: "repository.root"}
			claims[definition.ID] = claim
			drifted = true
		} else if reserved {
			coordinatedStatus := claim.Status
			if coordinatedStatus == "" {
				coordinatedStatus = "in_progress"
			}
			providerStatus := normalizeExternalStatus(state.Status)
			ownerClearedAtTerminal := providerStatus == "done" && state.Owner == ""
			if coordinatedStatus != providerStatus || (!ownerClearedAtTerminal && claim.Owner != state.Owner) {
				drifted = true
			}
			claim.Status, claim.Owner = providerStatus, state.Owner
			if ownerClearedAtTerminal {
				claim.Owner = claims[definition.ID].Owner
			}
			claims[definition.ID] = claim
		}
		if len(definitions) == 1 {
			view.ItemID, view.Owner, view.Status = state.ItemID, state.Owner, state.Status
		}
	}
	sort.Strings(revisions)
	digest := sha256.Sum256([]byte(strings.Join(revisions, "\n")))
	view.Revision = "sha256:" + hex.EncodeToString(digest[:])
	if drifted {
		return view, claims, fmt.Errorf("external task owner or status differs from the Git semantic reservation")
	}
	view.Confidence = Confirmed
	return view, claims, nil
}

func normalizeExternalStatus(value string) string {
	if value == "closed" {
		return "done"
	}
	return value
}

func externalStatusNeedsReservation(value string) bool {
	switch normalizeExternalStatus(value) {
	case "in_progress", "blocked", "review", "integrated":
		return true
	default:
		return false
	}
}

func mergeLiveWork(values []WorkView, claims map[string]providerpkg.GitLocalClaim) []WorkView {
	result := make([]WorkView, 0, len(values))
	for _, value := range values {
		if claim, found := claims[value.ID]; found {
			value.State = claim.Status
			if value.State == "" {
				value.State = "in_progress"
			}
			value.Owner = claim.Owner
			value.Branch = claim.Branch
			value.LiveRevision = providerpkg.ClaimRevision(claim)
		}
		if value.State == "done" || value.State == "closed" || value.State == "released" {
			continue
		}
		result = append(result, value)
	}
	return result
}

func findWorkDefinition(definitions []workpkg.Definition, id string) (workpkg.Definition, bool) {
	for _, definition := range definitions {
		if definition.ID == id {
			return definition, true
		}
	}
	return workpkg.Definition{}, false
}

func collectActiveWork(root string) ([]WorkView, error) {
	result := []WorkView{}
	for _, relative := range []string{filepath.Join(".harness", "work", "definitions"), filepath.Join(".harness", "work", "items")} {
		directory := filepath.Join(root, relative)
		entries, err := os.ReadDir(directory)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || (filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml") {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil, readErr
			}
			value, decodeErr := schema.DecodeYAML[map[string]any](data)
			if decodeErr != nil {
				return nil, decodeErr
			}
			id, _ := value["id"].(string)
			if id == "" {
				return nil, fmt.Errorf("work definition %s has no stable ID", filepath.ToSlash(path))
			}
			state, _ := value["status"].(string)
			if state == "" {
				state, _ = value["state"].(string)
			}
			if state == "" {
				state, _ = value["readiness"].(string)
			}
			if state == "done" || state == "closed" || state == "released" {
				continue
			}
			title, _ := value["title"].(string)
			digest := sha256.Sum256(data)
			result = append(result, WorkView{ID: id, Title: title, State: state, DefinitionFingerprint: "sha256:" + hex.EncodeToString(digest[:])})
		}
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ID < result[right].ID })
	return result, nil
}

func collectRelease(root string) ReleaseView {
	for _, relative := range []string{filepath.Join(".harness", "local", "release", "candidate.json"), filepath.Join(".harness", "release", "candidate.json"), filepath.Join(".harness", "state", "release-candidate.json")} {
		data, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			continue
		}
		candidate, decodeErr := schema.DecodeJSON[release.Candidate](data)
		if decodeErr == nil && candidate.Digest != "" {
			return ReleaseView{CandidateDigest: candidate.Digest, Confidence: LocalOnly}
		}
	}
	return ReleaseView{Confidence: Unknown}
}

func canonicalFingerprint(projectID string, snapshot contextpkg.Snapshot) string {
	ids := make([]string, 0, len(snapshot.Index))
	for id := range snapshot.Index {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	hash := sha256.New()
	_, _ = hash.Write([]byte(projectID))
	for _, id := range ids {
		entry := snapshot.Index[id]
		_, _ = hash.Write([]byte("\x00" + id + "\x00" + entry.Fingerprint))
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func normalizeIssues(items []domain.Item) []domain.Item {
	seen := map[string]bool{}
	result := make([]domain.Item, 0, len(items))
	for _, item := range items {
		refs := append([]string(nil), item.Refs...)
		sort.Strings(refs)
		item.Refs = refs
		key := item.Code + "\x00" + strings.Join(refs, "\x00")
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, item)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Code == result[right].Code {
			return strings.Join(result[left].Refs, "\x00") < strings.Join(result[right].Refs, "\x00")
		}
		return result[left].Code < result[right].Code
	})
	return result
}
