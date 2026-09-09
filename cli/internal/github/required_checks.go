package github

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type RequiredCheck struct {
	Context string `json:"context"`
	AppID   int64  `json:"app_id,omitempty"`
}

// RequiredChecks unions classic protection and active repository/organization
// rules. Rules endpoint semantics: https://docs.github.com/en/rest/repos/rules#get-rules-for-a-branch
// Missing classic protection is accepted only after authenticated repository access.
func (c *Client) RequiredChecks(ctx context.Context, baseRef string) ([]RequiredCheck, error) {
	if strings.TrimSpace(baseRef) == "" {
		return nil, errors.New("PR target branch unavailable")
	}
	if _, err := c.Repository(ctx); err != nil {
		return nil, err
	}
	type classicChecks struct {
		Contexts []string `json:"contexts"`
		Checks   []struct {
			Context string `json:"context"`
			AppID   *int64 `json:"app_id"`
		} `json:"checks"`
	}
	var classic *classicChecks
	err := c.api(ctx, c.path("branches/"+url.PathEscape(baseRef)+"/protection/required_status_checks"), &classic)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	required := []RequiredCheck{}
	if err == nil {
		if classic == nil || (classic.Contexts == nil && classic.Checks == nil) {
			return nil, errors.New("invalid GitHub required checks response")
		}
		named := map[string]bool{}
		for _, check := range classic.Checks {
			app := int64(0)
			if check.AppID != nil && *check.AppID > 0 {
				app = *check.AppID
			}
			required = append(required, RequiredCheck{check.Context, app})
			named[check.Context] = true
		}
		for _, context := range classic.Contexts {
			if !named[context] {
				required = append(required, RequiredCheck{Context: context})
			}
		}
	}
	var pages [][]struct {
		Type       string `json:"type"`
		Parameters struct {
			Checks []struct {
				Context       string `json:"context"`
				IntegrationID *int64 `json:"integration_id"`
			} `json:"required_status_checks"`
		} `json:"parameters"`
	}
	if err = c.api(ctx, c.path("rules/branches/"+url.PathEscape(baseRef)+"?per_page=100"), &pages, "--paginate", "--slurp"); err != nil {
		return nil, err
	}
	if pages == nil {
		return nil, errors.New("invalid GitHub effective rules response")
	}
	for _, page := range pages {
		if page == nil {
			return nil, errors.New("invalid GitHub effective rules page")
		}
		for _, rule := range page {
			if rule.Type == "" {
				return nil, errors.New("invalid GitHub effective rule")
			}
			if rule.Type != "required_status_checks" {
				continue
			}
			if rule.Parameters.Checks == nil {
				return nil, errors.New("required status check rule has no contexts")
			}
			for _, check := range rule.Parameters.Checks {
				app := int64(0)
				if check.IntegrationID != nil && *check.IntegrationID > 0 {
					app = *check.IntegrationID
				}
				required = append(required, RequiredCheck{check.Context, app})
			}
		}
	}
	out := []RequiredCheck{}
	seen := map[string]bool{}
	for _, check := range required {
		if strings.TrimSpace(check.Context) == "" {
			return nil, errors.New("required status check has no name")
		}
		key := fmt.Sprintf("%s\x00%d", check.Context, check.AppID)
		if !seen[key] {
			seen[key] = true
			out = append(out, check)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Context == out[j].Context {
			return out[i].AppID < out[j].AppID
		}
		return out[i].Context < out[j].Context
	})
	return out, nil
}
