package database

import (
	"fmt"
	"path/filepath"
	"regexp"

	"fullstack-orchestrator/cli/internal/operation"
)

var safeIdentifier = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// DBDiagramConfig contains identifiers only; the token remains in an environment variable.
type DBDiagramConfig struct {
	Root             string
	OperationID      string
	Executable       string
	ProjectID        string
	TokenEnvironment string
}

// PullPlan targets an isolated scratch directory and never canonical DBML.
func PullPlan(config DBDiagramConfig) (operation.Plan, error) {
	if config.OperationID == "" || !safeIdentifier.MatchString(config.OperationID) || config.ProjectID == "" || !safeIdentifier.MatchString(config.ProjectID) {
		return operation.Plan{}, fmt.Errorf("safe operation and dbdiagram project IDs are required")
	}
	if config.Executable == "" {
		config.Executable = "db2"
	}
	if config.TokenEnvironment == "" {
		return operation.Plan{}, fmt.Errorf("token environment variable name is required")
	}
	scratch := filepath.Join(config.Root, ".harness", "local", "dbdiagram", config.OperationID)
	return operation.Plan{ID: "dbdiagram-pull-" + config.OperationID, Root: config.Root, Commands: []operation.CommandStep{{Program: config.Executable, Args: []string{"pull", "--project", config.ProjectID, "--output", "candidate.dbml"}, Directory: scratch, ApprovalClass: "C"}}}, nil
}
