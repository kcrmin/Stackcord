package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	contextpkg "github.com/kcrmin/Stackcord/cli/internal/context"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/policy"
	"github.com/kcrmin/Stackcord/cli/internal/project"
	"github.com/kcrmin/Stackcord/cli/internal/provider"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
	workpkg "github.com/kcrmin/Stackcord/cli/internal/work"
	"github.com/kcrmin/Stackcord/cli/internal/workspace"
	"github.com/spf13/cobra"
)

func newWorkCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "work", Short: "Choose, claim, verify, and transfer collaborative work"}
	parent.AddCommand(newWorkDefine(version, jsonOutput), newWorkProvider(version, jsonOutput), newWorkNext(version, jsonOutput), newWorkConflict(version, jsonOutput), newWorkStart(version, jsonOutput), newWorkEvidence(version, jsonOutput), newWorkTransition(version, jsonOutput, ""), newWorkTransition(version, jsonOutput, workpkg.Done), newWorkHandoff(version, jsonOutput))
	return parent
}

func newWorkNext(version string, jsonOutput *bool) *cobra.Command {
	var root string
	command := &cobra.Command{Use: "next", Short: "Recommend the next dependency-ready work item", RunE: func(cmd *cobra.Command, _ []string) error {
		providerConfig, err := loadTaskProvider(root)
		if err != nil {
			return err
		}
		items, err := loadWorkItems(root)
		if err != nil {
			return err
		}
		claims, liveStatuses, providerComplete, providerIssues, err := loadClaimsAndStatuses(cmd.Context(), root)
		if err != nil {
			return err
		}
		if providerConfig.LiveStatusSource != "git-local" && !providerComplete {
			result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "work.next", OperationID: "work-next-read-only", Status: domain.StatusUnknown, ExitCode: domain.ExitUnavailable, Summary: "The selected external task source has not been freshly observed for every canonical work item.", Facts: []domain.Item{{Code: "work.provider", Message: providerConfig.LiveStatusSource}}, Warnings: providerIssues, NextActions: []domain.Item{{Code: "work.provider-check", Message: "Read changed or missing items through the selected connector, reconcile them, then ask for the next task again."}}}
			return writeResult(cmd, *jsonOutput, result)
		}
		snapshot, issues := contextpkg.Refresh(cmd.Context(), root, contextpkg.ReadOnly)
		done := map[string]bool{}
		for _, item := range items {
			state, observed := liveStatuses[item.ID]
			if providerConfig.LiveStatusSource != "git-local" && !observed {
				continue
			}
			done[item.ID] = item.Status == domain.WorkDone || (observed && (state == workpkg.Integrated || state == workpkg.Done))
		}
		var ready []domain.WorkItem
		for _, item := range items {
			state, observed := liveStatuses[item.ID]
			if providerConfig.LiveStatusSource != "git-local" && !observed {
				continue
			}
			if observed && state != workpkg.Proposed && state != workpkg.ReadyState {
				continue
			}
			if item.Status != domain.WorkReady && item.Status != domain.WorkProposed {
				continue
			}
			if workClaimed(item.ID, claims, time.Now().UTC()) || refsUncertain(item.Refs, snapshot) {
				continue
			}
			unblocked := true
			for _, dependency := range item.Dependencies {
				if !done[dependency] {
					unblocked = false
				}
			}
			if unblocked {
				ready = append(ready, item)
			}
		}
		sort.Slice(ready, func(i, j int) bool { return ready[i].ID < ready[j].ID })
		result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "work.next", OperationID: "work-next-read-only", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Dependency-ready work was evaluated from the selected live task source and Git semantic reservations.", Warnings: providerIssues}
		for _, issue := range issues {
			if strings.HasPrefix(issue.Code, "context.error") {
				result.Blockers = append(result.Blockers, issue)
			}
		}
		if len(result.Blockers) > 0 {
			result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitBlocked, "Work cannot be selected until project context is valid."
			return writeResult(cmd, *jsonOutput, result)
		}
		if len(ready) == 0 {
			result.Status, result.ExitCode, result.Summary = domain.StatusUnknown, domain.ExitUnavailable, "No dependency-ready local work item was found; inspect the configured live provider."
			result.NextActions = []domain.Item{{Code: "work.provider-check", Message: "Restore live provider visibility or create an approved local work item."}}
		} else {
			item := ready[0]
			result.Facts = []domain.Item{{Code: "work.recommended", Message: item.Title, Refs: append([]string{item.ID}, item.Refs...)}}
			result.NextActions = []domain.Item{{Code: "work.start", Message: "Run conflict preflight and start the recommended item.", Refs: []string{item.ID}}}
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&root, "root", ".", "project root")
	return command
}

func newWorkConflict(version string, jsonOutput *bool) *cobra.Command {
	var root, candidatePath string
	command := &cobra.Command{Use: "conflict", Short: "Check filesystem and semantic overlap before implementation", RunE: func(cmd *cobra.Command, _ []string) error {
		candidate, err := loadYAML[policy.Candidate](candidatePath)
		if err != nil {
			return err
		}
		claims, err := loadClaims(cmd.Context(), root)
		if err != nil {
			return err
		}
		snapshot, _ := contextpkg.Refresh(cmd.Context(), root, contextpkg.ReadOnly)
		report := policy.CheckConflict(candidate, claims, snapshot)
		result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "work.conflict", OperationID: "work-conflict-read-only", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "No active collaboration conflict was found.", Facts: []domain.Item{{Code: "conflict.level", Message: string(report.Level)}}}
		if report.Level != policy.ConflictClear {
			result.Summary = report.NextAction
			result.Blockers = report.Reasons
			result.Status, result.ExitCode = domain.StatusBlocked, domain.ExitBlocked
			if report.Level == policy.ConflictUnknown {
				result.Status, result.ExitCode = domain.StatusUnknown, domain.ExitUnavailable
			} else if report.Level == policy.ConflictCoordinate {
				result.Status, result.ExitCode = domain.StatusWarning, domain.ExitSuccess
			}
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&root, "root", ".", "project root")
	command.Flags().StringVar(&candidatePath, "candidate", "", "candidate scope YAML")
	_ = command.MarkFlagRequired("candidate")
	return command
}

func newWorkStart(version string, jsonOutput *bool) *cobra.Command {
	var request project.StartWorkRequest
	var paths, policies, scenarios, contracts, entities, migrations, flows, dependencies, stableIDs []string
	var lease time.Duration
	var apply bool
	command := &cobra.Command{Use: "start", Short: "Create a time-bounded semantic claim and branch checkpoint", RunE: func(cmd *cobra.Command, _ []string) error {
		located, err := workspace.FindRoot(cmd.Context(), request.Root)
		if err != nil {
			return err
		}
		request.Root = located.Path
		now := time.Now().UTC()
		request.Candidate = policy.Candidate{Repository: "repository.root", Paths: paths, PolicyIDs: policies, ScenarioIDs: scenarios, ContractIDs: contracts, DBEntities: entities, MigrationSlots: migrations, UIFlows: flows, DependencyMajors: dependencies, StableIDs: stableIDs, Now: now}
		definition, definitionFound, err := loadStartDefinition(request.Root, request.WorkID)
		if err != nil {
			return err
		}
		if definitionFound {
			request.Candidate = candidateFromDefinition(definition, now)
		}
		providerConfig, err := loadTaskProvider(request.Root)
		if err != nil {
			return err
		}
		claims, err := loadClaims(cmd.Context(), request.Root)
		if err != nil {
			return err
		}
		if apply && providerConfig.LiveStatusSource != "git-local" {
			filtered := claims[:0]
			for _, claim := range claims {
				if claim.ID != "claim.external-provider-unobservable" {
					filtered = append(filtered, claim)
				}
			}
			claims = filtered
		}
		request.ActiveClaims = claims
		request.Snapshot, _ = contextpkg.Refresh(cmd.Context(), request.Root, contextpkg.ReadOnly)
		plan := project.StartWork(request)
		if apply && len(plan.Blockers) == 0 {
			config := providerConfig
			if config.LiveStatusSource != "git-local" {
				if !definitionFound {
					return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.start", "work.definition-required", "External task coordination requires an executable canonical work definition.", request.WorkID))
				}
				observation, observationErr := loadExternalProviderObservation(request.Root, config, definition, time.Now().UTC())
				if observationErr != nil || observation.State.Confidence != provider.Confirmed {
					return writeResult(cmd, *jsonOutput, externalObservationBlocked(version, "work.start", observation, observationErr))
				}
				if observation.State.Status != string(workpkg.InProgress) {
					return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.start", "provider.status-not-active", "Assign the item and move it to in_progress in the selected task source before reserving implementation scope.", observation.State.Status, observation.State.ItemID))
				}
				if strings.TrimSpace(observation.State.Owner) != strings.TrimSpace(request.Owner) {
					return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.start", "provider.owner-mismatch", "The requested collaborator must match the freshly observed task owner.", request.Owner, observation.State.Owner))
				}
				return applyCoordinatedStart(cmd, *jsonOutput, version, request, definition, plan, config, &observation)
			}
			if !definitionFound {
				_, remoteErr := provider.NewGitLocalStore(request.Root, config.Remote, config.CoordinationBranch).Read(cmd.Context())
				if remoteErr == nil {
					result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "work.start", OperationID: "work-start-definition-read-only", Status: domain.StatusBlocked, ExitCode: domain.ExitBlocked, Summary: "Collaborative work needs an executable canonical definition before it can be claimed.", Blockers: []domain.Item{{Code: "work.definition-required", Message: "Define outcome, acceptance, semantic scope, merge order, first failing test, and required evidence before claiming this work."}}}
					return writeResult(cmd, *jsonOutput, result)
				}
				if !errors.Is(remoteErr, provider.ErrNoRemote) {
					return writeResult(cmd, *jsonOutput, gitLocalFailureResult(version, "provider.live-read-failed", "Live Git-local coordination could not be read safely.", remoteErr))
				}
				result := operation.Apply(cmd.Context(), plan)
				result.ToolVersion, result.Command = version, "work.start"
				result.Warnings = append(result.Warnings, domain.Item{Code: "provider.single-user-local", Message: "No executable work definition and remote coordination source were confirmed; this is a local advisory claim only."})
				if result.Status == domain.StatusPassed {
					result.NextActions = append(result.NextActions, domain.Item{Code: "git.create-worktree", Message: "Create and verify the conventional branch in an isolated worktree from the reviewed base before editing.", Refs: []string{request.Branch}})
				}
				return writeResult(cmd, *jsonOutput, result)
			}
			return applyCoordinatedStart(cmd, *jsonOutput, version, request, definition, plan, config, nil)
		}
		return writeResult(cmd, *jsonOutput, planResult(version, "work.start.plan", plan, "Work claim and branch checkpoint plan is ready."))
	}}
	command.Flags().StringVar(&request.Root, "root", ".", "project root")
	command.Flags().StringVar(&request.WorkID, "work-id", "", "work stable instance ID")
	command.Flags().StringVar(&request.ClaimID, "claim-id", "", "claim stable instance ID")
	command.Flags().StringVar(&request.Owner, "owner", "", "claim owner")
	command.Flags().StringVar(&request.Branch, "branch", "", "conventional branch name")
	command.Flags().DurationVar(&lease, "lease", 24*time.Hour, "claim lease duration")
	command.PreRun = func(_ *cobra.Command, _ []string) { request.ExpiresAt = time.Now().UTC().Add(lease) }
	command.Flags().StringSliceVar(&paths, "path", nil, "path scope")
	command.Flags().StringSliceVar(&policies, "policy", nil, "policy stable ID")
	command.Flags().StringSliceVar(&scenarios, "scenario", nil, "scenario stable ID")
	command.Flags().StringSliceVar(&contracts, "contract", nil, "contract stable ID")
	command.Flags().StringSliceVar(&entities, "db-entity", nil, "database entity")
	command.Flags().StringSliceVar(&migrations, "migration-slot", nil, "migration slot")
	command.Flags().StringSliceVar(&flows, "ui-flow", nil, "UI flow")
	command.Flags().StringSliceVar(&dependencies, "dependency-major", nil, "dependency major transition")
	command.Flags().StringSliceVar(&stableIDs, "ref", nil, "related stable product or contract ID")
	command.Flags().BoolVar(&apply, "apply", false, "write the reviewed claim plan")
	for _, flag := range []string{"work-id", "claim-id", "owner", "branch"} {
		_ = command.MarkFlagRequired(flag)
	}
	return command
}

func workClaimed(workID string, claims []policy.Claim, now time.Time) bool {
	for _, claim := range claims {
		if claim.WorkID == workID && (claim.ExpiresAt.IsZero() || claim.ExpiresAt.After(now)) {
			return true
		}
	}
	return false
}

func refsUncertain(refs []string, snapshot contextpkg.Snapshot) bool {
	for _, ref := range refs {
		if _, exists := snapshot.Index[ref]; !exists {
			return true
		}
		for _, uncertain := range append(append([]string(nil), snapshot.Stale...), snapshot.Unknown...) {
			if uncertain == ref || strings.HasPrefix(uncertain, ref+".") {
				return true
			}
		}
	}
	return false
}

func loadWorkItems(root string) ([]domain.WorkItem, error) {
	definitions, err := workpkg.LoadDefinitions(root)
	if err != nil {
		return nil, err
	}
	itemsByID := map[string]domain.WorkItem{}
	for _, definition := range definitions {
		status := domain.WorkProposed
		if definition.Readiness == workpkg.Ready {
			status = domain.WorkReady
		}
		itemsByID[definition.ID] = domain.WorkItem{SchemaVersion: 1, ID: definition.ID, Title: definition.Title, Status: status, Refs: definition.Refs, Dependencies: definition.Dependencies}
	}
	directory := filepath.Join(root, ".harness", "work", "items")
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return sortedWorkItems(itemsByID), nil
	}
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		item, loadErr := loadYAML[domain.WorkItem](filepath.Join(directory, entry.Name()))
		if loadErr != nil {
			return nil, loadErr
		}
		if issues := schema.Validate("work-item", item); len(issues) > 0 {
			return nil, fmt.Errorf("validate %s: %s", entry.Name(), issues[0].Message)
		}
		if _, canonical := itemsByID[item.ID]; !canonical {
			itemsByID[item.ID] = item
		}
	}
	return sortedWorkItems(itemsByID), nil
}

