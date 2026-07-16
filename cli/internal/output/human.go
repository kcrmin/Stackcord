package output

import (
	"fmt"
	"io"

	"fullstack-orchestrator/cli/internal/domain"
)

// WriteHuman prints a concise result whose first line is always the outcome.
func WriteHuman(w io.Writer, result domain.Result) error {
	_, err := fmt.Fprintln(w, result.Summary)
	return err
}
