package github

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// ErrNotFound means GitHub returned HTTP 404. GitHub also uses 404 to conceal
// inaccessible resources; callers must verify repository access independently.
var ErrNotFound = errors.New("GitHub resource not found or inaccessible")
var shaPattern = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

func (c *Client) Repository(ctx context.Context) (Repository, error) {
	var r Repository
	e := c.api(ctx, "repos/"+c.repo, &r)
	if e == nil && !strings.EqualFold(r.FullName, c.repo) {
		return r, errors.New("GitHub repository identity changed")
	}
	return r, e
}
func (c *Client) BranchHead(ctx context.Context, ref string) (string, error) {
	if strings.TrimSpace(ref) == "" {
		return "", errors.New("branch required")
	}
	var commit struct {
		SHA string `json:"sha"`
	}
	e := c.api(ctx, c.path("commits/"+url.PathEscape(ref)), &commit)
	if e == nil && !shaPattern.MatchString(commit.SHA) {
		e = errors.New("GitHub returned invalid commit SHA")
	}
	return commit.SHA, e
}
func (c *Client) ReadFile(ctx context.Context, filename, sha string) ([]byte, error) {
	if !shaPattern.MatchString(sha) {
		return nil, errors.New("immutable 40-character base commit required")
	}
	if filename == "" || path.Clean(filename) != filename || strings.HasPrefix(filename, "/") || strings.HasPrefix(filename, "../") || strings.Contains(filename, "\\") {
		return nil, errors.New("repository relative file path required")
	}
	parts := strings.Split(filename, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	var file struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if e := c.api(ctx, c.path("contents/"+strings.Join(parts, "/")+"?ref="+sha), &file); e != nil {
		return nil, e
	}
	if file.Encoding != "base64" {
		return nil, errors.New("unsupported GitHub file encoding")
	}
	b, e := base64.StdEncoding.DecodeString(file.Content)
	if e != nil {
		return nil, errors.New("invalid GitHub file content")
	}
	return b, nil
}

type File struct {
	Filename         string `json:"filename"`
	PreviousFilename string `json:"previous_filename,omitempty"`
	Status           string `json:"status"`
	Patch            string `json:"patch,omitempty"`
}

func (c *Client) Files(ctx context.Context, n int) ([]File, error) {
	if n < 1 {
		return nil, errors.New("positive PR number required")
	}
	files, err := pages[File](ctx, c, c.path(fmt.Sprintf("pulls/%d/files?per_page=100", n)))
	if err == nil && len(files) >= 3000 {
		return nil, errors.New("GitHub file limit reached; complete PR scope unavailable")
	}
	return files, err
}

type Check struct {
	App struct {
		ID int64 `json:"id"`
	} `json:"app"`
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Context    string `json:"context,omitempty"`
	State      string `json:"state,omitempty"`
}

func (c *Client) Checks(ctx context.Context, sha string) ([]Check, error) {
	if !shaPattern.MatchString(sha) {
		return nil, errors.New("immutable commit SHA required")
	}
	var runs []struct {
		Runs []Check `json:"check_runs"`
	}
	if e := c.api(ctx, c.path("commits/"+sha+"/check-runs?per_page=100&filter=latest"), &runs, "--paginate", "--slurp"); e != nil {
		return nil, e
	}
	out := []Check{}
	for _, page := range runs {
		out = append(out, page.Runs...)
	}
	var statuses []struct {
		Statuses []Check `json:"statuses"`
	}
	if e := c.api(ctx, c.path("commits/"+sha+"/status?per_page=100"), &statuses, "--paginate", "--slurp"); e != nil {
		return nil, e
	}
	latest := map[string]Check{}
	for _, page := range statuses {
		for _, check := range page.Statuses {
			if old, ok := latest[check.Context]; !ok || check.ID > old.ID {
				latest[check.Context] = check
			}
		}
	}
	for _, check := range latest {
		check.Name = check.Context
		check.Status = "completed"
		check.Conclusion = check.State
		out = append(out, check)
	}
	return out, nil
}

type ReadyState struct {
	Required []RequiredCheck `json:"required_checks"`
	Ready    bool            `json:"ready"`
	Head     string          `json:"head"`
	Reasons  []string        `json:"reasons"`
	Checks   []Check         `json:"checks"`
}

// Readiness is the pre-review CI/conflict gate, not policy authorization or a
// replacement for GitHub branch protections. Missing checks fail closed.
func (c *Client) Readiness(ctx context.Context, n int, expectedHead string) (ReadyState, error) {
	r := ReadyState{Reasons: []string{}}
	p, e := c.PullRequest(ctx, n)
	if e != nil {
		return r, e
	}
	r.Head = p.Head.SHA
	if expectedHead == "" || p.Head.SHA != expectedHead {
		return r, errors.New("PR head changed or expected head missing")
	}
	checks, e := c.Checks(ctx, p.Head.SHA)
	if e != nil {
		return r, e
	}
	r.Checks = checks
	required, e := c.RequiredChecks(ctx, p.Base.Ref)
	if e != nil {
		return r, e
	}
	r.Required = required
	if p.State != "open" {
		r.Reasons = append(r.Reasons, "PR is not open")
	}
	if p.Draft {
		r.Reasons = append(r.Reasons, "PR is draft")
	}
	if p.Mergeable == nil || !*p.Mergeable {
		r.Reasons = append(r.Reasons, "mergeability is unknown or conflicting")
	}
	if p.MergeableState != "clean" && p.MergeableState != "blocked" && p.MergeableState != "unstable" {
		r.Reasons = append(r.Reasons, "merge state is unknown or requires updating")
	}
	if len(checks) == 0 {
		r.Reasons = append(r.Reasons, "checks unavailable")
	}
	successes := 0
	for _, check := range checks {
		if check.Status == "completed" && check.Conclusion == "success" {
			successes++
			continue
		}
		// GitHub considers success, skipped and neutral successful; review policy
		// can still block merging without making CI unready for human review.
		// https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/troubleshooting-required-status-checks
		if check.Status == "completed" && (check.Conclusion == "skipped" || check.Conclusion == "neutral") {
			continue
		}
		if check.Status != "completed" || check.Conclusion != "success" {
			r.Reasons = append(r.Reasons, "check is not successful: "+check.Name)
		}
	}
	if successes == 0 {
		r.Reasons = append(r.Reasons, "no successful check observed")
	}
	for _, required := range required {
		found := false
		for _, check := range checks {
			if check.Name == required.Context && (required.AppID == 0 || check.App.ID == required.AppID) {
				found = true
			}
		}
		if !found {
			r.Reasons = append(r.Reasons, "required check missing or from a different app: "+required.Context)
		}
	}
	fresh, e := c.PullRequest(ctx, n)
	if e != nil {
		return r, e
	}
	if fresh.Head.SHA != expectedHead {
		return r, errors.New("PR head changed during verification")
	}
	if fresh.Base.Ref != p.Base.Ref || fresh.State != "open" || fresh.Draft || fresh.Mergeable == nil || !*fresh.Mergeable || (fresh.MergeableState != "clean" && fresh.MergeableState != "blocked" && fresh.MergeableState != "unstable") {
		r.Reasons = append(r.Reasons, "PR state changed during verification")
	}
	r.Ready = len(r.Reasons) == 0
	return r, nil
}
