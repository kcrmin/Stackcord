package github

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIssueObservationRejectsPRAndReadsPaginatedDependencies(t *testing.T) {
	c := client(t, &fakeTransport{outputs: []string{`{"number":7,"pull_request":{"url":"pr"}}`}})
	_, err := c.Issue(context.Background(), 7)
	require.Error(t, err)
	f := &fakeTransport{outputs: []string{`[[{"number":1,"html_url":"https://github.com/owner/repo/issues/1"}],[{"number":2,"html_url":"https://github.com/owner/repo/issues/2"}]]`}}
	dependencies, err := client(t, f).IssueDependencies(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, dependencies, 2)
	require.Contains(t, f.calls[0], "repos/owner/repo/issues/7/dependencies/blocked_by?per_page=100")
}
