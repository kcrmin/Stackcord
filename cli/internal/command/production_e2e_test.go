package command_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

type productionDogfoodReport struct {
	SchemaVersion int    `json:"schema_version"`
	Scenario      string `json:"scenario"`
	Status        string `json:"status"`
	Assertions    []struct {
		Code   string `json:"code"`
		Status string `json:"status"`
	} `json:"assertions"`
}

func TestProductionE2EMultiRepositoryContinuity(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	binary := focusedBuildNativeCLI(t)
	resultPath := filepath.Join(t.TempDir(), "result.json")
	fixtureRoot := filepath.Join(t.TempDir(), "fixture")

	var process *exec.Cmd
	if runtime.GOOS == "windows" {
		process = exec.Command(
			"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
			"-File", filepath.Join(repositoryRoot, "dogfood", "run.ps1"),
			"-Binary", binary, "-Output", resultPath, "-Workspace", fixtureRoot,
		)
	} else {
		process = exec.Command(
			"bash", filepath.Join(repositoryRoot, "dogfood", "run.sh"),
			"--binary", binary, "--output", resultPath, "--workspace", fixtureRoot,
		)
	}
	process.Dir = repositoryRoot
	process.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := process.CombinedOutput()
	require.NoError(t, err, "production dogfood failed: %s", output)

	data, err := os.ReadFile(resultPath)
	require.NoError(t, err)
	var report productionDogfoodReport
	require.NoError(t, json.Unmarshal(data, &report))
	require.Equal(t, 1, report.SchemaVersion)
	require.Equal(t, "scenario.multi-repository-continuity", report.Scenario)
	require.Equal(t, "passed", report.Status)

	expectedData, err := os.ReadFile(filepath.Join(repositoryRoot, "dogfood", "expected-results.json"))
	require.NoError(t, err)
	var expected struct {
		RequiredAssertions []string `json:"required_assertions"`
	}
	require.NoError(t, json.Unmarshal(expectedData, &expected))
	actual := map[string]string{}
	for _, assertion := range report.Assertions {
		actual[assertion.Code] = assertion.Status
	}
	for _, code := range expected.RequiredAssertions {
		require.Equal(t, "passed", actual[code], code)
	}
}
