package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/gitx"
)

const evidenceOutputLimit = 4 << 20

var (
	evidenceIDPattern     = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z0-9][a-z0-9-]*)+$`)
	evidenceDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// Run executes exactly one approved direct command and binds the result to a clean commit.
func Run(ctx context.Context, request Request) (Record, domain.Result) {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: "work.evidence", OperationID: "evidence-run", Status: domain.StatusFailed, ExitCode: domain.ExitInternal, Summary: "Evidence execution failed safely."}
	if err := validateRequest(request); err != nil {
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitInvalid, "Evidence request is not an approved direct workspace command."
		result.Blockers = []domain.Item{{Code: "evidence.request-invalid", Message: err.Error()}}
		return Record{}, result
	}
	repository := request.Repository
	if repository == "" {
		repository = request.Workspace
	}
	before, err := gitx.Inspect(ctx, repository)
	if err != nil {
		result.Blockers = []domain.Item{{Code: "evidence.workspace-invalid", Message: err.Error()}}
		return Record{}, result
	}
	workspace, err := canonicalEvidencePath(request.Workspace)
	repository, repositoryErr := canonicalEvidencePath(repository)
	if err != nil || repositoryErr != nil || before.Root != repository || !evidencePathWithin(repository, workspace) {
		result.Status, result.ExitCode = domain.StatusBlocked, domain.ExitBlocked
		result.Blockers = []domain.Item{{Code: "evidence.workspace-mismatch", Message: "Evidence must run from the exact workspace root."}}
		return Record{}, result
	}
	if before.Dirty {
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitBlocked, "Dirty workspace cannot produce reusable evidence."
		result.Blockers = []domain.Item{{Code: "evidence.workspace-dirty", Message: "Commit or intentionally discard reviewed changes before recording evidence."}}
		return Record{}, result
	}
	executable, argv, err := resolveApprovedCommand(workspace, request.Command.Argv)
	if err != nil {
		result.Status, result.ExitCode = domain.StatusBlocked, domain.ExitBlocked
		result.Blockers = []domain.Item{{Code: "evidence.command-invalid", Message: err.Error()}}
		return Record{}, result
	}
	timeout := time.Duration(request.Command.TimeoutSeconds) * time.Second
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	command := exec.CommandContext(commandCtx, executable, argv...)
	command.Dir = workspace
	command.Env = evidenceEnvironment(request.Command.Environment)
	var stdout, stderr boundedEvidenceBuffer
	command.Stdout, command.Stderr = &stdout, &stderr
	started := time.Now().UTC()
	runErr := command.Run()
	finished := time.Now().UTC()
	exitCode := 0
	if runErr != nil {
		exitCode = -1
		var exitError *exec.ExitError
		if errors.As(runErr, &exitError) {
			exitCode = exitError.ExitCode()
		}
	}
	after, inspectErr := gitx.Inspect(ctx, repository)
	if inspectErr != nil || after.Head != before.Head || after.Dirty {
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitVerification, "Evidence command changed workspace identity or left generated changes."
		result.Blockers = []domain.Item{{Code: "evidence.workspace-changed", Message: "Workspace must remain on the same clean commit after evidence execution."}}
		return Record{}, result
	}
	outputHash := sha256.New()
	_, _ = outputHash.Write(stdout.Bytes())
	_, _ = outputHash.Write([]byte{0})
	_, _ = outputHash.Write(stderr.Bytes())
	record := Record{
		SchemaVersion: 1, Kind: request.Command.Kind, WorkID: request.WorkID, WorkspaceID: request.WorkspaceID,
		Command: append([]string{request.Command.Argv[0]}, argv...), StartedAt: started, FinishedAt: finished, ExitCode: exitCode, Commit: before.Head,
		DefinitionFingerprint: request.DefinitionFingerprint, ContractFingerprint: request.ContractFingerprint,
		OutputDigest: "sha256:" + hex.EncodeToString(outputHash.Sum(nil)), ArtifactDigests: normalizedArtifacts(request.ArtifactDigests),
	}
	record.ID = evidenceRecordID(record)
	result.OperationID = record.ID
	result.Facts = []domain.Item{{Code: "evidence.id", Message: record.ID}, {Code: "evidence.kind", Message: record.Kind}, {Code: "evidence.commit", Message: record.Commit}, {Code: "evidence.output-digest", Message: record.OutputDigest}}
	if runErr != nil {
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitVerification, "Approved evidence command did not pass."
		code := "evidence.command-failed"
		if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
			code = "evidence.command-timeout"
		}
		result.Blockers = []domain.Item{{Code: code, Message: fmt.Sprintf("approved command exited with code %d", exitCode)}}
		return record, result
	}
	result.Status, result.ExitCode, result.Summary = domain.StatusPassed, domain.ExitSuccess, "Evidence passed on the exact clean workspace commit."
	result.Evidence = []domain.Item{{Code: "evidence.verified", Message: record.ID, Refs: []string{record.WorkID, record.WorkspaceID, record.Commit}}}
	return record, result
}

func validateRequest(request Request) error {
	if !evidenceIDPattern.MatchString(request.WorkID) || !strings.HasPrefix(request.WorkID, "work.") || !evidenceIDPattern.MatchString(request.WorkspaceID) || !strings.HasPrefix(request.WorkspaceID, "workspace.") {
		return fmt.Errorf("work and workspace stable IDs are required")
	}
	if !evidenceDigestPattern.MatchString(request.DefinitionFingerprint) || !evidenceDigestPattern.MatchString(request.ContractFingerprint) {
		return fmt.Errorf("current definition and contract fingerprints are required")
	}
	if !evidenceIDPattern.MatchString(request.Command.ID) || !strings.HasPrefix(request.Command.ID, "command.") || !validEvidenceKind(request.Command.Kind) || len(request.Command.Argv) == 0 || request.Command.TimeoutSeconds < 1 || request.Command.TimeoutSeconds > 1800 {
		return fmt.Errorf("approved command identity, kind, argv, or timeout is invalid")
	}
	for _, argument := range request.Command.Argv {
		if argument == "" || strings.ContainsAny(argument, "\x00\r\n") || len(argument) > 8192 {
			return fmt.Errorf("approved command contains an unsafe argument")
		}
	}
	for name, digest := range request.ArtifactDigests {
		if !evidenceIDPattern.MatchString("artifact."+name) || !evidenceDigestPattern.MatchString(digest) {
			return fmt.Errorf("artifact digest is invalid")
		}
	}
	return nil
}

func validEvidenceKind(value string) bool {
	switch value {
	case "test", "review", "integration", "migration", "rollback", "user", "child-merge", "root-pointer", "security":
		return true
	default:
		return false
	}
}

func resolveApprovedCommand(workspace string, values []string) (string, []string, error) {
	program := values[0]
	base := strings.ToLower(filepath.Base(program))
	for _, shell := range []string{"sh", "bash", "zsh", "fish", "cmd", "cmd.exe", "powershell", "powershell.exe", "pwsh", "pwsh.exe"} {
		if base == shell {
			return "", nil, fmt.Errorf("shell interpreters are not reusable evidence commands")
		}
	}
	var executable string
	var err error
	if strings.ContainsAny(program, "/\\") {
		if filepath.IsAbs(program) {
			return "", nil, fmt.Errorf("approved repository executables must be relative")
		}
		clean := filepath.Clean(filepath.FromSlash(program))
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return "", nil, fmt.Errorf("approved executable escapes the workspace")
		}
		executable = filepath.Join(workspace, clean)
		info, statErr := os.Lstat(executable)
		if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return "", nil, fmt.Errorf("approved repository executable is not a regular non-symlink file")
		}
	} else {
		executable, err = exec.LookPath(program)
		if err != nil {
			return "", nil, fmt.Errorf("approved executable is unavailable")
		}
	}
	return executable, append([]string(nil), values[1:]...), nil
}

func evidenceEnvironment(extra []string) []string {
	keys := []string{"PATH", "HOME", "SystemRoot", "SYSTEMROOT", "TEMP", "TMP", "TMPDIR", "USERPROFILE", "JAVA_HOME", "GOPATH"}
	seen := map[string]bool{}
	result := []string{"CI=1", "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1"}
	for _, key := range append(keys, extra...) {
		if seen[key] || !safeEnvironmentName(key) {
			continue
		}
		seen[key] = true
		if value, ok := os.LookupEnv(key); ok {
			result = append(result, key+"="+value)
		}
	}
	return result
}

func safeEnvironmentName(value string) bool {
	if value == "" {
		return false
	}
	upper := strings.ToUpper(value)
	for _, marker := range []string{"TOKEN", "PASSWORD", "SECRET", "PRIVATE_KEY", "CREDENTIAL"} {
		if strings.Contains(upper, marker) {
			return false
		}
	}
	for _, char := range value {
		if (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return true
}

func canonicalEvidencePath(value string) (string, error) {
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}

func evidencePathWithin(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func normalizedArtifacts(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func evidenceRecordID(record Record) string {
	copy := record
	copy.ID = ""
	data, _ := json.Marshal(copy)
	digest := sha256.Sum256(data)
	return "evidence." + hex.EncodeToString(digest[:12])
}

type boundedEvidenceBuffer struct{ bytes.Buffer }

func (buffer *boundedEvidenceBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := evidenceOutputLimit - buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
		}
		_, _ = buffer.Buffer.Write(data)
	}
	return original, nil
}
