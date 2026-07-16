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

func operationDirectory(plan Plan) string {
	return filepath.Join(plan.Root, ".harness", "local", "operations", plan.ID)
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
	if _, err := os.Stat(filepath.Join(directory, "receipt.json")); err == nil {
		return true, nil
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return false, err
	}
	metadataPath := filepath.Join(directory, "plan.json")
	want := planFingerprint(plan)
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
