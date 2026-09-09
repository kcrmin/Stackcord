package command

import (
	"fmt"
	"github.com/kcrmin/Stackcord/cli/internal/controlcenter"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	github "github.com/kcrmin/Stackcord/cli/internal/github"
	"github.com/kcrmin/Stackcord/cli/internal/governance"
	"github.com/spf13/cobra"
	"time"
)

func newReviewCommand(version string, jsonOutput *bool) *cobra.Command {
	var root, repo, head, account string
	var number int
	var approve, apply bool
	cmd := &cobra.Command{Use: "review", Short: "Verify live PR readiness and policy approval", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if repo == "" {
			repo = controlcenter.DetectRepository(cmd.Context(), root)
		}
		c, err := github.New(repo, nil)
		if err != nil {
			return err
		}
		if number < 1 {
			return fmt.Errorf("positive PR number required")
		}
		pr, err := c.PullRequest(cmd.Context(), number)
		if err != nil {
			return err
		}
		if head == "" {
			head = pr.Head.SHA
		}
		ready, err := c.Readiness(cmd.Context(), number, head)
		if err != nil {
			return err
		}
		verified, err := governance.ReviewLive(cmd.Context(), c, number, time.Now().UTC())
		if err != nil {
			return err
		}
		if verified.HeadCommit != head {
			return fmt.Errorf("head changed; refresh")
		}
		result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "review", OperationID: "live-review", Status: domain.StatusPassed, Summary: "Live policy review and observed checks verified.", Facts: []domain.Item{{Code: "review.url", Message: pr.HTMLURL}, {Code: "review.head", Message: head}, {Code: "review.policy-base", Message: verified.BaseCommit}}}
		for _, reason := range ready.Reasons {
			result.Blockers = append(result.Blockers, domain.Item{Code: "review.checks", Message: reason})
		}
		for _, reason := range verified.Reasons {
			result.Blockers = append(result.Blockers, domain.Item{Code: "review.policy", Message: reason})
		}
		if approve {
			if !apply || account == "" {
				return fmt.Errorf("approval requires --apply and the expected --account; GitHub authors cannot self-approve")
			}
			if !ready.Ready {
				return fmt.Errorf("finish CI and resolve conflicts before asking for approval")
			}
			if !governance.Eligible(verified.Policy, "user:"+account, verified.Kinds, verified.PolicyChange, time.Now().UTC()) {
				return fmt.Errorf("account is not eligible under the trusted policy")
			}
			if _, err = c.Approve(cmd.Context(), number, head, account, "Approved through Stackcord after live policy and check verification."); err != nil {
				return err
			}
			result.Summary = "GitHub review submitted for this exact head. Refresh review status before merging."
			result.Blockers = nil
		} else if !ready.Ready || !verified.Approved {
			result.Status = domain.StatusBlocked
			result.ExitCode = domain.ExitBlocked
			result.Summary = "PR needs the listed checks or eligible policy reviews before merge."
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	cmd.Flags().StringVar(&root, "root", ".", "project directory")
	cmd.Flags().StringVar(&repo, "repo", "", "GitHub owner/repository; defaults to origin")
	cmd.Flags().IntVar(&number, "pr", 0, "PR number")
	cmd.Flags().StringVar(&head, "head", "", "expected commit")
	cmd.Flags().StringVar(&account, "account", "", "expected authenticated reviewer login")
	cmd.Flags().BoolVar(&approve, "approve", false, "submit an eligible GitHub PR review")
	cmd.Flags().BoolVar(&apply, "apply", false, "explicitly authorize the review submission")
	return cmd
}
