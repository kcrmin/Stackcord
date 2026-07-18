package command_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"fullstack-orchestrator/cli/internal/command"
	"fullstack-orchestrator/cli/internal/project"
	"fullstack-orchestrator/cli/internal/release"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestFocusedJourneyCheckpointsInitializesClonesAndRecoversWithoutPlugin(t *testing.T) {
	parent := t.TempDir()
	checkpointPath := filepath.Join(parent, "checkpoint.yaml")
	checkpointData, err := yaml.Marshal(focusedCheckpoint())
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(checkpointPath, checkpointData, 0o600))

	checkpoint := runFocusedCommand(t, "project", "checkpoint", "--parent", parent, "--id", "01JFOCUSED", "--locale", "en", "--input", checkpointPath, "--apply", "--json")
	require.Contains(t, checkpoint, `"status":"passed"`)

	root := filepath.Join(parent, "service")
	draft := filepath.Join(parent, ".harness-drafts", "01JFOCUSED")
	initialized := runFocusedCommand(t, "project", "init", "--root", root, "--id", "project.focused-service", "--name", "Focused Service", "--locale", "en", "--draft", draft, "--apply", "--json")
	require.Contains(t, initialized, `"status":"passed"`)
	require.FileExists(t, filepath.Join(root, ".agents", "skills", "use-project-harness", "SKILL.md"))
	fallback := focusedRead(t, filepath.Join(root, ".agents", "skills", "use-project-harness", "references", "fallback.md"))
	require.Contains(t, fallback, "reduced coverage")
	require.NotContains(t, focusedTree(t, root), "User said")

	focusedGit(t, root, "init", "--initial-branch=main")
	focusedGit(t, root, "config", "user.email", "fixture@example.invalid")
	focusedGit(t, root, "config", "user.name", "Fixture User")
	focusedGit(t, root, "add", ".")
	focusedGit(t, root, "commit", "-m", "chore: initialize project")

	clone := filepath.Join(parent, "clone")
	focusedGit(t, "", "clone", root, clone)
	audit := runFocusedCommand(t, "context", "audit", "--root", clone, "--json")
	require.Contains(t, audit, `"status":"passed"`)
	require.Contains(t, audit, `"context.documents"`)
	require.Contains(t, runFocusedCommand(t, "git", "inspect", "--root", clone, "--json"), `"git.branch","message":"main"`)
	require.Contains(t, runFocusedCommand(t, "change", "plan", "--root", clone, "--objective", "Add recovery retry feedback", "--ref", "policy.account.recovery-proof", "--json"), `"change.source"`)
}

