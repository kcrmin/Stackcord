package command

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"fullstack-orchestrator/cli/internal/contract"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/evidence"
	"fullstack-orchestrator/cli/internal/gitx"
	"fullstack-orchestrator/cli/internal/governance"
	"fullstack-orchestrator/cli/internal/integration"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/provider"
	"fullstack-orchestrator/cli/internal/release"
	workpkg "fullstack-orchestrator/cli/internal/work"
	"fullstack-orchestrator/cli/internal/workspace"
	"github.com/spf13/cobra"
)

func newIntegrateCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "integrate", Short: "Plan and verify exact service integration"}
	parent.AddCommand(newIntegratePlan(version, jsonOutput), newIntegrateVerify(version, jsonOutput))
	return parent
}

func newIntegratePlan(version string, jsonOutput *bool) *cobra.Command {
	var root, outputPath string
	var apply bool
	command := &cobra.Command{Use: "plan", RunE: func(cmd *cobra.Command, _ []string) error {
		located, err := workspace.FindRoot(cmd.Context(), root)
		if err != nil {
			return err
		}
		definitions, err := workpkg.LoadDefinitions(located.Path)
		if err != nil {
			return err
		}
		providerStates, providerIssues := collectIntegrationProviders(cmd, located.Path, definitions)
		workspaceStates, workspaceIssues := collectIntegrationWorkspaces(cmd, located.Path, located.Manifest)
		plan := integration.Plan(definitions, providerStates, workspaceStates)
		plan.Blockers = append(plan.Blockers, providerIssues...)
		plan.Blockers = append(plan.Blockers, workspaceIssues...)
		plan.ContractFingerprint, err = currentContractFingerprint(located.Path)
		if err != nil {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "integrate.contract-unknown", Message: err.Error()})
		}
		if registry, registryErr := contract.LoadRegistry(located.Path); registryErr != nil {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "integrate.contract-invalid", Message: registryErr.Error()})
		} else {
			plan.Blockers = append(plan.Blockers, integration.CheckCompatibility(definitions, registry)...)
		}
		applyGovernanceToIntegration(cmd, located.Path, &plan)
		result := integrationPlanResult(version, plan)
		if len(plan.Blockers) > 0 || !apply {
			return writeResult(cmd, *jsonOutput, result)
		}
		data, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return err
		}
		writePlan := operation.Plan{ID: "integration-plan", Root: located.Path, Files: []operation.FileChange{{Path: outputPath, Content: append(data, '\n'), Mode: 0o644}}}
		writePlan.InitialStateFingerprint, err = operation.StateFingerprint(writePlan)
		if err != nil {
			return err
		}
		applied := operation.Apply(cmd.Context(), writePlan)
		applied.ToolVersion, applied.Command = version, "integrate.plan"
		if applied.Status == domain.StatusPassed {
			applied.Summary = "Exact contract, child, consumer, UI, migration, and root-pointer order is recorded."
			applied.Facts = result.Facts
			applied.NextActions = []domain.Item{{Code: "integrate.execute", Message: "Merge and record reviewed evidence in the displayed order, then verify integration."}}
		}
		return writeResult(cmd, *jsonOutput, applied)
	}}
	command.Flags().StringVar(&root, "root", ".", "root orchestration repository")
	command.Flags().StringVar(&outputPath, "output", ".harness/local/integration/plan.json", "local integration plan path relative to root")
	command.Flags().BoolVar(&apply, "apply", false, "record the exact integration plan")
	return command
}

