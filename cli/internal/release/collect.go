package release

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"fullstack-orchestrator/cli/internal/contract"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/evidence"
	"fullstack-orchestrator/cli/internal/gitx"
	"fullstack-orchestrator/cli/internal/integration"
	"fullstack-orchestrator/cli/internal/provider"
	"fullstack-orchestrator/cli/internal/schema"
	"fullstack-orchestrator/cli/internal/work"
	"fullstack-orchestrator/cli/internal/workspace"
)

const maxReleaseSourceBytes = 64 << 20

// EvidenceStore supplies local commit-bound command results. It never supplies live task status.
type EvidenceStore interface {
	Load(root string) ([]evidence.Record, error)
}

// ProviderReader supplies a fresh normalized observation from the one selected live source.
type ProviderReader interface {
	Read(context.Context, string, []work.Definition) ([]integration.ProviderState, []domain.Item)
}

// CollectOptions contains deterministic release choices, not hand-authored repository identities.
type CollectOptions struct {
	Version             string
	Profile             Profile
	EvidenceStore       EvidenceStore
	ProviderReader      ProviderReader
	StrictEvidence      *StrictEvidence
	ToolVersions        map[string]string
	OrchestratorVersion string
	WorkIDs             []string
}

// LocalEvidenceStore reads only safe evidence records produced by reviewed commands.
type LocalEvidenceStore struct{}

// Load reads local evidence recursively and rejects links or malformed records.
func (LocalEvidenceStore) Load(root string) ([]evidence.Record, error) {
	directory := filepath.Join(root, ".harness", "local", "evidence")
	if info, err := os.Lstat(directory); errors.Is(err, os.ErrNotExist) {
		return []evidence.Record{}, nil
	} else if err != nil {
		return nil, err
	} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("release evidence directory must be a real directory")
	}
	records := []evidence.Record{}
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("release evidence cannot contain symlinks")
		}
		if entry.IsDir() || (filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml") {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxReleaseSourceBytes {
			return fmt.Errorf("release evidence record is unsafe: %s", filepath.Base(path))
		}
		record, err := schema.LoadYAML[evidence.Record](path)
		if err != nil {
			return err
		}
		if issues := schema.Validate("evidence", record); len(issues) > 0 {
			return fmt.Errorf("invalid release evidence %s: %s", filepath.Base(path), issues[0].Message)
		}
		records = append(records, record)
		return nil
	})
	sort.Slice(records, func(left, right int) bool { return records[left].ID < records[right].ID })
	return records, err
}

