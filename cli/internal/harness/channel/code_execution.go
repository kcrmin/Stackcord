package channel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) jobPath(id string) string {
	return filepath.Join(s.root, ".harness", "local", "channel-jobs", id)
}

// The host may write the mailbox, not the original checkout or sibling code jobs.
// Custom commands remain unchanged and manage their own tool permissions.
func codeRunnerArguments(r RunnerConfig, mailbox string) []string {
	args := append([]string(nil), r.Argv...)
	if r.Host == "codex" && len(args) > 1 {
		prompt := args[len(args)-1]
		args = append(args[:len(args)-1], "--add-dir", mailbox, prompt)
	} else if r.Host == "claude" {
		args = append(args, "--add-dir", mailbox)
	}
	return args
}

// Every code Git operation is confined to a separate clone, never the user's index.
func codeGitRun(ctx context.Context, root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	options := []string{"-c", "core.hooksPath=" + filepath.Join(root, ".disabled-hooks"), "-c", "commit.gpgSign=false", "-c", "protocol.allow=never", "-c", "protocol.file.allow=always", "-c", "protocol.https.allow=always", "-c", "protocol.ssh.allow=always", "-C", root}
	cmd := exec.CommandContext(ctx, "git", append(options, args...)...)
	cmd.WaitDelay = 2 * time.Second
	cmd.Env = append(codeEnvironment(), "GIT_TERMINAL_PROMPT=0", "GIT_AUTHOR_NAME=Project worker", "GIT_AUTHOR_EMAIL=worker@localhost", "GIT_COMMITTER_NAME=Project worker", "GIT_COMMITTER_EMAIL=worker@localhost")
	output := limitedBuffer{limit: 4 * 1024 * 1024}
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil || errors.Is(err, exec.ErrWaitDelay) {
			return "", errCodeInterrupted
		}
		return "", fmt.Errorf("code Git operation %s failed", args[0])
	}
	if output.Len() >= output.limit {
		return "", errors.New("code Git output exceeds verification limit")
	}
	return strings.TrimRight(output.String(), "\r\n"), nil
}

func codeEnvironment() []string {
	env := []string{}
	for _, value := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(value, "=", 2)[0])
		switch key {
		case "GIT_DIR", "GIT_COMMON_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "STACKCORD_CHANNEL_ROOT", "STACKCORD_ACTIVE_REQUEST":
			continue
		}
		env = append(env, value)
	}
	return env
}

func runnerEnvironment(root, id string) []string {
	return append(codeEnvironment(), "STACKCORD_CHANNEL_ROOT="+root, "STACKCORD_ACTIVE_REQUEST="+id)
}

