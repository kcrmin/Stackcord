package gitx

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const outputLimit = 4 << 20

type runner struct{}

func (runner) read(ctx context.Context, directory string, args ...string) (string, error) {
	if err := validateReadArgs(args); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = directory
	command.Env = safeEnvironment()
	var stdout, stderr limitedBuffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", args[0], err, redact(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func validateReadArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("git command is required")
	}
	allowed := map[string]bool{"rev-parse": true, "status": true, "rev-list": true, "ls-tree": true, "for-each-ref": true, "cat-file": true, "worktree": true, "remote": true, "branch": true}
	allowed["config"] = true
	if !allowed[args[0]] {
		return fmt.Errorf("git command %q is outside the read-only allowlist", args[0])
	}
	if args[0] == "worktree" && (len(args) != 4 || args[1] != "list" || args[2] != "--porcelain" || args[3] != "-z") {
		return fmt.Errorf("only read-only Git worktree listing is allowed")
	}
	if args[0] == "config" {
		if len(args) != 3 || args[1] != "--get" || !safeSubmoduleConfigKey(args[2]) {
			return fmt.Errorf("only an exact submodule URL lookup is allowed")
		}
	}
	if args[0] == "remote" && (len(args) != 3 || args[1] != "get-url" || args[2] == "" || strings.HasPrefix(args[2], "-") || strings.ContainsAny(args[2], "\x00\r\n")) {
		return fmt.Errorf("only an exact Git remote URL lookup is allowed")
	}
	if args[0] == "branch" && (len(args) != 4 || args[1] != "-r" || args[2] != "--contains" || !objectIDPatternForMutation(args[3])) {
		return fmt.Errorf("only an exact remote-branch containment lookup is allowed")
	}
	for _, argument := range args {
		lower := strings.ToLower(argument)
		if strings.ContainsAny(argument, "\x00\r\n") || argument == "-c" || strings.HasPrefix(lower, "--config-env") || strings.Contains(lower, "coresshcommand") || strings.Contains(lower, "hookspath") || strings.Contains(lower, "protocol.allow") {
			return fmt.Errorf("unsafe git argument rejected")
		}
	}
	return nil
}

func safeEnvironment() []string {
	keys := []string{"PATH", "HOME", "SystemRoot", "SYSTEMROOT", "TEMP", "TMP", "TMPDIR", "USERPROFILE"}
	result := make([]string, 0, len(keys)+3)
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			result = append(result, key+"="+value)
		}
	}
	return append(result, "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_NOSYSTEM=1", "GIT_ALLOW_PROTOCOL=https:ssh:git:file", "GCM_INTERACTIVE=Never", "LC_ALL=C")
}

func safeSubmoduleConfigKey(value string) bool {
	if !strings.HasPrefix(value, "submodule.") || !strings.HasSuffix(value, ".url") || strings.ContainsAny(value, "\x00\r\n") {
		return false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(value, "submodule."), ".url")
	if name == "" {
		return false
	}
	for _, char := range name {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && !strings.ContainsRune("._/-", char) {
			return false
		}
	}
	return true
}

func redact(value string) string {
	for _, marker := range []string{"token=", "password=", "secret="} {
		if index := strings.Index(strings.ToLower(value), marker); index >= 0 {
			end := strings.IndexAny(value[index:], " &\r\n")
			if end < 0 {
				end = len(value) - index
			}
			value = value[:index+len(marker)] + "[REDACTED]" + value[index+end:]
		}
	}
	return value
}

type limitedBuffer struct{ bytes.Buffer }

func (buffer *limitedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := outputLimit - buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
		}
		_, _ = buffer.Buffer.Write(data)
	}
	return original, nil
}
