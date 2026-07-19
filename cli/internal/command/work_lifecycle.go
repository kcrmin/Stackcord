package command

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/evidence"
	"github.com/kcrmin/Stackcord/cli/internal/gitx"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/provider"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
	workpkg "github.com/kcrmin/Stackcord/cli/internal/work"
	"github.com/kcrmin/Stackcord/cli/internal/workspace"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

func newWorkEvidence(version string, jsonOutput *bool) *cobra.Command {
	var root, workID, workspaceID, commandID string
	var artifacts []string
	var apply bool
	command := &cobra.Command{Use: "evidence", Short: "Run one reviewed workspace command and bind its result to current meaning", RunE: func(cmd *cobra.Command, _ []string) error {
		located, err := workspace.FindRoot(cmd.Context(), root)
		if err != nil {
			return err
		}
		definition, found, err := loadStartDefinition(located.Path, workID)
		if err != nil {
			return err
		}
		if !found {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.evidence", "work.definition-required", "Evidence requires a current executable work definition.", workID))
		}
		entry, found := findWorkspaceEntry(located.Manifest, workspaceID)
		if !found || !containsString(definition.Workspaces, workspaceID) {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.evidence", "evidence.workspace-undeclared", "The requested workspace is not in the work definition and root manifest.", workspaceID))
		}
		approved, loadErr := loadApprovedCommand(located.Path, entry, commandID)
		if loadErr != nil {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.evidence", "evidence.command-unapproved", loadErr.Error(), commandID))
		}
		workspacePath := filepath.Join(located.Path, filepath.FromSlash(entry.Path))
		repositoryPath := workspacePath
		if entry.Kind == "root" || entry.Kind == "directory" {
			repositoryPath = located.Path
		}
		artifactValues, artifactErr := collectArtifactDigests(workspacePath, artifacts)
		if artifactErr != nil {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.evidence", "evidence.artifact-invalid", artifactErr.Error()))
		}
		if !apply {
			facts := []domain.Item{{Code: "evidence.command", Message: approved.ID, Refs: approved.Argv}, {Code: "evidence.workspace", Message: workspaceID}}
			for name := range artifactValues {
				facts = append(facts, domain.Item{Code: "evidence.artifact", Message: name})
			}
			sort.Slice(facts, func(left, right int) bool {
				return facts[left].Code+facts[left].Message < facts[right].Code+facts[right].Message
			})
			result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "work.evidence.plan", OperationID: "evidence-plan-" + strings.ReplaceAll(workID, ".", "-"), Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Reviewed evidence command and artifact inputs are ready to run on a clean workspace.", Facts: facts, NextActions: []domain.Item{{Code: "work.evidence.apply", Message: "Run the exact reviewed command and record its commit-bound digests."}}}
			return writeResult(cmd, *jsonOutput, result)
		}
		contractFingerprint, err := currentContractFingerprint(located.Path)
		if err != nil {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.evidence", "evidence.contract-unavailable", err.Error()))
		}
		record, result := evidence.Run(cmd.Context(), evidence.Request{Workspace: workspacePath, Repository: repositoryPath, WorkspaceID: workspaceID, WorkID: workID, DefinitionFingerprint: definition.Fingerprint, ContractFingerprint: contractFingerprint, Command: approved, ArtifactDigests: artifactValues})
		result.ToolVersion = version
		if result.Status != domain.StatusPassed {
			return writeResult(cmd, *jsonOutput, result)
		}
		if issues := schema.Validate("evidence", record); len(issues) > 0 {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.evidence", "evidence.record-invalid", issues[0].Message))
		}
		data, err := yaml.Marshal(record)
		if err != nil {
			return err
		}
		plan := operation.Plan{ID: "record-" + record.ID, Root: located.Path, Files: []operation.FileChange{{Path: filepath.ToSlash(filepath.Join(".harness", "local", "evidence", workID, record.ID+".yaml")), Content: data, Mode: 0o600}}}
		plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
		if err != nil {
			return err
		}
		applied := operation.Apply(cmd.Context(), plan)
		applied.ToolVersion, applied.Command = version, "work.evidence"
		if applied.Status == domain.StatusPassed {
			applied.Summary = result.Summary
			applied.Facts = result.Facts
			applied.Evidence = append(applied.Evidence, result.Evidence...)
		}
		return writeResult(cmd, *jsonOutput, applied)
	}}
	command.Flags().StringVar(&root, "root", ".", "project root or child path")
	command.Flags().StringVar(&workID, "work-id", "", "work stable ID")
	command.Flags().StringVar(&workspaceID, "workspace", "", "workspace stable ID")
	command.Flags().StringVar(&commandID, "command-id", "", "approved command stable ID")
	command.Flags().StringSliceVar(&artifacts, "artifact", nil, "verified artifact name=workspace-relative-path")
	command.Flags().BoolVar(&apply, "apply", false, "run and record the reviewed command")
	for _, flag := range []string{"work-id", "workspace", "command-id"} {
		_ = command.MarkFlagRequired(flag)
	}
	return command
}

