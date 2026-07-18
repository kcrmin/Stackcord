package database

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/schema"
	"go.yaml.in/yaml/v3"
)

// MigrationImpact names the semantic coordination required before canonical DBML changes.
type MigrationImpact struct {
	Contracts      []string `json:"contracts" yaml:"contracts"`
	Entities       []string `json:"entities" yaml:"entities"`
	MigrationOrder []string `json:"migration_order" yaml:"migration_order"`
	Tests          []string `json:"tests" yaml:"tests"`
	Rollback       []string `json:"rollback" yaml:"rollback"`
}

// Proposal is an isolated, fingerprint-bound database change from an external visual source.
type Proposal struct {
	SchemaVersion   int             `json:"schema_version" yaml:"schema_version"`
	ID              string          `json:"id" yaml:"id"`
	Kind            string          `json:"kind" yaml:"kind"`
	Authority       string          `json:"authority" yaml:"authority"`
	Source          string          `json:"source" yaml:"source"`
	SourceVersion   string          `json:"source_version" yaml:"source_version"`
	ContentHash     string          `json:"content_hash" yaml:"content_hash"`
	BaseFingerprint string          `json:"base_fingerprint" yaml:"base_fingerprint"`
	FetchedAt       time.Time       `json:"fetched_at" yaml:"fetched_at"`
	MappedRefs      []string        `json:"mapped_refs" yaml:"mapped_refs"`
	Consumers       []string        `json:"consumers" yaml:"consumers"`
	CanonicalPath   string          `json:"canonical_path" yaml:"canonical_path"`
	ProjectID       string          `json:"project_id" yaml:"project_id"`
	Action          string          `json:"action" yaml:"action"`
	Tool            string          `json:"tool" yaml:"tool"`
	Diff            Diff            `json:"diff" yaml:"diff"`
	Impact          MigrationImpact `json:"impact" yaml:"impact"`
	CandidatePath   string          `json:"-" yaml:"-"`
	RecordPath      string          `json:"-" yaml:"-"`
}

// ProposalRequest describes already-fetched candidate bytes; credentials never enter the request.
type ProposalRequest struct {
	Root, OperationID, Entry, Tool, ToolVersion, ProjectID, Action string
	Candidate                                                      []byte
	FetchedAt                                                      time.Time
	ContractIDs, MigrationIDs, TestIDs, RollbackIDs                []string
	ExpectedBaseFingerprint                                        string
}

// ReconcileRequest points at one isolated proposal record.
type ReconcileRequest struct{ Root, ProposalPath string }

// PrepareProposal records semantic diff and provenance without touching canonical DBML.
func PrepareProposal(request ProposalRequest) (Proposal, operation.Plan, error) {
	if request.Root == "" || !safeIdentifier.MatchString(request.OperationID) || request.Tool != "dbdiagram" || strings.TrimSpace(request.ToolVersion) == "" || !safeIdentifier.MatchString(request.ProjectID) || (request.Action != "pull" && request.Action != "push") || request.FetchedAt.IsZero() {
		return Proposal{}, operation.Plan{}, fmt.Errorf("safe operation, official dbdiagram identity, version, project, action, and fetch time are required")
	}
	if len(request.Candidate) == 0 || len(request.Candidate) > maxDBMLEntryBytes {
		return Proposal{}, operation.Plan{}, fmt.Errorf("candidate DBML must be non-empty and no larger than %d bytes", maxDBMLEntryBytes)
	}
	canonical, err := readCanonicalDBML(request.Root, request.Entry)
	if err != nil {
		return Proposal{}, operation.Plan{}, err
	}
	if request.ExpectedBaseFingerprint != "" && dbDigest(canonical) != request.ExpectedBaseFingerprint {
		return Proposal{}, operation.Plan{}, fmt.Errorf("canonical DBML changed after dbdiagram preparation")
	}
	diff, err := SemanticDiff(canonical, request.Candidate)
	if err != nil {
		return Proposal{}, operation.Plan{}, fmt.Errorf("parse candidate DBML: %w", err)
	}
	canonicalPath, err := canonicalDBMLRelative(request.Root, request.Entry)
	if err != nil {
		return Proposal{}, operation.Plan{}, err
	}
	id := "db.proposal." + strings.ToLower(strings.ReplaceAll(request.OperationID, "_", "-"))
	base := filepath.Join(".harness", "local", "dbdiagram", request.OperationID)
	proposal := Proposal{
		SchemaVersion: 1, ID: id, Kind: "dbml", Authority: "proposal", Source: "dbdiagram", SourceVersion: request.ToolVersion,
		ContentHash: dbDigest(request.Candidate), BaseFingerprint: dbDigest(canonical), FetchedAt: request.FetchedAt.UTC(),
		MappedRefs: normalizedDBRefs(request.ContractIDs), Consumers: normalizedDBRefs(append(append([]string(nil), request.TestIDs...), request.RollbackIDs...)),
		CanonicalPath: canonicalPath, ProjectID: request.ProjectID, Action: request.Action, Tool: request.Tool, Diff: diff,
		Impact:        MigrationImpact{Contracts: normalizedDBRefs(request.ContractIDs), Entities: diffEntities(diff), MigrationOrder: normalizedDBRefs(request.MigrationIDs), Tests: normalizedDBRefs(request.TestIDs), Rollback: normalizedDBRefs(request.RollbackIDs)},
		CandidatePath: filepath.Join(request.Root, base, "candidate.dbml"), RecordPath: filepath.Join(request.Root, base, "proposal.yaml"),
	}
	if issues := schema.Validate("external-source", proposal); len(issues) > 0 {
		return Proposal{}, operation.Plan{}, fmt.Errorf("validate database proposal: %s", issues[0].Message)
	}
	data, err := yaml.Marshal(proposal)
	if err != nil {
		return Proposal{}, operation.Plan{}, err
	}
	plan := operation.Plan{ID: "db-proposal-" + request.OperationID, Root: request.Root, Files: []operation.FileChange{
		{Path: filepath.ToSlash(filepath.Join(base, "candidate.dbml")), Content: append([]byte(nil), request.Candidate...), Mode: 0o600},
		{Path: filepath.ToSlash(filepath.Join(base, "proposal.yaml")), Content: data, Mode: 0o600},
	}}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return proposal, plan, err
}

