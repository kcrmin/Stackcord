package governance

import (
	"context"
	"fmt"
	github "github.com/kcrmin/Stackcord/cli/internal/github"
	"sort"
	"strings"
	"time"
)

// LiveReview binds authorization to the current default-branch policy, never the PR policy.
type LiveReview struct {
	Policy       Policy   `json:"policy"`
	BaseCommit   string   `json:"base_commit"`
	HeadCommit   string   `json:"head_commit"`
	Approved     bool     `json:"approved"`
	PolicyChange bool     `json:"policy_change"`
	Kinds        []string `json:"kinds"`
	Approvers    []string `json:"approvers"`
	Reasons      []string `json:"reasons"`
}

func ReviewLive(ctx context.Context, c *github.Client, number int, now time.Time) (LiveReview, error) {
	out := LiveReview{Approvers: []string{}, Reasons: []string{}, Kinds: []string{}}
	repo, err := c.Repository(ctx)
	if err != nil {
		return out, err
	}
	sha, err := c.BranchHead(ctx, repo.DefaultBranch)
	if err != nil {
		return out, err
	}
	out.BaseCommit = sha
	data, err := c.ReadFile(ctx, ".harness/governance.yaml", sha)
	if err != nil {
		return out, fmt.Errorf("trusted target policy unavailable: %w", err)
	}
	policy, err := DecodePolicy(data)
	if err != nil {
		return out, err
	}
	out.Policy = policy
	if policy.Repository != repo.FullName || policy.Provider != "github" {
		return out, fmt.Errorf("trusted repository identity differs from policy")
	}
	pr, err := c.PullRequest(ctx, number)
	if err != nil {
		return out, err
	}
	out.HeadCommit = pr.Head.SHA
	if pr.State != "open" || pr.Base.Ref != repo.DefaultBranch || pr.Base.Repo == nil || pr.Base.Repo.FullName != repo.FullName {
		return out, fmt.Errorf("PR must target the authenticated repository default branch")
	}
	files, err := c.Files(ctx, number)
	if err != nil {
		return out, err
	}
	kinds := map[string]bool{}
	protected := map[string]bool{}
	for _, kind := range policy.ProtectedKinds {
		protected[kind] = true
	}
	for _, f := range files {
		for _, path := range []string{f.Filename, f.PreviousFilename} {
			switch {
			case path == ".harness/governance.yaml":
				out.PolicyChange = true
				kinds["policy"] = true
			case strings.HasPrefix(path, "contracts/"):
				if protected["contract"] {
					kinds["contract"] = true
				}
			case strings.HasPrefix(path, "specs/"):
				for _, kind := range []string{"product", "policy", "business"} {
					if protected[kind] {
						kinds[kind] = true
					}
				}
			}
		}
	}
	for k := range kinds {
		out.Kinds = append(out.Kinds, k)
	}
	sort.Strings(out.Kinds)
	if SecurityMode(policy) != "weak" && len(out.Kinds) > 0 {
		reviews, err := c.Reviews(ctx, number)
		if err != nil {
			return out, err
		}
		for _, r := range github.EffectiveReviews(reviews, pr.Head.SHA) {
			if r.State == "CHANGES_REQUESTED" {
				out.Reasons = append(out.Reasons, "Changes requested by "+r.User.Login)
			}
			if r.State == "APPROVED" && !strings.EqualFold(r.User.Login, pr.User.Login) && Eligible(policy, "user:"+r.User.Login, out.Kinds, out.PolicyChange, now) {
				out.Approvers = append(out.Approvers, r.User.Login)
			}
		}
		if len(out.Approvers) < policy.Approval.Minimum {
			out.Reasons = append(out.Reasons, "Approval from eligible reviewers is required under the trusted policy.")
		}
	}
	again, err := c.PullRequest(ctx, number)
	if err != nil {
		return out, err
	}
	currentRepo, err := c.Repository(ctx)
	if err != nil {
		return out, err
	}
	if currentRepo.DefaultBranch != repo.DefaultBranch || currentRepo.FullName != repo.FullName {
		return out, fmt.Errorf("repository target changed during verification; refresh")
	}
	latest, err := c.BranchHead(ctx, repo.DefaultBranch)
	if err != nil {
		return out, err
	}
	if again.Head.SHA != out.HeadCommit || latest != sha || again.State != "open" || again.Base.Ref != repo.DefaultBranch || again.Base.Repo == nil || again.Base.Repo.FullName != repo.FullName {
		return out, fmt.Errorf("head or target policy moved during verification; refresh")
	}
	out.Approved = len(out.Reasons) == 0
	return out, nil
}
