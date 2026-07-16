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
			merged := strings.TrimRight(string(existing), "\r\n") + "\n" + string(file.Content)
			file.Content = []byte(uniqueLines(merged))
			plan.Files = append(plan.Files, file)
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

func uniqueLines(value string) string {
	seen := map[string]bool{}
	result := []string{}
	for _, line := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
		if !seen[line] {
			seen[line] = true
			result = append(result, line)
		}
	}
	return strings.TrimRight(strings.Join(result, "\n"), "\n") + "\n"
}
