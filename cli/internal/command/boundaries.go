package command

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"

	contextpkg "fullstack-orchestrator/cli/internal/context"
	"fullstack-orchestrator/cli/internal/contract"
	"fullstack-orchestrator/cli/internal/database"
	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	uiimport "fullstack-orchestrator/cli/internal/ui"
	"github.com/spf13/cobra"
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
		addDBFacts("db.column-changed", diff.ChangedColumns)
		addDBFacts("db.relation-added", diff.AddedRelations)
		addDBFacts("db.relation-removed", diff.RemovedRelations)
		addDBFacts("db.index-added", diff.AddedIndexes)
		addDBFacts("db.index-removed", diff.RemovedIndexes)
		addDBFacts("db.note-added", diff.AddedNotes)
		addDBFacts("db.note-removed", diff.RemovedNotes)
		addDBFacts("db.definition-added", diff.AddedDefinitions)
		addDBFacts("db.definition-removed", diff.RemovedDefinitions)
		addDBFacts("db.definition-changed", diff.ChangedDefinitions)
		return writeResult(cmd, *jsonOutput, result)
	}}
	diffCommand.Flags().StringVar(&before, "before", "", "canonical DBML file")
	diffCommand.Flags().StringVar(&after, "after", "", "candidate DBML file")
	_ = diffCommand.MarkFlagRequired("before")
	_ = diffCommand.MarkFlagRequired("after")
	var config database.DBDiagramConfig
	var apply bool
	diagram := &cobra.Command{Use: "diagram", Short: "Prepare and reconcile isolated dbdiagram proposals"}
	prepare := &cobra.Command{Use: "prepare", RunE: func(cmd *cobra.Command, _ []string) error {
		plan, err := database.SyncPlan(config)
		if err != nil {
			return err
		}
		planned := planResult(version, "db.diagram.plan", plan, "Isolated official dbdiagram sync plan is ready; canonical DBML will not change.")
		if !apply {
			return writeResult(cmd, *jsonOutput, planned)
		}
		result := operation.Apply(cmd.Context(), plan)
		result.ToolVersion, result.Command = version, "db.diagram.prepare"
		if result.Status == domain.StatusPassed {
			result.Summary = "The isolated DBML copy is ready; external dbdiagram commands remain explicit."
			result.NextActions = planned.NextActions
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	prepare.Flags().StringVar(&config.Root, "root", ".", "project root")
	prepare.Flags().StringVar(&config.OperationID, "operation", "", "operation ID")
	prepare.Flags().StringVar(&config.Action, "action", "pull", "isolated dbdiagram action: push or pull")
	prepare.Flags().StringVar(&config.Entry, "entry", "", "canonical DBML path inside the project")
	prepare.Flags().StringVar(&config.Executable, "executable", "dbdiagram", "official dbdiagram CLI executable")
	prepare.Flags().StringVar(&config.ToolVersion, "tool-version", "", "observed official dbdiagram CLI version")
	prepare.Flags().StringVar(&config.ProjectID, "project-id", "", "dbdiagram project ID")
	prepare.Flags().StringVar(&config.TokenEnvironment, "token-env", "DBDIAGRAM_TOKEN", "credential environment variable name")
	prepare.Flags().BoolVar(&apply, "apply", false, "write only the isolated DBML copy and provenance; never run external commands")
	_ = prepare.MarkFlagRequired("operation")
	_ = prepare.MarkFlagRequired("entry")
	_ = prepare.MarkFlagRequired("project-id")
	_ = prepare.MarkFlagRequired("tool-version")

	var reconcileRoot, reconcileOperation string
	var contractIDs, migrationIDs, testIDs, rollbackIDs []string
	var record, reconcileApply bool
	reconcile := &cobra.Command{Use: "reconcile", RunE: func(cmd *cobra.Command, _ []string) error {
		if record && reconcileApply {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "db.diagram.reconcile", "db.mode-invalid", "Record and canonical apply are separate review steps."))
		}
		proposalPath := filepath.Join(reconcileRoot, ".harness", "local", "dbdiagram", reconcileOperation, "proposal.yaml")
		if reconcileApply {
			proposal, plan, issues, err := database.ReconcileProposal(database.ReconcileRequest{Root: reconcileRoot, ProposalPath: proposalPath})
			if err != nil {
				return err
			}
			if len(issues) > 0 {
				result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "db.diagram.reconcile", OperationID: "db-reconcile-read-only", Status: domain.StatusBlocked, ExitCode: domain.ExitVerification, Summary: "Database proposal is stale or no longer identical.", Blockers: issues}
				return writeResult(cmd, *jsonOutput, result)
			}
			result := operation.Apply(cmd.Context(), plan)
			result.ToolVersion, result.Command = version, "db.diagram.reconcile"
			if result.Status == domain.StatusPassed {
				result.Summary = "Reviewed DBML proposal replaced the exact unchanged canonical base."
				result.Facts = databaseProposalFacts(proposal)
			}
			return writeResult(cmd, *jsonOutput, result)
		}
		preparation, err := database.LoadPreparation(reconcileRoot, reconcileOperation)
		if err != nil {
			return err
		}
		candidate, fetchedAt, err := database.LoadPreparedCandidate(reconcileRoot, reconcileOperation)
		if err != nil {
			return err
		}
		proposal, plan, err := database.PrepareProposal(database.ProposalRequest{
			Root: reconcileRoot, OperationID: reconcileOperation, Entry: preparation.CanonicalPath, Candidate: candidate,
			Tool: preparation.Tool, ToolVersion: preparation.ToolVersion, ProjectID: preparation.ProjectID, Action: preparation.Action, FetchedAt: fetchedAt, ExpectedBaseFingerprint: preparation.BaseFingerprint,
			ContractIDs: contractIDs, MigrationIDs: migrationIDs, TestIDs: testIDs, RollbackIDs: rollbackIDs,
		})
		if err != nil {
			return err
		}
		result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "db.diagram.reconcile", OperationID: "db-proposal-read-only", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "Isolated DBML proposal and semantic impact are ready for review.", Facts: databaseProposalFacts(proposal), NextActions: []domain.Item{{Code: "db.proposal-record", Message: "Record the reviewed proposal provenance without changing canonical DBML."}}}
		if record {
			result = operation.Apply(cmd.Context(), plan)
			result.ToolVersion, result.Command = version, "db.diagram.reconcile"
			if result.Status == domain.StatusPassed {
				result.Summary = "Proposal provenance was recorded; canonical DBML is unchanged."
				result.Facts = databaseProposalFacts(proposal)
				result.NextActions = []domain.Item{{Code: "db.proposal-approve", Message: "Review semantic diff, migration order, tests, and rollback before canonical apply."}}
			}
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	reconcile.Flags().StringVar(&reconcileRoot, "root", ".", "project root")
	reconcile.Flags().StringVar(&reconcileOperation, "operation", "", "prepared dbdiagram operation ID")
	reconcile.Flags().StringSliceVar(&contractIDs, "contract", nil, "affected data contract stable ID")
	reconcile.Flags().StringSliceVar(&migrationIDs, "migration", nil, "required migration order stable ID")
	reconcile.Flags().StringSliceVar(&testIDs, "test", nil, "required database test stable ID")
	reconcile.Flags().StringSliceVar(&rollbackIDs, "rollback", nil, "required rollback stable ID")
	reconcile.Flags().BoolVar(&record, "record", false, "record local proposal provenance without canonical mutation")
	reconcile.Flags().BoolVar(&reconcileApply, "apply", false, "apply an already-recorded proposal to its exact unchanged canonical base")
	_ = reconcile.MarkFlagRequired("operation")
	diagram.AddCommand(prepare, reconcile)
	parent.AddCommand(diffCommand, diagram)
	return parent
}

func newUICommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "ui", Short: "Quarantine and register external UI sources"}
	var source uiimport.Source
	var apply bool
	importCommand := &cobra.Command{Use: "import", RunE: func(cmd *cobra.Command, _ []string) error {
		registration, plan, err := uiimport.Register(source)
		if err != nil {
			return err
		}
		if apply {
			result := operation.Apply(cmd.Context(), plan)
			result.ToolVersion, result.Command = version, "ui.import"
			if result.Status == domain.StatusPassed {
				result.Facts = []domain.Item{{Code: "ui.source", Message: registration.ID, Refs: []string{registration.Authority, registration.ContentHash}}}
			}
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
	importCommand.Flags().StringSliceVar(&source.MappedRefs, "ref", nil, "mapped UI flow or product stable ID")
	importCommand.Flags().StringSliceVar(&source.Consumers, "consumer", nil, "consumer workspace or component stable ID")
	importCommand.Flags().BoolVar(&apply, "apply", false, "write only to local quarantine")
	_ = importCommand.MarkFlagRequired("archive")
	_ = importCommand.MarkFlagRequired("id")
	var reconcileRoot, reconcileID, reconcileArchive, reconcileVersion, reconcileLicense string
	var reconcileApply bool
	reconcile := &cobra.Command{Use: "reconcile", RunE: func(cmd *cobra.Command, _ []string) error {
		current, err := uiimport.LoadRegistration(reconcileRoot, reconcileID)
		if err != nil {
			return err
		}
		if reconcileVersion == "" {
			reconcileVersion = current.SourceVersion
		}
		if reconcileLicense == "" {
			reconcileLicense = current.License
		}
		next, plan, err := uiimport.Register(uiimport.Source{Root: reconcileRoot, Archive: reconcileArchive, ID: current.ID, Kind: current.Kind, Authority: current.Authority, Version: reconcileVersion, License: reconcileLicense, MappedRefs: current.MappedRefs, Consumers: current.Consumers, BaselineFingerprint: current.BaselineFingerprint})
		if err != nil {
			return err
		}
		state := uiimport.Reconcile(current, next)
		result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "ui.reconcile", OperationID: "ui-reconcile-read-only", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "External UI source is unchanged.", Facts: []domain.Item{{Code: "ui.source", Message: current.ID, Refs: []string{current.Authority, next.ContentHash}}}}
		for _, blocker := range state.Blockers {
			result.Blockers = append(result.Blockers, domain.Item{Code: blocker, Message: "External UI source identity or authority changed outside the approved mapping."})
		}
		if len(result.Blockers) > 0 {
			result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitBlocked, "External UI source cannot be reconciled safely."
			return writeResult(cmd, *jsonOutput, result)
		}
		if state.Changed {
			result.Status, result.Summary = domain.StatusWarning, "External UI source changed and requires review before its registration is updated."
			result.Changes = []domain.Item{{Code: "ui.source-changed", Message: next.ContentHash, Refs: state.StaleRefs}}
			for _, ref := range state.StaleRefs {
				result.Warnings = append(result.Warnings, domain.Item{Code: "ui.mapping-stale", Message: "Mapped canonical UI or consumer needs reconciliation.", Refs: []string{ref}})
			}
		}
		if reconcileApply && state.Changed {
			applied := operation.Apply(cmd.Context(), plan)
			applied.ToolVersion, applied.Command = version, "ui.reconcile"
			if applied.Status == domain.StatusPassed {
				applied.Summary = "Reviewed external UI registration and quarantine were updated; mapped canonical UI remains stale until integrated."
				applied.Warnings = result.Warnings
				applied.Facts = result.Facts
			}
			return writeResult(cmd, *jsonOutput, applied)
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	reconcile.Flags().StringVar(&reconcileRoot, "root", ".", "project root")
	reconcile.Flags().StringVar(&reconcileID, "id", "", "registered external UI stable ID")
	reconcile.Flags().StringVar(&reconcileArchive, "archive", "", "new source ZIP archive")
	reconcile.Flags().StringVar(&reconcileVersion, "version", "", "new source version")
	reconcile.Flags().StringVar(&reconcileLicense, "license", "", "reviewed source license")
	reconcile.Flags().BoolVar(&reconcileApply, "apply", false, "update reviewed source registration and quarantine only")
	_ = reconcile.MarkFlagRequired("id")
	_ = reconcile.MarkFlagRequired("archive")
	var integrateRoot, integrateID, integrateWorkID string
	var integrateApply bool
	integrate := &cobra.Command{Use: "integrate", RunE: func(cmd *cobra.Command, _ []string) error {
		definition, found, err := loadStartDefinition(integrateRoot, integrateWorkID)
		if err != nil {
			return err
		}
		if !found {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "ui.integrate", "work.definition-required", "UI baseline integration requires executable work scope.", integrateWorkID))
		}
		registration, plan, err := uiimport.AcceptIntegratedBaseline(integrateRoot, integrateID, definition)
		if err != nil {
			return err
		}
		if !integrateApply {
			result := planResult(version, "ui.integrate.plan", plan, "UI baseline acknowledgement is ready to commit with the implemented work.")
			result.Facts = []domain.Item{{Code: "ui.baseline", Message: registration.ContentHash, Refs: []string{registration.ID, integrateWorkID}}}
			return writeResult(cmd, *jsonOutput, result)
		}
		result := operation.Apply(cmd.Context(), plan)
		result.ToolVersion, result.Command = version, "ui.integrate"
		if result.Status == domain.StatusPassed {
			result.Summary = "UI baseline is acknowledged inside the executable work scope; commit it with implementation, then record integration evidence."
			result.Facts = []domain.Item{{Code: "ui.baseline", Message: registration.ContentHash, Refs: []string{registration.ID, integrateWorkID}}}
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	integrate.Flags().StringVar(&integrateRoot, "root", ".", "project root")
	integrate.Flags().StringVar(&integrateID, "id", "", "registered external UI stable ID")
	integrate.Flags().StringVar(&integrateWorkID, "work-id", "", "ready work definition covering every UI mapping and consumer")
	integrate.Flags().BoolVar(&integrateApply, "apply", false, "write the reviewed baseline acknowledgement for the implementation commit")
	_ = integrate.MarkFlagRequired("id")
	_ = integrate.MarkFlagRequired("work-id")
	parent.AddCommand(importCommand, reconcile, integrate)
	return parent
}

func databaseProposalFacts(proposal database.Proposal) []domain.Item {
	facts := []domain.Item{
		{Code: "db.proposal", Message: proposal.ID, Refs: []string{proposal.BaseFingerprint, proposal.ContentHash}},
		{Code: "db.entities", Message: fmtInt(len(proposal.Impact.Entities)), Refs: proposal.Impact.Entities},
		{Code: "db.migrations", Message: fmtInt(len(proposal.Impact.MigrationOrder)), Refs: proposal.Impact.MigrationOrder},
		{Code: "db.tests", Message: fmtInt(len(proposal.Impact.Tests)), Refs: proposal.Impact.Tests},
		{Code: "db.rollback", Message: fmtInt(len(proposal.Impact.Rollback)), Refs: proposal.Impact.Rollback},
	}
	for _, item := range []struct {
		code string
		refs []string
	}{
		{"db.tables.added", proposal.Diff.AddedTables}, {"db.tables.removed", proposal.Diff.RemovedTables},
		{"db.columns.added", proposal.Diff.AddedColumns}, {"db.columns.removed", proposal.Diff.RemovedColumns}, {"db.columns.changed", proposal.Diff.ChangedColumns},
		{"db.relations.added", proposal.Diff.AddedRelations}, {"db.relations.removed", proposal.Diff.RemovedRelations},
		{"db.indexes.added", proposal.Diff.AddedIndexes}, {"db.indexes.removed", proposal.Diff.RemovedIndexes},
		{"db.notes.added", proposal.Diff.AddedNotes}, {"db.notes.removed", proposal.Diff.RemovedNotes},
		{"db.definitions.added", proposal.Diff.AddedDefinitions}, {"db.definitions.removed", proposal.Diff.RemovedDefinitions}, {"db.definitions.changed", proposal.Diff.ChangedDefinitions},
	} {
		if len(item.refs) > 0 {
			facts = append(facts, domain.Item{Code: item.code, Message: fmtInt(len(item.refs)), Refs: item.refs})
		}
	}
	return facts
}

func fmtInt(value int) string { return strconv.Itoa(value) }
