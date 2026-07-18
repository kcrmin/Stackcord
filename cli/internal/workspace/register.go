package workspace

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/gitx"
	"fullstack-orchestrator/cli/internal/operation"
	"go.yaml.in/yaml/v3"
)

var registrationIDPattern = regexp.MustCompile(`^workspace\.[a-z0-9]+(?:[.-][a-z0-9]+)*$`)

// RegistrationRequest adds one recoverable service workspace boundary.
type RegistrationRequest struct {
	Root, ID, Kind, Path, Repository, Remote, RootRemote, Initialize string
	Responsibilities, Dependencies, Consumers                        []string
}

// PlanRegistration updates canonical topology without replacing existing workspace content.
func PlanRegistration(ctx context.Context, request RegistrationRequest) (operation.Plan, error) {
	root, err := filepath.Abs(request.Root)
	if err != nil {
		return operation.Plan{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return operation.Plan{}, err
	}
	plan := operation.Plan{ID: "workspace-register-" + strings.ReplaceAll(request.ID, ".", "-"), Root: root}
	add := func(code, message string, refs ...string) {
		plan.Blockers = append(plan.Blockers, domain.Item{Code: code, Message: message, Refs: refs})
	}
	manifest, err := Load(root)
	if err != nil {
		return operation.Plan{}, err
	}
	request.Path, err = safeRelative(request.Path)
	if err != nil || request.Path == "." || !registrationIDPattern.MatchString(request.ID) {
		add("workspace.registration-invalid", "Workspace needs a stable ID and a safe non-root path.", request.ID, request.Path)
		return plan, nil
	}
	if request.Kind != "directory" && request.Kind != "submodule" {
		add("workspace.kind-invalid", "Registered workspace kind must be directory or submodule.", request.Kind)
	}
	request.Responsibilities = normalizedWorkspaceValues(request.Responsibilities)
	request.Dependencies = normalizedWorkspaceValues(request.Dependencies)
	request.Consumers = normalizedWorkspaceValues(request.Consumers)
	if len(request.Responsibilities) == 0 {
		add("workspace.responsibility-required", "Workspace needs at least one responsibility.")
	}
	for _, entry := range manifest.Workspaces {
		if entry.ID == request.ID {
			add("workspace.id-duplicate", "Workspace ID is already registered.", request.ID)
		}
		if filepath.ToSlash(filepath.Clean(filepath.FromSlash(entry.Path))) == request.Path {
			add("workspace.path-duplicate", "Workspace path is already registered.", request.Path)
		}
	}
	target := filepath.Join(root, filepath.FromSlash(request.Path))
	if info, statErr := os.Lstat(target); statErr == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
		add("workspace.path-unsafe", "Workspace path must be a directory without symlink indirection.", request.Path)
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return operation.Plan{}, statErr
	}

	if request.Repository == "" {
		request.Repository = "repository." + strings.TrimPrefix(request.ID, "workspace.")
	}
	if request.Kind == "submodule" {
		state, inspectErr := gitx.Inspect(ctx, root)
		if inspectErr != nil || state.Root != root {
			add("workspace.submodule-unavailable", "Submodule workspace requires an exact inspectable orchestration root.", request.Path)
		} else {
			found := false
			for _, submodule := range state.Submodules {
				if submodule.Path == request.Path {
					found = true
					if !submodule.Initialized || submodule.Dirty || submodule.PointerDiff || submodule.UnsafeURL {
						add("workspace.submodule-unsafe", "Submodule must be initialized, clean, and at the root pointer.", request.Path)
					}
					if request.Remote == "" {
						request.Remote = submodule.URL
					}
					if request.Remote != submodule.URL || !gitx.SafeRemoteURL(request.Remote) {
						add("workspace.remote-mismatch", "Workspace remote differs from the reviewed submodule declaration.", request.Path)
					}
				}
			}
			if !found {
				add("workspace.submodule-missing", "Workspace path is not an initialized root submodule.", request.Path)
			}
		}
		if request.RootRemote == "" {
			request.RootRemote = manifest.RootRemote
		}
		if !gitx.SafeRemoteURL(request.RootRemote) {
			add("workspace.root-remote-required", "A credential-free orchestration root remote is required for child recovery.")
		}
	}
	if len(plan.Blockers) > 0 {
		return plan, nil
	}

	entry := Entry{ID: request.ID, Kind: request.Kind, Path: request.Path, Repository: request.Repository, Remote: request.Remote, Responsibilities: request.Responsibilities, Dependencies: request.Dependencies}
	manifest.Workspaces = append(manifest.Workspaces, entry)
	byID := map[string]*Entry{}
	for index := range manifest.Workspaces {
		byID[manifest.Workspaces[index].ID] = &manifest.Workspaces[index]
	}
	for _, consumer := range request.Consumers {
		current, exists := byID[consumer]
		if !exists {
			add("workspace.consumer-missing", "UI consumer workspace is not registered.", consumer)
			continue
		}
		current.Dependencies = normalizedWorkspaceValues(append(current.Dependencies, request.ID))
	}
	for index := range manifest.Workspaces {
		manifest.Workspaces[index].Responsibilities = normalizedWorkspaceValues(manifest.Workspaces[index].Responsibilities)
		manifest.Workspaces[index].Dependencies = normalizedWorkspaceValues(manifest.Workspaces[index].Dependencies)
	}
	sort.Slice(manifest.Workspaces, func(i, j int) bool { return manifest.Workspaces[i].ID < manifest.Workspaces[j].ID })
	if err := validateManifest(manifest); err != nil {
		add("workspace.manifest-invalid", err.Error())
	}
	if cycle := workspaceDependencyCycle(manifest); len(cycle) > 0 {
		add("workspace.dependency-cycle", "Workspace dependencies contain a cycle.", cycle...)
	}
	if len(plan.Blockers) > 0 {
		return plan, nil
	}
	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		return operation.Plan{}, err
	}
	plan.Files = append(plan.Files, operation.FileChange{Path: ".harness/workspaces.yaml", Content: manifestData, Mode: 0o644})

	if request.Kind == "submodule" {
		bridge := Bridge{SchemaVersion: 1, ProjectID: manifest.ProjectID, RootRemote: request.RootRemote, WorkspaceID: request.ID, Discovery: "git-superproject", CommandsPath: ".harness/commands.yaml"}
		bridgePath := filepath.Join(target, ".harness", "bridge.yaml")
		if _, statErr := os.Lstat(bridgePath); statErr == nil {
			current, loadErr := loadBridge(bridgePath)
			if loadErr != nil || current != bridge {
				add("workspace.bridge-conflict", "Existing child bridge differs from the requested service identity.", request.Path)
			}
		} else if !os.IsNotExist(statErr) {
			return operation.Plan{}, statErr
		} else {
			bridgeData, marshalErr := yaml.Marshal(bridge)
			if marshalErr != nil {
				return operation.Plan{}, marshalErr
			}
			plan.Files = append(plan.Files, operation.FileChange{Path: filepath.ToSlash(filepath.Join(request.Path, ".harness", "bridge.yaml")), Content: bridgeData, Mode: 0o644})
		}
	}
	if request.Initialize == "ui" {
		for path, content := range uiWorkspaceStarter(request.Path, request.ID) {
			absolute := filepath.Join(root, filepath.FromSlash(path))
			if _, statErr := os.Lstat(absolute); os.IsNotExist(statErr) {
				plan.Files = append(plan.Files, operation.FileChange{Path: path, Content: []byte(content), Mode: 0o644})
			} else if statErr != nil {
				return operation.Plan{}, statErr
			}
		}
	} else if request.Initialize != "" {
		add("workspace.initializer-invalid", "Only the framework-neutral ui initializer is supported.", request.Initialize)
	}
	if len(plan.Blockers) > 0 {
		plan.Files = nil
		return plan, nil
	}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return plan, err
}

