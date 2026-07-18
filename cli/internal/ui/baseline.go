package ui

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/domain"
)

var baselineIDPattern = regexp.MustCompile(`^ui\.baseline\.[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
var baselineWorkspacePattern = regexp.MustCompile(`^workspace\.[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
var baselineStableIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z0-9][a-z0-9-]*)+$`)
var baselineObjectIDPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var baselineDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var scpRemotePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+@[A-Za-z0-9.-]+:[A-Za-z0-9._/-]+$`)

// Baseline binds approved UI meaning to an exact recoverable workspace commit.
type Baseline struct {
	SchemaVersion   int      `json:"schema_version" yaml:"schema_version"`
	ID              string   `json:"id" yaml:"id"`
	WorkspaceID     string   `json:"workspace_id" yaml:"workspace_id"`
	WorkspaceCommit string   `json:"workspace_commit" yaml:"workspace_commit"`
	WorkspaceRemote string   `json:"workspace_remote" yaml:"workspace_remote"`
	SourceIDs       []string `json:"source_ids" yaml:"source_ids"`
	MappedRefs      []string `json:"mapped_refs" yaml:"mapped_refs"`
	Consumers       []string `json:"consumers" yaml:"consumers"`
	Fingerprint     string   `json:"fingerprint" yaml:"fingerprint"`
}

// ValidateBaseline checks intrinsic identity before repository state is inspected.
func ValidateBaseline(baseline Baseline) []domain.Item {
	issues := []domain.Item{}
	add := func(code, message string, refs ...string) {
		issues = append(issues, domain.Item{Code: code, Message: message, Refs: refs})
	}
	if baseline.SchemaVersion != 1 {
		add("ui.baseline-schema-invalid", "UI baseline schema_version must be 1.")
	}
	if !baselineIDPattern.MatchString(baseline.ID) {
		add("ui.baseline-id-invalid", "UI baseline ID is invalid.", baseline.ID)
	}
	if !baselineWorkspacePattern.MatchString(baseline.WorkspaceID) {
		add("ui.baseline-workspace-invalid", "UI baseline workspace ID is invalid.", baseline.WorkspaceID)
	}
	if !baselineObjectIDPattern.MatchString(baseline.WorkspaceCommit) {
		add("ui.baseline-commit-invalid", "UI baseline commit must be an exact Git object ID.", baseline.WorkspaceCommit)
	}
	if !safeBaselineRemote(baseline.WorkspaceRemote) {
		add("ui.baseline-remote-unsafe", "UI baseline remote must be a credential-free HTTPS or SSH Git URL.")
	}
	for _, group := range [][]string{baseline.SourceIDs, baseline.MappedRefs, baseline.Consumers} {
		seen := map[string]bool{}
		for _, value := range group {
			if !baselineStableIDPattern.MatchString(value) {
				add("ui.baseline-ref-invalid", "UI baseline references must be stable IDs.", value)
			}
			if seen[value] {
				add("ui.baseline-ref-duplicate", "UI baseline reference sets must not contain duplicates.", value)
			}
			seen[value] = true
		}
	}
	if len(baseline.MappedRefs) == 0 || len(baseline.Consumers) == 0 {
		add("ui.baseline-scope-required", "UI baseline needs mapped UI meaning and at least one consumer.")
	}
	want := BaselineFingerprint(baseline)
	if baseline.Fingerprint != "" && (!baselineDigestPattern.MatchString(baseline.Fingerprint) || baseline.Fingerprint != want) {
		add("ui.baseline-fingerprint-mismatch", "UI baseline fingerprint differs from normalized identity.", baseline.Fingerprint, want)
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Code == issues[j].Code {
			return strings.Join(issues[i].Refs, "\x00") < strings.Join(issues[j].Refs, "\x00")
		}
		return issues[i].Code < issues[j].Code
	})
	return issues
}

// BaselineFingerprint hashes normalized baseline meaning without its fingerprint field.
func BaselineFingerprint(baseline Baseline) string {
	baseline.SourceIDs = normalizedBaselineSet(baseline.SourceIDs)
	baseline.MappedRefs = normalizedBaselineSet(baseline.MappedRefs)
	baseline.Consumers = normalizedBaselineSet(baseline.Consumers)
	baseline.Fingerprint = ""
	data, _ := json.Marshal(baseline)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func normalizedBaselineSet(values []string) []string {
	result := append([]string(nil), values...)
	for index := range result {
		result[index] = strings.TrimSpace(result[index])
	}
	sort.Strings(result)
	return result
}

func safeBaselineRemote(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\x00\r\n") || strings.HasPrefix(value, "-") {
		return false
	}
	if scpRemotePattern.MatchString(value) {
		return !strings.Contains(value, "..")
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "ssh") || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return false
	}
	return parsed.Path != "" && !strings.Contains(parsed.Path, "..")
}
