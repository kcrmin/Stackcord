package command

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/policy"
	"fullstack-orchestrator/cli/internal/project"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

func newWorkCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "work", Short: "Choose, claim, verify, and transfer collaborative work"}
	parent.AddCommand(newWorkNext(version, jsonOutput), newWorkConflict(version, jsonOutput), newWorkStart(version, jsonOutput), newWorkFinish(version, jsonOutput), newWorkHandoff(version, jsonOutput))
	return parent
}

func newWorkNext(version string, jsonOutput *bool) *cobra.Command {
	var root string
	command := &cobra.Command{Use: "next", Short: "Recommend the next dependency-ready work item", RunE: func(cmd *cobra.Command, _ []string) error {
		items, err := loadWorkItems(root)
		if err != nil {
			return err
		}
		done := map[string]bool{}
		for _, item := range items {
			done[item.ID] = item.Status == domain.WorkDone
		}
		var ready []domain.WorkItem
		for _, item := range items {
			if item.Status != domain.WorkReady && item.Status != domain.WorkProposed {
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
		if len(ready) == 0 {
			result.Status, result.Summary = domain.StatusUnknown, "No dependency-ready local work item was found; inspect the configured live provider."
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
		claims, err := loadClaims(root)
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
	var paths, policies, scenarios, contracts, entities, migrations, flows, dependencies []string
	var apply bool
	command := &cobra.Command{Use: "start", Short: "Create a time-bounded semantic claim and branch checkpoint", RunE: func(cmd *cobra.Command, _ []string) error {
		request.Candidate = policy.Candidate{Repository: "root", Paths: paths, PolicyIDs: policies, ScenarioIDs: scenarios, ContractIDs: contracts, DBEntities: entities, MigrationSlots: migrations, UIFlows: flows, DependencyMajors: dependencies, Now: time.Now().UTC()}
		claims, err := loadClaims(request.Root)
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
	command.Flags().DurationVar(&claimDuration, "lease", 24*time.Hour, "claim lease duration")
	command.PreRun = func(_ *cobra.Command, _ []string) { request.ExpiresAt = time.Now().UTC().Add(claimDuration) }
	command.Flags().StringSliceVar(&paths, "path", nil, "path scope")
	command.Flags().StringSliceVar(&policies, "policy", nil, "policy stable ID")
	command.Flags().StringSliceVar(&scenarios, "scenario", nil, "scenario stable ID")
	command.Flags().StringSliceVar(&contracts, "contract", nil, "contract stable ID")
	command.Flags().StringSliceVar(&entities, "db-entity", nil, "database entity")
	command.Flags().StringSliceVar(&migrations, "migration-slot", nil, "migration slot")
	command.Flags().StringSliceVar(&flows, "ui-flow", nil, "UI flow")
	command.Flags().StringSliceVar(&dependencies, "dependency-major", nil, "dependency major transition")
	command.Flags().BoolVar(&apply, "apply", false, "write the reviewed claim plan")
	for _, flag := range []string{"work-id", "claim-id", "owner", "branch"} {
		_ = command.MarkFlagRequired(flag)
	}
	return command
}

var claimDuration time.Duration

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
	directory := filepath.Join(root, ".harness", "work", "items")
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return []domain.WorkItem{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []domain.WorkItem{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		item, loadErr := loadYAML[domain.WorkItem](filepath.Join(directory, entry.Name()))
		if loadErr != nil {
			return nil, loadErr
		}
		items = append(items, item)
	}
	return items, nil
}

func loadClaims(root string) ([]policy.Claim, error) {
	directory := filepath.Join(root, ".harness", "work", "claims")
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return []policy.Claim{}, nil
	}
	if err != nil {
		return nil, err
	}
	claims := []policy.Claim{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		claim, loadErr := loadYAML[policy.Claim](filepath.Join(directory, entry.Name()))
		if loadErr != nil {
			return nil, loadErr
		}
		claim.Observable = true
		claims = append(claims, claim)
	}
	return claims, nil
}

func loadYAML[T any](path string) (T, error) {
	var value T
	data, err := os.ReadFile(path)
	if err != nil {
		return value, err
	}
	if err := yaml.Unmarshal(data, &value); err != nil {
		return value, fmt.Errorf("decode %s: %w", path, err)
	}
	return value, nil
}
