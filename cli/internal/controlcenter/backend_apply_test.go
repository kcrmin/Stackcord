package controlcenter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/convention"
	"github.com/kcrmin/Stackcord/cli/internal/dashboard"
	"github.com/kcrmin/Stackcord/cli/internal/governance"
	"github.com/stretchr/testify/require"
)

type settingsApplyTransport struct {
	replies []string
	calls   [][]string
}

func (f *settingsApplyTransport) Run(_ context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, args)
	if len(f.replies) == 0 {
		return nil, errors.New("unexpected API call")
	}
	reply := f.replies[0]
	f.replies = f.replies[1:]
	return []byte(reply), nil
}

func TestSettingsApplyPersistsReloadableConfigurationAndPreservesPolicy(t *testing.T) {
	root := t.TempDir()
	policy := []byte("schema_version: 2\nmode: strong\nenabled: true\nprovider: github\nrepository: acme/app\nproduct_authorities: [user:alice, user:bob]\nprotected_kinds: [policy]\napproval: {minimum: 2, authority_self_approval: false}\n")
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".harness"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".harness/governance.yaml"), policy, 0600))
	settings, revision, err := ReadSettings(root)
	require.NoError(t, err)
	settings.Language = "ko"
	f := &settingsApplyTransport{replies: []string{`{"full_name":"acme/app","default_branch":"main"}`, `{"sha":"` + strings.Repeat("a", 40) + `"}`, `{"encoding":"base64","content":"` + base64.StdEncoding.EncodeToString(policy) + `"}`}}
	backend := &Backend{Root: root, Transport: f}
	payload, err := json.Marshal(map[string]any{"revision": revision, "settings": settings})
	require.NoError(t, err)
	action, err := json.Marshal(map[string]any{"kind": "settings.apply", "payload": json.RawMessage(payload)})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:43210/api/action", bytes.NewReader(action))
	request.Header.Set("Authorization", "Bearer fixture-session")
	request.Header.Set("Origin", "http://127.0.0.1:43210")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	dashboard.New(backend, "fixture-session", "127.0.0.1:43210").ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	reloaded, _, err := ReadSettings(root)
	require.NoError(t, err, "saving the form must not make the next dashboard refresh fail")
	require.Equal(t, "ko", reloaded.Language)
	_, present, err := convention.Load(root)
	require.NoError(t, err)
	require.True(t, present)
	saved, err := governance.LoadPolicy(root)
	require.NoError(t, err)
	require.Equal(t, 2, saved.Approval.Minimum)
	require.Equal(t, []string{"policy"}, saved.ProtectedKinds)
	for _, call := range f.calls {
		require.NotContains(t, call, "POST")
		require.NotContains(t, call, "PUT")
	}
}
