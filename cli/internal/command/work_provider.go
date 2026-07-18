package command

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/policy"
	"fullstack-orchestrator/cli/internal/project"
	"fullstack-orchestrator/cli/internal/provider"
	"fullstack-orchestrator/cli/internal/work"
	"fullstack-orchestrator/cli/internal/workspace"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

func newWorkProvider(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "provider", Short: "Reconcile one selected live task source without copying live state into Git"}
	parent.AddCommand(newWorkProviderReconcile(version, jsonOutput))
	return parent
}

func newWorkProviderReconcile(version string, jsonOutput *bool) *cobra.Command {
	var root, mappingPath, snapshotPath string
	var apply bool
	command := &cobra.Command{
		Use:   "reconcile",
		Short: "Compare canonical work with a fresh normalized provider observation",
		RunE: func(cmd *cobra.Command, _ []string) error {
			located, err := workspace.FindRoot(cmd.Context(), root)
			if err != nil {
				return err
			}
			if err := provider.ValidateSnapshotLocation(located.Path, snapshotPath); err != nil {
				return err
			}
			mapping, err := provider.LoadMapping(mappingPath)
			if err != nil {
				return err
			}
			selected, err := loadTaskProvider(located.Path)
			if err != nil {
				return err
			}
			if selected.Provider != selected.LiveStatusSource || mapping.Provider != selected.LiveStatusSource {
				state := provider.State{
					Confidence: provider.Unknown,
					Provider:   mapping.Provider,
					ItemID:     mapping.ItemID,
					Issues: []domain.Item{{
						Code:    "provider.selected-mismatch",
						Message: "Provider mapping differs from the project's only selected live status source.",
						Refs:    []string{selected.LiveStatusSource, mapping.Provider},
					}},
				}
				return writeResult(cmd, *jsonOutput, providerResult(version, state))
			}
			snapshot, err := provider.LoadSnapshot(snapshotPath)
			if err != nil {
				return err
			}
			definitions, err := work.LoadDefinitions(located.Path)
			if err != nil {
				return err
			}
			definition, found := findDefinition(definitions, mapping.WorkID)
			if !found {
				return fmt.Errorf("provider mapping references missing work definition %s", mapping.WorkID)
			}
			expectation := provider.Expectation{WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Dependencies: definition.Dependencies}
			state := provider.Reconcile(expectation, mapping, snapshot, time.Now().UTC())
			result := providerResult(version, state)
			if state.Confidence != provider.Confirmed || !apply {
				if state.Confidence == provider.Confirmed {
					data, marshalErr := yaml.Marshal(mapping)
					if marshalErr != nil {
						return marshalErr
					}
					snapshotData, marshalErr := yaml.Marshal(snapshot)
					if marshalErr != nil {
						return marshalErr
					}
					plan, planErr := providerReconcilePlan(located.Path, mapping, data, snapshotData)
					if planErr != nil {
						return planErr
					}
					result.Changes = []domain.Item{{Code: "provider.mapping-planned", Message: plan.Files[0].Path}, {Code: "provider.observation-local-planned", Message: plan.Files[1].Path}}
				}
				return writeResult(cmd, *jsonOutput, result)
			}
			data, err := yaml.Marshal(mapping)
			if err != nil {
				return err
			}
			snapshotData, err := yaml.Marshal(snapshot)
			if err != nil {
				return err
			}
			plan, err := providerReconcilePlan(located.Path, mapping, data, snapshotData)
			if err != nil {
				return err
			}
			applied := operation.Apply(cmd.Context(), plan)
			applied.ToolVersion, applied.Command = version, "work.provider.reconcile"
			applied.Facts = result.Facts
			applied.Evidence = append(applied.Evidence, domain.Item{Code: "provider.live-revision", Message: state.Revision, Refs: []string{state.Provider, state.ItemID}})
			return writeResult(cmd, *jsonOutput, applied)
		},
	}
	command.Flags().StringVar(&root, "root", ".", "project root or child path")
	command.Flags().StringVar(&mappingPath, "mapping", "", "strict provider mapping YAML or JSON")
	command.Flags().StringVar(&snapshotPath, "snapshot", "", "fresh normalized snapshot under local provider state or a temporary path")
	command.Flags().BoolVar(&apply, "apply", false, "write the stable mapping and ignored local observation")
	_ = command.MarkFlagRequired("mapping")
	_ = command.MarkFlagRequired("snapshot")
	return command
}

