// Package github provides authenticated github.com observations through gh.
// Credentials remain with gh; callers must separately authorize remote writes.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

type Transport interface {
	Run(context.Context, ...string) ([]byte, error)
}
type ExecTransport struct{}

func (ExecTransport) Run(ctx context.Context, args ...string) ([]byte, error) {
	b, e := exec.CommandContext(ctx, "gh", args...).Output()
	var exit *exec.ExitError
	if errors.As(e, &exit) && strings.Contains(string(exit.Stderr), "(HTTP 404)") {
		return nil, ErrNotFound
	}
	return b, e
}

type Client struct {
	repo      string
	transport Transport
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

func New(repo string, transport Transport) (*Client, error) {
	p := strings.Split(repo, "/")
	if len(p) != 2 || !namePattern.MatchString(p[0]) || !namePattern.MatchString(p[1]) {
		return nil, errors.New("repository must be owner/name on github.com")
	}
	if transport == nil {
		transport = ExecTransport{}
	}
	return &Client{repo, transport}, nil
}

type Account struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
	Type  string `json:"type"`
}
type Repository struct {
	ID            int64           `json:"id"`
	FullName      string          `json:"full_name"`
	DefaultBranch string          `json:"default_branch"`
	Permissions   map[string]bool `json:"permissions"`
	HTMLURL       string          `json:"html_url"`
}
type Ref struct {
	SHA  string      `json:"sha"`
	Ref  string      `json:"ref"`
	Repo *Repository `json:"repo"`
}
type PullRequest struct {
	RequestedReviewers []Account `json:"requested_reviewers"`
	Number             int       `json:"number"`
	Title              string    `json:"title"`
	Body               string    `json:"body"`
	State              string    `json:"state"`
	HTMLURL            string    `json:"html_url"`
	User               Account   `json:"user"`
	Head               Ref       `json:"head"`
	Base               Ref       `json:"base"`
	Draft              bool      `json:"draft"`
	Mergeable          *bool     `json:"mergeable"`
	MergeableState     string    `json:"mergeable_state"`
}
type Review struct {
	ID          int64   `json:"id"`
	User        Account `json:"user"`
	State       string  `json:"state"`
	CommitID    string  `json:"commit_id"`
	SubmittedAt string  `json:"submitted_at"`
	HTMLURL     string  `json:"html_url"`
}
type Issue struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	State       string          `json:"state"`
	HTMLURL     string          `json:"html_url"`
	User        Account         `json:"user"`
	Assignees   []Account       `json:"assignees"`
	PullRequest json.RawMessage `json:"pull_request,omitempty"`
}

func (c *Client) api(ctx context.Context, endpoint string, out any, args ...string) error {
	argv := []string{"api", "--hostname", "github.com", endpoint}
	argv = append(argv, args...)
	b, e := c.transport.Run(ctx, argv...)
	if e != nil {
		if errors.Is(e, ErrNotFound) {
			return ErrNotFound
		}
		return errors.New("GitHub request failed; check gh authentication, permissions and connectivity")
	}
	if e = json.Unmarshal(b, out); e != nil {
		return errors.New("GitHub returned an invalid response")
	}
	return nil
}
func (c *Client) path(s string) string { return "repos/" + c.repo + "/" + s }
func (c *Client) Identity(ctx context.Context) (Account, error) {
	var a Account
	e := c.api(ctx, "user", &a)
	if e == nil && a.Login == "" {
		e = errors.New("GitHub identity unavailable")
	}
	return a, e
}
func pages[T any](ctx context.Context, c *Client, endpoint string) ([]T, error) {
	var ps [][]T
	if e := c.api(ctx, endpoint, &ps, "--paginate", "--slurp"); e != nil {
		return nil, e
	}
	out := []T{}
	for _, p := range ps {
		out = append(out, p...)
	}
	return out, nil
}
func (c *Client) OpenPullRequests(ctx context.Context) ([]PullRequest, error) {
	return pages[PullRequest](ctx, c, c.path("pulls?state=open&per_page=100"))
}
func (c *Client) PullRequest(ctx context.Context, n int) (PullRequest, error) {
	var p PullRequest
	if n < 1 {
		return p, errors.New("positive PR number required")
	}
	e := c.api(ctx, c.path(fmt.Sprintf("pulls/%d", n)), &p)
	return p, e
}
func (c *Client) Reviews(ctx context.Context, n int) ([]Review, error) {
	if n < 1 {
		return nil, errors.New("positive PR number required")
	}
	return pages[Review](ctx, c, c.path(fmt.Sprintf("pulls/%d/reviews?per_page=100", n)))
}
func (c *Client) Issues(ctx context.Context, state string) ([]Issue, error) {
	if state != "open" && state != "closed" && state != "all" {
		return nil, errors.New("invalid issue state")
	}
	is, e := pages[Issue](ctx, c, c.path("issues?state="+state+"&per_page=100"))
	out := []Issue{}
	for _, i := range is {
		if len(i.PullRequest) == 0 || string(i.PullRequest) == "null" {
			out = append(out, i)
		}
	}
	return out, e
}

