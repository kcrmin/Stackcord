package command

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
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
					plan, planErr := providerMappingPlan(located.Path, mapping.WorkID, data)
					if planErr != nil {
						return planErr
					}
					result.Changes = []domain.Item{{Code: "provider.mapping-planned", Message: plan.Files[0].Path}}
				}
				return writeResult(cmd, *jsonOutput, result)
			}
			data, err := yaml.Marshal(mapping)
			if err != nil {
				return err
			}
			plan, err := providerMappingPlan(located.Path, mapping.WorkID, data)
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
	command.Flags().BoolVar(&apply, "apply", false, "write only the stable provider mapping")
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

func providerMappingPlan(root, workID string, data []byte) (operation.Plan, error) {
	plan := operation.Plan{
		ID:   "provider-map-" + strings.ReplaceAll(workID, ".", "-"),
		Root: root,
		Files: []operation.FileChange{{
			Path: filepath.ToSlash(filepath.Join(".harness", "work", "mappings", workID+".yaml")), Content: data, Mode: 0o644,
		}},
	}
	var err error
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return plan, err
}
