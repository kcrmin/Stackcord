package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
)

// PlanAdopt adds only missing files and managed sections, preserving existing project content.
func PlanAdopt(request InitRequest) (operation.Plan, error) {
	if request.Root == "" || request.ProjectID == "" || (request.Locale != "en" && request.Locale != "ko") {
		return operation.Plan{}, fmt.Errorf("root, stable project ID, and locale en|ko are required")
	}
	if !projectIDPattern.MatchString(request.ProjectID) {
		return operation.Plan{}, fmt.Errorf("project ID must be a lowercase dot-separated stable ID")
	}
	generated, err := render(request)
	if err != nil {
		return operation.Plan{}, err
	}
	plan := operation.Plan{ID: "adopt-" + request.ProjectID, Root: request.Root}
	for _, file := range generated {
		path := filepath.Join(request.Root, filepath.FromSlash(file.Path))
		existing, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			plan.Files = append(plan.Files, file)
			continue
		}
		if err != nil {
			return operation.Plan{}, err
		}
		switch file.Path {
		case "README.md", "AGENTS.md":
			file.Content = []byte(mergeManaged(string(existing), string(file.Content)))
			if string(file.Content) != string(existing) {
				plan.Files = append(plan.Files, file)
			}
		case ".gitattributes":
			if !strings.Contains(string(existing), "* text=auto eol=lf") {
				plan.Blockers = append(plan.Blockers, domain.Item{Code: "project.tooling-conflict", Message: "Existing .gitattributes conflicts with cross-platform LF normalization.", Refs: []string{file.Path}})
			}
		case ".editorconfig":
			if strings.Contains(string(existing), "end_of_line") && !strings.Contains(string(existing), "end_of_line = lf") {
				plan.Blockers = append(plan.Blockers, domain.Item{Code: "project.tooling-conflict", Message: "Existing .editorconfig uses a different line-ending policy.", Refs: []string{file.Path}})
			}
		case ".gitignore":
			file.Content = []byte(mergeGitignore(string(existing), string(file.Content)))
			if string(file.Content) != string(existing) {
				plan.Files = append(plan.Files, file)
			}
		case ".harness/workspaces.yaml":
			upgraded, changed, upgradeErr := upgradeWorkspaceIdentity(string(existing), request.ProjectID)
			if upgradeErr != nil {
				plan.Blockers = append(plan.Blockers, domain.Item{Code: "project.workspace-identity-conflict", Message: upgradeErr.Error(), Refs: []string{file.Path}})
			} else if changed {
				file.Content = []byte(upgraded)
				plan.Files = append(plan.Files, file)
			}
		default:
			// Existing authored project files always win. Adoption does not replace them.
		}
	}
	if len(plan.Blockers) > 0 {
		plan.Files = nil
		return plan, nil
	}
	fingerprint, err := operation.StateFingerprint(plan)
	plan.InitialStateFingerprint = fingerprint
	return plan, err
}

func upgradeWorkspaceIdentity(existing, projectID string) (string, bool, error) {
	value, err := schema.DecodeYAML[map[string]any]([]byte(existing))
	if err != nil {
		return "", false, fmt.Errorf("decode existing workspace manifest: %w", err)
	}
	if current, exists := value["project_id"]; exists {
		currentID, ok := current.(string)
		if !ok || currentID != projectID {
			return "", false, fmt.Errorf("existing workspace project ID %q differs from requested %q", current, projectID)
		}
		return existing, false, nil
	}
	if version, ok := value["schema_version"].(int); !ok || version != 1 {
		return "", false, fmt.Errorf("legacy workspace manifest must use schema_version 1")
	}
	lineEnding := "\n"
	if strings.Contains(existing, "\r\n") {
		lineEnding = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(existing, "\r\n", "\n"), "\n")
	for index, line := range lines {
		if strings.TrimSpace(line) == "schema_version: 1" {
			lines = append(lines[:index+1], append([]string{"project_id: " + projectID}, lines[index+1:]...)...)
			return strings.Join(lines, lineEnding), true, nil
		}
	}
	return "", false, fmt.Errorf("legacy workspace manifest has no schema_version field")
}

func mergeGitignore(existing, generated string) string {
	normalizedExisting := strings.ReplaceAll(strings.ReplaceAll(existing, "\r\n", "\n"), "\r", "\n")
	seen := map[string]bool{}
	for _, line := range strings.Split(normalizedExisting, "\n") {
		seen[line] = true
	}
	missing := []string{}
	for _, line := range strings.Split(strings.ReplaceAll(generated, "\r\n", "\n"), "\n") {
		if line != "" && !seen[line] {
			seen[line] = true
			missing = append(missing, line)
		}
	}
	if len(missing) == 0 {
		return existing
	}
	lineEnding := "\n"
	if strings.Contains(existing, "\r\n") {
		lineEnding = "\r\n"
	}
	result := existing
	if result != "" && !strings.HasSuffix(result, "\n") && !strings.HasSuffix(result, "\r") {
		result += lineEnding
	}
	return result + strings.Join(missing, lineEnding) + lineEnding
}