func normalizedWorkspaceValues(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func workspaceDependencyCycle(manifest Manifest) []string {
	dependencies := map[string][]string{}
	for _, entry := range manifest.Workspaces {
		dependencies[entry.ID] = entry.Dependencies
	}
	state := map[string]int{}
	stack := []string{}
	var visit func(string) []string
	visit = func(id string) []string {
		if state[id] == 1 {
			for index, value := range stack {
				if value == id {
					return append(append([]string(nil), stack[index:]...), id)
				}
			}
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		stack = append(stack, id)
		for _, dependency := range dependencies[id] {
			if cycle := visit(dependency); len(cycle) > 0 {
				return cycle
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = 2
		return nil
	}
	ids := make([]string, 0, len(dependencies))
	for id := range dependencies {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if cycle := visit(id); len(cycle) > 0 {
			return cycle
		}
	}
	return nil
}

func uiWorkspaceStarter(root, id string) map[string]string {
	return map[string]string{
		filepath.ToSlash(filepath.Join(root, "README.md")):              "# UI baseline\n\nThis workspace owns editable product flows, screens, states, tokens, accessibility expectations, approved assets, and source provenance. Production frontend code belongs in the frontend workspace.\n",
		filepath.ToSlash(filepath.Join(root, "AGENTS.md")):              "# UI workspace\n\nPreserve product intent and source provenance. Record loading, empty, error, success, permission, responsive, destructive-action, and accessibility behavior before frontend implementation.\n",
		filepath.ToSlash(filepath.Join(root, "coverage", "index.yaml")): "schema_version: 1\nworkspace_id: " + id + "\nroles: []\njourneys: []\nflows: []\nopen_questions: []\n",
	}
}
