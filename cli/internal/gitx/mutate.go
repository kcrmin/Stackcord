package gitx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/schema"
)

type mutationKind string

const (
	mutationWorktree  mutationKind = "worktree-add"
	mutationSubmodule mutationKind = "submodule-update"
)

var safeBasePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)

// CreateWorktreeRequest identifies one isolated conventional branch checkout.
type CreateWorktreeRequest struct {
	Root   string
	Branch string
	Base   string
	Target string
}

// CreateWorktree applies one allow-listed Git worktree mutation and verifies its exact postcondition.
func CreateWorktree(ctx context.Context, request CreateWorktreeRequest) domain.Result {
	result := gitMutationResult("git.worktree-create", "worktree-create-"+strings.ReplaceAll(request.Branch, "/", "-"))
	if ValidateBranch(request.Branch) != nil || !safeBaseRef(request.Base) {
		return blockGitMutation(result, "git.worktree-request-invalid", "A conventional branch and safe base ref are required.")
	}
	state, err := Inspect(ctx, request.Root)
	if err != nil {
		return failGitMutation(result, "git.repository-invalid", err)
	}
	root, err := canonicalPath(request.Root)
	if err != nil || root != state.Root {
		return blockGitMutation(result, "git.root-mismatch", "The requested path is not the repository root.")
	}
	if state.Dirty {
		return blockGitMutation(result, "git.base-dirty", "The base worktree has uncommitted changes.")
	}
	if state.Diverged {
		return blockGitMutation(result, "git.base-diverged", "The base branch has diverged from its upstream.")
	}
	if state.Detached {
		return blockGitMutation(result, "git.base-detached", "A detached checkout cannot create shared branch work.")
	}
	git := runner{}
	baseHead, err := git.read(ctx, root, "rev-parse", request.Base+"^{commit}")
	if err != nil || !objectIDPatternForMutation(baseHead) {
		return blockGitMutation(result, "git.base-missing", "The requested base commit is unavailable.")
	}
	if _, branchErr := git.read(ctx, root, "rev-parse", "--verify", "refs/heads/"+request.Branch); branchErr == nil {
		return blockGitMutation(result, "git.branch-in-use", "The branch already exists or is checked out.")
	}
	for _, worktree := range state.Worktrees {
		if worktree.Branch == request.Branch {
			return blockGitMutation(result, "git.branch-in-use", "The branch is already checked out in another worktree.")
		}
	}
	target, err := worktreeTarget(ctx, git, root, request.Branch, request.Target)
	if err != nil {
		return blockGitMutation(result, "git.worktree-target-unsafe", err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return failGitMutation(result, "git.worktree-parent-failed", err)
	}
	args := []string{"worktree", "add", "-b", request.Branch, target, request.Base}
	if _, err = git.mutate(ctx, root, mutationWorktree, args...); err != nil {
		removeEmptyDirectory(target)
		return failGitMutation(result, "git.worktree-create-failed", err)
	}
	created, err := Inspect(ctx, target)
	if err != nil || created.Root != target || created.Branch != request.Branch || created.Head != baseHead || created.Detached {
		return blockGitMutation(result, "git.worktree-postcondition", "Created worktree path, branch, or HEAD differs from the reviewed request.")
	}
	result.Status, result.ExitCode, result.Summary = domain.StatusPassed, domain.ExitSuccess, "Isolated worktree was created and verified."
	result.Facts = []domain.Item{
		{Code: "git.worktree-path", Message: target},
		{Code: "git.worktree-branch", Message: request.Branch},
		{Code: "git.worktree-head", Message: baseHead},
	}
	result.Evidence = []domain.Item{{Code: "git.worktree-verified", Message: target, Refs: []string{request.Branch, baseHead}}}
	return result
}

// SyncPinnedSubmodules initializes only explicit paths and verifies each root gitlink postcondition.
func SyncPinnedSubmodules(ctx context.Context, root string, paths []string) domain.Result {
	result := gitMutationResult("git.submodule-sync", "submodule-sync")
	state, err := Inspect(ctx, root)
	if err != nil {
		return failGitMutation(result, "git.repository-invalid", err)
	}
	canonicalRoot, err := canonicalPath(root)
	if err != nil || canonicalRoot != state.Root {
		return blockGitMutation(result, "git.root-mismatch", "Submodules can only be synchronized from the exact orchestration root.")
	}
	paths, pathErr := normalizedMutationPaths(paths)
	if pathErr != nil {
		return blockGitMutation(result, "git.submodule-path-invalid", pathErr.Error())
	}
	if len(paths) == 0 {
		return blockGitMutation(result, "git.submodule-path-required", "At least one explicit submodule path is required.")
	}
	declared, declarationErr := declaredSubmodulePaths(canonicalRoot)
	if declarationErr != nil {
		return blockGitMutation(result, "git.workspace-manifest-invalid", "The canonical workspace manifest could not be verified.")
	}
	for _, path := range paths {
		if !declared[path] {
			return blockGitMutation(result, "git.submodule-not-in-workspace-manifest", "Requested submodule is not declared by the service workspace manifest: "+path)
		}
	}
	byPath := map[string]Submodule{}
	for _, submodule := range state.Submodules {
		byPath[submodule.Path] = submodule
	}
	git := runner{}
	selected := make([]Submodule, 0, len(paths))
	for _, path := range paths {
		submodule, found := byPath[path]
		if !found {
			return blockGitMutation(result, "git.submodule-undeclared", "Requested path is not an explicit root submodule: "+path)
		}
		if submodule.UnsafeURL {
			return blockGitMutation(result, "git.submodule-unsafe-url", "Submodule URL requires explicit security review: "+path)
		}
		effectiveURL, urlErr := git.read(ctx, canonicalRoot, "config", "--get", "submodule."+submodule.Name+".url")
		if urlErr == nil && (unsafeSubmoduleURL(effectiveURL) || effectiveURL != submodule.URL) {
			return blockGitMutation(result, "git.submodule-url-mismatch", "Effective submodule URL differs from the reviewed root declaration: "+path)
		}
		if submodule.Dirty {
			return blockGitMutation(result, "git.submodule-dirty", "Submodule has local changes: "+path)
		}
		if submodule.Initialized && submodule.PointerDiff {
			return blockGitMutation(result, "git.submodule-pointer-mismatch", "Submodule HEAD differs from the root gitlink: "+path)
		}
		selected = append(selected, submodule)
	}
	for _, submodule := range selected {
		if !submodule.Initialized {
			if _, err = git.mutate(ctx, canonicalRoot, mutationSubmodule, "submodule", "update", "--init", "--checkout", "--", submodule.Path); err != nil {
				return failGitMutation(result, "git.submodule-sync-failed", err)
			}
		}
		child, inspectErr := Inspect(ctx, filepath.Join(canonicalRoot, filepath.FromSlash(submodule.Path)))
		if inspectErr != nil || child.Head != submodule.ExpectedSHA || child.Dirty {
			return blockGitMutation(result, "git.submodule-postcondition", "Submodule did not finish at the exact clean root pointer: "+submodule.Path)
		}
		result.Evidence = append(result.Evidence, domain.Item{Code: "git.submodule-pinned", Message: submodule.ExpectedSHA, Refs: []string{submodule.Path}})
	}
	result.Status, result.ExitCode, result.Summary = domain.StatusPassed, domain.ExitSuccess, "Explicit submodules were initialized at their exact root pointers."
	return result
}

func (git runner) mutate(ctx context.Context, directory string, kind mutationKind, args ...string) (string, error) {
	if err := validateMutationArgs(kind, args); err != nil {
		return "", err
	}
	executable, err := exec.LookPath("git")
	if err != nil {
		return "", err
	}
	base := strings.ToLower(filepath.Base(executable))
	if base != "git" && base != "git.exe" {
		return "", fmt.Errorf("resolved executable is not Git")
	}
	hooks, err := os.MkdirTemp("", "service-git-hooks-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(hooks)
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	commandArgs := append([]string{"-c", "core.hooksPath=" + hooks}, args...)
	command := exec.CommandContext(ctx, executable, commandArgs...)
	command.Dir = directory
	command.Env = safeEnvironment()
	var stdout, stderr limitedBuffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		digest := sha256.Sum256([]byte(stderr.String()))
		return "", fmt.Errorf("restricted Git %s failed: %w (stderr sha256:%s)", kind, err, hex.EncodeToString(digest[:]))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func validateMutationArgs(kind mutationKind, args []string) error {
	for _, argument := range args {
		if strings.ContainsAny(argument, "\x00\r\n") {
			return fmt.Errorf("unsafe Git mutation argument")
		}
	}
	switch kind {
	case mutationWorktree:
		if len(args) != 6 || args[0] != "worktree" || args[1] != "add" || args[2] != "-b" || ValidateBranch(args[3]) != nil || !safeBaseRef(args[5]) || !filepath.IsAbs(args[4]) {
			return fmt.Errorf("invalid worktree mutation")
		}
	case mutationSubmodule:
		if len(args) != 6 || args[0] != "submodule" || args[1] != "update" || args[2] != "--init" || args[3] != "--checkout" || args[4] != "--" || !safeMutationPath(args[5]) {
			return fmt.Errorf("invalid submodule mutation")
		}
	default:
		return fmt.Errorf("unsupported Git mutation")
	}
	return nil
}

func worktreeTarget(ctx context.Context, git runner, root, branch, requested string) (string, error) {
	target := requested
	if target == "" {
		placement := filepath.Dir(root)
		if superproject, err := git.read(ctx, root, "rev-parse", "--show-superproject-working-tree"); err == nil && superproject != "" {
			placement = filepath.Dir(superproject)
		}
		target = filepath.Join(placement, ".orchestrator-worktrees", filepath.Base(root), strings.ReplaceAll(branch, "/", "-"))
	}
	target, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if withinPath(root, target) {
		return "", fmt.Errorf("worktree target must stay outside the repository")
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		return "", fmt.Errorf("worktree target already exists")
	}
	ancestor, err := existingAncestor(filepath.Dir(target))
	if err != nil {
		return "", err
	}
	if symlinkOnPath(ancestor, filepath.Dir(target)) {
		return "", fmt.Errorf("worktree target crosses a symlink")
	}
	if containing, inspectErr := git.read(ctx, ancestor, "rev-parse", "--show-toplevel"); inspectErr == nil && containing != "" {
		return "", fmt.Errorf("worktree target is inside another repository")
	}
	resolvedAncestor, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(ancestor, target)
	if err != nil {
		return "", err
	}
	target = filepath.Join(resolvedAncestor, relative)
	return target, nil
}

func canonicalPath(value string) (string, error) {
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}

func existingAncestor(value string) (string, error) {
	current := filepath.Clean(value)
	for {
		if _, err := os.Lstat(current); err == nil {
			return current, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no existing worktree target ancestor")
		}
		current = parent
	}
}

func symlinkOnPath(ancestor, target string) bool {
	relative, err := filepath.Rel(ancestor, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return true
	}
	current := ancestor
	if info, err := os.Lstat(current); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "." || component == "" {
			continue
		}
		current = filepath.Join(current, component)
		if info, err := os.Lstat(current); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

func removeEmptyDirectory(path string) {
	entries, err := os.ReadDir(path)
	if err == nil && len(entries) == 0 {
		_ = os.Remove(path)
	}
}

func safeBaseRef(value string) bool {
	return safeBasePattern.MatchString(value) && !strings.Contains(value, "..") && !strings.Contains(value, "@{") && !strings.HasPrefix(value, "-")
}

func objectIDPatternForMutation(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func safeMutationPath(value string) bool {
	if value == "" || filepath.IsAbs(value) || strings.ContainsAny(value, "\x00\r\n") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	return clean != ".." && !strings.HasPrefix(clean, "../") && clean == filepath.ToSlash(value)
}

func normalizedMutationPaths(values []string) ([]string, error) {
	seen, result := map[string]bool{}, []string{}
	for _, value := range values {
		value = filepath.ToSlash(strings.TrimSpace(value))
		if !safeMutationPath(value) {
			return nil, fmt.Errorf("submodule path is unsafe: %s", value)
		}
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result, nil
}

func declaredSubmodulePaths(root string) (map[string]bool, error) {
	value, err := schema.LoadYAML[map[string]any](filepath.Join(root, ".harness", "workspaces.yaml"))
	if err != nil {
		return nil, err
	}
	entries, ok := value["workspaces"].([]any)
	if !ok {
		return nil, fmt.Errorf("workspaces must be a list")
	}
	result := map[string]bool{}
	for _, raw := range entries {
		entry, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("workspace entry must be an object")
		}
		kind, _ := entry["kind"].(string)
		path, _ := entry["path"].(string)
		path = filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
		if kind == "submodule" {
			if !safeMutationPath(path) {
				return nil, fmt.Errorf("submodule workspace path is unsafe")
			}
			result[path] = true
		}
	}
	return result, nil
}

func withinPath(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func gitMutationResult(command, operationID string) domain.Result {
	return domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: command, OperationID: operationID, Status: domain.StatusFailed, ExitCode: domain.ExitInternal, Summary: "Git mutation failed safely."}
}

func blockGitMutation(result domain.Result, code, message string) domain.Result {
	result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitBlocked, message
	result.Blockers = []domain.Item{{Code: code, Message: message}}
	return result
}

func failGitMutation(result domain.Result, code string, err error) domain.Result {
	if errors.Is(err, context.DeadlineExceeded) {
		result.Status, result.ExitCode = domain.StatusUnknown, domain.ExitUnavailable
	}
	result.Summary = "The restricted Git operation failed without an automatic destructive recovery."
	result.Blockers = []domain.Item{{Code: code, Message: err.Error()}}
	return result
}