func providerResult(version string, state provider.State) domain.Result {
	result := domain.Result{
		SchemaVersion: "1.0", ToolVersion: version, Command: "work.provider.reconcile", OperationID: "provider-reconcile-read-only",
		Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Provider item, revision, owner, dependencies, and work fingerprint were confirmed live.",
		Facts: []domain.Item{
			{Code: "provider.name", Message: state.Provider},
			{Code: "provider.item", Message: state.ItemID},
			{Code: "provider.revision", Message: state.Revision},
			{Code: "provider.status", Message: state.Status},
			{Code: "provider.owner", Message: state.Owner},
		},
	}
	if state.Confidence == provider.Confirmed {
		return result
	}
	result.Status, result.ExitCode, result.Summary = domain.StatusUnknown, domain.ExitUnavailable, "Provider state is stale, cached, drifted, or not strongly observable."
	for _, issue := range state.Issues {
		if providerIssueBlocks(issue.Code) {
			result.Blockers = append(result.Blockers, issue)
		} else {
			result.Warnings = append(result.Warnings, issue)
		}
	}
	if len(result.Blockers) > 0 {
		result.Status, result.ExitCode = domain.StatusBlocked, domain.ExitBlocked
	}
	result.NextActions = []domain.Item{{Code: "provider.refresh", Message: "Read the selected provider again through its chosen connector, then reconcile the new revision."}}
	return result
}

func providerIssueBlocks(code string) bool {
	return strings.Contains(code, "mismatch") || strings.Contains(code, "drift") || strings.Contains(code, "schema") || strings.Contains(code, "capability-invalid")
}

func findDefinition(definitions []work.Definition, id string) (work.Definition, bool) {
	for _, definition := range definitions {
		if definition.ID == id {
			return definition, true
		}
	}
	return work.Definition{}, false
}

func providerReconcilePlan(root string, mapping provider.Mapping, mappingData, snapshotData []byte) (operation.Plan, error) {
	digest := sha256.Sum256(snapshotData)
	plan := operation.Plan{
		ID:   "provider-reconcile-" + strings.ReplaceAll(mapping.WorkID, ".", "-") + "-" + hex.EncodeToString(digest[:6]),
		Root: root,
		Files: []operation.FileChange{
			{Path: filepath.ToSlash(filepath.Join(".harness", "work", "mappings", mapping.WorkID+".yaml")), Content: mappingData, Mode: 0o644},
			{Path: filepath.ToSlash(filepath.Join(".harness", "local", "providers", mapping.Provider, mapping.WorkID+".yaml")), Content: snapshotData, Mode: 0o600},
		},
	}
	var err error
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return plan, err
}

func loadStartDefinition(root, workID string) (work.Definition, bool, error) {
	definitions, err := work.LoadDefinitions(root)
	if err != nil {
		return work.Definition{}, false, err
	}
	definition, found := findDefinition(definitions, workID)
	return definition, found, nil
}

func candidateFromDefinition(definition work.Definition, now time.Time) policy.Candidate {
	repository := "repository.root"
	if len(definition.Scope.Repositories) > 0 {
		repository = strings.Join(definition.Scope.Repositories, ",")
	}
	return policy.Candidate{
		Repository:       repository,
		Paths:            append([]string(nil), definition.Scope.Paths...),
		PolicyIDs:        append([]string(nil), definition.Scope.PolicyIDs...),
		ScenarioIDs:      append([]string(nil), definition.Scope.ScenarioIDs...),
		ContractIDs:      append([]string(nil), definition.Scope.ContractIDs...),
		DBEntities:       append([]string(nil), definition.Scope.DBEntities...),
		MigrationSlots:   append([]string(nil), definition.Scope.MigrationSlots...),
		UIFlows:          append([]string(nil), definition.Scope.UIFlows...),
		DependencyMajors: append([]string(nil), definition.Scope.DependencyMajors...),
		StableIDs:        append([]string(nil), definition.Refs...),
		RootPointer:      len(definition.Scope.RootPointers) > 0,
		Now:              now,
	}
}