func TestFocusedJourneyCoordinatesSubmoduleContractsDBMLUIAndConflicts(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "backend")
	focusedGit(t, "", "init", "--initial-branch=main", child)
	focusedGit(t, child, "config", "user.email", "fixture@example.invalid")
	focusedGit(t, child, "config", "user.name", "Fixture User")
	require.NoError(t, os.WriteFile(filepath.Join(child, "README.md"), []byte("backend\n"), 0o600))
	focusedGit(t, child, "add", "README.md")
	focusedGit(t, child, "commit", "-m", "chore: initialize backend")

	root := filepath.Join(parent, "root")
	runFocusedCommand(t, "project", "init", "--root", root, "--id", "project.multi-repo", "--locale", "en", "--apply", "--json")
	focusedGit(t, root, "init", "--initial-branch=main")
	focusedGit(t, root, "config", "user.email", "fixture@example.invalid")
	focusedGit(t, root, "config", "user.name", "Fixture User")
	focusedGit(t, root, "add", ".")
	focusedGit(t, root, "commit", "-m", "chore: initialize root")
	focusedGit(t, root, "-c", "protocol.file.allow=always", "submodule", "add", child, "workspaces/backend")
	focusedGit(t, root, "commit", "-am", "build: add backend workspace")

	inspect := runFocusedCommand(t, "git", "inspect", "--root", root, "--json")
	require.Contains(t, inspect, `"git.submodules","message":"1"`)
	require.Contains(t, inspect, `"git.submodule.initialized","message":"true"`)
	require.Contains(t, inspect, `"git.submodule.dirty","message":"false"`)
	require.Contains(t, inspect, `"git.submodule.pointer-mismatch","message":"false"`)
	require.Contains(t, inspect, `"git.detached","message":"false"`)

	require.NoError(t, os.WriteFile(filepath.Join(root, "workspaces", "backend", "README.md"), []byte("dirty backend\n"), 0o600))
	dirtyInspect := runFocusedCommand(t, "git", "inspect", "--root", root, "--json")
	require.Contains(t, dirtyInspect, `"status":"warning"`)
	require.Contains(t, dirtyInspect, `"git.submodule.dirty","message":"true"`)
	require.Contains(t, dirtyInspect, `"git.submodule-dirty"`)
	require.NoError(t, os.WriteFile(filepath.Join(root, "workspaces", "backend", "README.md"), []byte("backend\n"), 0o600))
	require.Contains(t, runFocusedCommand(t, "integrate", "plan", "--root", root, "--json"), "additive contract, providers, consumers, UI connection")

	contractPath := filepath.Join(parent, "contract.yaml")
	contractYAML := "id: contract.identity.recovery.v1\nfields:\n  account_id:\n    type: string\n    required: true\nerrors:\n  RATE_LIMITED: retry later\nretry: safe\nidempotency: required\ntimeout: 5s\npartial_failure: reject\ncompensation: not-required\n"
	require.NoError(t, os.WriteFile(contractPath, []byte(contractYAML), 0o600))
	require.Contains(t, runFocusedCommand(t, "contract", "check", "--file", contractPath, "--json"), `"status":"passed"`)

	beforeDBML := filepath.Join(parent, "before.dbml")
	afterDBML := filepath.Join(parent, "after.dbml")
	require.NoError(t, os.WriteFile(beforeDBML, []byte("Table users {\n id int [pk]\n}\n"), 0o600))
	require.NoError(t, os.WriteFile(afterDBML, []byte("Table users {\n id int [pk]\n email varchar [not null]\n}\n"), 0o600))
	require.Contains(t, runFocusedCommand(t, "db", "diff", "--before", beforeDBML, "--after", afterDBML, "--json"), "users.email")
	dbEntry := filepath.Join(root, "schema.dbml")
	require.NoError(t, os.WriteFile(dbEntry, []byte("Table users {\n id int [pk]\n}\n"), 0o600))
	diagramPlan := runFocusedCommand(t, "db", "diagram", "--root", root, "--operation", "01JDBE2E", "--action", "push", "--entry", "schema.dbml", "--project-id", "diagram-1", "--apply", "--json")
	require.Contains(t, diagramPlan, ".harness/local/dbdiagram")
	require.Contains(t, diagramPlan, "dbdiagram init --entry candidate.dbml --diagram-id diagram-1")
	require.Contains(t, diagramPlan, "dbdiagram push")
	require.Equal(t, focusedRead(t, dbEntry), focusedRead(t, filepath.Join(root, ".harness", "local", "dbdiagram", "01JDBE2E", "candidate.dbml")))

	archive := focusedUIArchive(t)
	uiPlan := runFocusedCommand(t, "ui", "import", "--root", root, "--archive", archive, "--id", "ui.external.recovery", "--authority", "reference", "--json")
	require.Contains(t, uiPlan, ".harness/local/imports")

	started := runFocusedCommand(t, "work", "start", "--root", root, "--work-id", "work.backend-recovery", "--claim-id", "claim.backend-recovery", "--owner", "alex", "--branch", "feature/backend-recovery", "--contract", "contract.identity.recovery.v1", "--apply", "--json")
	require.Contains(t, started, `"status":"passed"`)
	candidate := filepath.Join(parent, "candidate.yaml")
	candidateYAML := "repository: repository.root\npaths: []\npolicy_ids: []\nscenario_ids: []\ncontract_ids: [contract.identity.recovery.v1]\ndb_entities: []\nmigration_slots: []\nui_flows: []\ndependency_majors: []\nstable_ids: []\nroot_pointer: false\n"
	require.NoError(t, os.WriteFile(candidate, []byte(candidateYAML), 0o600))
	conflict := runFocusedCommand(t, "work", "conflict", "--root", root, "--candidate", candidate, "--json")
	require.Contains(t, conflict, `"conflict.level","message":"unknown"`)
	require.Contains(t, conflict, "conflict.claim-unobservable")
}