// CollectInput rebuilds release identity from actual Git, canonical meaning, live provider state, and current evidence.
func CollectInput(ctx context.Context, start string, options CollectOptions) (Input, []domain.Item) {
	input := Input{
		Profile: options.Profile, Version: strings.TrimSpace(options.Version), WorkspaceCommits: map[string]string{}, WorkspaceRemotes: map[string]string{},
		ProviderRevisions: map[string]string{}, ToolVersions: cloneMap(options.ToolVersions), ArtifactDigests: map[string]string{}, TDDEvidence: map[string]string{}, IntegrationEvidence: map[string]string{},
		StrictEvidence: options.StrictEvidence,
	}
	issues := []domain.Item{}
	located, err := workspace.FindRoot(ctx, start)
	if err != nil {
		return input, []domain.Item{releaseIssue("release.root-unavailable", err.Error())}
	}
	root := located.Path
	profile, profileErr := loadReleaseProfile(root)
	if profileErr != nil {
		issues = append(issues, releaseIssue("release.profile-invalid", profileErr.Error()))
	} else if input.Profile == "" {
		input.Profile = profile
	} else if profile != "" && input.Profile != profile {
		issues = append(issues, releaseIssue("release.profile-mismatch", "Requested release profile differs from committed project profile.", string(profile), string(input.Profile)))
	}
	if input.Profile == "" {
		input.Profile = ProfileCore
	}
	orchestratorVersion := strings.TrimSpace(options.OrchestratorVersion)
	if orchestratorVersion == "" {
		orchestratorVersion = input.Version
	}
	input.ToolVersions["orchestrator"] = orchestratorVersion
	if gitVersion, versionErr := commandOutput(ctx, root, "git", "--version"); versionErr == nil {
		input.ToolVersions["git"] = strings.TrimSpace(strings.TrimPrefix(gitVersion, "git version "))
	} else {
		issues = append(issues, releaseIssue("release.git-version-unknown", "Git version could not be observed."))
	}
	rootGit, err := gitx.Inspect(ctx, root)
	if err != nil {
		return input, append(issues, releaseIssue("release.git-unavailable", err.Error()))
	}
	input.RootCommit = rootGit.Head
	workspaceStates, workspaceIssues := collectReleaseWorkspaces(ctx, root, located.Manifest, rootGit, &input)
	issues = append(issues, workspaceIssues...)

	specsPath, specsErr := authoredReleasePath(root, "specs", "specs")
	if specsErr != nil {
		issues = append(issues, releaseIssue("release.product-invalid", specsErr.Error()))
	} else {
		input.ProductFingerprint, err = hashReleaseSources(root, []string{".harness/manifest.yaml", specsPath})
		if err != nil {
			issues = append(issues, releaseIssue("release.product-invalid", err.Error()))
		}
	}
	docsPath, docsErr := authoredReleasePath(root, "docs", "docs")
	if docsErr != nil {
		issues = append(issues, releaseIssue("release.docs-invalid", docsErr.Error()))
	} else {
		input.DocsFingerprint, err = hashReleaseSources(root, []string{docsPath})
		if err != nil {
			issues = append(issues, releaseIssue("release.docs-invalid", err.Error()))
		}
	}
	contractsPath, contractsErr := authoredReleasePath(root, "contracts", "contracts")
	if contractsErr != nil {
		issues = append(issues, releaseIssue("release.contract-invalid", contractsErr.Error()))
	} else {
		input.ContractFingerprint, err = evidence.FingerprintTree(root, contractsPath)
		if err != nil {
			issues = append(issues, releaseIssue("release.contract-invalid", err.Error()))
		}
	}
	registry, registryErr := contract.LoadRegistry(root)
	if registryErr != nil {
		issues = append(issues, releaseIssue("release.contract-stale", registryErr.Error()))
	}
	contextSnapshot, contextIssues := contextpkg.Refresh(ctx, root, contextpkg.ReadOnly)
	for _, item := range contextIssues {
		if strings.HasPrefix(item.Code, "context.error") {
			issues = append(issues, releaseIssue("release.context-invalid", item.Message, item.Refs...))
		}
	}
	if len(contextSnapshot.Stale) > 0 {
		issues = append(issues, releaseIssue("release.context-stale", "Canonical product dependents are stale.", contextSnapshot.Stale...))
	}
	if len(contextSnapshot.Unknown) > 0 {
		issues = append(issues, releaseIssue("release.context-unknown", "Canonical product meaning has unresolved references.", contextSnapshot.Unknown...))
	}

	definitions, err := work.LoadDefinitions(root)
	if err != nil {
		issues = append(issues, releaseIssue("release.work-invalid", err.Error()))
	}
	definitions, selectionIssues := selectReleaseDefinitions(definitions, options.WorkIDs)
	issues = append(issues, selectionIssues...)
	if registryErr == nil {
		for _, compatibilityIssue := range integration.CheckCompatibility(definitions, registry) {
			issues = append(issues, releaseIssue("release.contract-incompatible", compatibilityIssue.Message, compatibilityIssue.Refs...))
		}
	}
	reader := options.ProviderReader
	if reader == nil {
		reader = selectedProviderReader{}
	}
	providerStates, providerIssues := reader.Read(ctx, root, definitions)
	issues = append(issues, providerIssues...)
	providerByWork := map[string]integration.ProviderState{}
	for _, state := range providerStates {
		providerByWork[state.WorkID] = state
		if !state.Confirmed || state.Revision == "" || (state.Status != "review" && state.Status != "integrated" && state.Status != "done") {
			issues = append(issues, releaseIssue("release.provider-unknown", "Work is not freshly confirmed as reviewable by the selected live provider.", state.WorkID))
			continue
		}
		input.ProviderRevisions[state.WorkID] = state.Revision
	}
	for _, definition := range definitions {
		state, exists := providerByWork[definition.ID]
		if !exists || !state.Confirmed {
			issues = append(issues, releaseIssue("release.provider-unknown", "No fresh selected-provider state exists for release work.", definition.ID))
		} else if state.DefinitionFingerprint != definition.Fingerprint {
			issues = append(issues, releaseIssue("release.provider-stale", "Selected-provider state references an older work definition.", definition.ID))
		}
		if definition.Evidence.MigrationRequired || definition.Evidence.RollbackRequired {
			input.MigrationRequired = true
		}
	}

	store := options.EvidenceStore
	if store == nil {
		store = LocalEvidenceStore{}
	}
	records, err := store.Load(root)
	if err != nil {
		issues = append(issues, releaseIssue("release.evidence-invalid", err.Error()))
	} else {
		issues = append(issues, collectReleaseEvidence(root, located.Manifest, definitions, workspaceStates, input.ContractFingerprint, records, &input)...)
	}
	if blockers := validateInput(input); len(blockers) > 0 {
		for _, blocker := range blockers {
			issues = append(issues, releaseIssue("release.input-incomplete", blocker.Message, blocker.Refs...))
		}
	}
	return input, normalizeReleaseIssues(issues)
}

