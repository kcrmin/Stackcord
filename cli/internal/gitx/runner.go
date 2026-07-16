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
	allowed := map[string]bool{"rev-parse": true, "status": true, "rev-list": true, "ls-tree": true}
	if !allowed[args[0]] {
		return fmt.Errorf("git command %q is outside the read-only allowlist", args[0])
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
	return append(result, "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_NOSYSTEM=1")
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