const maxEvidenceArtifactBytes = 64 << 20

var evidenceArtifactNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+)*$`)

func collectArtifactDigests(workspacePath string, values []string) (map[string]string, error) {
	result := map[string]string{}
	workspace, err := filepath.EvalSymlinks(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("artifact workspace is unavailable")
	}
	for _, value := range values {
		name, relative, found := strings.Cut(value, "=")
		name, relative = strings.TrimSpace(name), strings.TrimSpace(relative)
		if !found || name == "" || relative == "" {
			return nil, fmt.Errorf("artifact must use name=relative-path")
		}
		if !evidenceArtifactNamePattern.MatchString(name) {
			return nil, fmt.Errorf("artifact name must be a lowercase stable identifier: %s", name)
		}
		if _, duplicate := result[name]; duplicate {
			return nil, fmt.Errorf("artifact name is duplicated: %s", name)
		}
		if filepath.IsAbs(relative) {
			return nil, fmt.Errorf("artifact path must be workspace-relative: %s", relative)
		}
		clean := filepath.Clean(filepath.FromSlash(relative))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("artifact path escapes the workspace: %s", relative)
		}
		path := filepath.Join(workspace, clean)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() == 0 || info.Size() > maxEvidenceArtifactBytes {
			return nil, fmt.Errorf("artifact must be a non-empty regular file within the size limit: %s", relative)
		}
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, fmt.Errorf("artifact cannot be resolved safely: %s", relative)
		}
		if inside, err := filepath.Rel(workspace, canonical); err != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("artifact resolves outside the workspace: %s", relative)
		}
		file, err := os.Open(canonical)
		if err != nil {
			return nil, fmt.Errorf("artifact cannot be read: %s", relative)
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, io.LimitReader(file, maxEvidenceArtifactBytes+1))
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			return nil, fmt.Errorf("artifact cannot be hashed: %s", relative)
		}
		result[name] = "sha256:" + hex.EncodeToString(hash.Sum(nil))
	}
	return result, nil
}

func newWorkTransition(version string, jsonOutput *bool, aliasTarget workpkg.State) *cobra.Command {
	var root, workID, targetValue string
	var apply bool
	use, short := "transition", "Verify and change work lifecycle state"
	if aliasTarget == workpkg.Done {
		use, short, targetValue = "finish", "Finish work only after current evidence and integration are proven", string(workpkg.Done)
	}
	command := &cobra.Command{Use: use, Short: short, RunE: func(cmd *cobra.Command, _ []string) error {
		located, err := workspace.FindRoot(cmd.Context(), root)
		if err != nil {
			return err
		}
		definition, found, err := loadStartDefinition(located.Path, workID)
		if err != nil {
			return err
		}
		if !found {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work."+use, "work.definition-required", "Lifecycle transitions require a current executable work definition.", workID))
		}
		target, valid := parseWorkState(targetValue)
		if !valid {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work."+use, "work.target-invalid", "Target must be ready, in_progress, blocked, review, integrated, or done.", targetValue))
		}
		config, err := loadTaskProvider(located.Path)
		if err != nil {
			return err
		}
		store := provider.NewGitLocalStore(located.Path, config.Remote, config.CoordinationBranch)
		current, err := store.Read(cmd.Context())
		if err != nil {
			return writeResult(cmd, *jsonOutput, gitLocalFailureResult(version, "provider.live-read-failed", "Live Git-local lifecycle could not be read safely.", err))
		}
		claimIndex := -1
		for index, claim := range current.Claims {
			if claim.WorkID == workID {
				claimIndex = index
				break
			}
		}
		if claimIndex < 0 {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work."+use, "work.live-state-missing", "No selected-provider lifecycle state exists for this work.", workID))
		}
		records, warnings, err := loadCurrentEvidence(located.Path, located.Manifest, definition)
		if err != nil {
			return err
		}
		claim := current.Claims[claimIndex]
		liveRevision := current.Revision
		var external *externalProviderObservation
		if config.LiveStatusSource != "git-local" {
			observation, observationErr := loadExternalProviderObservation(located.Path, config, definition, time.Now().UTC())
			if observationErr != nil || observation.State.Confidence != provider.Confirmed {
				return writeResult(cmd, *jsonOutput, externalObservationBlocked(version, "work."+use, observation, observationErr))
			}
			observedStatus, statusValid := providerWorkState(observation.State.Status)
			if !statusValid {
				return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work."+use, "provider.status-unknown", "The external task status is not mapped to the executable lifecycle.", observation.State.Status))
			}
			currentStatus := gitLocalWorkState(claim.Status)
			if observedStatus != currentStatus && observedStatus != target {
				return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work."+use, "provider.status-mismatch", "The external task status differs from both the last coordinated state and the requested target.", string(currentStatus), observation.State.Status, string(target)))
			}
			terminalWithoutOwner := target == workpkg.Done && observedStatus == target && strings.TrimSpace(observation.State.Owner) == ""
			if !terminalWithoutOwner && strings.TrimSpace(observation.State.Owner) != strings.TrimSpace(claim.Owner) {
				return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work."+use, "provider.owner-mismatch", "The freshly observed task owner differs from the semantic reservation owner.", claim.Owner, observation.State.Owner))
			}
			if apply && observedStatus != target {
				return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work."+use, "provider.target-not-confirmed", "Change the external task status, re-read it through the connector, and reconcile the exact target before applying.", string(target), observation.State.Status))
			}
			external = &observation
			liveRevision = observationRevision(observation)
		}
		live := workpkg.LiveState{Status: gitLocalWorkState(claim.Status), Owner: claim.Owner, Revision: liveRevision, Confirmed: true, Children: childStates(located.Path, definition.ID, current.Claims), RootPointersConfirmed: rootPointersConfirmed(cmd, located.Path, located.Manifest, definition)}
		result := workpkg.Transition(definition, live, records, target)
		result.ToolVersion, result.Command = version, "work."+use
		result.Warnings = append(result.Warnings, warnings...)
		if target == workpkg.Review && result.Status == domain.StatusPassed && !evidenceOnClaimedBranch(cmd, located.Path, located.Manifest, records, claim.Branch) {
			result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitVerification, "Review evidence is not bound to the clean claimed branch."
			result.Blockers = []domain.Item{{Code: "work.evidence-branch-mismatch", Message: "Record implementation evidence from the claimed branch workspace before review.", Refs: []string{claim.Branch}}}
		}
		if result.Status != domain.StatusPassed || !apply {
			if result.Status == domain.StatusPassed {
				if external == nil {
					result.Summary = "Lifecycle transition is verified and ready to publish to the selected provider."
					result.NextActions = []domain.Item{{Code: "work.transition.apply", Message: "Publish the verified transition and re-read its live revision."}}
				} else if observedStatus, valid := providerWorkState(external.State.Status); valid && observedStatus == target {
					result.Summary = "The external target revision and lifecycle evidence are verified; semantic coordination is ready to synchronize."
					result.NextActions = []domain.Item{{Code: "work.transition.apply", Message: "Synchronize the verified target with the Git semantic reservation."}}
				} else {
					result.Summary = "Lifecycle evidence is verified before changing the external task source."
					result.NextActions = []domain.Item{{Code: "provider.transition", Message: "Change the selected task item, re-read it through the connector, reconcile it, then rerun with --apply.", Refs: []string{config.LiveStatusSource, string(target)}}}
				}
			}
			return writeResult(cmd, *jsonOutput, result)
		}
		next := current
		next.Claims = append([]provider.GitLocalClaim(nil), current.Claims...)
		next.Claims[claimIndex].Status = string(target)
		revision, err := store.CompareAndSwap(cmd.Context(), current.Revision, next)
		if err != nil {
			return writeResult(cmd, *jsonOutput, gitLocalFailureResult(version, "provider.transition-failed", "Verified lifecycle transition was not published.", err))
		}
		observed, err := store.Read(cmd.Context())
		if err != nil || observed.Revision != revision || !observedClaimStatus(observed, workID, string(target)) {
			if err == nil {
				err = errors.New("live lifecycle postcondition differs from the requested target")
			}
			return writeResult(cmd, *jsonOutput, gitLocalFailureResult(version, "provider.transition-postcondition", "Published lifecycle state could not be confirmed.", err))
		}
		if external == nil {
			result.Summary = "Lifecycle transition was verified, published, and re-read from the selected provider."
			result.Evidence = append(result.Evidence, domain.Item{Code: "provider.live-revision", Message: revision, Refs: []string{workID, string(target)}})
		} else {
			result.Summary = "The external lifecycle revision was verified and its semantic reservation was synchronized through Git CAS."
			result.Evidence = append(result.Evidence,
				domain.Item{Code: "provider.live-revision", Message: observationRevision(*external), Refs: []string{external.State.Provider, external.State.ItemID, string(target)}},
				domain.Item{Code: "coordination.semantic-reservation", Message: revision, Refs: []string{workID, string(target)}},
			)
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&root, "root", ".", "project root or child path")
	command.Flags().StringVar(&workID, "work-id", "", "work stable ID")
	if aliasTarget == "" {
		command.Flags().StringVar(&targetValue, "target", "", "target lifecycle state")
		_ = command.MarkFlagRequired("target")
	}
	command.Flags().BoolVar(&apply, "apply", false, "publish and re-read the verified transition")
	_ = command.MarkFlagRequired("work-id")
	return command
}

func newWorkHandoff(version string, jsonOutput *bool) *cobra.Command {
	var root, workID, workspaceID, owner, nextAction string
	var blockers []string
	var lease time.Duration
	var apply bool
	command := &cobra.Command{Use: "handoff", Short: "Transfer live ownership with exact Git and evidence identity", RunE: func(cmd *cobra.Command, _ []string) error {
		located, err := workspace.FindRoot(cmd.Context(), root)
		if err != nil {
			return err
		}
		definition, found, err := loadStartDefinition(located.Path, workID)
		if err != nil {
			return err
		}
		if !found || !containsString(definition.Workspaces, workspaceID) {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.handoff", "work.definition-required", "Handoff requires a current definition and one affected workspace.", workID, workspaceID))
		}
		entry, found := findWorkspaceEntry(located.Manifest, workspaceID)
		if !found {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.handoff", "work.workspace-missing", "Handoff workspace is not declared by the orchestration root.", workspaceID))
		}
		config, err := loadTaskProvider(located.Path)
		if err != nil {
			return err
		}
		store := provider.NewGitLocalStore(located.Path, config.Remote, config.CoordinationBranch)
		current, err := store.Read(cmd.Context())
		if err != nil {
			return writeResult(cmd, *jsonOutput, gitLocalFailureResult(version, "provider.live-read-failed", "Live ownership could not be read safely.", err))
		}
		claimIndex := -1
		for index, claim := range current.Claims {
			if claim.WorkID == workID {
				claimIndex = index
				break
			}
		}
		if claimIndex < 0 {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.handoff", "work.live-state-missing", "No live selected-provider claim exists for handoff.", workID))
		}
		claim := current.Claims[claimIndex]
		if strings.TrimSpace(owner) == "" || owner == claim.Owner {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.handoff", "work.owner-unchanged", "Handoff is only for a real change of owner.", claim.Owner))
		}
		if claim.Status == "integrated" || claim.Status == "done" {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.handoff", "work.handoff-terminal", "Integrated or completed work no longer has implementation ownership to transfer.", workID))
		}
		var external *externalProviderObservation
		if config.LiveStatusSource != "git-local" {
			observation, observationErr := loadExternalProviderObservation(located.Path, config, definition, time.Now().UTC())
			if observationErr != nil || observation.State.Confidence != provider.Confirmed {
				return writeResult(cmd, *jsonOutput, externalObservationBlocked(version, "work.handoff", observation, observationErr))
			}
			if observation.State.Status != string(gitLocalWorkState(claim.Status)) {
				return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.handoff", "provider.status-mismatch", "Ownership cannot change while the external lifecycle differs from the coordinated reservation.", claim.Status, observation.State.Status))
			}
			observedOwner := strings.TrimSpace(observation.State.Owner)
			if observedOwner != strings.TrimSpace(claim.Owner) && observedOwner != strings.TrimSpace(owner) {
				return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.handoff", "provider.owner-mismatch", "The external owner differs from both the current and requested owners.", claim.Owner, owner, observation.State.Owner))
			}
			if apply && observedOwner != strings.TrimSpace(owner) {
				return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.handoff", "provider.handoff-not-confirmed", "Transfer the external task owner, re-read it through the connector, and reconcile it before applying the handoff.", owner, observation.State.Owner))
			}
			external = &observation
		}
		for _, blocker := range blockers {
			if !validHandoffRef(blocker) {
				return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.handoff", "work.handoff-blocker-invalid", "Handoff blockers must use stable IDs, not free-form issue text.", blocker))
			}
		}
		workspacePath := filepath.Join(located.Path, filepath.FromSlash(entry.Path))
		repositoryPath := workspacePath
		if entry.Kind == "root" || entry.Kind == "directory" {
			repositoryPath = located.Path
		}
		gitState, err := gitx.Inspect(cmd.Context(), repositoryPath)
		if err != nil {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.handoff", "git.workspace-unavailable", err.Error(), workspaceID))
		}
		if gitState.Branch != claim.Branch || gitState.Detached || gitState.Dirty {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.handoff", "work.handoff-git-mismatch", "Run handoff from the clean claimed branch in the selected workspace.", claim.Branch, gitState.Branch))
		}
		if gitState.Upstream == "" || gitState.Ahead > 0 || gitState.Diverged {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "work.handoff", "work.handoff-local-only", "The exact handoff commit must be visible on its upstream before ownership changes.", gitState.Head))
		}
		records, warnings, err := loadCurrentEvidence(located.Path, located.Manifest, definition)
		if err != nil {
			return err
		}
		evidenceIDs := make([]string, 0, len(records))
		for _, record := range records {
			if record.Commit == gitState.Head {
				evidenceIDs = append(evidenceIDs, record.ID)
			}
		}
		sort.Strings(evidenceIDs)
		now := time.Now().UTC()
		handoff := &provider.GitLocalHandoff{FromOwner: claim.Owner, ToOwner: owner, Workspace: workspaceID, Branch: claim.Branch, Commit: gitState.Head, LocalOnly: false, Evidence: evidenceIDs, Blockers: append([]string(nil), blockers...), NextAction: strings.TrimSpace(nextAction), RecordedAt: now}
		result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "work.handoff", OperationID: "work-handoff-" + strings.ReplaceAll(workID, ".", "-"), Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Ownership-transfer checkpoint is verified and ready to publish.", Facts: []domain.Item{{Code: "work.handoff-owner", Message: owner, Refs: []string{claim.Owner}}, {Code: "git.branch", Message: gitState.Branch}, {Code: "git.commit", Message: gitState.Head}, {Code: "work.handoff-local-only", Message: "false"}}, Warnings: warnings}
		if !apply {
			if external == nil {
				result.NextActions = []domain.Item{{Code: "work.handoff.apply", Message: "Publish the exact ownership change and re-read the selected provider."}}
			} else if external.State.Owner == owner {
				result.NextActions = []domain.Item{{Code: "work.handoff.apply", Message: "Synchronize the verified external owner with the Git semantic reservation."}}
			} else {
				result.NextActions = []domain.Item{{Code: "provider.handoff", Message: "Transfer the selected task item, re-read it through the connector, reconcile it, then rerun with --apply.", Refs: []string{config.LiveStatusSource, owner}}}
			}
			return writeResult(cmd, *jsonOutput, result)
		}
		next := current
		next.Claims = append([]provider.GitLocalClaim(nil), current.Claims...)
		next.Claims[claimIndex].Owner = owner
		next.Claims[claimIndex].Handoff = handoff
		next.Claims[claimIndex].StartsAt = now
		next.Claims[claimIndex].ExpiresAt = now.Add(lease)
		revision, err := store.CompareAndSwap(cmd.Context(), current.Revision, next)
		if err != nil {
			return writeResult(cmd, *jsonOutput, gitLocalFailureResult(version, "provider.handoff-failed", "Verified ownership transfer was not published.", err))
		}
		observed, err := store.Read(cmd.Context())
		if err != nil || observed.Revision != revision || !observedHandoff(observed, workID, owner, gitState.Head) {
			if err == nil {
				err = errors.New("live handoff postcondition differs from the requested owner or commit")
			}
			return writeResult(cmd, *jsonOutput, gitLocalFailureResult(version, "provider.handoff-postcondition", "Published ownership transfer could not be confirmed.", err))
		}
		if external == nil {
			result.Summary = "Ownership changed with exact branch, commit, evidence, blockers, next action, and live revision confirmed."
			result.Evidence = []domain.Item{{Code: "provider.live-revision", Message: revision, Refs: []string{workID, owner}}}
		} else {
			result.Summary = "The external ownership change and exact Git checkpoint were verified, then semantic coordination was synchronized."
			result.Evidence = []domain.Item{
				{Code: "provider.live-revision", Message: observationRevision(*external), Refs: []string{external.State.Provider, external.State.ItemID, owner}},
				{Code: "coordination.semantic-reservation", Message: revision, Refs: []string{workID, owner}},
			}
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&root, "root", ".", "project root or claimed workspace")
	command.Flags().StringVar(&workID, "work-id", "", "work stable ID")
	command.Flags().StringVar(&workspaceID, "workspace", "", "workspace whose branch is transferred")
	command.Flags().StringVar(&owner, "owner", "", "receiving owner")
	command.Flags().StringVar(&nextAction, "next-action", "", "one reproducible next action")
	command.Flags().StringSliceVar(&blockers, "blocker", nil, "stable blocker ID")
	command.Flags().DurationVar(&lease, "lease", 24*time.Hour, "renewed ownership lease")
	command.Flags().BoolVar(&apply, "apply", false, "publish and re-read the ownership change")
	for _, flag := range []string{"work-id", "workspace", "owner", "next-action"} {
		_ = command.MarkFlagRequired(flag)
	}
	return command
}

func loadApprovedCommand(root string, entry workspace.Entry, commandID string) (evidence.ApprovedCommand, error) {
	if entry.CommandsPath == "" {
		return evidence.ApprovedCommand{}, fmt.Errorf("workspace has no approved command manifest; select the technology and record reviewed commands first")
	}
	path := filepath.Join(root, filepath.FromSlash(entry.Path), filepath.FromSlash(entry.CommandsPath))
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return evidence.ApprovedCommand{}, fmt.Errorf("approved command manifest is unavailable or unsafe")
	}
	manifest, err := schema.LoadYAML[evidence.CommandManifest](path)
	if err != nil {
		return evidence.ApprovedCommand{}, err
	}
	if issues := schema.Validate("commands", manifest); len(issues) > 0 {
		return evidence.ApprovedCommand{}, fmt.Errorf("approved command manifest is invalid: %s", issues[0].Message)
	}
	if manifest.WorkspaceID != entry.ID {
		return evidence.ApprovedCommand{}, fmt.Errorf("approved command manifest workspace identity differs from the root manifest")
	}
	seen := map[string]bool{}
	for _, command := range manifest.Commands {
		if seen[command.ID] {
			return evidence.ApprovedCommand{}, fmt.Errorf("approved command IDs must be unique")
		}
		seen[command.ID] = true
		if command.ID == commandID {
			return command, nil
		}
	}
	return evidence.ApprovedCommand{}, fmt.Errorf("command is not present in the reviewed workspace manifest")
}

func loadCurrentEvidence(root string, manifest workspace.Manifest, definition workpkg.Definition) ([]evidence.Record, []domain.Item, error) {
	directory := filepath.Join(root, ".harness", "local", "evidence", definition.ID)
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return []evidence.Record{}, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	contractFingerprint, err := currentContractFingerprint(root)
	if err != nil {
		return nil, nil, err
	}
	current, warnings := []evidence.Record{}, []domain.Item{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		info, statErr := os.Lstat(path)
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			warnings = append(warnings, domain.Item{Code: "evidence.record-unsafe", Message: "Ignored an unsafe local evidence record.", Refs: []string{entry.Name()}})
			continue
		}
		record, loadErr := schema.LoadYAML[evidence.Record](path)
		if loadErr != nil || len(schema.Validate("evidence", record)) > 0 {
			warnings = append(warnings, domain.Item{Code: "evidence.record-invalid", Message: "Ignored an invalid local evidence record.", Refs: []string{entry.Name()}})
			continue
		}
		workspaceEntry, found := findWorkspaceEntry(manifest, record.WorkspaceID)
		if !found {
			warnings = append(warnings, domain.Item{Code: "evidence.workspace-missing", Message: "Evidence refers to a workspace no longer in the root manifest.", Refs: []string{record.ID, record.WorkspaceID}})
			continue
		}
		workspacePath := filepath.Join(root, filepath.FromSlash(workspaceEntry.Path))
		repositoryPath := workspacePath
		if workspaceEntry.Kind == "root" || workspaceEntry.Kind == "directory" {
			repositoryPath = root
		}
		issues := evidence.VerifyCurrent(record, evidence.Actual{Workspace: workspacePath, Repository: repositoryPath, DefinitionFingerprint: definition.Fingerprint, ContractFingerprint: contractFingerprint})
		if len(issues) > 0 {
			warnings = append(warnings, issues...)
			continue
		}
		current = append(current, record)
	}
	sort.Slice(current, func(left, right int) bool { return current[left].ID < current[right].ID })
	return current, warnings, nil
}

func currentContractFingerprint(root string) (string, error) {
	manifest, err := schema.LoadYAML[map[string]any](filepath.Join(root, ".harness", "manifest.yaml"))
	if err != nil {
		return "", err
	}
	contracts := "contracts"
	if paths, ok := manifest["paths"].(map[string]any); ok {
		if configured, ok := paths["contracts"].(string); ok && configured != "" {
			contracts = configured
		}
	}
	return evidence.FingerprintTree(root, contracts)
}

func rootPointersConfirmed(cmd *cobra.Command, root string, manifest workspace.Manifest, definition workpkg.Definition) bool {
	if len(definition.Scope.RootPointers) == 0 {
		return true
	}
	state, err := gitx.Inspect(cmd.Context(), root)
	if err != nil {
		return false
	}
	byPath := map[string]gitx.Submodule{}
	for _, submodule := range state.Submodules {
		byPath[submodule.Path] = submodule
	}
	for _, id := range definition.Scope.RootPointers {
		entry, found := findWorkspaceEntry(manifest, id)
		if !found || entry.Kind != "submodule" {
			return false
		}
		submodule, found := byPath[entry.Path]
		if !found || !submodule.Initialized || submodule.Dirty || submodule.PointerDiff || submodule.UnsafeURL || submodule.Head == "" || submodule.Head != submodule.ExpectedSHA {
			return false
		}
	}
	return true
}

func evidenceOnClaimedBranch(cmd *cobra.Command, root string, manifest workspace.Manifest, records []evidence.Record, branch string) bool {
	found := false
	for _, record := range records {
		if record.Kind != "test" {
			continue
		}
		entry, exists := findWorkspaceEntry(manifest, record.WorkspaceID)
		if !exists {
			return false
		}
		repositoryPath := filepath.Join(root, filepath.FromSlash(entry.Path))
		if entry.Kind == "root" || entry.Kind == "directory" {
			repositoryPath = root
		}
		state, err := gitx.Inspect(cmd.Context(), repositoryPath)
		if err != nil || state.Dirty || state.Detached || state.Branch != branch || state.Head != record.Commit {
			return false
		}
		found = true
	}
	return found
}

func childStates(root, parentID string, claims []provider.GitLocalClaim) map[string]workpkg.State {
	definitions, err := workpkg.LoadDefinitions(root)
	if err != nil {
		return map[string]workpkg.State{}
	}
	result := map[string]workpkg.State{}
	for _, definition := range definitions {
		if definition.ParentID != parentID {
			continue
		}
		result[definition.ID] = workpkg.ReadyState
		for _, claim := range claims {
			if claim.WorkID == definition.ID {
				result[definition.ID] = gitLocalWorkState(claim.Status)
			}
		}
	}
	return result
}

func parseWorkState(value string) (workpkg.State, bool) {
	state := workpkg.State(value)
	switch state {
	case workpkg.ReadyState, workpkg.InProgress, workpkg.Blocked, workpkg.Review, workpkg.Integrated, workpkg.Done:
		return state, true
	default:
		return "", false
	}
}

func gitLocalWorkState(value string) workpkg.State {
	if value == "" {
		return workpkg.InProgress
	}
	state, valid := parseWorkState(value)
	if !valid {
		return workpkg.Proposed
	}
	return state
}

func observedClaimStatus(snapshot provider.SnapshotSet, workID, status string) bool {
	for _, claim := range snapshot.Claims {
		if claim.WorkID == workID && claim.Status == status {
			return true
		}
	}
	return false
}

func observedHandoff(snapshot provider.SnapshotSet, workID, owner, commit string) bool {
	for _, claim := range snapshot.Claims {
		if claim.WorkID == workID && claim.Owner == owner && claim.Handoff != nil && claim.Handoff.ToOwner == owner && claim.Handoff.Commit == commit {
			return true
		}
	}
	return false
}

func validHandoffRef(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for index, char := range part {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && (index == 0 || char != '-') {
				return false
			}
		}
	}
	return true
}

func findWorkspaceEntry(manifest workspace.Manifest, id string) (workspace.Entry, bool) {
	for _, entry := range manifest.Workspaces {
		if entry.ID == id {
			return entry, true
		}
	}
	return workspace.Entry{}, false
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func lifecycleBlocked(version, command, code, message string, refs ...string) domain.Result {
	return domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: command, OperationID: strings.ReplaceAll(command, ".", "-") + "-read-only", Status: domain.StatusBlocked, ExitCode: domain.ExitBlocked, Summary: message, Blockers: []domain.Item{{Code: code, Message: message, Refs: refs}}}
}