func selectReleaseDefinitions(definitions []work.Definition, selected []string) ([]work.Definition, []domain.Item) {
	if len(selected) == 0 {
		if len(definitions) == 0 {
			return nil, []domain.Item{releaseIssue("release.work-missing", "At least one executable work definition must be selected for release.")}
		}
		return definitions, nil
	}
	wanted := map[string]bool{}
	for _, id := range selected {
		id = strings.TrimSpace(id)
		if id != "" {
			wanted[id] = true
		}
	}
	result := []work.Definition{}
	for _, definition := range definitions {
		if wanted[definition.ID] {
			result = append(result, definition)
			delete(wanted, definition.ID)
		}
	}
	issues := []domain.Item{}
	for id := range wanted {
		issues = append(issues, releaseIssue("release.work-missing", "Selected release work has no executable definition.", id))
	}
	if len(result) == 0 {
		issues = append(issues, releaseIssue("release.work-missing", "No executable work remains in the selected release scope."))
	}
	return result, normalizeReleaseIssues(issues)
}

func authoredReleasePath(root, key, fallback string) (string, error) {
	manifest, err := schema.LoadYAML[map[string]any](filepath.Join(root, ".harness", "manifest.yaml"))
	if err != nil {
		return "", err
	}
	value := fallback
	if paths, ok := manifest["paths"].(map[string]any); ok {
		if configured, ok := paths[key].(string); ok && strings.TrimSpace(configured) != "" {
			value = configured
		}
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("authored %s path must be repository-relative", key)
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("authored %s path escapes repository", key)
	}
	return filepath.ToSlash(clean), nil
}

func loadReleaseProfile(root string) (Profile, error) {
	path := filepath.Join(root, ".harness", "profile.yaml")
	profile, err := schema.LoadYAML[map[string]any](path)
	if errors.Is(err, os.ErrNotExist) {
		return ProfileCore, nil
	}
	if err != nil {
		return "", err
	}
	if issues := schema.Validate("profile", profile); len(issues) > 0 {
		return "", fmt.Errorf("committed release profile is invalid: %s", issues[0].Message)
	}
	releaseValue, ok := profile["release"].(string)
	if !ok {
		return "", fmt.Errorf("committed release profile has no release mode")
	}
	return Profile(releaseValue), nil
}

