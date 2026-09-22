package github

import (
	"context"
	"errors"
	"fmt"
)

func (c *Client) Issue(ctx context.Context, number int) (Issue, error) {
	var issue Issue
	if number < 1 {
		return issue, errors.New("positive issue number required")
	}
	err := c.api(ctx, c.path(fmt.Sprintf("issues/%d", number)), &issue)
	if err == nil && (issue.Number != number || len(issue.PullRequest) > 0) {
		err = errors.New("GitHub item is not the expected issue")
	}
	return issue, err
}

func (c *Client) IssueDependencies(ctx context.Context, number int) ([]Issue, error) {
	if number < 1 {
		return nil, errors.New("positive issue number required")
	}
	return pages[Issue](ctx, c, c.path(fmt.Sprintf("issues/%d/dependencies/blocked_by?per_page=100", number)))
}
