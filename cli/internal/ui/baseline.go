package ui

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

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/gitx"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/schema"
	"fullstack-orchestrator/cli/internal/workspace"
	"go.yaml.in/yaml/v3"
)

var baselineIDPattern = regexp.MustCompile(`^ui\.baseline\.[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
var baselineWorkspacePattern = regexp.MustCompile(`^workspace\.[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
var baselineStableIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z0-9][a-z0-9-]*)+$`)
var baselineObjectIDPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var baselineDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// Baseline binds approved UI meaning to an exact recoverable workspace commit.
type Baseline struct {
	SchemaVersion      int               `json:"schema_version" yaml:"schema_version"`
	ID                 string            `json:"id" yaml:"id"`
	WorkspaceID        string            `json:"workspace_id" yaml:"workspace_id"`
	WorkspaceCommit    string            `json:"workspace_commit" yaml:"workspace_commit"`
	WorkspaceRemote    string            `json:"workspace_remote" yaml:"workspace_remote"`
	SourceIDs          []string          `json:"source_ids" yaml:"source_ids"`
	SourceFingerprints map[string]string `json:"source_fingerprints" yaml:"source_fingerprints"`
	MappedRefs         []string          `json:"mapped_refs" yaml:"mapped_refs"`
	Consumers          []string          `json:"consumers" yaml:"consumers"`
	Fingerprint        string            `json:"fingerprint" yaml:"fingerprint"`
}

// ValidateBaseline checks intrinsic identity before repository state is inspected.
func ValidateBaseline(baseline Baseline) []domain.Item {
	issues := []domain.Item{}
	add := func(code, message string, refs ...string) {
		issues = append(issues, domain.Item{Code: code, Message: message, Refs: refs})
	}
	if baseline.SchemaVersion != 1 {
		add("ui.baseline-schema-invalid", "UI baseline schema_version must be 1.")
	}
	if !baselineIDPattern.MatchString(baseline.ID) {
		add("ui.baseline-id-invalid", "UI baseline ID is invalid.", baseline.ID)
	}
	if !baselineWorkspacePattern.MatchString(baseline.WorkspaceID) {
		add("ui.baseline-workspace-invalid", "UI baseline workspace ID is invalid.", baseline.WorkspaceID)
	}
	if !baselineObjectIDPattern.MatchString(baseline.WorkspaceCommit) {
		add("ui.baseline-commit-invalid", "UI baseline commit must be an exact Git object ID.", baseline.WorkspaceCommit)
	}
	if !gitx.SafeRemoteURL(baseline.WorkspaceRemote) {
		add("ui.baseline-remote-unsafe", "UI baseline remote must be a credential-free HTTPS or SSH Git URL.")
	}
	for _, group := range [][]string{baseline.SourceIDs, baseline.MappedRefs, baseline.Consumers} {
		seen := map[string]bool{}
		for _, value := range group {
			if !baselineStableIDPattern.MatchString(value) {
				add("ui.baseline-ref-invalid", "UI baseline references must be stable IDs.", value)
			}
			if seen[value] {
				add("ui.baseline-ref-duplicate", "UI baseline reference sets must not contain duplicates.", value)
			}
			seen[value] = true
		}
	}
	if len(baseline.SourceIDs) != len(baseline.SourceFingerprints) {
		add("ui.baseline-source-fingerprint-missing", "Every baseline source needs its exact imported fingerprint.")
	}
	for _, id := range baseline.SourceIDs {
		if fingerprint, exists := baseline.SourceFingerprints[id]; !exists || !baselineDigestPattern.MatchString(fingerprint) {
			add("ui.baseline-source-fingerprint-invalid", "Baseline source fingerprint is missing or invalid.", id)
		}
	}
	for id := range baseline.SourceFingerprints {
		if !containsBaselineValue(baseline.SourceIDs, id) {
			add("ui.baseline-source-fingerprint-extra", "Baseline source fingerprint has no matching source ID.", id)
		}
	}
	if len(baseline.MappedRefs) == 0 || len(baseline.Consumers) == 0 {
		add("ui.baseline-scope-required", "UI baseline needs mapped UI meaning and at least one consumer.")
	}
	want := BaselineFingerprint(baseline)
	if baseline.Fingerprint != "" && (!baselineDigestPattern.MatchString(baseline.Fingerprint) || baseline.Fingerprint != want) {
		add("ui.baseline-fingerprint-mismatch", "UI baseline fingerprint differs from normalized identity.", baseline.Fingerprint, want)
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Code == issues[j].Code {
			return strings.Join(issues[i].Refs, "\x00") < strings.Join(issues[j].Refs, "\x00")
		}
		return issues[i].Code < issues[j].Code
	})
	return issues
}

// BaselineFingerprint hashes normalized baseline meaning without its fingerprint field.
func BaselineFingerprint(baseline Baseline) string {
	baseline.SourceIDs = normalizedBaselineSet(baseline.SourceIDs)
	baseline.MappedRefs = normalizedBaselineSet(baseline.MappedRefs)
	baseline.Consumers = normalizedBaselineSet(baseline.Consumers)
	baseline.Fingerprint = ""
	data, _ := json.Marshal(baseline)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func normalizedBaselineSet(values []string) []string {
	result := append([]string{}, values...)
	for index := range result {
		result[index] = strings.TrimSpace(result[index])
	}
	sort.Strings(result)
	return result
}

func containsBaselineValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// BaselineRequest binds approved UI meaning to the current UI workspace commit.
type BaselineRequest struct {
	Root, ID, WorkspaceID            string
	SourceIDs, MappedRefs, Consumers []string
}

// PlanBaseline verifies actual Git and source state before writing root-owned identity.
func PlanBaseline(ctx context.Context, request BaselineRequest) (Baseline, operation.Plan, []domain.Item, error) {
	root, err := filepath.Abs(request.Root)
	if err != nil {
		return Baseline{}, operation.Plan{}, nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return Baseline{}, operation.Plan{}, nil, err
	}
	plan := operation.Plan{ID: "ui-baseline-bind-" + strings.ReplaceAll(request.ID, ".", "-"), Root: root}
	warnings := []domain.Item{}
	add := func(code, message string, refs ...string) {
		plan.Blockers = append(plan.Blockers, domain.Item{Code: code, Message: message, Refs: refs})
	}
	manifest, err := workspace.Load(root)
	if err != nil {
		return Baseline{}, operation.Plan{}, nil, err
	}
	entry, found := uiWorkspace(manifest, request.WorkspaceID)
	if !found {
		add("ui.baseline-workspace-missing", "Baseline requires a declared UI workspace.", request.WorkspaceID)
		return Baseline{}, plan, warnings, nil
	}
	workspaceRoot := filepath.Join(root, filepath.FromSlash(entry.Path))
	if info, statErr := os.Lstat(workspaceRoot); statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		add("ui.baseline-workspace-unsafe", "UI workspace must be an existing non-symlink directory.", request.WorkspaceID)
		return Baseline{}, plan, warnings, nil
	}
	state, inspectErr := gitx.Inspect(ctx, workspaceRoot)
	if inspectErr != nil {
		add("ui.baseline-git-unknown", "UI workspace Git identity is unavailable.", request.WorkspaceID)
		return Baseline{}, plan, warnings, nil
	}
	remote, remoteErr := gitx.RemoteURL(ctx, workspaceRoot, "origin")
	if remoteErr != nil || !gitx.SafeRemoteURL(remote) {
		add("ui.baseline-remote-unsafe", "UI workspace needs a credential-free origin remote.", request.WorkspaceID)
	}
	if entry.Remote != "" && entry.Remote != remote {
		add("ui.baseline-remote-mismatch", "UI workspace origin differs from canonical workspace identity.", request.WorkspaceID)
	}
	if state.Dirty {
		add("ui.baseline-workspace-dirty", "Commit UI workspace changes before binding a baseline.", request.WorkspaceID)
	}
	if state.Detached && entry.Kind != "submodule" {
		add("ui.baseline-workspace-detached", "Standalone UI workspace must use an attached branch.", request.WorkspaceID)
	}
	if state.Diverged {
		add("ui.baseline-workspace-diverged", "UI workspace branch has diverged from its upstream.", request.WorkspaceID)
	}
	if !gitx.CommitPublished(ctx, workspaceRoot, state.Head) {
		add("ui.baseline-local-only", "UI baseline commit is not available from a remote-tracking branch.", request.WorkspaceID, state.Head)
	}
	sourceFingerprints := map[string]string{}
	sourceUpdates := []Registration{}
	for _, sourceID := range normalizedBaselineSet(request.SourceIDs) {
		registration, loadErr := LoadRegistration(root, sourceID)
		if loadErr != nil || (registration.Authority != "seed" && registration.Authority != "canonical") {
			add("ui.baseline-source-invalid", "Baseline sources must be promoted seed or canonical material.", sourceID)
			continue
		}
		sourceFingerprints[sourceID] = registration.ContentHash
		registration.BaselineFingerprint = registration.ContentHash
		sourceUpdates = append(sourceUpdates, registration)
		promotionPath := filepath.Join(workspaceRoot, "sources", sourceID, "promotion.yaml")
		promotion, loadErr := schema.LoadYAML[PromotionRecord](promotionPath)
		if loadErr != nil || promotion.SourceID != sourceID || promotion.WorkspaceID != request.WorkspaceID || promotion.ContentHash != registration.ContentHash {
			add("ui.baseline-source-unpromoted", "Baseline source was not promoted unchanged into this UI workspace.", sourceID)
		}
	}
	baseline := Baseline{SchemaVersion: 1, ID: request.ID, WorkspaceID: request.WorkspaceID, WorkspaceCommit: state.Head, WorkspaceRemote: remote, SourceIDs: normalizedBaselineSet(request.SourceIDs), SourceFingerprints: sourceFingerprints, MappedRefs: normalizedBaselineSet(request.MappedRefs), Consumers: normalizedBaselineSet(request.Consumers)}
	baseline.Fingerprint = BaselineFingerprint(baseline)
	plan.Blockers = append(plan.Blockers, ValidateBaseline(baseline)...)
	if schemaIssues := schema.Validate("ui-baseline", baseline); len(schemaIssues) > 0 {
		add("ui.baseline-schema-invalid", schemaIssues[0].Message)
	}
	if entry.Kind == "submodule" {
		rootState, rootErr := gitx.Inspect(ctx, root)
		if rootErr != nil {
			add("ui.baseline-root-git-unknown", "Orchestration root Git state is unavailable.")
		} else {
			for _, submodule := range rootState.Submodules {
				if submodule.Path == entry.Path && submodule.ExpectedSHA != state.Head {
					warnings = append(warnings, domain.Item{Code: "ui.baseline-pointer-pending", Message: "Commit the UI gitlink and baseline record together in the orchestration root.", Refs: []string{entry.ID, submodule.ExpectedSHA, state.Head}})
				}
			}
		}
	}
	if len(plan.Blockers) > 0 {
		return baseline, plan, warnings, nil
	}
	data, err := yaml.Marshal(baseline)
	if err != nil {
		return Baseline{}, operation.Plan{}, nil, err
	}
	plan.Files = []operation.FileChange{{Path: filepath.ToSlash(filepath.Join(".harness", "ui", "baselines", baseline.ID+".yaml")), Content: data, Mode: 0o644}}
	for _, registration := range sourceUpdates {
		registrationData, marshalErr := yaml.Marshal(registration)
		if marshalErr != nil {
			return Baseline{}, operation.Plan{}, nil, marshalErr
		}
		plan.Files = append(plan.Files, operation.FileChange{Path: filepath.ToSlash(filepath.Join(".harness", "sources", "ui", registration.ID+".yaml")), Content: registrationData, Mode: 0o644})
	}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return baseline, plan, warnings, err
}

// LoadBaseline reads one committed UI baseline without following links.
func LoadBaseline(root, id string) (Baseline, error) {
	if !baselineIDPattern.MatchString(id) {
		return Baseline{}, fmt.Errorf("UI baseline ID is invalid")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return Baseline{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return Baseline{}, err
	}
	path := filepath.Join(root, ".harness", "ui", "baselines", id+".yaml")
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Baseline{}, fmt.Errorf("UI baseline must be a regular non-symlink file")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(path) {
		return Baseline{}, fmt.Errorf("UI baseline cannot use symlinked storage")
	}
	baseline, err := schema.LoadYAML[Baseline](path)
	if err != nil {
		return Baseline{}, err
	}
	if issues := ValidateBaseline(baseline); len(issues) > 0 {
		return Baseline{}, fmt.Errorf("invalid UI baseline: %s", issues[0].Code)
	}
	if issues := schema.Validate("ui-baseline", baseline); len(issues) > 0 {
		return Baseline{}, fmt.Errorf("validate UI baseline: %s", issues[0].Message)
	}
	return baseline, nil
}