func collectReleaseWorkspaces(ctx context.Context, root string, manifest workspace.Manifest, rootGit gitx.State, input *Input) ([]integration.WorkspaceState, []domain.Item) {
	states := []integration.WorkspaceState{}
	issues := []domain.Item{}
	submodules := map[string]gitx.Submodule{}
	for _, submodule := range rootGit.Submodules {
		submodules[filepath.ToSlash(filepath.Clean(filepath.FromSlash(submodule.Path)))] = submodule
	}
	for _, entry := range manifest.Workspaces {
		path := root
		state := rootGit
		expected, actual := "", rootGit.Head
		if entry.Kind == "submodule" {
			path = filepath.Join(root, filepath.FromSlash(entry.Path))
			submodule, exists := submodules[filepath.ToSlash(filepath.Clean(filepath.FromSlash(entry.Path)))]
			if !exists || !submodule.Initialized {
				issues = append(issues, releaseIssue("release.workspace-missing", "Declared submodule is not initialized.", entry.ID))
				continue
			}
			if submodule.UnsafeURL {
				issues = append(issues, releaseIssue("release.submodule-url-unsafe", "Submodule URL is unsafe for release recovery.", entry.ID))
			}
			expected, actual = submodule.ExpectedSHA, submodule.Head
			if submodule.Dirty || submodule.PointerDiff || expected == "" || expected != actual {
				issues = append(issues, releaseIssue("release.pointer-mismatch", "Submodule checkout differs from the exact root gitlink.", entry.ID, expected, actual))
			}
			var err error
			state, err = gitx.Inspect(ctx, path)
			if err != nil {
				issues = append(issues, releaseIssue("release.workspace-unavailable", err.Error(), entry.ID))
				continue
			}
		} else if entry.Kind == "external" {
			path = filepath.Join(root, filepath.FromSlash(entry.Path))
			var err error
			state, err = gitx.Inspect(ctx, path)
			if err != nil {
				issues = append(issues, releaseIssue("release.workspace-unavailable", err.Error(), entry.ID))
				continue
			}
		}
		remote, err := releaseRemote(ctx, path)
		if err != nil {
			issues = append(issues, releaseIssue("release.remote-unknown", err.Error(), entry.ID))
		}
		expectedRemote := entry.Remote
		if entry.Kind == "root" && expectedRemote == "" {
			expectedRemote = manifest.RootRemote
		}
		if expectedRemote != "" && remote != expectedRemote {
			issues = append(issues, releaseIssue("release.remote-mismatch", "Actual workspace remote differs from the orchestration manifest.", entry.ID))
		}
		published := releaseCommitPublished(ctx, path, state.Head)
		if state.Dirty {
			issues = append(issues, releaseIssue("release.workspace-dirty", "Release workspace has uncommitted changes.", entry.ID))
		}
		if state.Diverged || state.Ahead > 0 || state.Behind > 0 || !published {
			issues = append(issues, releaseIssue("release.workspace-unpublished", "Release commit is not the exact current published workspace state.", entry.ID, state.Head))
		}
		input.WorkspaceCommits[entry.ID] = state.Head
		input.WorkspaceRemotes[entry.ID] = remote
		states = append(states, integration.WorkspaceState{ID: entry.ID, Kind: entry.Kind, Commit: state.Head, Remote: remote, Clean: !state.Dirty, Published: published && !state.Diverged && state.Ahead == 0 && state.Behind == 0, ExpectedPointer: expected, ActualPointer: actual})
	}
	sort.Slice(states, func(left, right int) bool { return states[left].ID < states[right].ID })
	return states, issues
}