func newIntegrateVerify(version string, jsonOutput *bool) *cobra.Command {
	var root, planPath string
	command := &cobra.Command{Use: "verify", RunE: func(cmd *cobra.Command, _ []string) error {
		located, err := workspace.FindRoot(cmd.Context(), root)
		if err != nil {
			return err
		}
		plan, err := readProjectJSON[integration.MergePlan](located.Path, planPath)
		if err != nil {
			return err
		}
		definitions, err := workpkg.LoadDefinitions(located.Path)
		if err != nil {
			return err
		}
		providerStates, providerIssues := collectIntegrationProviders(cmd, located.Path, definitions)
		workspaceStates, workspaceIssues := collectIntegrationWorkspaces(cmd, located.Path, located.Manifest)
		currentPlan := integration.Plan(definitions, providerStates, workspaceStates)
		contractFingerprint, contractErr := currentContractFingerprint(located.Path)
		currentPlan.ContractFingerprint = contractFingerprint
		if contractErr != nil {
			currentPlan.Blockers = append(currentPlan.Blockers, domain.Item{Code: "integrate.contract-unknown", Message: contractErr.Error()})
		}
		if registry, registryErr := contract.LoadRegistry(located.Path); registryErr != nil {
			currentPlan.Blockers = append(currentPlan.Blockers, domain.Item{Code: "integrate.contract-invalid", Message: registryErr.Error()})
		} else {
			currentPlan.Blockers = append(currentPlan.Blockers, integration.CheckCompatibility(definitions, registry)...)
		}
		applyGovernanceToIntegration(cmd, located.Path, &currentPlan)
		if integrationPlanIdentity(plan) != integrationPlanIdentity(currentPlan) {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "integrate.plan-stale", Message: "Current work, provider revision, workspace commit, or integration order differs from the recorded plan."})
		}
		plan.Blockers = append(plan.Blockers, providerIssues...)
		plan.Blockers = append(plan.Blockers, workspaceIssues...)
		records, err := (release.LocalEvidenceStore{}).Load(located.Path)
		if err != nil {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "integrate.evidence-invalid", Message: err.Error()})
		}
		evidenceValues := integrationEvidenceForPlan(plan, records, contractFingerprint)
		result := integration.Verify(plan, evidenceValues, workspaceStates, contractFingerprint)
		result.ToolVersion = version
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&root, "root", ".", "root orchestration repository")
	command.Flags().StringVar(&planPath, "plan", ".harness/local/integration/plan.json", "recorded local integration plan path relative to root")
	return command
}

func applyGovernanceToIntegration(cmd *cobra.Command, root string, plan *integration.MergePlan) {
	report := governance.Check(cmd.Context(), root, "", time.Now().UTC())
	plan.GovernanceFingerprint = report.ProtectedFingerprint
	plan.GovernanceApprovalRevision = report.ApprovalRevision
	if report.Enabled && report.Status != governance.Approved {
		if len(report.Issues) == 0 {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "integrate.governance-unapproved", Message: "Protected product meaning requires approval from a configured product authority."})
		} else {
			for _, item := range report.Issues {
				plan.Blockers = append(plan.Blockers, domain.Item{Code: "integrate." + item.Code, Message: item.Message, Refs: item.Refs})
			}
		}
	}
}

func integrationPlanResult(version string, plan integration.MergePlan) domain.Result {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "integrate.plan", OperationID: "integration-plan-read-only", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Exact service integration order is ready."}
	for index, step := range plan.Steps {
		result.Facts = append(result.Facts, domain.Item{Code: "integrate.step", Message: fmt.Sprintf("%d. %s", index+1, step.Kind), Refs: []string{step.WorkID, step.Ref, step.WorkspaceID, step.Commit, step.RequiredEvidence}})
	}
	if len(plan.Blockers) > 0 {
		result.Status, result.ExitCode, result.Summary, result.Blockers = domain.StatusBlocked, domain.ExitVerification, "Service integration cannot start from current state.", plan.Blockers
		result.NextActions = []domain.Item{{Code: "integrate.refresh", Message: "Refresh the selected provider and exact workspace state, resolve blockers, then plan again."}}
		return result
	}
	result.NextActions = []domain.Item{{Code: "integrate.plan-record", Message: "Record this exact order before merging across repositories."}}
	return result
}

