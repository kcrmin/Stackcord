package command

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/continuity"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/workspace"
)

func doctorFacts(ctx context.Context, root, version string) ([]domain.Item, []domain.Item) {
	facts := []domain.Item{
		{Code: "environment.os", Message: runtime.GOOS},
		{Code: "environment.arch", Message: runtime.GOARCH},
		{Code: "environment.go", Message: runtime.Version()},
		{Code: "environment.cli-version", Message: version},
	}
	warnings := []domain.Item{}
	if executable, err := os.Executable(); err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
			executable = resolved
		}
		facts = append(facts, domain.Item{Code: "environment.cli-path", Message: executable})
	} else {
		facts = append(facts, domain.Item{Code: "environment.cli-path", Message: "unavailable"})
		warnings = append(warnings, domain.Item{Code: "environment.cli-path-unknown", Message: "The running CLI path could not be resolved."})
	}
	gitVersion := "unavailable"
	if output, err := exec.CommandContext(ctx, "git", "--version").Output(); err == nil {
		gitVersion = strings.TrimSpace(string(output))
	} else {
		warnings = append(warnings, domain.Item{Code: "environment.git-unavailable", Message: "Git is unavailable; collaboration and verifiable release checks are reduced."})
	}
	facts = append(facts, domain.Item{Code: "environment.git-version", Message: gitVersion})

	dbdiagramCommand := firstAvailable("dbdiagram", "dbdocs", "dbml2sql")
	facts = append(facts, domain.Item{Code: "environment.dbdiagram-available", Message: strconv.FormatBool(dbdiagramCommand != "")})
	if dbdiagramCommand != "" {
		facts = append(facts, domain.Item{Code: "environment.dbdiagram-command", Message: dbdiagramCommand})
	}

	located, err := workspace.FindRoot(ctx, root)
	if err != nil {
		facts = append(facts, domain.Item{Code: "environment.project-detected", Message: "false"})
		return facts, warnings
	}
	facts = append(facts,
		domain.Item{Code: "environment.project-detected", Message: "true"},
		domain.Item{Code: "project.id", Message: located.Manifest.ProjectID},
		domain.Item{Code: "project.root", Message: located.Path},
		domain.Item{Code: "project.current-workspace", Message: located.CurrentWorkspaceID},
	)
	snapshot := continuity.Collect(ctx, located.Path, continuity.Options{})
	facts = append(facts, domain.Item{Code: "provider.selected", Message: snapshot.Provider.Name})
	connector := providerConnector(snapshot.Provider.Name)
	facts = append(facts, domain.Item{Code: "provider.connector-available", Message: strconv.FormatBool(connector != "")})
	if connector != "" {
		facts = append(facts, domain.Item{Code: "provider.connector", Message: connector})
	} else if snapshot.Provider.Name != "" {
		warnings = append(warnings, domain.Item{Code: "provider.connector-unavailable", Message: "The selected task source has no detected local connector.", Refs: []string{snapshot.Provider.Name}})
	}
	if dbdiagramCommand == "" && projectHasDBML(located.Path) {
		warnings = append(warnings, domain.Item{Code: "database.visualization-reduced", Message: "DBML remains canonical, but no supported dbdiagram-compatible CLI was detected."})
	}
	return facts, warnings
}

func firstAvailable(names ...string) string {
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

func providerConnector(provider string) string {
	switch provider {
	case "git-local":
		return "built-in"
	case "github":
		return firstAvailable("gh")
	case "jira":
		return firstAvailable("jira", "acli")
	default:
		return ""
	}
}

func projectHasDBML(root string) bool {
	for _, directory := range []string{"contracts", "db", "database"} {
		matched, _ := filepath.Glob(filepath.Join(root, directory, "*.dbml"))
		if len(matched) > 0 {
			return true
		}
		matched, _ = filepath.Glob(filepath.Join(root, directory, "*", "*.dbml"))
		if len(matched) > 0 {
			return true
		}
	}
	return false
}
