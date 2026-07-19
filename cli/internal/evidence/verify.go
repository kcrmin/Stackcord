package evidence

import (
	"context"
	"sort"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/gitx"
)

// VerifyCurrent checks whether a record still proves the current workspace and meaning.
func VerifyCurrent(record Record, actual Actual) []domain.Item {
	issues := []domain.Item{}
	add := func(code, message string) {
		issues = append(issues, domain.Item{Code: code, Message: message, Refs: []string{record.ID}})
	}
	repository := actual.Repository
	if repository == "" {
		repository = actual.Workspace
	}
	state, err := gitx.Inspect(context.Background(), repository)
	if err != nil {
		add("evidence.workspace-unavailable", "Evidence workspace can no longer be inspected.")
	} else {
		if state.Dirty {
			add("evidence.workspace-dirty", "Workspace is no longer clean.")
		}
		if actual.Head != "" && actual.Head != state.Head {
			add("evidence.actual-head-mismatch", "Supplied actual HEAD differs from inspected Git state.")
		}
		if record.Commit != state.Head {
			add("evidence.commit-changed", "Workspace HEAD changed after evidence was recorded.")
		}
	}
	if record.ExitCode != 0 {
		add("evidence.command-failed", "Evidence command did not pass.")
	}
	if record.DefinitionFingerprint != actual.DefinitionFingerprint {
		add("evidence.definition-changed", "Work definition changed after evidence was recorded.")
	}
	if record.ContractFingerprint != actual.ContractFingerprint {
		add("evidence.contract-changed", "Contract set changed after evidence was recorded.")
	}
	if !evidenceDigestPattern.MatchString(record.OutputDigest) || (!record.FinishedAt.IsZero() && record.FinishedAt.Before(record.StartedAt)) {
		add("evidence.record-invalid", "Evidence record integrity fields are invalid.")
	}
	sort.Slice(issues, func(left, right int) bool {
		if issues[left].Code == issues[right].Code {
			return strings.Join(issues[left].Refs, "\x00") < strings.Join(issues[right].Refs, "\x00")
		}
		return issues[left].Code < issues[right].Code
	})
	return issues
}
