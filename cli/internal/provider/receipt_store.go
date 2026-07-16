package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var receiptComponent = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// ReceiptStore persists idempotency evidence and coordinates concurrent writers.
type ReceiptStore interface {
	Load(providerID, operationID string) (ExecutionReceipt, bool, error)
	Save(ExecutionReceipt) error
	Acquire(providerID, operationID string) (func(), error)
}

// FileReceiptStore keeps redacted receipts in a caller-selected local state directory.
type FileReceiptStore struct{ root string }

// NewFileReceiptStore prepares a private cross-process receipt directory.
func NewFileReceiptStore(root string) (*FileReceiptStore, error) {
	if root == "" {
		return nil, fmt.Errorf("receipt store root is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, err
	}
	realRoot, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, err
	}
	return &FileReceiptStore{root: realRoot}, nil
}

func (store *FileReceiptStore) Load(providerID, operationID string) (ExecutionReceipt, bool, error) {
	path, err := store.receiptPath(providerID, operationID)
	if err != nil {
		return ExecutionReceipt{}, false, err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return ExecutionReceipt{}, false, nil
	}
	if err != nil {
		return ExecutionReceipt{}, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return ExecutionReceipt{}, false, fmt.Errorf("provider receipt must be a regular file, not a symlink")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ExecutionReceipt{}, false, err
	}
	var receipt ExecutionReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return ExecutionReceipt{}, false, fmt.Errorf("decode provider receipt: %w", err)
	}
	if receipt.OperationID != operationID || receipt.Provider != providerID || receipt.Fingerprint == "" {
		return ExecutionReceipt{}, false, fmt.Errorf("provider receipt identity mismatch")
	}
	return receipt, true, nil
}

func (store *FileReceiptStore) Save(receipt ExecutionReceipt) error {
	path, err := store.receiptPath(receipt.Provider, receipt.OperationID)
	if err != nil {
		return err
	}
	if existing, found, err := store.Load(receipt.Provider, receipt.OperationID); err != nil {
		return err
	} else if found {
		if existing.Fingerprint == receipt.Fingerprint {
			return nil
		}
		return fmt.Errorf("operation ID already has a different provider receipt")
	}
	if err := store.ensureProviderDirectory(receipt.Provider); err != nil {
		return err
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		_ = file.Close()
		if cleanup {
			_ = os.Remove(temporary)
		}
	}()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func (store *FileReceiptStore) Acquire(providerID, operationID string) (func(), error) {
	receiptPath, err := store.receiptPath(providerID, operationID)
	if err != nil {
		return nil, err
	}
	if err := store.ensureProviderDirectory(providerID); err != nil {
		return nil, err
	}
	lockPath := receiptPath + ".lock"
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("provider operation is already running or needs lock recovery: %w", err)
	}
	_ = file.Close()
	return func() { _ = os.Remove(lockPath) }, nil
}

func (store *FileReceiptStore) receiptPath(providerID, operationID string) (string, error) {
	if !receiptComponent.MatchString(providerID) || !receiptComponent.MatchString(operationID) {
		return "", fmt.Errorf("safe provider and operation IDs are required")
	}
	directory := filepath.Join(store.root, providerID)
	if info, err := os.Lstat(directory); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("provider receipt directory must not be a symlink")
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return filepath.Join(directory, operationID+".json"), nil
}

func (store *FileReceiptStore) ensureProviderDirectory(providerID string) error {
	if !receiptComponent.MatchString(providerID) {
		return fmt.Errorf("safe provider ID is required")
	}
	directory := filepath.Join(store.root, providerID)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("provider receipt directory must not be a symlink")
	}
	return nil
}