func applyCoordinatedStart(cmd *cobra.Command, jsonOutput bool, version string, request project.StartWorkRequest, definition work.Definition, plan operation.Plan, config taskProviderConfig, external *externalProviderObservation) error {
	store := provider.NewGitLocalStore(request.Root, config.Remote, config.CoordinationBranch)
	current, err := store.Read(cmd.Context())
	if errors.Is(err, provider.ErrNoRemote) {
		if external != nil {
			return writeResult(cmd, jsonOutput, lifecycleBlocked(version, "work.start", "coordination.remote-required", "External task assignment cannot reserve cross-repository semantic scope without a reachable Git coordination remote.", config.Remote))
		}
		result := operation.Apply(cmd.Context(), plan)
		result.ToolVersion, result.Command = version, "work.start"
		result.Warnings = append(result.Warnings, domain.Item{Code: "provider.single-user-local", Message: "No Git coordination remote was available; this claim is local advisory state and is not a team lock."})
		if result.Status == domain.StatusPassed {
			result.NextActions = append(result.NextActions, domain.Item{Code: "git.create-worktree", Message: "Create and verify the conventional branch in an isolated worktree from the reviewed base before editing.", Refs: []string{request.Branch}})
		}
		return writeResult(cmd, jsonOutput, result)
	}
	if err != nil {
		return writeResult(cmd, jsonOutput, gitLocalFailureResult(version, "provider.live-read-failed", "Live Git-local coordination could not be read safely.", err))
	}
	for _, claim := range current.Claims {
		if claim.WorkID == request.WorkID && (claim.Status == "integrated" || claim.Status == "done") {
			return writeResult(cmd, jsonOutput, domain.Result{
				SchemaVersion: "1.0", ToolVersion: version, Command: "work.start", OperationID: "work-start-terminal-read-only",
				Status: domain.StatusBlocked, ExitCode: domain.ExitBlocked, Summary: "Completed or integrated work cannot be claimed again under the same stable ID.",
				Blockers: []domain.Item{{Code: "work.already-terminal", Message: "Define a new work ID if additional product behavior is required.", Refs: []string{request.WorkID, claim.Status}}},
			})
		}
	}
	liveClaims := make([]provider.GitLocalClaim, 0, len(current.Claims))
	for _, claim := range current.Claims {
		if provider.GitLocalClaimActive(claim, request.Candidate.Now) {
			liveClaims = append(liveClaims, claim)
		}
	}
	active := make([]policy.Claim, 0, len(liveClaims))
	for _, claim := range liveClaims {
		active = append(active, policyClaimFromGitLocal(claim))
	}
	report := policy.CheckConflict(request.Candidate, active, request.Snapshot)
	if report.Level != policy.ConflictClear {
		return writeResult(cmd, jsonOutput, domain.Result{
			SchemaVersion: "1.0", ToolVersion: version, Command: "work.start", OperationID: "work-start-conflict-read-only",
			Status: domain.StatusBlocked, ExitCode: domain.ExitBlocked, Summary: report.NextAction, Blockers: report.Reasons,
			NextActions: []domain.Item{{Code: "work.refresh-preflight", Message: "Refresh the live claim set, agree ownership and merge order, then retry."}},
		})
	}
	next := provider.SnapshotSet{SchemaVersion: 1, Claims: append([]provider.GitLocalClaim(nil), liveClaims...)}
	next.Claims = append(next.Claims, gitLocalClaimFromStart(request, definition))
	revision, err := store.CompareAndSwap(cmd.Context(), current.Revision, next)
	if err != nil {
		code, message := "provider.claim-failed", "Git-local claim publication could not be verified."
		if errors.Is(err, provider.ErrCASConflict) {
			code, message = "provider.claim-race", "Another collaborator changed live claims first; no branch work was started."
		}
		return writeResult(cmd, jsonOutput, gitLocalFailureResult(version, code, message, err))
	}
	result := operation.Apply(cmd.Context(), plan)
	result.ToolVersion, result.Command = version, "work.start"
	result.Facts = append(result.Facts, domain.Item{Code: "provider.name", Message: config.LiveStatusSource})
	if external == nil {
		result.Evidence = append(result.Evidence, domain.Item{Code: "provider.git-local-claim", Message: revision, Refs: []string{request.WorkID, request.Owner}})
	} else {
		result.Evidence = append(result.Evidence,
			domain.Item{Code: "provider.live-revision", Message: observationRevision(*external), Refs: []string{external.State.Provider, external.State.ItemID}},
			domain.Item{Code: "coordination.semantic-reservation", Message: revision, Refs: []string{request.WorkID, request.Owner}},
		)
		if external.Snapshot.Capabilities.Claim == "advisory" {
			result.Warnings = append(result.Warnings, domain.Item{Code: "provider.assignment-advisory", Message: "The task provider assignment is advisory; the verified Git CAS reservation is the exclusive semantic lock.", Refs: []string{external.State.Provider, external.State.ItemID}})
		}
	}
	if result.Status != domain.StatusPassed {
		result.Warnings = append(result.Warnings, domain.Item{Code: "provider.claim-active", Message: "The remote claim succeeded but the local checkpoint failed; resume the same work instead of claiming again.", Refs: []string{request.WorkID, revision}})
	} else {
		result.NextActions = append(result.NextActions, domain.Item{Code: "git.create-worktree", Message: "Create and verify the conventional branch in an isolated worktree from the reviewed base before editing.", Refs: []string{request.Branch}})
	}
	return writeResult(cmd, jsonOutput, result)
}

