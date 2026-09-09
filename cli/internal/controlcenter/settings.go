// Package controlcenter shares local UI preferences and reviewed project proposals across hosts.
package controlcenter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/kcrmin/Stackcord/cli/internal/convention"
	github "github.com/kcrmin/Stackcord/cli/internal/github"
	"github.com/kcrmin/Stackcord/cli/internal/governance"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"go.yaml.in/yaml/v3"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const personalPath = ".harness/local/control-center.json"

var statePaths = []string{personalPath, ".harness/governance.yaml", ".harness/git-conventions.yaml"}

type Settings struct {
	Language         string                `json:"language"`
	SecurityMode     string                `json:"securityMode"`
	Repository       string                `json:"repository"`
	Admins           []string              `json:"admins"`
	Delegates        []governance.Delegate `json:"delegates"`
	BranchPattern    string                `json:"branchPattern"`
	CommitConvention string                `json:"commitConvention"`
	UIPreference     string                `json:"uiPreference"`
	TargetBranch     string                `json:"targetBranch"`
}
type personal struct {
	Language     string `json:"language"`
	UIPreference string `json:"uiPreference"`
}

func revision(root string) (string, error) {
	p := operation.Plan{Root: root}
	for _, path := range statePaths {
		p.Files = append(p.Files, operation.FileChange{Path: path})
	}
	base, err := operation.StateFingerprint(p)
	if err != nil {
		return "", err
	}
	remote, _ := exec.Command("git", "-C", root, "remote", "get-url", "origin").Output()
	sum := sha256.Sum256(append([]byte(base), remote...))
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
func ReadSettings(root string) (Settings, string, error) {
	s := Settings{Language: "en", SecurityMode: "strong", Admins: []string{}, Delegates: []governance.Delegate{}, BranchPattern: "{type}/{description}", CommitConvention: "{type}: {subject}", UIPreference: "unset"}
	rev, err := revision(root)
	if err != nil {
		return s, "", err
	}
	data, err := os.ReadFile(filepath.Join(root, personalPath))
	if err == nil {
		var p personal
		if err = json.Unmarshal(data, &p); err != nil {
			return s, rev, err
		}
		s.Language = p.Language
		s.UIPreference = p.UIPreference
	} else if !os.IsNotExist(err) {
		return s, rev, err
	}
	if _, err = os.Stat(filepath.Join(root, ".harness/governance.yaml")); err == nil {
		p, e := governance.LoadPolicy(root)
		if e != nil {
			return s, rev, e
		}
		s.SecurityMode = governance.SecurityMode(p)
		s.Repository = p.Repository
		s.Delegates = p.Delegates
		for _, a := range p.ProductAuthorities {
			s.Admins = append(s.Admins, strings.TrimPrefix(a, "user:"))
		}
	} else if !os.IsNotExist(err) {
		return s, rev, err
	}
	cfg, ok, err := convention.Load(root)
	if err != nil {
		return s, rev, err
	}
	if ok {
		if cfg.Branch != nil {
			s.BranchPattern = cfg.Branch.Format
		}
		if cfg.Commit != nil {
			s.CommitConvention = cfg.Commit.TitleFormat
		}
	}
	return s, rev, nil
}
func validateSettings(s Settings) error {
	if s.Language != "en" && s.Language != "ko" {
		return fmt.Errorf("language must be en or ko")
	}
	if s.UIPreference != "enabled" && s.UIPreference != "declined" && s.UIPreference != "unset" {
		return fmt.Errorf("invalid UI preference")
	}
	if s.SecurityMode != "strong" && s.SecurityMode != "medium" && s.SecurityMode != "weak" {
		return fmt.Errorf("invalid security mode")
	}
	if _, e := github.New(s.Repository, nil); e != nil {
		return e
	}
	if s.SecurityMode != "weak" && len(s.Admins) == 0 {
		return fmt.Errorf("register at least one administrator")
	}
	branch := strings.NewReplacer("{type}", "feature", "{description}", "sample", "{issue}", "7").Replace(s.BranchPattern)
	if err := convention.ValidateBranchIdentity(branch); err != nil {
		return err
	}
	if !strings.Contains(s.BranchPattern, "{description}") || !strings.Contains(s.CommitConvention, "{subject}") {
		return fmt.Errorf("branch format requires description and commit format requires subject")
	}
	title := strings.NewReplacer("{type}", "feat", "{subject}", "sample", "{scope}", "core", "{issue}", "7").Replace(s.CommitConvention)
	// The same immutable naming rule applies to UI-authored formats.
	for _, marker := range []string{"codex", "claude", "openai", "[ai]", "ai:"} {
		if strings.Contains(strings.ToLower(title), marker) {
			return fmt.Errorf("commit format contains tool attribution")
		}
	}
	_, err := policyBytes(s)
	return err
}
func policyBytes(s Settings) ([]byte, error) {
	admins := []string{}
	seen := map[string]bool{}
	for _, a := range s.Admins {
		a = strings.TrimPrefix(strings.TrimSpace(a), "user:")
		if seen[strings.ToLower(a)] {
			return nil, fmt.Errorf("duplicate administrator")
		}
		seen[strings.ToLower(a)] = true
		admins = append(admins, "user:"+a)
	}
	p := governance.Policy{SchemaVersion: 2, Mode: s.SecurityMode, Enabled: s.SecurityMode != "weak", Provider: "github", Repository: s.Repository, ProductAuthorities: admins, ProtectedKinds: []string{"product", "policy", "business", "contract"}, Approval: governance.ApprovalPolicy{Minimum: 1}, Delegates: s.Delegates}
	data, err := yaml.Marshal(p)
	if err != nil {
		return nil, err
	}
	_, err = governance.DecodePolicy(data)
	return data, err
}
func personalBytes(s Settings) []byte {
	data, _ := json.MarshalIndent(personal{s.Language, s.UIPreference}, "", "  ")
	return append(data, '\n')
}
func apply(ctx context.Context, root, expected string, files []operation.FileChange) (string, error) {
	unlock, err := lockRoot(root)
	if err != nil {
		return "", err
	}
	defer unlock()

	current, err := revision(root)
	if err != nil {
		return "", err
	}
	if expected == "" || current != expected {
		return "", fmt.Errorf("settings changed in another host; refresh before applying")
	}
	// Include unchanged files in the operation precondition to detect concurrent host writes.
	existing := map[string]bool{}
	for _, f := range files {
		existing[f.Path] = true
	}
	for _, path := range statePaths {
		if existing[path] {
			continue
		}
		data, e := os.ReadFile(filepath.Join(root, path))
		if e == nil {
			files = append(files, operation.FileChange{Path: path, Content: data, Mode: 0600})
		} else if !os.IsNotExist(e) {
			return "", e
		}
	}
	payload, _ := json.Marshal(files)
	sum := sha256.Sum256(append([]byte(expected), payload...))
	p := operation.Plan{Root: root, ID: "settings-" + hex.EncodeToString(sum[:16]), Files: files}
	p.InitialStateFingerprint, err = operation.StateFingerprint(p)
	if err != nil {
		return "", err
	}
	// Recheck after constructing the precondition to close the read/preflight window.
	latest, err := revision(root)
	if err != nil || latest != expected {
		return "", fmt.Errorf("settings changed during preview; refresh")
	}
	result := operation.Apply(ctx, p)
	if result.ExitCode != 0 {
		return "", fmt.Errorf("%s: %v", result.Summary, result.Blockers)
	}
	return revision(root)
}
func SavePersonal(ctx context.Context, root, expected string, s Settings) (string, error) {
	if s.Language != "en" && s.Language != "ko" {
		return "", fmt.Errorf("invalid language")
	}
	if s.UIPreference != "enabled" && s.UIPreference != "declined" && s.UIPreference != "unset" {
		return "", fmt.Errorf("invalid UI preference")
	}
	return apply(ctx, root, expected, []operation.FileChange{{Path: personalPath, Content: personalBytes(s), Mode: 0600}})
}
func SaveProposal(ctx context.Context, root, expected string, s Settings) (string, error) {
	if err := validateSettings(s); err != nil {
		return "", err
	}
	p, err := policyBytes(s)
	if err != nil {
		return "", err
	}
	// Preserve policy fields the form does not expose; a language edit cannot lower quorum.
	if _, e := os.Stat(filepath.Join(root, ".harness/governance.yaml")); e == nil {
		old, e := governance.LoadPolicy(root)
		if e != nil {
			return "", e
		}
		proposed, e := governance.DecodePolicy(p)
		if e != nil {
			return "", e
		}
		proposed.Approval.Minimum = old.Approval.Minimum
		proposed.ProtectedKinds = old.ProtectedKinds
		p, e = yaml.Marshal(proposed)
		if e != nil {
			return "", e
		}
		if _, e = governance.DecodePolicy(p); e != nil {
			return "", e
		}
	} else if !os.IsNotExist(e) {
		return "", e
	}
	cfg, _, err := convention.Load(root)
	if err != nil {
		return "", err
	}
	cfg.SchemaVersion = 1
	if cfg.Branch == nil {
		cfg.Branch = &convention.BranchConvention{Types: []string{"feature", "fix", "docs", "chore", "release"}}
	}
	cfg.Branch.Format = s.BranchPattern
	if cfg.Commit == nil {
		cfg.Commit = &convention.CommitConvention{Types: []string{"feat", "fix", "docs", "chore", "refactor", "test"}, MaxLength: 100}
	}
	cfg.Commit.TitleFormat = s.CommitConvention
	if err = convention.ValidateConfig(cfg); err != nil {
		return "", err
	}
	c, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return apply(ctx, root, expected, []operation.FileChange{{Path: personalPath, Content: personalBytes(s), Mode: 0600}, {Path: ".harness/governance.yaml", Content: p, Mode: 0644}, {Path: ".harness/git-conventions.yaml", Content: c, Mode: 0644}})
}

func lockRoot(root string) (func(), error) {
	path := ".harness/local/control-center.lock"
	_, err := operation.StateFingerprint(operation.Plan{Root: root, Files: []operation.FileChange{{Path: path}}})
	if err != nil {
		return nil, err
	}
	full := filepath.Join(root, path)
	if err = os.MkdirAll(filepath.Dir(full), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("another host operation or interrupted operation requires inspection: %w", err)
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	return func() { _ = os.Remove(full) }, nil
}
