package operation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type journalEvent struct {
	Time   time.Time `json:"time"`
	Event  string    `json:"event"`
	Path   string    `json:"path,omitempty"`
	Detail string    `json:"detail,omitempty"`
}

type operationMetadata struct {
	PlanFingerprint string `json:"plan_fingerprint"`
}

type operationReceipt struct {
	SchemaVersion   int    `json:"schema_version"`
	OperationID     string `json:"operation_id"`
	PlanFingerprint string `json:"plan_fingerprint"`
	Status          string `json:"status"`
}

func operationDirectory(plan Plan) string {
	return filepath.Join(plan.Root, ".harness", "local", "operations", plan.ID)
}

func acquireApplyLock(plan Plan) (func(), error) {
	lockPath, err := safeTarget(plan.Root, filepath.ToSlash(filepath.Join(".harness", "local", "operations", plan.ID, "apply.lock")))
	if err != nil {
		return nil, err
	}
	directory := filepath.Dir(lockPath)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("operation lock exists and must be inspected before retry: %w", err)
	}
	if _, err := file.WriteString(time.Now().UTC().Format(time.RFC3339Nano) + "\n"); err != nil {
		_ = file.Close()
		_ = os.Remove(lockPath)
		return nil, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(lockPath)
		return nil, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(lockPath)
		return nil, err
	}
	return func() { _ = os.Remove(lockPath) }, nil
}

func appendJournal(directory, event, path, detail string) error {
	file, err := os.OpenFile(filepath.Join(directory, "journal.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := json.Marshal(journalEvent{Time: time.Now().UTC(), Event: event, Path: path, Detail: detail})
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return file.Sync()
}

func initializeOrResume(plan Plan) (bool, error) {
	directory := operationDirectory(plan)
	want := planFingerprint(plan)
	if data, err := os.ReadFile(filepath.Join(directory, "receipt.json")); err == nil {
		var receipt operationReceipt
		if json.Unmarshal(data, &receipt) != nil || receipt.SchemaVersion != 1 || receipt.OperationID != plan.ID || receipt.PlanFingerprint != want || receipt.Status != "completed" {
			return false, fmt.Errorf("operation ID %s has an invalid or mismatched completion receipt", plan.ID)
		}
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return false, err
	}
	metadataPath := filepath.Join(directory, "plan.json")
	if data, err := os.ReadFile(metadataPath); err == nil {
		var existing operationMetadata
		if json.Unmarshal(data, &existing) != nil || existing.PlanFingerprint != want {
			return false, fmt.Errorf("operation ID %s already belongs to a different plan", plan.ID)
		}
		return false, nil
	}
	data, _ := json.MarshalIndent(operationMetadata{PlanFingerprint: want}, "", "  ")
	if err := os.WriteFile(metadataPath, append(data, '\n'), 0o600); err != nil {
		return false, err
	}
	return false, appendJournal(directory, "started", "", "")
}
