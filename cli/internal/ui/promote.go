package ui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/workspace"
	"go.yaml.in/yaml/v3"
)

// PromotionRequest moves an already-inspected source into an editable UI workspace.
type PromotionRequest struct {
	Root, SourceID, WorkspaceID, Mode string
	Paths                             []string
}

// PromotionRecord preserves why and how reviewed material entered the UI workspace.
type PromotionRecord struct {
	SchemaVersion int      `json:"schema_version" yaml:"schema_version"`
	SourceID      string   `json:"source_id" yaml:"source_id"`
	WorkspaceID   string   `json:"workspace_id" yaml:"workspace_id"`
	Authority     string   `json:"authority" yaml:"authority"`
	SourceVersion string   `json:"source_version" yaml:"source_version"`
	License       string   `json:"license" yaml:"license"`
	ContentHash   string   `json:"content_hash" yaml:"content_hash"`
	Mode          string   `json:"mode" yaml:"mode"`
	Paths         []string `json:"paths" yaml:"paths"`
}

// Promote returns an atomic reviewed copy plan; it never invents canonical UI structure.
func Promote(request PromotionRequest) (operation.Plan, error) {
	plan := operation.Plan{ID: "ui-promote-" + strings.ReplaceAll(request.SourceID, ".", "-"), Root: request.Root}
	root, err := filepath.Abs(request.Root)
	if err != nil {
		return operation.Plan{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return operation.Plan{}, err
	}
	plan.Root = root
	add := func(code, message string, refs ...string) {
		plan.Blockers = append(plan.Blockers, domain.Item{Code: code, Message: message, Refs: refs})
	}
	if request.Mode != "whole" && request.Mode != "selected" && request.Mode != "reference-only" {
		add("ui.promotion-mode-invalid", "Promotion mode must be whole, selected, or reference-only.", request.Mode)
		return plan, nil
	}
	registration, err := LoadRegistration(root, request.SourceID)
	if err != nil {
		return operation.Plan{}, err
	}
	manifest, err := workspace.Load(root)
	if err != nil {
		return operation.Plan{}, err
	}
	entry, found := uiWorkspace(manifest, request.WorkspaceID)
	if !found {
		add("ui.workspace-invalid", "Promotion target must be a declared UI baseline workspace.", request.WorkspaceID)
		return plan, nil
	}
	workspaceRoot := filepath.Join(root, filepath.FromSlash(entry.Path))
	info, err := os.Lstat(workspaceRoot)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		add("ui.workspace-unsafe", "UI workspace must be an existing non-symlink directory.", request.WorkspaceID)
		return plan, nil
	}
	quarantine := filepath.Join(root, ".harness", "local", "imports", request.SourceID)
	files, contentHash, err := inspectedSourceFiles(quarantine)
	if err != nil {
		return operation.Plan{}, err
	}
	if contentHash != registration.ContentHash {
		add("ui.quarantine-stale", "Inspected source content no longer matches its registered hash.", request.SourceID)
		return plan, nil
	}
	if request.Mode == "reference-only" {
		plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
		return plan, err
	}

	selected := map[string]bool{}
	if request.Mode == "whole" {
		for path := range files {
			selected[path] = true
		}
	} else {
		for _, raw := range request.Paths {
			path := filepath.ToSlash(strings.TrimSpace(raw))
			if !safePromotionPath(path) {
				add("ui.promotion-path-invalid", "Selected UI source path is unsafe.", raw)
				continue
			}
			if _, exists := files[path]; !exists {
				add("ui.promotion-path-missing", "Selected UI source file is not present in the inspected import.", path)
				continue
			}
			selected[path] = true
		}
		if len(request.Paths) == 0 {
			add("ui.promotion-path-required", "Selected promotion requires at least one exact source file.")
		}
	}
	if len(plan.Blockers) > 0 {
		return plan, nil
	}
	paths := make([]string, 0, len(selected))
	for path := range selected {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	destinationRoot := filepath.ToSlash(filepath.Join(entry.Path, "sources", request.SourceID))
	for _, path := range paths {
		destination := filepath.ToSlash(filepath.Join(destinationRoot, filepath.FromSlash(path)))
		absolute := filepath.Join(root, filepath.FromSlash(destination))
		if info, statErr := os.Lstat(absolute); statErr == nil && (info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular()) {
			add("ui.promotion-target-unsafe", "Promotion target must be a regular non-symlink file.", destination)
			continue
		} else if statErr != nil && !os.IsNotExist(statErr) {
			add("ui.promotion-target-unsafe", "Promotion target cannot be inspected safely.", destination)
			continue
		}
		if current, readErr := os.ReadFile(absolute); readErr == nil {
			if string(current) != string(files[path]) {
				add("ui.promotion-overwrite", "Promotion will not overwrite edited UI workspace files.", destination)
			}
			continue
		} else if !os.IsNotExist(readErr) {
			add("ui.promotion-target-unsafe", "Promotion target cannot be read safely.", destination)
			continue
		}
		plan.Files = append(plan.Files, operation.FileChange{Path: destination, Content: files[path], Mode: 0o644})
	}
	record := PromotionRecord{SchemaVersion: 1, SourceID: registration.ID, WorkspaceID: request.WorkspaceID, Authority: registration.Authority, SourceVersion: registration.SourceVersion, License: registration.License, ContentHash: registration.ContentHash, Mode: request.Mode, Paths: paths}
	recordData, err := yaml.Marshal(record)
	if err != nil {
		return operation.Plan{}, err
	}
	recordPath := filepath.ToSlash(filepath.Join(destinationRoot, "promotion.yaml"))
	recordAbsolute := filepath.Join(root, filepath.FromSlash(recordPath))
	if info, statErr := os.Lstat(recordAbsolute); statErr == nil && (info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular()) {
		add("ui.promotion-target-unsafe", "Promotion provenance must be a regular non-symlink file.", recordPath)
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return operation.Plan{}, statErr
	} else if current, readErr := os.ReadFile(recordAbsolute); readErr == nil && string(current) != string(recordData) {
		add("ui.promotion-overwrite", "Existing promotion provenance differs from the reviewed request.", recordPath)
	} else if os.IsNotExist(readErr) {
		plan.Files = append(plan.Files, operation.FileChange{Path: recordPath, Content: recordData, Mode: 0o644})
	} else if readErr != nil {
		return operation.Plan{}, readErr
	}
	if len(plan.Blockers) > 0 {
		plan.Files = nil
		return plan, nil
	}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return plan, err
}

func uiWorkspace(manifest workspace.Manifest, id string) (workspace.Entry, bool) {
	for _, entry := range manifest.Workspaces {
		if entry.ID != id {
			continue
		}
		for _, responsibility := range entry.Responsibilities {
			if responsibility == "ui-baseline" {
				return entry, true
			}
		}
	}
	return workspace.Entry{}, false
}

func inspectedSourceFiles(root string) (map[string][]byte, string, error) {
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("inspected UI source cannot contain symlinks")
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == "source.yaml" {
			return nil
		}
		if !safePromotionPath(relative) {
			return fmt.Errorf("inspected UI source path is unsafe: %s", relative)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[relative] = data
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, path := range paths {
		_, _ = hash.Write([]byte(path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(files[path])
		_, _ = hash.Write([]byte{0})
	}
	return files, "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func safePromotionPath(value string) bool {
	if value == "" || filepath.IsAbs(value) || strings.ContainsAny(value, "\x00\r\n") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	return clean == value && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}
