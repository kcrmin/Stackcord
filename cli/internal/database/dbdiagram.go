package database

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"fullstack-orchestrator/cli/internal/operation"
)

const maxDBMLEntryBytes = 16 << 20

var safeIdentifier = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// DBDiagramConfig contains identifiers only; the token remains in an environment variable.
type DBDiagramConfig struct {
	Root             string
	OperationID      string
	Action           string
	Entry            string
	Executable       string
	ProjectID        string
	TokenEnvironment string
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
	if !environmentName.MatchString(config.TokenEnvironment) {
		return operation.Plan{}, fmt.Errorf("token environment variable name is invalid")
	}
	entry, err := readCanonicalDBML(config.Root, config.Entry)
	if err != nil {
		return operation.Plan{}, err
	}
	scratchRelative := filepath.Join(".harness", "local", "dbdiagram", config.OperationID)
	scratch := filepath.Join(config.Root, scratchRelative)
	plan := operation.Plan{
		ID:   "dbdiagram-" + config.Action + "-" + config.OperationID,
		Root: config.Root,
		Files: []operation.FileChange{{
			Path:    filepath.ToSlash(filepath.Join(scratchRelative, "candidate.dbml")),
			Content: entry,
			Mode:    0o600,
		}},
		Commands: []operation.CommandStep{
			{Program: config.Executable, Args: []string{"init", "--entry", "candidate.dbml", "--diagram-id", config.ProjectID}, Directory: scratch, ApprovalClass: "C"},
			{Program: config.Executable, Args: []string{config.Action}, Directory: scratch, ApprovalClass: "C"},
		},
	}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return plan, err
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
