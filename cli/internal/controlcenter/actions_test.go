package controlcenter

import (
	github "github.com/kcrmin/Stackcord/cli/internal/github"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestActionsUseAuthenticatedAccount(t *testing.T) {
	issues := []github.Issue{{Title: "Mine", HTMLURL: "https://github.com/a/b/issues/1", Assignees: []github.Account{{Login: "Alice"}}}, {Title: "Other", Assignees: []github.Account{{Login: "bob"}}}}
	a := accountActions("alice", issues, nil)
	require.Len(t, a, 1)
	require.Equal(t, "Mine", a[0]["title"])
	require.Empty(t, accountActions("", issues, nil))
}
