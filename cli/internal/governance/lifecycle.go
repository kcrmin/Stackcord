package governance

import (
	"context"
	"fmt"
	github "github.com/kcrmin/Stackcord/cli/internal/github"
	"os/exec"
	"strings"
	"time"
)

func committedPolicy(ctx context.Context, root string) (Policy, bool) {
	data, e := exec.CommandContext(ctx, "git", "-C", root, "show", "HEAD:.harness/governance.yaml").Output()
	if e != nil {
		return Policy{}, false
	}
	p, e := DecodePolicy(data)
	return p, e == nil
}
func useLive(ctx context.Context, root string, p Policy) bool {
	if p.SchemaVersion == 2 {
		return true
	}
	committed, ok := committedPolicy(ctx, root)
	if ok && committed.SchemaVersion == 2 {
		return true
	}
	// Tracking refs remember prior enrollment even after the candidate removes
	// or downgrades policy. They select live verification, never grant approval.
	refs, err := exec.CommandContext(ctx, "git", "-C", root, "for-each-ref", "--format=%(refname)", "refs/remotes/origin").Output()
	if err == nil {
		for _, ref := range strings.Fields(string(refs)) {
			data, e := exec.CommandContext(ctx, "git", "-C", root, "show", ref+":.harness/governance.yaml").Output()
			if e != nil {
				continue
			}
			previous, e := DecodePolicy(data)
			if e == nil && previous.SchemaVersion == 2 {
				return true
			}
		}
	}
	// Enabled legacy GitHub projects with an actual GitHub origin use the live
	// adapter. Disabled, unselected legacy projects retain offline behavior.
	if !p.Enabled || p.Provider != "github" {
		return false
	}
	if raw, err := exec.CommandContext(ctx, "git", "-C", root, "remote", "get-url", "origin").Output(); err == nil {
		remote := strings.TrimSpace(string(raw))
		for _, prefix := range []string{"https://github.com/", "git@github.com:", "ssh://git@github.com/"} {
			if strings.HasPrefix(remote, prefix) {
				return true
			}
		}
	}
	return false
}
func checkGitHub(ctx context.Context, root string, now time.Time) Report {
	return checkGitHubWithTransport(ctx, root, now, nil)
}

func checkGitHubWithTransport(ctx context.Context, root string, now time.Time, transport github.Transport) Report {
	out := Report{Enabled: true, Status: Unknown, Approvers: []string{}, Authorities: []string{}}
	fail := func(e error) Report {
		out.Issues = append(out.Issues, issue("governance.live-required", e.Error()))
		return out
	}
	raw, e := exec.CommandContext(ctx, "git", "-C", root, "remote", "get-url", "origin").Output()
	if e != nil {
		return fail(fmt.Errorf("GitHub origin is required for live policy verification"))
	}
	remote := strings.TrimSpace(string(raw))
	repo := ""
	for _, prefix := range []string{"https://github.com/", "git@github.com:", "ssh://git@github.com/"} {
		if strings.HasPrefix(remote, prefix) {
			repo = strings.TrimSuffix(strings.TrimPrefix(remote, prefix), ".git")
		}
	}
	c, e := github.New(repo, transport)
	if e != nil {
		return fail(e)
	}
	metadata, e := c.Repository(ctx)
	if e != nil {
		return fail(e)
	}
	base, e := c.BranchHead(ctx, metadata.DefaultBranch)
	if e != nil {
		return fail(e)
	}
	data, e := c.ReadFile(ctx, ".harness/governance.yaml", base)
	if e != nil {
		return fail(fmt.Errorf("trusted policy missing or inaccessible; an existing protected project cannot rebootstrap"))
	}
	policy, e := DecodePolicy(data)
	if e != nil {
		return fail(e)
	}
	if policy.Provider != "github" || policy.Repository != repo {
		return fail(fmt.Errorf("trusted policy repository mismatch"))
	}
	out.Enabled = policy.Enabled
	out.Authorities = policy.ProductAuthorities
	out.ProtectedFingerprint, e = ProtectedFingerprint(root)
	if e != nil {
		return fail(e)
	}
	if SecurityMode(policy) == "weak" {
		out.Status = Disabled
		out.ApprovalRevision = base
		return out
	}
	headBytes, e := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "HEAD").Output()
	if e != nil {
		return fail(e)
	}
	head := strings.TrimSpace(string(headBytes))
	dirty, e := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain", "--", ".harness/governance.yaml", "specs", "contracts").Output()
	if e != nil || len(dirty) > 0 {
		out.Status = Blocked
		return fail(fmt.Errorf("commit protected changes before requesting exact-head review"))
	}
	if head == base {
		freshRepo, err := c.Repository(ctx)
		if err != nil {
			return fail(err)
		}
		freshBase, err := c.BranchHead(ctx, metadata.DefaultBranch)
		if err != nil {
			return fail(err)
		}
		if freshRepo.DefaultBranch != metadata.DefaultBranch || freshBase != base {
			return fail(fmt.Errorf("trusted target moved during verification; refresh"))
		}
		if err := verifyLocalCandidate(ctx, root, head); err != nil {
			return fail(err)
		}
		out.Status = Approved
		out.ApprovalRevision = base
		return out
	}
	prs, e := c.OpenPullRequests(ctx)
	if e != nil {
		return fail(e)
	}
	for _, pr := range prs {
		if pr.Head.SHA != head {
			continue
		}
		r, e := ReviewLive(ctx, c, pr.Number, now)
		if e != nil {
			return fail(e)
		}
		if r.HeadCommit != head {
			return fail(fmt.Errorf("PR head changed from the local candidate during verification"))
		}
		if err := verifyLocalCandidate(ctx, root, head); err != nil {
			return fail(err)
		}
		out.Approvers = r.Approvers
		out.ApprovalRevision = r.HeadCommit
		if r.Approved {
			out.Status = Approved
		} else {
			out.Status = Proposed
			for _, reason := range r.Reasons {
				out.Issues = append(out.Issues, issue("governance.approval-required", reason))
			}
		}
		return out
	}
	out.Status = Proposed
	return fail(fmt.Errorf("open a PR for this exact commit to obtain policy approval"))
}

func verifyLocalCandidate(ctx context.Context, root, expected string) error {
	head, err := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != expected {
		return fmt.Errorf("local candidate changed during verification; refresh")
	}
	dirty, err := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain", "--", ".harness/governance.yaml", "specs", "contracts").Output()
	if err != nil || len(dirty) != 0 {
		return fmt.Errorf("protected local files changed during verification; refresh")
	}
	return nil
}
