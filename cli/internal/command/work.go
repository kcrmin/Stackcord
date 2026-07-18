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

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/gitx"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/policy"
	"fullstack-orchestrator/cli/internal/project"
	"fullstack-orchestrator/cli/internal/schema"
	workpkg "fullstack-orchestrator/cli/internal/work"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

func newWorkCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "work", Short: "Choose, claim, verify, and transfer collaborative work"}
	parent.AddCommand(newWorkDefine(version, jsonOutput), newWorkNext(version, jsonOutput), newWorkConflict(version, jsonOutput), newWorkStart(version, jsonOutput), newWorkFinish(version, jsonOutput), newWorkHandoff(version, jsonOutput))
	return parent
}

func newWorkNext(version string, jsonOutput *bool) *cobra.Command {
	var root string
	command := &cobra.Command{Use: "next", Short: "Recommend the next dependency-ready work item", RunE: func(cmd *cobra.Command, _ []string) error {
		providerConfig, err := loadTaskProvider(root)
		if err != nil {
			return err
		}
		if providerConfig.LiveStatusSource != "git-local" {
			result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "work.next", OperationID: "work-next-read-only", Status: domain.StatusUnknown, ExitCode: domain.ExitUnavailable, Summary: "The configured external task provider must be queried before recommending work.", Facts: []domain.Item{{Code: "work.provider", Message: providerConfig.LiveStatusSource}}, NextActions: []domain.Item{{Code: "work.provider-check", Message: "Use the configured provider adapter and rerun after live status is observable."}}}
			return writeResult(cmd, *jsonOutput, result)
		}
		items, err := loadWorkItems(root)
		if err != nil {
			return err
		}
		claims, err := loadClaims(cmd.Context(), root)
		if err != nil {
			return err
		}
		snapshot, issues := contextpkg.Refresh(cmd.Context(), root, contextpkg.ReadOnly)
		done := map[string]bool{}
		for _, item := range items {
			done[item.ID] = item.Status == domain.WorkDone
		}
		var ready []domain.WorkItem
		for _, item := range items {
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
		result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "work.next", OperationID: "work-next-read-only", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Dependency-ready work was evaluated from the selected local provider."}
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
		request.Candidate = policy.Candidate{Repository: "root", Paths: paths, PolicyIDs: policies, ScenarioIDs: scenarios, ContractIDs: contracts, DBEntities: entities, MigrationSlots: migrations, UIFlows: flows, DependencyMajors: dependencies, StableIDs: stableIDs, Now: time.Now().UTC()}
		claims, err := loadClaims(cmd.Context(), request.Root)
		if err != nil {
			return err
		}
		request.ActiveClaims = claims
		request.Snapshot, _ = contextpkg.Refresh(cmd.Context(), request.Root, contextpkg.ReadOnly)
		plan := project.StartWork(request)
		if apply && len(plan.Blockers) == 0 {
			result := operation.Apply(cmd.Context(), plan)
			result.ToolVersion, result.Command = version, "work.start"
			return writeResult(cmd, *jsonOutput, result)
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

func newWorkFinish(version string, jsonOutput *bool) *cobra.Command {
	var workID string
	var evidence []string
	command := &cobra.Command{Use: "finish", Short: "Verify reproducible evidence before completing work", RunE: func(cmd *cobra.Command, _ []string) error {
		result := project.FinishWork(project.FinishWorkRequest{WorkID: workID, Evidence: evidence})
		result.ToolVersion = version
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&workID, "work-id", "", "work ID")
	command.Flags().StringSliceVar(&evidence, "evidence", nil, "verification receipt or command")
	_ = command.MarkFlagRequired("work-id")
	return command
}

func newWorkHandoff(version string, jsonOutput *bool) *cobra.Command {
	var root, workID, owner, current, next string
	var evidence []string
	var apply bool
	command := &cobra.Command{Use: "handoff", Short: "Record an explicit ownership-transfer checkpoint", RunE: func(cmd *cobra.Command, _ []string) error {
		content, _ := yaml.Marshal(map[string]any{"schema_version": 1, "work_id": workID, "receiving_owner": owner, "current_state": current, "next_action": next, "evidence": evidence})
		plan := operation.Plan{ID: "handoff-" + strings.ReplaceAll(workID, ".", "-"), Root: root, Files: []operation.FileChange{{Path: filepath.ToSlash(filepath.Join(".harness", "work", "handoffs", workID+".yaml")), Content: content, Mode: 0o644}}}
		plan.InitialStateFingerprint, _ = operation.StateFingerprint(plan)
		if apply {
			result := operation.Apply(cmd.Context(), plan)
			result.ToolVersion, result.Command = version, "work.handoff"
			return writeResult(cmd, *jsonOutput, result)
		}
		return writeResult(cmd, *jsonOutput, planResult(version, "work.handoff.plan", plan, "Ownership-transfer checkpoint plan is ready."))
	}}
	command.Flags().StringVar(&root, "root", ".", "project root")
	command.Flags().StringVar(&workID, "work-id", "", "work ID")
	command.Flags().StringVar(&owner, "owner", "", "receiving owner")
	command.Flags().StringVar(&current, "current-state", "", "normalized current state")
	command.Flags().StringVar(&next, "next-action", "", "one reproducible next action")
	command.Flags().StringSliceVar(&evidence, "evidence", nil, "evidence receipt")
	command.Flags().BoolVar(&apply, "apply", false, "write the reviewed handoff")
	for _, flag := range []string{"work-id", "owner", "next-action"} {
		_ = command.MarkFlagRequired(flag)
	}
	return command
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
	providerConfig, err := loadTaskProvider(root)
	if err != nil {
		return nil, err
	}
	directory := filepath.Join(root, ".harness", "work", "claims")
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		entries = nil
	} else if err != nil {
		return nil, err
	}
	claims := []policy.Claim{}
	if providerConfig.LiveStatusSource != "git-local" {
		claims = append(claims, policy.Claim{ID: "claim.external-provider-unobservable", Observable: false})
	}
	byID := map[string]int{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		claim, loadErr := loadYAML[policy.Claim](filepath.Join(directory, entry.Name()))
		if loadErr != nil {
			return nil, loadErr
		}
		if issues := schema.Validate("claim", claim); len(issues) > 0 {
			return nil, fmt.Errorf("validate %s: %s", entry.Name(), issues[0].Message)
		}
		claim.Observable = true
		byID[claim.ID] = len(claims)
		claims = append(claims, claim)
	}
	if _, err := os.Lstat(filepath.Join(root, ".git")); err == nil {
		remoteFiles, readErr := gitx.ReadRemoteFiles(ctx, root, ".harness/work/claims", ".yaml")
		if readErr != nil {
			return nil, readErr
		}
		for _, remoteFile := range remoteFiles {
			claim, decodeErr := schema.DecodeYAML[policy.Claim](remoteFile.Data)
			if decodeErr != nil {
				return nil, fmt.Errorf("decode %s:%s: %w", remoteFile.Ref, remoteFile.Path, decodeErr)
			}
			if issues := schema.Validate("claim", claim); len(issues) > 0 {
				return nil, fmt.Errorf("validate %s:%s: %s", remoteFile.Ref, remoteFile.Path, issues[0].Message)
			}
			claim.Observable = true
			if index, exists := byID[claim.ID]; exists {
				if !sameClaim(claims[index], claim) {
					claims[index].Observable = false
				}
				continue
			}
			byID[claim.ID] = len(claims)
			claims = append(claims, claim)
		}
	}
	return claims, nil
}

type taskProviderConfig struct {
	SchemaVersion    int    `yaml:"schema_version"`
	Provider         string `yaml:"provider"`
	LiveStatusSource string `yaml:"live_status_source"`
}

func loadTaskProvider(root string) (taskProviderConfig, error) {
	path := filepath.Join(root, ".harness", "work", "provider.yaml")
	config, err := schema.LoadYAML[taskProviderConfig](path)
	if errors.Is(err, os.ErrNotExist) {
		return taskProviderConfig{SchemaVersion: 1, Provider: "git-local", LiveStatusSource: "git-local"}, nil
	}
	if err != nil {
		return taskProviderConfig{}, err
	}
	if config.SchemaVersion != 1 || config.Provider == "" || config.LiveStatusSource == "" {
		return taskProviderConfig{}, fmt.Errorf("task provider configuration is incomplete")
	}
	return config, nil
}

func sameClaim(left, right policy.Claim) bool {
	left.Observable, right.Observable = true, true
	leftData, leftErr := yaml.Marshal(left)
	rightData, rightErr := yaml.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftData) == string(rightData)
}

func loadYAML[T any](path string) (T, error) {
	return schema.LoadYAML[T](path)
}