func gitLocalClaimFromStart(request project.StartWorkRequest, definition work.Definition) provider.GitLocalClaim {
	return provider.GitLocalClaim{
		ID: request.ClaimID, WorkID: request.WorkID, DefinitionFingerprint: definition.Fingerprint,
		Status: "in_progress",
		Owner:  request.Owner, Branch: request.Branch, Repository: request.Candidate.Repository, Workspace: request.Candidate.Workspace,
		Paths: append([]string(nil), request.Candidate.Paths...), PolicyIDs: append([]string(nil), request.Candidate.PolicyIDs...), ScenarioIDs: append([]string(nil), request.Candidate.ScenarioIDs...),
		ContractIDs: append([]string(nil), request.Candidate.ContractIDs...), DBEntities: append([]string(nil), request.Candidate.DBEntities...), MigrationSlots: append([]string(nil), request.Candidate.MigrationSlots...),
		UIFlows: append([]string(nil), request.Candidate.UIFlows...), DependencyMajors: append([]string(nil), request.Candidate.DependencyMajors...), StableIDs: append([]string(nil), request.Candidate.StableIDs...),
		RootPointer: request.Candidate.RootPointer, StartsAt: request.Candidate.Now.UTC(), ExpiresAt: request.ExpiresAt.UTC(),
	}
}

func policyClaimFromGitLocal(claim provider.GitLocalClaim) policy.Claim {
	return policy.Claim{
		SchemaVersion: 1, ID: claim.ID, WorkID: claim.WorkID, Repository: claim.Repository, Workspace: claim.Workspace,
		Owner: claim.Owner, Branch: claim.Branch, Paths: append([]string(nil), claim.Paths...), PolicyIDs: append([]string(nil), claim.PolicyIDs...),
		ScenarioIDs: append([]string(nil), claim.ScenarioIDs...), ContractIDs: append([]string(nil), claim.ContractIDs...), DBEntities: append([]string(nil), claim.DBEntities...),
		MigrationSlots: append([]string(nil), claim.MigrationSlots...), UIFlows: append([]string(nil), claim.UIFlows...), DependencyMajors: append([]string(nil), claim.DependencyMajors...),
		StableIDs: append([]string(nil), claim.StableIDs...), RootPointer: claim.RootPointer, StartsAt: claim.StartsAt, ExpiresAt: claim.ExpiresAt, Observable: true,
	}
}

func gitLocalFailureResult(version, code, message string, err error) domain.Result {
	return domain.Result{
		SchemaVersion: "1.0", ToolVersion: version, Command: "work.start", OperationID: "work-start-git-local",
		Status: domain.StatusUnknown, ExitCode: domain.ExitUnavailable, Summary: message,
		Blockers:    []domain.Item{{Code: code, Message: err.Error()}},
		NextActions: []domain.Item{{Code: "work.refresh-preflight", Message: "Refresh live coordination and actual Git state before retrying."}},
	}
}