func (s *Store) prepareCode(ctx context.Context, c diskConfig, r Event, events []Event) (string, string, error) {
	p := c.Runner.Code
	if p == nil || r.Code.Repository != p.Repository {
		return "", "", errors.New("code repository is not authorized by this worker")
	}
	job := s.jobPath(r.ID)
	if err := safePath(job); err != nil {
		return "", "", err
	}
	if _, err := os.Lstat(job); !os.IsNotExist(err) {
		return "", "", errors.New("code job already exists; inspect it before starting a new request")
	}
	if err := os.MkdirAll(filepath.Dir(job), 0700); err != nil {
		return "", "", err
	}
	if _, err := codeGitRun(ctx, s.root, "clone", "--no-local", "--no-checkout", "--", s.root, job); err != nil {
		return "", "", err
	}
	if _, err := codeGitRun(ctx, job, "remote", "set-url", "origin", p.Remote); err != nil {
		return "", "", err
	}
	if _, err := codeGitRun(ctx, job, "fetch", "--no-tags", "--", p.Remote, r.Code.BaseCommit); err != nil {
		return "", "", err
	}
	if _, err := codeGitRun(ctx, job, "checkout", "-b", "work/"+r.ID, r.Code.BaseCommit); err != nil {
		return "", "", err
	}
	results := map[string]*CodeArtifact{}
	requests := map[string]Event{}
	for _, e := range events {
		if e.Type == "request" {
			requests[e.ID] = e
		}
		if e.Type == "result" && e.Status == "success" {
			results[e.RequestID] = e.Artifact
		}
	}
	for id, artifact := range results {
		previous := requests[id]
		if artifact == nil || previous.Code == nil || previous.Code.Repository != r.Code.Repository || previous.Code.BaseCommit == r.Code.BaseCommit || !codeOverlap(r, previous) {
			continue
		}
		if _, err := codeGitRun(ctx, job, "fetch", "--no-tags", "--", p.Remote, artifact.Commit); err != nil {
			return "", "", fmt.Errorf("previous completed code is unavailable: %w", err)
		}
		if _, err := codeGitRun(ctx, job, "merge-base", "--is-ancestor", artifact.Commit, r.Code.BaseCommit); err != nil {
			return "", "", fmt.Errorf("new baseline omits completed overlapping code; integrate it before starting: %w", err)
		}
	}
	for _, id := range r.Dependencies {
		artifact := results[id]
		if artifact == nil || artifact.Repository != p.Repository || artifact.BaseCommit != r.Code.BaseCommit {
			return "", "", errors.New("prerequisite has no matching verified code artifact")
		}
		if _, err := codeGitRun(ctx, job, "fetch", "--no-tags", "--", p.Remote, artifact.Commit); err != nil {
			return "", "", fmt.Errorf("prerequisite code is not available at the configured remote: %w", err)
		}
		tree, err := codeGitRun(ctx, job, "rev-parse", artifact.Commit+"^{tree}")
		if err != nil {
			return "", "", err
		}
		if tree != artifact.Tree {
			return "", "", errors.New("prerequisite tree differs from signed evidence")
		}
		if _, err = codeGitRun(ctx, job, "merge-base", "--is-ancestor", r.Code.BaseCommit, artifact.Commit); err != nil {
			return "", "", fmt.Errorf("prerequisite does not contain the agreed base: %w", err)
		}
		if _, err = codeGitRun(ctx, job, "merge", "--no-edit", "--no-ff", artifact.Commit); err != nil {
			return "", "", fmt.Errorf("prerequisite integration conflict; downstream runner was not started: %w", err)
		}
	}
	head, err := codeGitRun(ctx, job, "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}
	// Verify even a dependency-free baseline, so existing breakage is not blamed on the AI.
	if _, err = s.verifyCode(ctx, p, job, head); err != nil {
		return "", "", fmt.Errorf("combined input verification failed: %w", err)
	}
	return job, head, nil
}

var errCodeInterrupted = errors.New("verification interrupted; remaining processes need inspection")

func (s *Store) verifyCode(ctx context.Context, p *CodePolicy, job, head string) (string, error) {
	verifyCtx, cancel := context.WithTimeout(ctx, time.Duration(p.VerifyTimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(verifyCtx, p.VerifyArgv[0], p.VerifyArgv[1:]...)
	cmd.Dir = job
	cmd.Env = codeEnvironment()
	cmd.WaitDelay = 2 * time.Second
	log := limitedBuffer{limit: 64 * 1024}
	cmd.Stdout = &log
	cmd.Stderr = &log
	runErr := cmd.Run()
	if verifyCtx.Err() != nil || errors.Is(runErr, exec.ErrWaitDelay) {
		return "", errCodeInterrupted
	}
	// Keep bounded diagnostics local; raw tool output is not sent to peers.
	logPath := filepath.Join(job, ".git", "stackcord-verification.log")
	if err := safePath(logPath); err != nil {
		return "", err
	}
	if err := os.WriteFile(logPath, log.Bytes(), 0600); err != nil {
		return "", err
	}
	if err := runErr; err != nil {
		return "", errors.New("configured verification command failed")
	}
	actual, err := codeGitRun(ctx, job, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	if actual != head {
		return "", errors.New("verification changed the candidate commit")
	}
	dirty, err := codeGitRun(ctx, job, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return "", err
	}
	if dirty != "" {
		return "", errors.New("verification left changes outside the tested commit")
	}
	tree, err := codeGitRun(ctx, job, "rev-parse", "HEAD^{tree}")
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(struct {
		Commit, Tree string
		Argv         []string
	}{head, tree, p.VerifyArgv})
	digest := sha256.Sum256(b)
	return hex.EncodeToString(digest[:]), nil
}

func (s *Store) finishCode(ctx context.Context, c diskConfig, r Event, job, start string) (*CodeArtifact, error) {
	if _, err := codeGitRun(ctx, job, "merge-base", "--is-ancestor", start, "HEAD"); err != nil {
		return nil, fmt.Errorf("runner rewrote the prepared input history: %w", err)
	}
	if _, err := codeGitRun(ctx, job, "add", "--all"); err != nil {
		return nil, err
	}
	changed, err := codeGitRun(ctx, job, "diff", "--cached", "--name-only", "-z", start, "--")
	if err != nil {
		return nil, err
	}
	for _, name := range strings.Split(changed, "\x00") {
		if name == "" {
			continue
		}
		allowed := false
		for _, scope := range r.Scope {
			if name == scope || strings.HasPrefix(name, scope+"/") {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("runner changed a path outside declared scope: %s", name)
		}
	}
	// Preserve commits already made by the runner; commit only its remaining changes.
	dirty, err := codeGitRun(ctx, job, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	if dirty != "" {
		if _, err = codeGitRun(ctx, job, "commit", "-m", "feat: complete registered request"); err != nil {
			return nil, err
		}
	}
	head, err := codeGitRun(ctx, job, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	proof, err := s.verifyCode(ctx, c.Runner.Code, job, head)
	if err != nil {
		return nil, err
	}
	tree, err := codeGitRun(ctx, job, "rev-parse", "HEAD^{tree}")
	if err != nil {
		return nil, err
	}
	return &CodeArtifact{Repository: r.Code.Repository, BaseCommit: r.Code.BaseCommit, Commit: head, Tree: tree, Verification: proof}, nil
}

func (s *Store) publishCode(ctx context.Context, c diskConfig, result Event) error {
	if result.Artifact == nil {
		return nil
	}
	if c.Runner.Code == nil || c.Runner.Code.Repository != result.Artifact.Repository {
		return errors.New("code publication policy changed; restore it before publication")
	}
	job := s.jobPath(result.RequestID)
	if err := safePath(job); err != nil {
		return err
	}
	tree, err := codeGitRun(ctx, job, "rev-parse", result.Artifact.Commit+"^{tree}")
	if err != nil {
		return err
	}
	if tree != result.Artifact.Tree {
		return errors.New("durable code artifact is unavailable")
	}
	ref := "refs/heads/work/" + result.RequestID
	existing, err := codeGitRun(ctx, job, "ls-remote", "--refs", c.Runner.Code.Remote, ref)
	if err != nil {
		return err
	}
	if existing != "" {
		if strings.Fields(existing)[0] != result.Artifact.Commit {
			return errors.New("code output branch already points to different work")
		}
		return nil
	}
	_, err = codeGitRun(ctx, job, "push", "--", c.Runner.Code.Remote, result.Artifact.Commit+":"+ref)
	return err
}