// ReconcileProposal verifies exact base and candidate identities before planning a canonical write.
func ReconcileProposal(request ReconcileRequest) (Proposal, operation.Plan, []domain.Item, error) {
	path, err := safeProposalPath(request.Root, request.ProposalPath)
	if err != nil {
		return Proposal{}, operation.Plan{}, nil, err
	}
	proposal, err := schema.LoadYAML[Proposal](path)
	if err != nil {
		return Proposal{}, operation.Plan{}, nil, err
	}
	if issues := schema.Validate("external-source", proposal); len(issues) > 0 {
		return Proposal{}, operation.Plan{}, nil, fmt.Errorf("validate database proposal: %s", issues[0].Message)
	}
	proposal.RecordPath = path
	proposal.CandidatePath = filepath.Join(filepath.Dir(path), "candidate.dbml")
	candidate, err := readProposalCandidate(request.Root, proposal.CandidatePath)
	if err != nil {
		return Proposal{}, operation.Plan{}, nil, err
	}
	canonical, err := readCanonicalDBML(request.Root, proposal.CanonicalPath)
	if err != nil {
		return Proposal{}, operation.Plan{}, nil, err
	}
	issues := []domain.Item{}
	if dbDigest(canonical) != proposal.BaseFingerprint {
		issues = append(issues, domain.Item{Code: "db.proposal-stale-base", Message: "Canonical DBML changed after the proposal was prepared.", Refs: []string{proposal.CanonicalPath}})
	}
	if dbDigest(candidate) != proposal.ContentHash {
		issues = append(issues, domain.Item{Code: "db.proposal-content-changed", Message: "Isolated candidate changed after provenance was recorded.", Refs: []string{proposal.ID}})
	}
	diff, diffErr := SemanticDiff(canonical, candidate)
	if diffErr != nil {
		return Proposal{}, operation.Plan{}, nil, diffErr
	}
	if !sameDBDiff(diff, proposal.Diff) {
		issues = append(issues, domain.Item{Code: "db.proposal-diff-changed", Message: "Current semantic diff differs from recorded proposal metadata.", Refs: []string{proposal.ID}})
	}
	issues = append(issues, proposalImpactIssues(proposal)...)
	if len(issues) > 0 {
		return proposal, operation.Plan{}, issues, nil
	}
	data, err := yaml.Marshal(proposal)
	if err != nil {
		return Proposal{}, operation.Plan{}, nil, err
	}
	planRoot, err := filepath.Abs(request.Root)
	if err != nil {
		return Proposal{}, operation.Plan{}, nil, err
	}
	planRoot, err = filepath.EvalSymlinks(planRoot)
	if err != nil {
		return Proposal{}, operation.Plan{}, nil, err
	}
	relativeRecord, err := filepath.Rel(planRoot, path)
	if err != nil || relativeRecord == ".." || strings.HasPrefix(relativeRecord, ".."+string(filepath.Separator)) {
		return Proposal{}, operation.Plan{}, nil, fmt.Errorf("proposal record escapes project root")
	}
	plan := operation.Plan{ID: "db-reconcile-" + strings.TrimPrefix(proposal.ID, "db.proposal."), Root: planRoot, Files: []operation.FileChange{
		{Path: proposal.CanonicalPath, Content: candidate, Mode: 0o644},
		{Path: filepath.ToSlash(relativeRecord), Content: data, Mode: 0o600},
	}}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return proposal, plan, nil, err
}

