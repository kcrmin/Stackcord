package ui

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/schema"
	"go.yaml.in/yaml/v3"
)

const maxImportBytes = 64 << 20
const maxImportEntries = 4096

var secretPattern = regexp.MustCompile(`(?i)(api[_-]?token|password|secret|private[_-]?key)\s*[:=]\s*[A-Za-z0-9_\-]{16,}`)
var sourceIDPattern = regexp.MustCompile(`^ui\.[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
var windowsAbsolutePath = regexp.MustCompile(`^[A-Za-z]:/`)

// Source describes an external UI artifact and its authority.
type Source struct {
	Root, Archive, ID, Kind, Authority, Version, License string
	MappedRefs, Consumers                                []string
	FetchedAt                                            time.Time
	BaselineFingerprint                                  string
}

// Registration is committed source identity; quarantined archive content remains local.
type Registration struct {
	SchemaVersion       int       `json:"schema_version" yaml:"schema_version"`
	ID                  string    `json:"id" yaml:"id"`
	Kind                string    `json:"kind" yaml:"kind"`
	Authority           string    `json:"authority" yaml:"authority"`
	Source              string    `json:"source" yaml:"source"`
	SourceVersion       string    `json:"source_version" yaml:"source_version"`
	License             string    `json:"license" yaml:"license"`
	ContentHash         string    `json:"content_hash" yaml:"content_hash"`
	BaselineFingerprint string    `json:"baseline_fingerprint" yaml:"baseline_fingerprint"`
	FetchedAt           time.Time `json:"fetched_at" yaml:"fetched_at"`
	MappedRefs          []string  `json:"mapped_refs" yaml:"mapped_refs"`
	Consumers           []string  `json:"consumers" yaml:"consumers"`
}

// Register validates one archive and returns both stable identity and exact quarantine plan.
func Register(source Source) (Registration, operation.Plan, error) {
	plan, err := registerPlan(source)
	if err != nil {
		return Registration{}, operation.Plan{}, err
	}
	for _, file := range plan.Files {
		if filepath.ToSlash(file.Path) == filepath.ToSlash(filepath.Join(".harness", "sources", "ui", source.ID+".yaml")) {
			registration, decodeErr := schema.DecodeYAML[Registration](file.Content)
			return registration, plan, decodeErr
		}
	}
	return Registration{}, operation.Plan{}, fmt.Errorf("UI registration was not produced")
}

// ImportPlan validates and quarantines an archive before any canonical UI change.
func ImportPlan(source Source) (operation.Plan, error) {
	_, plan, err := Register(source)
	return plan, err
}

func registerPlan(source Source) (operation.Plan, error) {
	if !sourceIDPattern.MatchString(source.ID) || !map[string]bool{"reference": true, "seed": true, "canonical": true}[source.Authority] {
		return operation.Plan{}, fmt.Errorf("source ID and authority reference|seed|canonical are required")
	}
	reader, err := zip.OpenReader(source.Archive)
	if err != nil {
		return operation.Plan{}, err
	}
	defer reader.Close()
	if len(reader.File) > maxImportEntries {
		return operation.Plan{}, fmt.Errorf("archive exceeds %d entries", maxImportEntries)
	}
	licenseFound := source.License != ""
	licenseValue := source.License
	total := int64(0)
	files := []operation.FileChange{}
	seenPaths := map[string]bool{}
	hash := sha256.New()
	sort.Slice(reader.File, func(i, j int) bool { return reader.File[i].Name < reader.File[j].Name })
	for _, entry := range reader.File {
		rawName := strings.ReplaceAll(entry.Name, "\\", "/")
		name := path.Clean(rawName)
		if name == "." || name == ".." || strings.HasPrefix(name, "../") || strings.HasPrefix(rawName, "/") || windowsAbsolutePath.MatchString(rawName) {
			return operation.Plan{}, fmt.Errorf("archive path escapes quarantine: %s", entry.Name)
		}
		pathKey := strings.ToLower(name)
		if seenPaths[pathKey] {
			return operation.Plan{}, fmt.Errorf("duplicate normalized archive path: %s", name)
		}
		seenPaths[pathKey] = true
		if entry.Mode()&os.ModeSymlink != 0 {
			return operation.Plan{}, fmt.Errorf("symlinks are not allowed: %s", entry.Name)
		}
		extension := strings.ToLower(filepath.Ext(name))
		if map[string]bool{".sh": true, ".bat": true, ".cmd": true, ".ps1": true, ".exe": true, ".dll": true, ".dylib": true}[extension] {
			return operation.Plan{}, fmt.Errorf("executable content is not allowed: %s", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		stream, openErr := entry.Open()
		if openErr != nil {
			return operation.Plan{}, openErr
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, maxImportBytes+1))
		_ = stream.Close()
		if readErr != nil || len(data) > maxImportBytes {
			return operation.Plan{}, fmt.Errorf("read archive entry %s", name)
		}
		total += int64(len(data))
		if total > maxImportBytes {
			return operation.Plan{}, fmt.Errorf("archive exceeds %d bytes", maxImportBytes)
		}
		if secretPattern.Match(data) {
			return operation.Plan{}, fmt.Errorf("secret-like content detected in %s", name)
		}
		if strings.HasPrefix(strings.ToUpper(filepath.Base(name)), "LICENSE") {
			licenseFound = true
			if licenseValue == "" {
				licenseValue = "archive:" + name
			}
		}
		_, _ = hash.Write([]byte(name))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(data)
		_, _ = hash.Write([]byte{0})
		files = append(files, operation.FileChange{Path: filepath.ToSlash(filepath.Join(".harness", "local", "imports", source.ID, name)), Content: data, Mode: 0o600})
	}
	if !licenseFound {
		return operation.Plan{}, fmt.Errorf("UI source license is required")
	}
	if source.Kind == "" {
		source.Kind = "mockup"
	}
	if source.Version == "" {
		source.Version = "unversioned"
	}
	if source.FetchedAt.IsZero() {
		if info, statErr := os.Stat(source.Archive); statErr == nil {
			source.FetchedAt = info.ModTime().UTC()
		} else {
			source.FetchedAt = time.Now().UTC()
		}
	}
	contentHash := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if source.BaselineFingerprint == "" {
		source.BaselineFingerprint = contentHash
	}
	registration := Registration{
		SchemaVersion: 1, ID: source.ID, Kind: source.Kind, Authority: source.Authority, Source: "archive", SourceVersion: source.Version,
		License: licenseValue, ContentHash: contentHash, BaselineFingerprint: source.BaselineFingerprint, FetchedAt: source.FetchedAt.UTC(),
		MappedRefs: normalizedUIRefs(source.MappedRefs), Consumers: normalizedUIRefs(source.Consumers),
	}
	if issues := schema.Validate("external-source", registration); len(issues) > 0 {
		return operation.Plan{}, fmt.Errorf("validate UI registration: %s", issues[0].Message)
	}
	manifest, err := yaml.Marshal(registration)
	if err != nil {
		return operation.Plan{}, fmt.Errorf("encode UI source manifest: %w", err)
	}
	files = append(files, operation.FileChange{Path: filepath.ToSlash(filepath.Join(".harness", "local", "imports", source.ID, "source.yaml")), Content: manifest, Mode: 0o600})
	files = append(files, operation.FileChange{Path: filepath.ToSlash(filepath.Join(".harness", "sources", "ui", source.ID+".yaml")), Content: manifest, Mode: 0o644})
	plan := operation.Plan{ID: "ui-import-" + strings.ReplaceAll(source.ID, ".", "-") + "-" + strings.TrimPrefix(contentHash, "sha256:")[:12], Root: source.Root, Files: files}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return plan, err
}

func normalizedUIRefs(values []string) []string {
	result, seen := []string{}, map[string]bool{}
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