// EffectiveReviews retains the latest decision per human account. Comments and
// pending reviews do not supersede decisions; stale approvals cannot authorize.
// Stale changes-requested decisions remain blockers until explicitly superseded.
func EffectiveReviews(reviews []Review, head string) []Review {
	rs := append([]Review(nil), reviews...)
	sort.SliceStable(rs, func(i, j int) bool {
		if rs[i].SubmittedAt != rs[j].SubmittedAt {
			return rs[i].SubmittedAt < rs[j].SubmittedAt
		}
		return rs[i].ID < rs[j].ID
	})
	latest := map[string]Review{}
	for _, r := range rs {
		if r.User.Login == "" || strings.EqualFold(r.User.Type, "Bot") || strings.HasSuffix(strings.ToLower(r.User.Login), "[bot]") {
			continue
		}
		switch r.State {
		case "APPROVED", "CHANGES_REQUESTED", "DISMISSED":
			latest[strings.ToLower(r.User.Login)] = r
		}
	}
	out := []Review{}
	for _, r := range latest {
		if r.State == "APPROVED" && (head == "" || r.CommitID != head) {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].User.Login) < strings.ToLower(out[j].User.Login) })
	return out
}

// Approve reauthenticates and pins the review to the observed expected head.
// The caller must enforce trusted policy eligibility before invoking this write.
func (c *Client) Approve(ctx context.Context, n int, head, account, body string) (Review, error) {
	var r Review
	if head == "" || account == "" {
		return r, errors.New("expected head and account required")
	}
	a, e := c.Identity(ctx)
	if e != nil {
		return r, e
	}
	if !strings.EqualFold(a.Login, account) {
		return r, errors.New("authenticated GitHub account changed")
	}
	p, e := c.PullRequest(ctx, n)
	if e != nil {
		return r, e
	}
	if p.State != "open" || p.Head.SHA != head {
		return r, errors.New("PR is closed or head changed")
	}
	if p.User.Login == "" {
		return r, errors.New("GitHub PR author unavailable")
	}
	if strings.EqualFold(p.User.Login, a.Login) {
		return r, errors.New("GitHub PR authors cannot approve their own PR")
	}
	e = c.api(ctx, c.path(fmt.Sprintf("pulls/%d/reviews", n)), &r, "--method", "POST", "-f", "event=APPROVE", "-f", "commit_id="+head, "-f", "body="+body)
	return r, e
}

// EnsureIssue is retry-idempotent using a stable marker across open and closed
// issues. Concurrent independent creators require caller-side serialization.
func (c *Client) EnsureIssue(ctx context.Context, id, title, body string) (Issue, error) {
	var out Issue
	if !namePattern.MatchString(id) || strings.TrimSpace(title) == "" {
		return out, errors.New("stable issue ID and title required")
	}
	marker := "<!-- stackcord:issue:" + id + " -->"
	is, e := c.Issues(ctx, "all")
	if e != nil {
		return out, e
	}
	for _, i := range is {
		if strings.Contains(i.Body, marker) {
			return i, nil
		}
	}
	e = c.api(ctx, c.path("issues"), &out, "--method", "POST", "-f", "title="+title, "-f", "body="+body+"\n\n"+marker)
	return out, e
}