func sortedWorkItems(items map[string]domain.WorkItem) []domain.WorkItem {
	result := make([]domain.WorkItem, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ID < result[right].ID })
	return result
}

func loadClaims(ctx context.Context, root string) ([]policy.Claim, error) {
	claims, _, _, _, err := loadClaimsAndStatuses(ctx, root)
	return claims, err
}

func loadClaimsAndStatuses(ctx context.Context, root string) ([]policy.Claim, map[string]workpkg.State, bool, []domain.Item, error) {
	providerConfig, err := loadTaskProvider(root)
	if err != nil {
		return nil, nil, false, nil, err
	}
	directory := filepath.Join(root, ".harness", "work", "claims")
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		entries = nil
	} else if err != nil {
		return nil, nil, false, nil, err
	}
	localClaims := []policy.Claim{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		claim, loadErr := loadYAML[policy.Claim](filepath.Join(directory, entry.Name()))
		if loadErr != nil {
			return nil, nil, false, nil, loadErr
		}
		if issues := schema.Validate("claim", claim); len(issues) > 0 {
			return nil, nil, false, nil, fmt.Errorf("validate %s: %s", entry.Name(), issues[0].Message)
		}
		claim.Observable = false
		localClaims = append(localClaims, claim)
	}
	observed, readErr := provider.NewGitLocalStore(root, providerConfig.Remote, providerConfig.CoordinationBranch).Read(ctx)
	if errors.Is(readErr, provider.ErrNoRemote) {
		if providerConfig.LiveStatusSource != "git-local" {
			localClaims = append(localClaims, policy.Claim{ID: "claim.external-provider-unobservable", Observable: false})
			externalStatuses, complete, issues, statusErr := externalProviderStatuses(root, providerConfig, time.Now().UTC())
			if statusErr != nil {
				return nil, nil, false, nil, statusErr
			}
			issues = append(issues, domain.Item{Code: "coordination.remote-unavailable", Message: "Work can be selected, but collaborative semantic scope cannot be reserved until the Git coordination remote is reachable.", Refs: []string{providerConfig.Remote}})
			return localClaims, externalStatuses, complete, issues, nil
		}
		return localClaims, map[string]workpkg.State{}, true, nil, nil
	}
	if readErr != nil {
		return nil, nil, false, nil, readErr
	}
	claims := make([]policy.Claim, 0, len(observed.Claims))
	statuses := make(map[string]workpkg.State, len(observed.Claims))
	for _, claim := range observed.Claims {
		if providerConfig.LiveStatusSource == "git-local" {
			statuses[claim.WorkID] = gitLocalWorkState(claim.Status)
		}
		if provider.GitLocalClaimActive(claim, time.Now().UTC()) {
			claims = append(claims, policyClaimFromGitLocal(claim))
		}
	}
	if providerConfig.LiveStatusSource == "git-local" {
		return claims, statuses, true, nil, nil
	}
	externalStatuses, complete, issues, err := externalProviderStatuses(root, providerConfig, time.Now().UTC())
	if err != nil {
		return nil, nil, false, nil, err
	}
	return claims, externalStatuses, complete, issues, nil
}

