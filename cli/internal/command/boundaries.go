package command

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"fullstack-orchestrator/cli/internal/contract"
	"fullstack-orchestrator/cli/internal/database"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	uiimport "fullstack-orchestrator/cli/internal/ui"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

func newChangeCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "change", Short: "Plan a product-aware cross-workspace change"}
	var root, objective string
	var refs []string
	plan := &cobra.Command{Use: "plan", RunE: func(cmd *cobra.Command, _ []string) error {
		snapshot, issues := contextpkg.Refresh(cmd.Context(), root, contextpkg.ReadOnly)
		result := contextResult(version, "change-plan", root, snapshot, issues, contextpkg.ReadOnly)
		result.Command, result.OperationID, result.Summary = "change.plan", "change-plan-read-only", "Product and integration impact plan is ready."
		result.Facts = append(result.Facts, domain.Item{Code: "change.objective", Message: objective, Refs: refs})
		for _, ref := range refs {
			if entry, exists := snapshot.Index[ref]; exists {
				result.Evidence = append(result.Evidence, domain.Item{Code: "change.source", Message: entry.Path, Refs: []string{ref, entry.Fingerprint}})
			} else {
				result.Warnings = append(result.Warnings, domain.Item{Code: "change.ref-unknown", Message: "Referenced product meaning is not indexed.", Refs: []string{ref}})
			}
		}
		result.NextActions = []domain.Item{{Code: "change.define-tdd", Message: "Record the failing behavior test, semantic claim, compatibility order, verification, and rollback before implementation."}}
		return writeResult(cmd, *jsonOutput, result)
	}}
	plan.Flags().StringVar(&root, "root", ".", "project root")
	plan.Flags().StringVar(&objective, "objective", "", "normalized change objective")
	plan.Flags().StringSliceVar(&refs, "ref", nil, "related stable ID")
	_ = plan.MarkFlagRequired("objective")
	parent.AddCommand(plan)
	return parent
}

func newContractCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "contract", Short: "Validate and analyze cross-component behavioral obligations"}
	var file string
	check := &cobra.Command{Use: "check", RunE: func(cmd *cobra.Command, _ []string) error {
		definition, err := loadYAML[contract.Definition](file)
		if err != nil {
			return err
		}
		issues := contract.Check(definition)
		result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "contract.check", OperationID: "contract-check-read-only", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Contract structural and behavioral obligations are valid.", Blockers: issues}
		if len(issues) > 0 {
			result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitVerification, "Contract obligations are incomplete."
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	check.Flags().StringVar(&file, "file", "", "normalized contract YAML")
	_ = check.MarkFlagRequired("file")
	var root, id string
	impact := &cobra.Command{Use: "impact", RunE: func(cmd *cobra.Command, _ []string) error {
		snapshot, issues := contextpkg.Refresh(cmd.Context(), root, contextpkg.ReadOnly)
		result := contextResult(version, "contract-impact", root, snapshot, issues, contextpkg.ReadOnly)
		result.Command, result.OperationID = "contract.impact", "contract-impact-read-only"
		entry, exists := snapshot.Index[id]
		if !exists {
			result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitInvalid, "Contract stable ID is not indexed."
			result.Blockers = append(result.Blockers, domain.Item{Code: "contract.not-found", Message: id})
		} else {
			result.Summary = "Contract providers, consumers, and dependent product meaning were resolved."
			result.Facts = append(result.Facts, domain.Item{Code: "contract.source", Message: entry.Path, Refs: append([]string{id}, snapshot.Impact[id]...)})
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	impact.Flags().StringVar(&root, "root", ".", "project root")
	impact.Flags().StringVar(&id, "id", "", "contract stable ID")
	_ = impact.MarkFlagRequired("id")
	parent.AddCommand(check, impact)
	return parent
}

func newDatabaseCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "db", Short: "Validate DBML semantics and isolate dbdiagram collaboration"}
	var before, after string
	diffCommand := &cobra.Command{Use: "diff", RunE: func(cmd *cobra.Command, _ []string) error {
		left, err := os.ReadFile(before)
		if err != nil {
			return err
		}
		right, err := os.ReadFile(after)
		if err != nil {
			return err
		}
		diff, err := database.SemanticDiff(left, right)
		if err != nil {
			return err
		}
		result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "db.diff", OperationID: "db-diff-read-only", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "DBML semantic diff completed."}
		addDBFacts := func(code string, values []string) {
			sort.Strings(values)
			for _, value := range values {
				result.Changes = append(result.Changes, domain.Item{Code: code, Message: value})
			}
		}
		addDBFacts("db.table-added", diff.AddedTables)
		addDBFacts("db.table-removed", diff.RemovedTables)
		addDBFacts("db.column-added", diff.AddedColumns)
		addDBFacts("db.column-removed", diff.RemovedColumns)
		addDBFacts("db.relation-added", diff.AddedRelations)
		addDBFacts("db.relation-removed", diff.RemovedRelations)
		return writeResult(cmd, *jsonOutput, result)
	}}
	diffCommand.Flags().StringVar(&before, "before", "", "canonical DBML file")
	diffCommand.Flags().StringVar(&after, "after", "", "candidate DBML file")
	_ = diffCommand.MarkFlagRequired("before")
	_ = diffCommand.MarkFlagRequired("after")
	var config database.DBDiagramConfig
	diagram := &cobra.Command{Use: "diagram", RunE: func(cmd *cobra.Command, _ []string) error {
		plan, err := database.PullPlan(config)
		if err != nil {
			return err
		}
		return writeResult(cmd, *jsonOutput, planResult(version, "db.diagram", plan, "Isolated dbdiagram pull plan is ready; canonical DBML will not change."))
	}}
	diagram.Flags().StringVar(&config.Root, "root", ".", "project root")
	diagram.Flags().StringVar(&config.OperationID, "operation", "", "operation ID")
	diagram.Flags().StringVar(&config.Executable, "executable", "db2", "official dbdiagram CLI executable")
	diagram.Flags().StringVar(&config.ProjectID, "project-id", "", "dbdiagram project ID")
	diagram.Flags().StringVar(&config.TokenEnvironment, "token-env", "DBDIAGRAM_TOKEN", "credential environment variable name")
	_ = diagram.MarkFlagRequired("operation")
	_ = diagram.MarkFlagRequired("project-id")
	parent.AddCommand(diffCommand, diagram)
	return parent
}

func newUICommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "ui", Short: "Quarantine and register external UI sources"}
	var source uiimport.Source
	var apply bool
	importCommand := &cobra.Command{Use: "import", RunE: func(cmd *cobra.Command, _ []string) error {
		plan, err := uiimport.ImportPlan(source)
		if err != nil {
			return err
		}
		if apply {
			result := operation.Apply(cmd.Context(), plan)
			result.ToolVersion, result.Command = version, "ui.import"
			return writeResult(cmd, *jsonOutput, result)
		}
		return writeResult(cmd, *jsonOutput, planResult(version, "ui.import.plan", plan, "External UI source passed quarantine and import is planned."))
	}}
	importCommand.Flags().StringVar(&source.Root, "root", ".", "project root")
	importCommand.Flags().StringVar(&source.Archive, "archive", "", "source ZIP archive")
	importCommand.Flags().StringVar(&source.ID, "id", "", "stable UI source ID")
	importCommand.Flags().StringVar(&source.Kind, "kind", "mockup", "source kind")
	importCommand.Flags().StringVar(&source.Authority, "authority", "reference", "reference, seed, or canonical")
	importCommand.Flags().StringVar(&source.Version, "version", "", "source version")
	importCommand.Flags().StringVar(&source.License, "license", "", "declared license when archive lacks LICENSE")
	importCommand.Flags().BoolVar(&apply, "apply", false, "write only to local quarantine")
	_ = importCommand.MarkFlagRequired("archive")
	_ = importCommand.MarkFlagRequired("id")
	parent.AddCommand(importCommand)
	return parent
}

func decodeContract(data []byte) (contract.Definition, error) {
	var definition contract.Definition
	if err := yaml.Unmarshal(data, &definition); err != nil {
		return definition, fmt.Errorf("decode contract: %w", err)
	}
	return definition, nil
}

func safeRelative(root, target string) string {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return target
	}
	return filepath.ToSlash(relative)
}

func joined(values []string) string { return strings.Join(values, ",") }
