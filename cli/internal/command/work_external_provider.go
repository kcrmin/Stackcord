package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/provider"
	workpkg "fullstack-orchestrator/cli/internal/work"
)

// externalProviderObservation is a fresh connector result reconciled against
// canonical work. It remains ignored local evidence; only Mapping is committed.
type externalProviderObservation struct {
	Mapping  provider.Mapping
	Snapshot provider.Snapshot
	State    provider.State
}

func loadExternalProviderObservation(root string, config taskProviderConfig, definition workpkg.Definition, now time.Time) (externalProviderObservation, error) {
	mappingPath := filepath.Join(root, ".harness", "work", "mappings", definition.ID+".yaml")
	if err := provider.ValidateCanonicalMappingLocation(root, mappingPath); err != nil {
		return externalProviderObservation{}, fmt.Errorf("validate provider mapping location: %w", err)
	}
	mapping, err := provider.LoadMapping(mappingPath)
	if err != nil {
		return externalProviderObservation{}, fmt.Errorf("load provider mapping: %w", err)
	}
	if mapping.Provider != config.LiveStatusSource || config.Provider != config.LiveStatusSource {
		return externalProviderObservation{}, fmt.Errorf("provider mapping differs from the selected live status source")
	}
	snapshotPath := filepath.Join(root, ".harness", "local", "providers", mapping.Provider, definition.ID+".yaml")
	if err := provider.ValidateCanonicalSnapshotLocation(root, snapshotPath); err != nil {
		return externalProviderObservation{}, fmt.Errorf("validate local provider observation location: %w", err)
	}
	snapshot, err := provider.LoadSnapshot(snapshotPath)
	if err != nil {
		return externalProviderObservation{}, fmt.Errorf("load local provider observation: %w", err)
	}
	expectation := provider.Expectation{WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Dependencies: definition.Dependencies}
	state := provider.Reconcile(expectation, mapping, snapshot, now)
	return externalProviderObservation{Mapping: mapping, Snapshot: snapshot, State: state}, nil
}

func externalProviderStatuses(root string, config taskProviderConfig, now time.Time) (map[string]workpkg.State, bool, []domain.Item, error) {
	definitions, err := workpkg.LoadDefinitions(root)
	if err != nil {
		return nil, false, nil, err
	}
	statuses := make(map[string]workpkg.State, len(definitions))
	complete := true
	issues := []domain.Item{}
	for _, definition := range definitions {
		observation, loadErr := loadExternalProviderObservation(root, config, definition, now)
		if loadErr != nil {
			complete = false
			code := "provider.observation-required"
			if !errors.Is(loadErr, os.ErrNotExist) {
				code = "provider.observation-invalid"
			}
			issues = append(issues, domain.Item{Code: code, Message: "Read this item through the selected connector and reconcile it before choosing work.", Refs: []string{definition.ID, config.LiveStatusSource}})
			continue
		}
		if observation.State.Confidence != provider.Confirmed {
			complete = false
			for _, issue := range observation.State.Issues {
				issue.Refs = append(issue.Refs, definition.ID)
				issues = append(issues, issue)
			}
			continue
		}
		status, valid := providerWorkState(observation.State.Status)
		if !valid {
			complete = false
			issues = append(issues, domain.Item{Code: "provider.status-unknown", Message: "The provider status is not mapped to the executable lifecycle.", Refs: []string{definition.ID, observation.State.Status}})
			continue
		}
		statuses[definition.ID] = status
	}
	return statuses, complete, issues, nil
}

func providerWorkState(value string) (workpkg.State, bool) {
	if value == "closed" {
		return workpkg.Done, true
	}
	return parseWorkState(value)
}

func observationRevision(observation externalProviderObservation) string {
	if strings.TrimSpace(observation.State.Revision) != "" {
		return observation.State.Revision
	}
	return observation.Snapshot.RawHash
}

func externalObservationBlocked(version, commandName string, observation externalProviderObservation, err error) domain.Result {
	result := domain.Result{
		SchemaVersion: "1.0", ToolVersion: version, Command: commandName, OperationID: strings.ReplaceAll(commandName, ".", "-") + "-provider-read-only",
		Status: domain.StatusUnknown, ExitCode: domain.ExitUnavailable, Summary: "The selected task provider must be read and reconciled again before this change.",
		NextActions: []domain.Item{{Code: "provider.refresh", Message: "Read the exact item through the selected connector, then run provider reconciliation."}},
	}
	if err != nil {
		result.Blockers = []domain.Item{{Code: "provider.observation-required", Message: "No valid fresh local observation is available for this canonical work definition."}}
		return result
	}
	for _, issue := range observation.State.Issues {
		if providerIssueBlocks(issue.Code) {
			result.Blockers = append(result.Blockers, issue)
		} else {
			result.Warnings = append(result.Warnings, issue)
		}
	}
	if len(result.Blockers) > 0 {
		result.Status, result.ExitCode = domain.StatusBlocked, domain.ExitBlocked
	}
	return result
}
