package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
)

// PlanAdopt adds only missing files and managed sections, preserving existing project content.
func PlanAdopt(request InitRequest) (operation.Plan, error) {
	if request.Root == "" || request.ProjectID == "" || (request.Locale != "en" && request.Locale != "ko") {
		return operation.Plan{}, fmt.Errorf("root, stable project ID, and locale en|ko are required")
	}
	if !projectIDPattern.MatchString(request.ProjectID) {
		return operation.Plan{}, fmt.Errorf("project ID must be a lowercase dot-separated stable ID")
	}
	generated := render(request)
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
