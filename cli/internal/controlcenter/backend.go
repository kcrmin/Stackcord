package controlcenter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	github "github.com/kcrmin/Stackcord/cli/internal/github"
	"github.com/kcrmin/Stackcord/cli/internal/governance"
	"github.com/kcrmin/Stackcord/cli/internal/project"
	"github.com/kcrmin/Stackcord/cli/internal/workspace"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Backend struct {
	Root      string
	Transport github.Transport
	mu        sync.Mutex
}

func (b *Backend) Snapshot(ctx context.Context) (any, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	s, rev, err := ReadSettings(b.Root)
	if err != nil {
		return nil, err
	}
	if s.Repository == "" {
		s.Repository = DetectRepository(ctx, b.Root)
	}
	branch, _ := exec.CommandContext(ctx, "git", "-C", b.Root, "branch", "--show-current").Output()
	state := map[string]any{"project": map[string]string{"name": filepath.Base(b.Root), "path": b.Root, "branch": strings.TrimSpace(string(branch))}, "settings": s, "revision": rev, "issues": []any{}, "pullRequests": []any{}, "diagnostics": []any{}, "errors": []string{}}
	errs := []string{}
	login := ""
	liveIssues := []github.Issue{}
	livePRs := []github.PullRequest{}
	c, err := github.New(s.Repository, b.Transport)
	if err != nil {
		state["identity"] = map[string]string{"error": "Select a GitHub repository in Settings to load your account and actions."}
	} else {
		account, e := c.Identity(ctx)
		if e != nil {
			state["identity"] = map[string]string{"error": e.Error()}
		} else {
			login = account.Login
			state["identity"] = map[string]string{"login": account.Login, "url": "https://github.com/" + account.Login}
		}
		repo, e := c.Repository(ctx)
		if e == nil {
			s.TargetBranch = repo.DefaultBranch
			state["settings"] = s
		} else {
			errs = append(errs, e.Error())
		}
		issues, e := c.Issues(ctx, "open")
		if e == nil {
			state["issues"] = issues
			liveIssues = issues
		} else {
			errs = append(errs, e.Error())
		}
		prs, e := c.OpenPullRequests(ctx)
		if e != nil {
			errs = append(errs, e.Error())
		} else {
			livePRs = prs
			cards := []map[string]any{}
			for index, pr := range prs {
				status := "Not refreshed; open GitHub for the current checks."
				reasons := []string{}
				if index < 10 {
					ready, e := c.Readiness(ctx, pr.Number, pr.Head.SHA)
					if e != nil {
						status = "unknown"
						reasons = append(reasons, e.Error())
					} else if ready.Ready {
						status = "Observed checks passed; ready for eligible policy review"
					} else {
						status = "Not ready for review"
						reasons = ready.Reasons
					}
				}
				cards = append(cards, map[string]any{"number": pr.Number, "title": pr.Title, "url": pr.HTMLURL, "review_url": pr.HTMLURL + "/files", "state": pr.State, "head": pr.Head.SHA, "author": pr.User.Login, "draft": pr.Draft, "readiness": status, "checks": reasons})
			}
			state["pullRequests"] = cards
		}
	}
	checkpoint, _, e := project.ReadDiscovery(b.Root, false)
	if e == nil {
		r, e := project.SummarizeDiscovery(checkpoint)
		if e == nil {
			state["discovery"] = r
		} else {
			errs = append(errs, e.Error())
		}
	} else {
		state["discovery"] = map[string]string{"status": "unknown", "message": "No initialized discovery state is available for this project."}
	}
	state["actions"] = accountActions(login, liveIssues, livePRs)
	state["errors"] = errs
	state["diagnostics"] = []any{map[string]string{"title": "Shared state", "message": "Project meaning and settings stay in Git. Personal UI preferences stay local. The dashboard runs only while this command is open."}, map[string]string{"title": "Approval boundary", "message": "Settings edits are proposals. Trusted target-branch policy and live GitHub reviews control approval; local Git names are not authentication."}}
	return state, nil
}
func (b *Backend) Action(ctx context.Context, kind string, payload json.RawMessage) (any, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var request struct {
		Revision string   `json:"revision"`
		Settings Settings `json:"settings"`
		Title    string   `json:"title"`
		Body     string   `json:"body"`
		ID       string   `json:"id"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		return nil, fmt.Errorf("invalid action payload")
	}
	current, rev, err := ReadSettings(b.Root)
	if err != nil {
		return nil, err
	}
	if rev != request.Revision {
		return nil, fmt.Errorf("state changed; refresh and preview again")
	}
	if current.Repository == "" {
		current.Repository = DetectRepository(ctx, b.Root)
	}
	switch kind {
	case "settings.preview", "settings.apply":
		if err = validateSettings(request.Settings); err != nil {
			return nil, err
		}
		if kind == "settings.preview" {
			return map[string]any{"preview": request.Settings, "message": "This writes a local policy proposal and personal preferences. Commit and open a PR; approval uses the current trusted target policy. No remote change is made."}, nil
		}
		if actual := DetectRepository(ctx, b.Root); actual != "" && actual != request.Settings.Repository {
			return nil, fmt.Errorf("selected repository differs from Git origin")
		}
		c, e := github.New(request.Settings.Repository, b.Transport)
		if e != nil {
			return nil, e
		}
		repo, e := c.Repository(ctx)
		if e != nil {
			return nil, e
		}
		if request.Settings.TargetBranch != "" && request.Settings.TargetBranch != repo.DefaultBranch {
			return nil, fmt.Errorf("target branch must match GitHub default branch")
		}
		sha, e := c.BranchHead(ctx, repo.DefaultBranch)
		if e != nil {
			return nil, e
		}
		data, e := c.ReadFile(ctx, ".harness/governance.yaml", sha)
		if errors.Is(e, github.ErrNotFound) {
			// Bootstrap is explicit and requires authenticated repository administration.
			if !repo.Permissions["admin"] {
				return nil, fmt.Errorf("initial policy requires authenticated repository administration")
			}
			a, identityErr := c.Identity(ctx)
			if identityErr != nil {
				return nil, identityErr
			}
			found := false
			for _, admin := range request.Settings.Admins {
				if strings.EqualFold(strings.TrimPrefix(admin, "user:"), a.Login) {
					found = true
				}
			}
			if request.Settings.SecurityMode != "weak" && !found {
				return nil, fmt.Errorf("include the authenticated account as initial administrator")
			}
		} else if e != nil {
			return nil, e
		} else {
			if _, e = governance.DecodePolicy(data); e != nil {
				return nil, fmt.Errorf("trusted policy is invalid: %w", e)
			}
		}
		revision, e := SaveProposal(ctx, b.Root, request.Revision, request.Settings)
		if e != nil {
			return nil, e
		}
		return map[string]any{"revision": revision, "message": "Proposal saved locally. Review the Git diff, commit it and open a policy PR. No approval or merge was performed."}, nil
	case "issues.preview", "issues.create":
		if strings.TrimSpace(request.Title) == "" {
			return nil, fmt.Errorf("issue title is required")
		}
		if kind == "issues.preview" {
			return map[string]any{"preview": map[string]string{"repository": current.Repository, "title": request.Title, "body": request.Body}, "message": "Create a GitHub Issue in the selected repository using the authenticated account."}, nil
		}
		issue, e := b.createIssue(ctx, request.Revision, request.ID, request.Title, request.Body)
		if e != nil {
			return nil, e
		}
		return map[string]any{"issue": issue, "message": "GitHub Issue is available at " + issue.HTMLURL}, nil
	default:
		return nil, fmt.Errorf("unsupported action")
	}
}

// createIssue holds the cross-host lock while checking the exact preview revision
// and writing remotely, so the selected repository cannot change between them.
func (b *Backend) createIssue(ctx context.Context, expected, id, title, body string) (github.Issue, error) {
	var empty github.Issue
	unlock, err := lockRoot(b.Root)
	if err != nil {
		return empty, err
	}
	defer unlock()
	current, rev, err := ReadSettings(b.Root)
	if err != nil {
		return empty, err
	}
	if expected == "" || rev != expected {
		return empty, fmt.Errorf("state changed; refresh and preview again")
	}
	if current.Repository == "" {
		current.Repository = DetectRepository(ctx, b.Root)
	}
	c, err := github.New(current.Repository, b.Transport)
	if err != nil {
		return empty, err
	}
	if id == "" {
		sum := sha256.Sum256([]byte(expected + title + body))
		id = hex.EncodeToString(sum[:16])
	}
	return c.EnsureIssue(ctx, id, title, body)
}

// DetectRepository derives github.com identity from origin, not OS/Git display names.
func DetectRepository(ctx context.Context, root string) string {
	b, e := exec.CommandContext(ctx, "git", "-C", root, "remote", "get-url", "origin").Output()
	if e != nil {
		return ""
	}
	s := strings.TrimSpace(string(b))
	for _, prefix := range []string{"https://github.com/", "git@github.com:", "ssh://git@github.com/"} {
		if strings.HasPrefix(s, prefix) {
			return strings.TrimSuffix(strings.TrimPrefix(s, prefix), ".git")
		}
	}
	return ""
}

// ResolveRoot follows orchestration roots before falling back to an ordinary Git checkout.
func ResolveRoot(ctx context.Context, path string) (string, error) {
	if located, e := workspace.FindRoot(ctx, path); e == nil {
		return located.Path, nil
	}
	if raw, e := exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "--show-toplevel").Output(); e == nil {
		return filepath.Abs(strings.TrimSpace(string(raw)))
	}
	return filepath.Abs(path)
}
