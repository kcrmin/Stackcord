package database

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/schema"
	"go.yaml.in/yaml/v3"
)

const maxDBMLEntryBytes = 16 << 20

var safeIdentifier = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var databaseDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// DBDiagramConfig contains identifiers only; the token remains in an environment variable.
type DBDiagramConfig struct {
	Root             string
	OperationID      string
	Action           string
	Entry            string
	Executable       string
	ToolVersion      string
	ProjectID        string
	TokenEnvironment string
	PreparedAt       time.Time
}

// Preparation is the immutable local identity captured before the official CLI runs.
type Preparation struct {
	SchemaVersion   int       `json:"schema_version" yaml:"schema_version"`
	Tool            string    `json:"tool" yaml:"tool"`
	ToolVersion     string    `json:"tool_version" yaml:"tool_version"`
	ProjectID       string    `json:"project_id" yaml:"project_id"`
	Action          string    `json:"action" yaml:"action"`
	CanonicalPath   string    `json:"canonical_path" yaml:"canonical_path"`
	BaseFingerprint string    `json:"base_fingerprint" yaml:"base_fingerprint"`
	PreparedAt      time.Time `json:"prepared_at" yaml:"prepared_at"`
}

// SyncPlan copies canonical DBML into an isolated scratch directory and plans the official CLI commands.
// The returned commands remain explicit: applying the local file plan never runs an external command.
func SyncPlan(config DBDiagramConfig) (operation.Plan, error) {
	if config.OperationID == "" || !safeIdentifier.MatchString(config.OperationID) || config.ProjectID == "" || !safeIdentifier.MatchString(config.ProjectID) {
		return operation.Plan{}, fmt.Errorf("safe operation and dbdiagram project IDs are required")
	}
	if config.Action == "" {
		config.Action = "pull"
	}
	if config.Action != "push" && config.Action != "pull" {
		return operation.Plan{}, fmt.Errorf("dbdiagram action must be push or pull")
	}
	if config.Executable == "" {
		config.Executable = "dbdiagram"
	}
	executableName := strings.ToLower(filepath.Base(config.Executable))
	if (executableName != "dbdiagram" && executableName != "dbdiagram.exe") || strings.TrimSpace(config.ToolVersion) == "" {
		return operation.Plan{}, fmt.Errorf("official dbdiagram executable and observed version are required")
	}
	if !environmentName.MatchString(config.TokenEnvironment) {
		return operation.Plan{}, fmt.Errorf("token environment variable name is invalid")
	}
	entry, err := readCanonicalDBML(config.Root, config.Entry)
	if err != nil {
		return operation.Plan{}, err
	}
	scratchRelative := filepath.Join(".harness", "local", "dbdiagram", config.OperationID)
	scratch := filepath.Join(config.Root, scratchRelative)
	canonicalPath, err := canonicalDBMLRelative(config.Root, config.Entry)
	if err != nil {
		return operation.Plan{}, err
	}
	if config.PreparedAt.IsZero() {
		if info, statErr := os.Stat(filepath.Join(config.Root, filepath.FromSlash(canonicalPath))); statErr == nil {
			config.PreparedAt = info.ModTime().UTC()
		}
	}
	baseData, err := yaml.Marshal(Preparation{SchemaVersion: 1, Tool: "dbdiagram", ToolVersion: config.ToolVersion, ProjectID: config.ProjectID, Action: config.Action, CanonicalPath: canonicalPath, BaseFingerprint: dbDigest(entry), PreparedAt: config.PreparedAt.UTC()})
	if err != nil {
		return operation.Plan{}, err
	}
	plan := operation.Plan{
		ID:   "dbdiagram-" + config.Action + "-" + config.OperationID,
		Root: config.Root,
		Files: []operation.FileChange{
			{Path: filepath.ToSlash(filepath.Join(scratchRelative, "candidate.dbml")), Content: entry, Mode: 0o600},
			{Path: filepath.ToSlash(filepath.Join(scratchRelative, "base.yaml")), Content: baseData, Mode: 0o600},
		},
		Commands: []operation.CommandStep{
			{Program: config.Executable, Args: []string{"init", "--entry", "candidate.dbml", "--diagram-id", config.ProjectID}, Directory: scratch, ApprovalClass: "C"},
			{Program: config.Executable, Args: []string{config.Action}, Directory: scratch, ApprovalClass: "C"},
		},
	}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return plan, err
}

// LoadPreparation reads only the safe operation-local base record.
func LoadPreparation(root, operationID string) (Preparation, error) {
	if !safeIdentifier.MatchString(operationID) {
		return Preparation{}, fmt.Errorf("safe dbdiagram operation ID is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return Preparation{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return Preparation{}, err
	}
	path := filepath.Join(root, ".harness", "local", "dbdiagram", operationID, "base.yaml")
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Preparation{}, fmt.Errorf("dbdiagram preparation must be a regular non-symlink file")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(path) {
		return Preparation{}, fmt.Errorf("dbdiagram preparation cannot use symlinked storage")
	}
	preparation, err := schema.LoadYAML[Preparation](resolved)
	if err != nil {
		return Preparation{}, err
	}
	if preparation.SchemaVersion != 1 || preparation.Tool != "dbdiagram" || preparation.ToolVersion == "" || !safeIdentifier.MatchString(preparation.ProjectID) || (preparation.Action != "pull" && preparation.Action != "push") || preparation.CanonicalPath == "" || !databaseDigestPattern.MatchString(preparation.BaseFingerprint) || preparation.PreparedAt.IsZero() {
		return Preparation{}, fmt.Errorf("dbdiagram preparation identity is invalid")
	}
	return preparation, nil
}

// LoadPreparedCandidate reads the official CLI output without following a replaced link.
func LoadPreparedCandidate(root, operationID string) ([]byte, time.Time, error) {
	if !safeIdentifier.MatchString(operationID) {
		return nil, time.Time{}, fmt.Errorf("safe dbdiagram operation ID is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, time.Time{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, time.Time{}, err
	}
	path := filepath.Join(root, ".harness", "local", "dbdiagram", operationID, "candidate.dbml")
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > maxDBMLEntryBytes {
		return nil, time.Time{}, fmt.Errorf("dbdiagram candidate must be a safe non-empty regular DBML file")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(path) {
		return nil, time.Time{}, fmt.Errorf("dbdiagram candidate cannot use symlinked storage")
	}
	data, err := os.ReadFile(resolved)
	return data, info.ModTime().UTC(), err
}

func readCanonicalDBML(root, entry string) ([]byte, error) {
	if root == "" || entry == "" {
		return nil, fmt.Errorf("canonical DBML entry is required")
	}
	rootAbsolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rootResolved, err := filepath.EvalSymlinks(rootAbsolute)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}
	entryAbsolute := entry
	if !filepath.IsAbs(entryAbsolute) {
		entryAbsolute = filepath.Join(rootResolved, entryAbsolute)
	}
	entryResolved, err := filepath.EvalSymlinks(entryAbsolute)
	if err != nil {
		return nil, fmt.Errorf("resolve canonical DBML entry: %w", err)
	}
	relative, err := filepath.Rel(rootResolved, entryResolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("canonical DBML entry must be inside the project root")
	}
	if strings.ToLower(filepath.Ext(entryResolved)) != ".dbml" {
		return nil, fmt.Errorf("canonical database entry must be a .dbml file")
	}
	info, err := os.Stat(entryResolved)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxDBMLEntryBytes {
		return nil, fmt.Errorf("canonical DBML entry must be a regular file no larger than %d bytes", maxDBMLEntryBytes)
	}
	return os.ReadFile(entryResolved)
}