func collectReleaseEvidence(root string, manifest workspace.Manifest, definitions []work.Definition, states []integration.WorkspaceState, contractFingerprint string, records []evidence.Record, input *Input) []domain.Item {
	issues := []domain.Item{}
	definitionsByID := map[string]work.Definition{}
	for _, definition := range definitions {
		definitionsByID[definition.ID] = definition
	}
	statesByID := map[string]integration.WorkspaceState{}
	for _, state := range states {
		statesByID[state.ID] = state
	}
	workspaceByID := map[string]workspace.Entry{}
	for _, entry := range manifest.Workspaces {
		workspaceByID[entry.ID] = entry
	}
	kindsByWork := map[string]map[string]bool{}
	migrationEvidence := map[string]string{}
	rollbackEvidence := map[string]string{}
	for _, record := range records {
		definition, definitionExists := definitionsByID[record.WorkID]
		if !definitionExists {
			continue
		}
		entry, workspaceExists := workspaceByID[record.WorkspaceID]
		state, stateExists := statesByID[record.WorkspaceID]
		if !workspaceExists || !stateExists || record.DefinitionFingerprint != definition.Fingerprint || record.ContractFingerprint != contractFingerprint || record.Commit != state.Commit || record.ExitCode != 0 {
			issues = append(issues, releaseIssue("release.evidence-stale", "Evidence differs from current work, contract, workspace, or commit identity.", record.ID))
			continue
		}
		workspacePath := filepath.Join(root, filepath.FromSlash(entry.Path))
		repositoryPath := workspacePath
		if entry.Kind == "root" || entry.Kind == "directory" {
			repositoryPath = root
		}
		if verification := evidence.VerifyCurrent(record, evidence.Actual{Workspace: workspacePath, Repository: repositoryPath, Head: state.Commit, DefinitionFingerprint: definition.Fingerprint, ContractFingerprint: contractFingerprint}); len(verification) > 0 {
			issues = append(issues, releaseIssue("release.evidence-stale", "Evidence no longer verifies against actual Git state.", record.ID))
			continue
		}
		if kindsByWork[record.WorkID] == nil {
			kindsByWork[record.WorkID] = map[string]bool{}
		}
		kindsByWork[record.WorkID][record.Kind] = true
		switch record.Kind {
		case "test":
			input.TDDEvidence[record.ID] = record.OutputDigest
		case "integration", "child-merge", "root-pointer":
			input.IntegrationEvidence[record.ID] = record.OutputDigest
		case "migration":
			migrationEvidence[record.ID] = record.OutputDigest
		case "rollback":
			rollbackEvidence[record.ID] = record.OutputDigest
		}
		for name, digest := range record.ArtifactDigests {
			key := record.WorkspaceID + "." + name
			if previous, exists := input.ArtifactDigests[key]; exists && previous != digest {
				issues = append(issues, releaseIssue("release.artifact-conflict", "Two evidence records disagree on an artifact digest.", key))
			} else {
				input.ArtifactDigests[key] = digest
			}
		}
	}
	for _, definition := range definitions {
		for _, kind := range definition.Evidence.Kinds {
			if !kindsByWork[definition.ID][kind] {
				issues = append(issues, releaseIssue("release.evidence-missing", "Required work evidence is absent.", definition.ID, kind))
			}
		}
		if definition.Evidence.IntegrationRequired && !hasReleaseEvidenceKind(kindsByWork[definition.ID], "integration", "child-merge", "root-pointer") {
			issues = append(issues, releaseIssue("release.integration-missing", "Work requires exact service integration evidence.", definition.ID))
		}
		if definition.Evidence.MigrationRequired && !kindsByWork[definition.ID]["migration"] {
			issues = append(issues, releaseIssue("release.migration-missing", "Work requires current migration evidence.", definition.ID))
		}
		if definition.Evidence.RollbackRequired && !kindsByWork[definition.ID]["rollback"] {
			issues = append(issues, releaseIssue("release.rollback-missing", "Work requires current rollback evidence.", definition.ID))
		}
	}
	input.MigrationEvidence = aggregateReleaseEvidence(migrationEvidence)
	input.RollbackEvidence = aggregateReleaseEvidence(rollbackEvidence)
	return issues
}

func hasReleaseEvidenceKind(values map[string]bool, kinds ...string) bool {
	for _, kind := range kinds {
		if values[kind] {
			return true
		}
	}
	return false
}

