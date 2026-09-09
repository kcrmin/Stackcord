package controlcenter

import (
	github "github.com/kcrmin/Stackcord/cli/internal/github"
	"strings"
)

func accountActions(login string, issues []github.Issue, prs []github.PullRequest) []map[string]string {
	out := []map[string]string{}
	if login == "" {
		return out
	}
	for _, i := range issues {
		for _, a := range i.Assignees {
			if strings.EqualFold(a.Login, login) {
				out = append(out, map[string]string{"title": i.Title, "message": "Assigned to your authenticated GitHub account", "url": i.HTMLURL})
				break
			}
		}
	}
	for _, p := range prs {
		if strings.EqualFold(p.User.Login, login) {
			out = append(out, map[string]string{"title": p.Title, "message": "Your PR: inspect checks and reviewer requests before the next step", "url": p.HTMLURL})
			continue
		}
		for _, a := range p.RequestedReviewers {
			if strings.EqualFold(a.Login, login) {
				out = append(out, map[string]string{"title": p.Title, "message": "GitHub requested your review; verify checks and policy eligibility before approval", "url": p.HTMLURL + "/files"})
				break
			}
		}
	}
	return out
}