func proposalImpactIssues(proposal Proposal) []domain.Item {
	if !hasDBChanges(proposal.Diff) {
		return nil
	}
	issues := []domain.Item{}
	if len(proposal.Impact.Contracts) == 0 {
		issues = append(issues, domain.Item{Code: "db.impact-contract-missing", Message: "Canonical DBML changes require an affected data contract.", Refs: []string{proposal.ID}})
	}
	if len(proposal.Impact.MigrationOrder) == 0 {
		issues = append(issues, domain.Item{Code: "db.impact-migration-missing", Message: "Canonical DBML changes require an explicit migration order.", Refs: []string{proposal.ID}})
	}
	if len(proposal.Impact.Tests) == 0 {
		issues = append(issues, domain.Item{Code: "db.impact-test-missing", Message: "Canonical DBML changes require database test evidence.", Refs: []string{proposal.ID}})
	}
	if len(proposal.Impact.Rollback) == 0 {
		issues = append(issues, domain.Item{Code: "db.impact-rollback-missing", Message: "Canonical DBML changes require a rollback definition.", Refs: []string{proposal.ID}})
	}
	return issues
}

func hasDBChanges(diff Diff) bool {
	return len(diff.AddedTables)+len(diff.RemovedTables)+len(diff.AddedColumns)+len(diff.RemovedColumns)+len(diff.ChangedColumns)+
		len(diff.AddedRelations)+len(diff.RemovedRelations)+len(diff.AddedIndexes)+len(diff.RemovedIndexes)+len(diff.AddedNotes)+
		len(diff.RemovedNotes)+len(diff.AddedDefinitions)+len(diff.RemovedDefinitions)+len(diff.ChangedDefinitions) > 0
}

func canonicalDBMLRelative(root, entry string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	entryPath := entry
	if !filepath.IsAbs(entryPath) {
		entryPath = filepath.Join(root, entryPath)
	}
	entryPath, err = filepath.EvalSymlinks(entryPath)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, entryPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("canonical DBML entry must be inside the project root")
	}
	return filepath.ToSlash(relative), nil
}

func safeProposalPath(root, value string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	path := value
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("proposal record must be a regular non-symlink file")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	allowed := filepath.Join(root, ".harness", "local", "dbdiagram")
	relative, err := filepath.Rel(allowed, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.Base(resolved) != "proposal.yaml" {
		return "", fmt.Errorf("proposal record must stay under .harness/local/dbdiagram")
	}
	return resolved, nil
}

func readProposalCandidate(root, path string) ([]byte, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > maxDBMLEntryBytes {
		return nil, fmt.Errorf("proposal candidate must be a safe DBML file")
	}
	allowed := filepath.Join(root, ".harness", "local", "dbdiagram")
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(allowed, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("proposal candidate escapes isolated storage")
	}
	return os.ReadFile(resolved)
}

func diffEntities(diff Diff) []string {
	set := map[string]bool{}
	for _, table := range append(append([]string(nil), diff.AddedTables...), diff.RemovedTables...) {
		set[table] = true
	}
	for _, value := range append(append(append([]string(nil), diff.AddedColumns...), diff.RemovedColumns...), diff.ChangedColumns...) {
		if table, _, found := strings.Cut(value, "."); found {
			set[table] = true
		}
	}
	result := []string{}
	for entity := range set {
		result = append(result, entity)
	}
	sort.Strings(result)
	return result
}

func normalizedDBRefs(values []string) []string {
	set := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = true
		}
	}
	result := []string{}
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func dbDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func sameDBDiff(left, right Diff) bool {
	a, _ := json.Marshal(left)
	b, _ := json.Marshal(right)
	return string(a) == string(b)
}
