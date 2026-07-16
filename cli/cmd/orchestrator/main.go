package main

import (
	"fmt"
	"os"

	"fullstack-orchestrator/cli/internal/command"
)

var version = "dev"

func main() {
	cmd := command.New(version, os.Stdout, os.Stderr)
	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(8)
	}
	os.Exit(command.ExitCode(cmd))
}