func collectIntegrationProviders(cmd *cobra.Command, root string, definitions []workpkg.Definition) ([]integration.ProviderState, []domain.Item) {
	config, err := loadTaskProvider(root)
	if err != nil {
		return nil, []domain.Item{{Code: "integrate.provider-invalid", Message: err.Error()}}
	}
	if config.Provider == "git-local" && config.LiveStatusSource == "git-local" {
		observed, readErr := provider.NewGitLocalStore(root, config.Remote, config.CoordinationBranch).Read(cmd.Context())
		if readErr != nil {
			return nil, []domain.Item{{Code: "integrate.provider-unknown", Message: "Git-local live state could not be read."}}
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
	if config.Provider != config.LiveStatusSource {
		return nil, []domain.Item{{Code: "integrate.provider-invalid", Message: "Project must use exactly one selected live task source."}}
	}
	coordinated, coordinateErr := provider.NewGitLocalStore(root, config.Remote, config.CoordinationBranch).Read(cmd.Context())
	if coordinateErr != nil {
		return nil, []domain.Item{{Code: "integrate.provider-unknown", Message: "Git semantic reservations could not be read for the selected task source."}}
	}
	claims := make(map[string]provider.GitLocalClaim, len(coordinated.Claims))
	for _, claim := range coordinated.Claims {
		claims[claim.WorkID] = claim
	}
	states, issues := []integration.ProviderState{}, []domain.Item{}
	for _, definition := range definitions {
		observation, observationErr := loadExternalProviderObservation(root, config, definition, time.Now().UTC())
		if observationErr != nil || observation.State.Confidence != provider.Confirmed {
			issues = append(issues, domain.Item{Code: "integrate.provider-unknown", Message: "Fresh selected-provider observation is unavailable.", Refs: []string{definition.ID, config.Provider}})
			continue
		}
		claim, exists := claims[definition.ID]
		claimStatus := claim.Status
		if claimStatus == "" {
			claimStatus = "in_progress"
		}
		observedStatus, valid := providerWorkState(observation.State.Status)
		ownerClearedAtTerminal := valid && observedStatus == workpkg.Done && observation.State.Owner == ""
		if !exists || claim.DefinitionFingerprint != definition.Fingerprint || !valid || string(observedStatus) != claimStatus || (!ownerClearedAtTerminal && observation.State.Owner != claim.Owner) {
			issues = append(issues, domain.Item{Code: "integrate.provider-drift", Message: "External task state differs from its Git semantic reservation.", Refs: []string{definition.ID, config.Provider}})
			continue
		}
		identity := observationRevision(observation) + "\x00" + provider.ClaimRevision(claim)
		digest := sha256.Sum256([]byte(identity))
		states = append(states, integration.ProviderState{WorkID: definition.ID, Status: string(observedStatus), Revision: "sha256:" + hex.EncodeToString(digest[:]), DefinitionFingerprint: definition.Fingerprint, Confirmed: true})
	}
	return states, issues
}

func collectIntegrationWorkspaces(cmd *cobra.Command, root string, manifest workspace.Manifest) ([]integration.WorkspaceState, []domain.Item) {
	rootState, err := gitx.Inspect(cmd.Context(), root)
	if err != nil {
		return nil, []domain.Item{{Code: "integrate.root-unavailable", Message: err.Error()}}
	}
	submodules := map[string]gitx.Submodule{}
	for _, submodule := range rootState.Submodules {
		submodules[filepath.ToSlash(filepath.Clean(filepath.FromSlash(submodule.Path)))] = submodule
	}
	states, issues := []integration.WorkspaceState{}, []domain.Item{}
	for _, entry := range manifest.Workspaces {
		path, state := root, rootState
		expected, actual := "", rootState.Head
		if entry.Kind == "submodule" {
			submodule, exists := submodules[filepath.ToSlash(filepath.Clean(filepath.FromSlash(entry.Path)))]
			if !exists || !submodule.Initialized {
				issues = append(issues, domain.Item{Code: "integrate.workspace-missing", Message: "Declared submodule is not initialized.", Refs: []string{entry.ID}})
				continue
			}
			if submodule.UnsafeURL {
				issues = append(issues, domain.Item{Code: "integrate.submodule-url-unsafe", Message: "Submodule URL is unsafe.", Refs: []string{entry.ID}})
			}
			expected, actual = submodule.ExpectedSHA, submodule.Head
			path = filepath.Join(root, filepath.FromSlash(entry.Path))
			state, err = gitx.Inspect(cmd.Context(), path)
			if err != nil {
				issues = append(issues, domain.Item{Code: "integrate.workspace-unavailable", Message: err.Error(), Refs: []string{entry.ID}})
				continue
			}
		} else if entry.Kind == "external" {
			path = filepath.Join(root, filepath.FromSlash(entry.Path))
			state, err = gitx.Inspect(cmd.Context(), path)
			if err != nil {
				issues = append(issues, domain.Item{Code: "integrate.workspace-unavailable", Message: err.Error(), Refs: []string{entry.ID}})
				continue
			}
		}
		remote, remoteErr := gitx.RemoteURL(cmd.Context(), path, "origin")
		if remoteErr != nil {
			issues = append(issues, domain.Item{Code: "integrate.remote-unknown", Message: "Workspace origin remote is unavailable.", Refs: []string{entry.ID}})
		}
		expectedRemote := entry.Remote
		if entry.Kind == "root" && expectedRemote == "" {
			expectedRemote = manifest.RootRemote
		}
		if expectedRemote != "" && remote != expectedRemote {
			issues = append(issues, domain.Item{Code: "integrate.remote-mismatch", Message: "Actual workspace remote differs from the manifest.", Refs: []string{entry.ID}})
		}
		published := gitx.CommitPublished(cmd.Context(), path, state.Head)
		states = append(states, integration.WorkspaceState{ID: entry.ID, Kind: entry.Kind, Commit: state.Head, Remote: remote, Clean: !state.Dirty, Published: published, ExpectedPointer: expected, ActualPointer: actual})
	}
	sort.Slice(states, func(left, right int) bool { return states[left].ID < states[right].ID })
	return states, issues
}

func integrationEvidenceForPlan(plan integration.MergePlan, records []evidence.Record, contractFingerprint string) []integration.Evidence {
	result := []integration.Evidence{}
	for _, step := range plan.Steps {
		if step.Kind == integration.ContractStep {
			result = append(result, integration.Evidence{StepID: step.ID, WorkID: step.WorkID, Kind: step.RequiredEvidence, WorkspaceID: step.WorkspaceID, DefinitionFingerprint: step.DefinitionFingerprint, ContractFingerprint: contractFingerprint, ProviderRevision: step.ProviderRevision, Commit: step.Commit, Digest: contractFingerprint})
			continue
		}
		matches := []evidence.Record{}
		for _, record := range records {
			if record.WorkID == step.WorkID && record.WorkspaceID == step.WorkspaceID && record.Kind == step.RequiredEvidence && record.Commit == step.Commit && record.DefinitionFingerprint == step.DefinitionFingerprint && record.ContractFingerprint == contractFingerprint && record.ExitCode == 0 {
				matches = append(matches, record)
			}
		}
		if len(matches) == 0 {
			continue
		}
		sort.Slice(matches, func(left, right int) bool {
			if matches[left].FinishedAt.Equal(matches[right].FinishedAt) {
				return matches[left].ID < matches[right].ID
			}
			return matches[left].FinishedAt.After(matches[right].FinishedAt)
		})
		record := matches[0]
		result = append(result, integration.Evidence{StepID: step.ID, WorkID: step.WorkID, Kind: step.RequiredEvidence, WorkspaceID: step.WorkspaceID, DefinitionFingerprint: step.DefinitionFingerprint, ContractFingerprint: contractFingerprint, ProviderRevision: step.ProviderRevision, Commit: step.Commit, Digest: record.OutputDigest})
	}
	return result
}

func integrationPlanIdentity(plan integration.MergePlan) string {
	copy := plan
	copy.Blockers = nil
	data, _ := json.Marshal(copy)
	return string(data)
}
