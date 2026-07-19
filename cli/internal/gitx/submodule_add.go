package gitx

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
)

var scpGitRemotePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+@[A-Za-z0-9.-]+:[A-Za-z0-9._/-]+$`)

// SubmoduleAddRequest identifies one reviewed existing repository boundary.
type SubmoduleAddRequest struct {
	Root   string
	Remote string
	Path   string
}

// SafeRemoteURL accepts credential-free network Git remotes and rejects local transports.
func SafeRemoteURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\x00\r\n") || strings.HasPrefix(value, "-") || strings.Contains(value, "..") {
		return false
	}
	if scpGitRemotePattern.MatchString(value) {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "ssh") || parsed.Host == "" || parsed.User != nil || parsed.Path == "" || parsed.Fragment != "" {
		return false
	}
	return true
}

// PlanSubmoduleAdd verifies local preconditions and describes the only Git mutation allowed.
func PlanSubmoduleAdd(ctx context.Context, request SubmoduleAddRequest) operation.Plan {
	plan := operation.Plan{ID: "submodule-add-" + strings.ReplaceAll(filepath.ToSlash(request.Path), "/", "-"), Root: request.Root}
	state, err := Inspect(ctx, request.Root)
	if err != nil {
		plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.repository-invalid", Message: err.Error()})
		return plan
	}
	root, err := canonicalPath(request.Root)
	if err != nil || root != state.Root {
		plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.root-mismatch", Message: "Submodules can only be added from the exact orchestration root."})
		return plan
	}
	plan.Root = root
	path := filepath.ToSlash(strings.TrimSpace(request.Path))
	if !SafeRemoteURL(request.Remote) {
		plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.submodule-remote-unsafe", Message: "Submodule remote must be a credential-free HTTPS or SSH URL."})
	}
	if !safeMutationPath(path) || path == "." {
		plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.submodule-path-invalid", Message: "Submodule path must be a new repository-relative path."})
	}
	if state.Dirty {
		plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.base-dirty", Message: "The orchestration root has uncommitted changes."})
	}
	if state.Detached {
		plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.base-detached", Message: "A detached checkout cannot add a shared repository boundary."})
	}
	if state.Diverged {
		plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.base-diverged", Message: "The orchestration branch has diverged from its upstream."})
	}
	if safeMutationPath(path) {
		if _, statErr := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); !os.IsNotExist(statErr) {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.submodule-target-exists", Message: "Submodule target already exists.", Refs: []string{path}})
		}
		for _, submodule := range state.Submodules {
			if submodule.Path == path {
				plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.submodule-already-declared", Message: "Submodule path is already declared.", Refs: []string{path}})
			}
		}
		if declared, declarationErr := declaredWorkspacePathSet(root); declarationErr != nil {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.workspace-manifest-invalid", Message: declarationErr.Error()})
		} else if declared[path] {
			plan.Blockers = append(plan.Blockers, domain.Item{Code: "git.workspace-path-in-use", Message: "Workspace path is already declared.", Refs: []string{path}})
		}
	}
	if len(plan.Blockers) == 0 {
		plan.Commands = []operation.CommandStep{{Program: "git", Args: []string{"submodule", "add", "--", request.Remote, path}, Directory: root, ApprovalClass: "C"}}
	}
	return plan
}

// AddSubmodule executes the reviewed command and verifies exact Git postconditions.
func AddSubmodule(ctx context.Context, request SubmoduleAddRequest) domain.Result {
	result := gitMutationResult("git.submodule-add", "submodule-add-"+strings.ReplaceAll(filepath.ToSlash(request.Path), "/", "-"))
	plan := PlanSubmoduleAdd(ctx, request)
	if len(plan.Blockers) > 0 {
		result.Status, result.ExitCode, result.Summary, result.Blockers = domain.StatusBlocked, domain.ExitBlocked, "Submodule addition is not safe in the current repository state.", plan.Blockers
		return result
	}
	path := filepath.ToSlash(strings.TrimSpace(request.Path))
	if _, err := (runner{}).mutate(ctx, plan.Root, mutationSubmoduleAdd, "submodule", "add", "--", request.Remote, path); err != nil {
		return failGitMutation(result, "git.submodule-add-failed", err)
	}
	state, err := Inspect(ctx, plan.Root)
	if err != nil {
		return failGitMutation(result, "git.submodule-postcondition", err)
	}
	for _, submodule := range state.Submodules {
		if submodule.Path == path && submodule.URL == request.Remote && submodule.Initialized && submodule.Head != "" && submodule.ExpectedSHA == submodule.Head && !submodule.Dirty && !submodule.UnsafeURL {
			result.Status, result.ExitCode, result.Summary = domain.StatusPassed, domain.ExitSuccess, "Submodule was added and its clean Git identity was verified; no commit or push was performed."
			result.Evidence = []domain.Item{{Code: "git.submodule-added", Message: path, Refs: []string{request.Remote, submodule.Head}}}
			return result
		}
	}
	return blockGitMutation(result, "git.submodule-postcondition", "Created submodule differs from the reviewed remote, path, or clean HEAD.")
}

func declaredWorkspacePathSet(root string) (map[string]bool, error) {
	value, err := schema.LoadYAML[map[string]any](filepath.Join(root, ".harness", "workspaces.yaml"))
	if err != nil {
		return nil, err
	}
	entries, ok := value["workspaces"].([]any)
	if !ok {
		return nil, os.ErrInvalid
	}
	result := map[string]bool{}
	for _, raw := range entries {
		entry, ok := raw.(map[string]any)
		if !ok {
			return nil, os.ErrInvalid
		}
		path, _ := entry["path"].(string)
		if path != "." {
			result[filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))] = true
		}
	}
	return result, nil
}