type taskProviderConfig struct {
	SchemaVersion      int    `yaml:"schema_version"`
	Provider           string `yaml:"provider"`
	LiveStatusSource   string `yaml:"live_status_source"`
	Remote             string `yaml:"remote,omitempty"`
	CoordinationBranch string `yaml:"coordination_branch,omitempty"`
}

func loadTaskProvider(root string) (taskProviderConfig, error) {
	path := filepath.Join(root, ".harness", "work", "provider.yaml")
	config, err := schema.LoadYAML[taskProviderConfig](path)
	if errors.Is(err, os.ErrNotExist) {
		return taskProviderConfig{SchemaVersion: 1, Provider: "git-local", LiveStatusSource: "git-local", Remote: "origin", CoordinationBranch: "coordination"}, nil
	}
	if err != nil {
		return taskProviderConfig{}, err
	}
	if config.SchemaVersion != 1 || config.Provider == "" || config.LiveStatusSource == "" {
		return taskProviderConfig{}, fmt.Errorf("task provider configuration is incomplete")
	}
	if config.Remote == "" {
		config.Remote = "origin"
	}
	if config.CoordinationBranch == "" {
		config.CoordinationBranch = "coordination"
	}
	return config, nil
}

func loadYAML[T any](path string) (T, error) {
	return schema.LoadYAML[T](path)
}