func TestFocusedJourneyVerifiesTechnicalAndUserEvidenceAgainstOneCandidate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "release-project")
	runFocusedCommand(t, "project", "init", "--root", root, "--id", "project.release", "--locale", "en", "--apply", "--json")
	input := release.Input{
		Profile:             release.ProfileCore,
		Version:             "1.0.0",
		RootCommit:          strings.Repeat("a", 40),
		WorkspaceCommits:    map[string]string{"workspace.root": strings.Repeat("a", 40)},
		ArtifactDigests:     map[string]string{"archive": focusedDigest("a")},
		ProductFingerprint:  focusedDigest("b"),
		DocsFingerprint:     focusedDigest("c"),
		ContractFingerprint: focusedDigest("d"),
		TDDEvidence:         map[string]string{"tests": focusedDigest("e")},
		IntegrationEvidence: map[string]string{"integration": focusedDigest("f")},
	}
	inputPath := filepath.Join(root, "release-input.json")
	focusedWriteJSON(t, inputPath, input)
	prepared := runFocusedCommand(t, "release", "prepare", "--root", root, "--input", inputPath, "--apply", "--json")
	require.Contains(t, prepared, `"status":"passed"`)

	candidatePath := filepath.Join(root, ".harness", "state", "release-candidate.json")
	var candidate release.Candidate
	require.NoError(t, json.Unmarshal([]byte(focusedRead(t, candidatePath)), &candidate))
	validation := release.UserValidation{SchemaVersion: 1, CandidateDigest: candidate.Digest, Confirmed: true, EvidenceDigest: focusedDigest("1"), VerifiedAt: time.Now().UTC().Format(time.RFC3339)}
	validationPath := filepath.Join(root, "user-validation.json")
	focusedWriteJSON(t, validationPath, validation)

	verified := runFocusedCommand(t, "release", "verify", "--candidate", candidatePath, "--input", inputPath, "--validation", validationPath, "--json")
	require.Contains(t, verified, `"status":"passed"`)
	require.Contains(t, verified, candidate.Digest)

	input.ProductFingerprint = focusedDigest("2")
	focusedWriteJSON(t, inputPath, input)
	changed := runFocusedCommand(t, "release", "verify", "--candidate", candidatePath, "--input", inputPath, "--validation", validationPath, "--json")
	require.Contains(t, changed, `"status":"blocked"`)
	require.Contains(t, changed, "product_fingerprint")
}

func TestFocusedJourneyNativeBinaryInitializesAdoptsRecoversAndVerifiesRelease(t *testing.T) {
	binary := focusedBuildNativeCLI(t)
	parent := t.TempDir()
	root := filepath.Join(parent, "new-project")

	require.Contains(t, focusedNative(t, binary, "project", "init", "--root", root, "--id", "project.native-new", "--locale", "en", "--apply", "--json"), `"status":"passed"`)
	focusedGit(t, root, "init", "--initial-branch=main")
	focusedGit(t, root, "config", "user.email", "fixture@example.invalid")
	focusedGit(t, root, "config", "user.name", "Fixture User")
	focusedGit(t, root, "add", ".")
	focusedGit(t, root, "commit", "-m", "chore: initialize project")
	require.Contains(t, focusedNative(t, binary, "context", "audit", "--root", root, "--json"), `"status":"passed"`)
	require.Contains(t, focusedNative(t, binary, "git", "inspect", "--root", root, "--json"), `"git.branch","message":"main"`)

	existing := filepath.Join(parent, "existing-project")
	require.NoError(t, os.MkdirAll(existing, 0o700))
	readme := "# Existing service\n\nUser-owned introduction.\n"
	require.NoError(t, os.WriteFile(filepath.Join(existing, "README.md"), []byte(readme), 0o600))
	require.Contains(t, focusedNative(t, binary, "project", "adopt", "--root", existing, "--id", "project.native-existing", "--locale", "en", "--apply", "--json"), `"status":"passed"`)
	require.Contains(t, focusedRead(t, filepath.Join(existing, "README.md")), readme)
	require.FileExists(t, filepath.Join(existing, ".agents", "skills", "use-project-harness", "SKILL.md"))

	commit := focusedGit(t, root, "rev-parse", "HEAD")
	input := release.Input{
		Profile:             release.ProfileCore,
		Version:             "1.0.0",
		RootCommit:          commit,
		WorkspaceCommits:    map[string]string{"workspace.root": commit},
		ArtifactDigests:     map[string]string{"archive": focusedDigest("a")},
		ProductFingerprint:  focusedDigest("b"),
		DocsFingerprint:     focusedDigest("c"),
		ContractFingerprint: focusedDigest("d"),
		TDDEvidence:         map[string]string{"tests": focusedDigest("e")},
		IntegrationEvidence: map[string]string{"integration": focusedDigest("f")},
	}
	inputPath := filepath.Join(parent, "release-input.json")
	focusedWriteJSON(t, inputPath, input)
	require.Contains(t, focusedNative(t, binary, "release", "prepare", "--root", root, "--input", inputPath, "--apply", "--json"), `"status":"passed"`)
	candidatePath := filepath.Join(root, ".harness", "state", "release-candidate.json")
	var candidate release.Candidate
	require.NoError(t, json.Unmarshal([]byte(focusedRead(t, candidatePath)), &candidate))
	validationPath := filepath.Join(parent, "user-validation.json")
	focusedWriteJSON(t, validationPath, release.UserValidation{
		SchemaVersion:   1,
		CandidateDigest: candidate.Digest,
		Confirmed:       true,
		EvidenceDigest:  focusedDigest("1"),
		VerifiedAt:      time.Now().UTC().Format(time.RFC3339),
	})
	require.Contains(t, focusedNative(t, binary, "release", "verify", "--candidate", candidatePath, "--input", inputPath, "--validation", validationPath, "--json"), `"status":"passed"`)
}

