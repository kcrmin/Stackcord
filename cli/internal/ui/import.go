package ui

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/operation"
)

const maxImportBytes = 64 << 20

var secretPattern = regexp.MustCompile(`(?i)(api[_-]?token|password|secret|private[_-]?key)\s*[:=]\s*[A-Za-z0-9_\-]{16,}`)

// Source describes an external UI artifact and its authority.
type Source struct{ Root, Archive, ID, Kind, Authority, Version, License string }

// ImportPlan validates and quarantines an archive before any canonical UI change.
func ImportPlan(source Source) (operation.Plan, error) {
	if source.ID == "" || !map[string]bool{"reference": true, "seed": true, "canonical": true}[source.Authority] {
		return operation.Plan{}, fmt.Errorf("source ID and authority reference|seed|canonical are required")
	}
	reader, err := zip.OpenReader(source.Archive)
	if err != nil {
		return operation.Plan{}, err
	}
	defer reader.Close()
	licenseFound := source.License != ""
	total := int64(0)
	files := []operation.FileChange{}
	hash := sha256.New()
	sort.Slice(reader.File, func(i, j int) bool { return reader.File[i].Name < reader.File[j].Name })
	for _, entry := range reader.File {
		name := filepath.ToSlash(filepath.Clean(entry.Name))
		if name == ".." || strings.HasPrefix(name, "../") || filepath.IsAbs(entry.Name) {
			return operation.Plan{}, fmt.Errorf("archive path escapes quarantine: %s", entry.Name)
		}
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
		total += int64(entry.UncompressedSize64)
		if total > maxImportBytes {
			return operation.Plan{}, fmt.Errorf("archive exceeds %d bytes", maxImportBytes)
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
		if secretPattern.Match(data) {
			return operation.Plan{}, fmt.Errorf("secret-like content detected in %s", name)
		}
		if strings.HasPrefix(strings.ToUpper(filepath.Base(name)), "LICENSE") {
			licenseFound = true
		}
		_, _ = hash.Write([]byte(name))
		_, _ = hash.Write(data)
		files = append(files, operation.FileChange{Path: filepath.ToSlash(filepath.Join(".harness", "local", "imports", source.ID, name)), Content: data, Mode: 0o600})
	}
	if !licenseFound {
		return operation.Plan{}, fmt.Errorf("UI source license is required")
	}
	manifest := fmt.Sprintf("schema_version: 1\nid: %s\nkind: %s\nauthority: %s\nversion: %s\nhash: sha256:%s\n", source.ID, source.Kind, source.Authority, source.Version, hex.EncodeToString(hash.Sum(nil)))
	files = append(files, operation.FileChange{Path: filepath.ToSlash(filepath.Join(".harness", "local", "imports", source.ID, "source.yaml")), Content: []byte(manifest), Mode: 0o600})
	plan := operation.Plan{ID: "ui-import-" + strings.ReplaceAll(source.ID, ".", "-"), Root: source.Root, Files: files}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return plan, err
}
