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

	"fullstack-orchestrator/cli/internal/operation"
	"go.yaml.in/yaml/v3"
)

const maxImportBytes = 64 << 20

var secretPattern = regexp.MustCompile(`(?i)(api[_-]?token|password|secret|private[_-]?key)\s*[:=]\s*[A-Za-z0-9_\-]{16,}`)
var sourceIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var windowsAbsolutePath = regexp.MustCompile(`^[A-Za-z]:/`)

// Source describes an external UI artifact and its authority.
type Source struct{ Root, Archive, ID, Kind, Authority, Version, License string }

// ImportPlan validates and quarantines an archive before any canonical UI change.
func ImportPlan(source Source) (operation.Plan, error) {
	if !sourceIDPattern.MatchString(source.ID) || !map[string]bool{"reference": true, "seed": true, "canonical": true}[source.Authority] {
		return operation.Plan{}, fmt.Errorf("source ID and authority reference|seed|canonical are required")
	}
	reader, err := zip.OpenReader(source.Archive)
	if err != nil {
		return operation.Plan{}, err
	}
	defer reader.Close()
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
		_, _ = hash.Write(data)
		files = append(files, operation.FileChange{Path: filepath.ToSlash(filepath.Join(".harness", "local", "imports", source.ID, name)), Content: data, Mode: 0o600})
	}
	if !licenseFound {
		return operation.Plan{}, fmt.Errorf("UI source license is required")
	}
	manifest, err := yaml.Marshal(struct {
		SchemaVersion int    `yaml:"schema_version"`
		ID            string `yaml:"id"`
		Kind          string `yaml:"kind"`
		Authority     string `yaml:"authority"`
		Version       string `yaml:"version"`
		License       string `yaml:"license"`
		Hash          string `yaml:"hash"`
	}{1, source.ID, source.Kind, source.Authority, source.Version, licenseValue, "sha256:" + hex.EncodeToString(hash.Sum(nil))})
	if err != nil {
		return operation.Plan{}, fmt.Errorf("encode UI source manifest: %w", err)
	}
	files = append(files, operation.FileChange{Path: filepath.ToSlash(filepath.Join(".harness", "local", "imports", source.ID, "source.yaml")), Content: manifest, Mode: 0o600})
	plan := operation.Plan{ID: "ui-import-" + strings.ReplaceAll(source.ID, ".", "-"), Root: source.Root, Files: files}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return plan, err
}
