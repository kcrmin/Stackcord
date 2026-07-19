package operation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
)

// Apply executes or resumes a local plan using atomic file replacements and an idempotency receipt.
func Apply(ctx context.Context, plan Plan) domain.Result {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: "operation.apply", OperationID: plan.ID, Status: domain.StatusFailed, ExitCode: domain.ExitInternal, Summary: "Operation failed."}
	if err := validatePlanIdentity(plan); err != nil {
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitInvalid, "Operation plan identity is invalid."
		result.Blockers = []domain.Item{{Code: "operation.plan-invalid", Message: err.Error()}}
		return result
	}
	releaseLock, err := acquireApplyLock(plan)
	if err != nil {
		result.Status, result.ExitCode, result.Summary = domain.StatusUnknown, domain.ExitUnavailable, "Another process may be applying this operation; no mutation was attempted."
		result.Blockers = []domain.Item{{Code: "operation.concurrent", Message: err.Error(), Refs: []string{plan.ID}}}
		return result
	}
	defer releaseLock()
	directory := operationDirectory(plan)
	alreadyCompleted, err := initializeOrResume(plan)
	if err != nil {
		result.Blockers = []domain.Item{{Code: "operation.plan-conflict", Message: err.Error()}}
		return result
	}
	if alreadyCompleted {
		for _, change := range plan.Files {
			target, targetErr := safeTarget(plan.Root, change.Path)
			if targetErr != nil || !sameFile(target, change.Content) {
				message := "completed operation target no longer matches its receipt"
				if targetErr != nil {
					message = targetErr.Error()
				}
				result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitBlocked, "Completed operation state changed after its receipt was recorded."
				result.Blockers = []domain.Item{{Code: "operation.completed-state-changed", Message: message, Refs: []string{change.Path}}}
				return result
			}
		}
		result.Status, result.ExitCode, result.Summary = domain.StatusPassed, domain.ExitSuccess, "Operation already completed; existing receipt was reused."
		result.Evidence = []domain.Item{{Code: "operation.receipt", Message: filepath.Join(directory, "receipt.json")}}
		return result
	}
	if !journalHasMutations(directory) {
		current, fingerprintErr := StateFingerprint(plan)
		if fingerprintErr != nil || current != plan.InitialStateFingerprint {
			message := "initial target state changed after the plan was created"
			if fingerprintErr != nil {
				message = fingerprintErr.Error()
			}
			_ = appendJournal(directory, "blocked", "", message)
			result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitBlocked, "Operation preflight no longer matches actual state."
			result.Blockers = []domain.Item{{Code: "operation.state-mismatch", Message: message}}
			return result
		}
	}

	for _, change := range plan.Files {
		if err := ctx.Err(); err != nil {
			return failedResult(result, directory, change.Path, err)
		}
		target, err := safeTarget(plan.Root, change.Path)
		if err != nil {
			return failedResult(result, directory, change.Path, err)
		}
		if sameFile(target, change.Content) {
			continue
		}
		if err := atomicReplace(target, change.Content, change.Mode, plan.ID); err != nil {
			return failedResult(result, directory, change.Path, err)
		}
		_ = appendJournal(directory, "file_completed", change.Path, digest(change.Content))
		result.Changes = append(result.Changes, domain.Item{Code: "operation.file-written", Message: change.Path})
	}

	receipt := map[string]any{"schema_version": 1, "operation_id": plan.ID, "plan_fingerprint": planFingerprint(plan), "status": "completed"}
	data, _ := json.MarshalIndent(receipt, "", "  ")
	if err := atomicReplace(filepath.Join(directory, "receipt.json"), append(data, '\n'), 0o600, plan.ID+"-receipt"); err != nil {
		return failedResult(result, directory, "receipt.json", err)
	}
	_ = appendJournal(directory, "completed", "", "")
	result.Status, result.ExitCode, result.Summary = domain.StatusPassed, domain.ExitSuccess, "Operation completed atomically and receipt was recorded."
	result.Evidence = []domain.Item{{Code: "operation.receipt", Message: filepath.Join(directory, "receipt.json")}}
	return result
}

func atomicReplace(target string, content []byte, mode os.FileMode, operationID string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	temporary := filepath.Join(filepath.Dir(target), "."+operationID+".tmp")
	_ = os.Remove(temporary)
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	clean := true
	defer func() {
		_ = file.Close()
		if clean {
			_ = os.Remove(temporary)
		}
	}()
	if _, err := file.Write(content); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, target); err != nil {
		return err
	}
	clean = false
	if mode == 0 {
		mode = 0o644
	}
	if err := os.Chmod(target, mode.Perm()); err != nil {
		return err
	}
	if directory, err := os.Open(filepath.Dir(target)); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return nil
}

func failedResult(result domain.Result, directory, path string, err error) domain.Result {
	_ = appendJournal(directory, "failed", path, err.Error())
	result.Status, result.ExitCode, result.Summary = domain.StatusFailed, domain.ExitInternal, "Operation stopped safely and can be resumed with the same operation ID."
	result.Blockers = []domain.Item{{Code: "operation.write-failed", Message: fmt.Sprintf("%s: %v", path, err)}}
	return result
}

func sameFile(path string, content []byte) bool {
	data, err := os.ReadFile(path)
	return err == nil && bytes.Equal(data, content)
}

func journalHasMutations(directory string) bool {
	data, err := os.ReadFile(filepath.Join(directory, "journal.jsonl"))
	return err == nil && bytes.Contains(data, []byte(`"event":"file_completed"`))
}