func aggregateReleaseEvidence(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hash := sha256.New()
	for _, key := range keys {
		_, _ = hash.Write([]byte(key))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(values[key]))
		_, _ = hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

type selectedProviderReader struct{}

type releaseProviderConfig struct {
	SchemaVersion      int    `yaml:"schema_version"`
	Provider           string `yaml:"provider"`
	LiveStatusSource   string `yaml:"live_status_source"`
	Remote             string `yaml:"remote,omitempty"`
	CoordinationBranch string `yaml:"coordination_branch,omitempty"`
}

func (selectedProviderReader) Read(ctx context.Context, root string, definitions []work.Definition) ([]integration.ProviderState, []domain.Item) {
	path := filepath.Join(root, ".harness", "work", "provider.yaml")
	config, err := schema.LoadYAML[releaseProviderConfig](path)
	if errors.Is(err, os.ErrNotExist) {
		config = releaseProviderConfig{SchemaVersion: 1, Provider: "git-local", LiveStatusSource: "git-local", Remote: "origin", CoordinationBranch: "coordination"}
	} else if err != nil {
		return nil, []domain.Item{releaseIssue("release.provider-invalid", err.Error())}
	}
	if config.SchemaVersion != 1 || config.Provider == "" || config.Provider != config.LiveStatusSource {
		return nil, []domain.Item{releaseIssue("release.provider-invalid", "Project must select exactly one live task source.")}
	}
	if config.Provider == "git-local" {
		store := provider.NewGitLocalStore(root, config.Remote, config.CoordinationBranch)
		observed, readErr := store.Read(ctx)
		if readErr != nil {
			return nil, []domain.Item{releaseIssue("release.provider-unknown", "Git-local live state could not be read.")}
		}
		claims := map[string]provider.GitLocalClaim{}
		for _, claim := range observed.Claims {
			claims[claim.WorkID] = claim
		}
		states := []integration.ProviderState{}
		for _, definition := range definitions {
			claim, exists := claims[definition.ID]
			revision := ""
			if exists {
				revision = provider.ClaimRevision(claim)
			}
			states = append(states, integration.ProviderState{WorkID: definition.ID, Status: claim.Status, Revision: revision, DefinitionFingerprint: claim.DefinitionFingerprint, Confirmed: exists && observed.Revision != ""})
		}
		return states, nil
	}
	if config.Remote == "" {
		config.Remote = "origin"
	}
	if config.CoordinationBranch == "" {
		config.CoordinationBranch = "coordination"
	}
	coordinated, coordinateErr := provider.NewGitLocalStore(root, config.Remote, config.CoordinationBranch).Read(ctx)
	if coordinateErr != nil {
		return nil, []domain.Item{releaseIssue("release.provider-unknown", "Git semantic reservations could not be read for the selected task source.")}
	}
	claims := make(map[string]provider.GitLocalClaim, len(coordinated.Claims))
	for _, claim := range coordinated.Claims {
		claims[claim.WorkID] = claim
	}
	states, issues := []integration.ProviderState{}, []domain.Item{}
	for _, definition := range definitions {
		mappingPath := filepath.Join(root, ".harness", "work", "mappings", definition.ID+".yaml")
		snapshotPath := filepath.Join(root, ".harness", "local", "providers", config.Provider, definition.ID+".yaml")
		if provider.ValidateCanonicalMappingLocation(root, mappingPath) != nil || provider.ValidateCanonicalSnapshotLocation(root, snapshotPath) != nil {
			issues = append(issues, releaseIssue("release.provider-unknown", "Provider mapping or observation escaped its canonical local boundary.", definition.ID, config.Provider))
			continue
		}
		mapping, mappingErr := provider.LoadMapping(mappingPath)
		snapshot, snapshotErr := provider.LoadSnapshot(snapshotPath)
		if mappingErr != nil || snapshotErr != nil {
			issues = append(issues, releaseIssue("release.provider-unknown", "Fresh selected-provider observation is unavailable.", definition.ID, config.Provider))
			continue
		}
		state := provider.Reconcile(provider.Expectation{WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Dependencies: definition.Dependencies}, mapping, snapshot, time.Now().UTC())
		claim, exists := claims[definition.ID]
		claimStatus := claim.Status
		if claimStatus == "" {
			claimStatus = "in_progress"
		}
		providerStatus := normalizeReleaseProviderStatus(state.Status)
		ownerClearedAtTerminal := providerStatus == "done" && state.Owner == ""
		if state.Confidence != provider.Confirmed || mapping.Provider != config.Provider || !exists || claim.DefinitionFingerprint != definition.Fingerprint || claimStatus != providerStatus || (!ownerClearedAtTerminal && claim.Owner != state.Owner) {
			issues = append(issues, releaseIssue("release.provider-drift", "External task state differs from its Git semantic reservation.", definition.ID, config.Provider))
			continue
		}
		revision := state.Revision
		if revision == "" {
			revision = snapshot.RawHash
		}
		digest := sha256.Sum256([]byte(revision + "\x00" + provider.ClaimRevision(claim)))
		states = append(states, integration.ProviderState{WorkID: definition.ID, Status: providerStatus, Revision: "sha256:" + hex.EncodeToString(digest[:]), DefinitionFingerprint: definition.Fingerprint, Confirmed: true})
	}
	return states, issues
}

func normalizeReleaseProviderStatus(value string) string {
	if value == "closed" {
		return "done"
	}
	return value
}

func hashReleaseSources(root string, sources []string) (string, error) {
	type fileValue struct {
		path string
		data []byte
	}
	files := []fileValue{}
	total := int64(0)
	for _, source := range sources {
		path := filepath.Join(root, filepath.FromSlash(source))
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("release source cannot be a symlink: %s", source)
		}
		if info.Mode().IsRegular() {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return "", readErr
			}
			files = append(files, fileValue{path: filepath.ToSlash(source), data: data})
			continue
		}
		err = filepath.WalkDir(path, func(current string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("release source cannot contain symlinks")
			}
			if entry.IsDir() {
				return nil
			}
			info, infoErr := entry.Info()
			if infoErr != nil || !info.Mode().IsRegular() {
				return fmt.Errorf("release source is not a regular file")
			}
			total += info.Size()
			if total > maxReleaseSourceBytes {
				return fmt.Errorf("release source set exceeds %d bytes", maxReleaseSourceBytes)
			}
			data, readErr := os.ReadFile(current)
			if readErr != nil {
				return readErr
			}
			relative, _ := filepath.Rel(root, current)
			files = append(files, fileValue{path: filepath.ToSlash(relative), data: data})
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	sort.Slice(files, func(left, right int) bool { return files[left].path < files[right].path })
	hash := sha256.New()
	for _, file := range files {
		_, _ = hash.Write([]byte(file.path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(file.data)
		_, _ = hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func releaseRemote(ctx context.Context, root string) (string, error) {
	value, err := gitx.RemoteURL(ctx, root, "origin")
	if err != nil || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("origin remote is unavailable")
	}
	value = strings.TrimSpace(value)
	if strings.Contains(value, "\n") || strings.HasPrefix(value, "-") || strings.HasPrefix(strings.ToLower(value), "file:") || filepath.IsAbs(value) {
		return "", fmt.Errorf("origin remote is unsafe")
	}
	if parsed, parseErr := url.Parse(value); parseErr == nil && parsed.Scheme != "" {
		if parsed.User != nil {
			if _, hasPassword := parsed.User.Password(); hasPassword {
				return "", fmt.Errorf("origin remote contains credentials")
			}
		}
		if parsed.Scheme != "https" && parsed.Scheme != "ssh" {
			return "", fmt.Errorf("origin remote scheme is unsupported")
		}
	}
	return value, nil
}

func releaseCommitPublished(ctx context.Context, root, commit string) bool {
	return gitx.CommitPublished(ctx, root, commit)
}

func commandOutput(ctx context.Context, root, program string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, program, args...)
	command.Dir = root
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.Output()
	return strings.TrimSpace(string(output)), err
}

func releaseIssue(code, message string, refs ...string) domain.Item {
	return domain.Item{Code: code, Message: message, Refs: refs}
}

func normalizeReleaseIssues(items []domain.Item) []domain.Item {
	sort.Slice(items, func(left, right int) bool {
		if items[left].Code == items[right].Code {
			return strings.Join(items[left].Refs, "\x00") < strings.Join(items[right].Refs, "\x00")
		}
		return items[left].Code < items[right].Code
	})
	result := []domain.Item{}
	for _, item := range items {
		if len(result) > 0 && result[len(result)-1].Code == item.Code && strings.Join(result[len(result)-1].Refs, "\x00") == strings.Join(item.Refs, "\x00") {
			continue
		}
		result = append(result, item)
	}
	return result
}