func focusedCheckpoint() project.DiscoveryCheckpoint {
	return project.DiscoveryCheckpoint{
		SchemaVersion:   1,
		Summary:         "Members can recover access safely.",
		CurrentFocus:    "Recovery proof and failure behavior",
		Roles:           []project.DiscoveryFact{{ID: "role.member", Summary: "Registered member"}},
		Journeys:        []project.DiscoveryFact{{ID: "journey.account.recovery", Summary: "Recover account access"}},
		Capabilities:    []project.DiscoveryFact{{ID: "capability.account.recovery", Summary: "Recover access"}},
		Policies:        []project.DiscoveryFact{{ID: "policy.account.recovery-proof", Summary: "Require verified proof"}},
		Scenarios:       []project.DiscoveryScenario{{ID: "scenario.account.recovery-success", Actor: "role.member", Trigger: "valid proof", Outcome: "access restored", Failure: "invalid proof is rejected"}},
		Quality:         []project.DiscoveryFact{{ID: "quality.account.accessibility", Summary: "Keyboard accessible"}},
		UICoverage:      []project.UICoverage{{ID: "ui.account.recovery", RoleID: "role.member", JourneyID: "journey.account.recovery", States: []string{"ready", "submitting", "success", "error"}}},
		TechnologyNeeds: []project.DiscoveryFact{{ID: "technology.need.secure-token", Summary: "Secure expiring token; implementation not selected"}},
		Decisions:       []project.DiscoveryDecision{{ID: "decision.account.channel", Choice: "verified link", Rationale: "Available to current users"}},
		Assumptions:     []project.DiscoveryFact{{ID: "assumption.account.email-verified", Summary: "Emails are verified"}},
		OpenQuestions:   []project.DiscoveryFact{{ID: "question.account.retention", Summary: "How long are attempts retained?"}},
	}
}

func runFocusedCommand(t *testing.T, args ...string) string {
	t.Helper()
	var output bytes.Buffer
	var errors bytes.Buffer
	cmd := command.New("1.0.0", &output, &errors)
	cmd.SetArgs(args)
	require.NoError(t, cmd.Execute(), "stderr: %s", errors.String())
	return output.String()
}

func focusedGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if root != "" {
		cmd.Dir = root
	}
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), output)
	return strings.TrimSpace(string(output))
}

func focusedUIArchive(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mockup.zip")
	file, err := os.Create(path)
	require.NoError(t, err)
	writer := zip.NewWriter(file)
	for name, content := range map[string]string{"LICENSE": "MIT", "screens/recovery.html": "<main>Recover</main>"} {
		entry, createErr := writer.Create(name)
		require.NoError(t, createErr)
		_, createErr = entry.Write([]byte(content))
		require.NoError(t, createErr)
	}
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())
	return path
}

func focusedWriteJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, append(data, '\n'), 0o600))
}

func focusedRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func focusedTree(t *testing.T, root string) string {
	t.Helper()
	var result strings.Builder
	require.NoError(t, filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator)) {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr == nil {
			result.Write(data)
		}
		return nil
	}))
	return result.String()
}

func focusedDigest(character string) string {
	return "sha256:" + strings.Repeat(character, 64)
}

func focusedBuildNativeCLI(t *testing.T) string {
	t.Helper()
	cliRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	name := "orchestrator"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	command := exec.Command("go", "build", "-trimpath", "-o", binary, "./cmd/orchestrator")
	command.Dir = cliRoot
	output, err := command.CombinedOutput()
	require.NoError(t, err, "build native CLI: %s", output)
	return binary
}

func focusedNative(t *testing.T, binary string, args ...string) string {
	t.Helper()
	command := exec.Command(binary, args...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "native CLI %s: %s", strings.Join(args, " "), output)
	return string(output)
}
