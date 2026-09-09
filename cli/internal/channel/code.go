package channel

import (
	"errors"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// CodeRequest binds implementation to an immutable source and declared ownership.
type CodeRequest struct {
	Repository string   `json:"repository"`
	BaseCommit string   `json:"base_commit"`
	Resources  []string `json:"resources,omitempty"`
}

// CodePolicy is locally authorized; messages never select remotes or commands.
type CodePolicy struct {
	Repository           string   `json:"repository"`
	Remote               string   `json:"remote"`
	VerifyArgv           []string `json:"verify_argv"`
	VerifyTimeoutSeconds int      `json:"verify_timeout_seconds"`
}

// CodeArtifact identifies the exact committed tree checked by a local verifier.
type CodeArtifact struct {
	Repository   string `json:"repository"`
	BaseCommit   string `json:"base_commit"`
	Commit       string `json:"commit"`
	Tree         string `json:"tree"`
	Verification string `json:"verification"`
}

var codeObject = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
var codeDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var codeSSHRemote = regexp.MustCompile(`^[A-Za-z0-9_.-]+@[A-Za-z0-9_.-]+:[^\s]+$`)

func validateCodePolicy(p *CodePolicy) error {
	if p == nil {
		return nil
	}
	if !filepath.IsAbs(p.Remote) && !strings.Contains(p.Remote, "://") && !codeSSHRemote.MatchString(p.Remote) {
		return errors.New("code remote must be an absolute path or Git URL, not a local alias")
	}
	if !identifier.MatchString(p.Repository) || !validRemote(p.Remote) || len(p.VerifyArgv) == 0 || !filepath.IsAbs(p.VerifyArgv[0]) || p.VerifyTimeoutSeconds < 1 || p.VerifyTimeoutSeconds > 3600 {
		return errors.New("code policy requires repository, remote, absolute verification command and timeout 1..3600")
	}
	return validateRunner(RunnerConfig{Argv: p.VerifyArgv, AllowedKinds: []string{"verification"}})
}

func safeCodeScope(p string) bool {
	return p != "" && p != "." && path.Clean(p) == p && !strings.HasPrefix(p, "/") && !strings.ContainsAny(p, `\:*?[]`+"\x00\r\n") && p != ".." && !strings.HasPrefix(p, "../") && !pathsOverlap(p, ".git") && !pathsOverlap(p, ".harness/local")
}
func pathsOverlap(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}
func codeOverlap(a, b Event) bool {
	for _, x := range a.Scope {
		for _, y := range b.Scope {
			if pathsOverlap(x, y) {
				return true
			}
		}
	}
	for _, x := range a.Code.Resources {
		for _, y := range b.Code.Resources {
			if x == y {
				return true
			}
		}
	}
	return false
}
func validateCodeEvent(e Event, history []Event) error {
	if e.Type == "request" && e.Code == nil && e.Artifact == nil {
		return nil
	}
	if e.Type == "result" && e.Code == nil && e.Artifact == nil {
		for i := len(history) - 1; i >= 0; i-- {
			if history[i].Type == "request" && history[i].ID == e.RequestID {
				if history[i].Code == nil {
					return nil
				}
				break
			}
		}
	}
	requests := map[string]Event{}
	results := map[string]string{}
	for _, old := range history {
		if old.Type == "request" {
			requests[old.ID] = old
		}
		if old.Type == "result" {
			results[old.RequestID] = old.Status
		}
	}
	if e.Type == "request" {
		if e.Artifact != nil {
			return errors.New("request cannot contain verification evidence")
		}
		if e.Code == nil {
			return nil
		}
		if !identifier.MatchString(e.Code.Repository) || !codeObject.MatchString(e.Code.BaseCommit) || len(e.Scope) == 0 || len(e.Code.Resources) > 32 {
			return errors.New("code request requires repository, exact base commit and path scope")
		}
		for _, p := range e.Scope {
			if !safeCodeScope(p) {
				return errors.New("invalid code path scope")
			}
		}
		for _, r := range e.Code.Resources {
			if !identifier.MatchString(r) {
				return errors.New("invalid code resource claim")
			}
		}
		ancestors := map[string]bool{}
		var visit func(string) error
		visit = func(id string) error {
			if ancestors[id] {
				return nil
			}
			ancestors[id] = true
			r, ok := requests[id]
			if !ok || r.Code == nil || r.Code.Repository != e.Code.Repository || r.Code.BaseCommit != e.Code.BaseCommit {
				return errors.New("code prerequisites require matching repository and base commit")
			}
			for _, d := range r.Dependencies {
				if err := visit(d); err != nil {
					return err
				}
			}
			return nil
		}
		for _, d := range e.Dependencies {
			if err := visit(d); err != nil {
				return err
			}
		}
		for _, r := range requests {
			if results[r.ID] == "failed" {
				continue
			}
			if r.Code != nil && r.Code.Repository == e.Code.Repository && results[r.ID] == "" && codeOverlap(e, r) && r.Code.BaseCommit != e.Code.BaseCommit {
				return errors.New("active conflicting code ownership uses another baseline")
			}
			// Completed work still matters: omitting it would silently fork from stale code.
			if r.Code != nil && r.Code.Repository == e.Code.Repository && r.Code.BaseCommit == e.Code.BaseCommit && codeOverlap(e, r) && !ancestors[r.ID] {
				return errors.New("overlapping code paths or resources require an explicit prerequisite")
			}
		}
		return nil
	}
	if e.Code != nil {
		return errors.New("only requests can contain code intent")
	}
	r := requests[e.RequestID]
	if r.Code == nil {
		if e.Artifact != nil {
			return errors.New("unexpected code artifact")
		}
		return nil
	}
	if e.Status != "success" {
		if e.Artifact != nil {
			return errors.New("failed result cannot contain successful code evidence")
		}
		return nil
	}
	a := e.Artifact
	if a == nil || a.Repository != r.Code.Repository || a.BaseCommit != r.Code.BaseCommit || !codeObject.MatchString(a.Commit) || !codeObject.MatchString(a.Tree) || !codeDigest.MatchString(a.Verification) {
		return errors.New("code success requires exact verified artifact; use the configured code worker")
	}
	return nil
}
