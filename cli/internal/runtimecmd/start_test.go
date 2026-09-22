package runtimecmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLocalSetupPreservesCredentialAndData(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "local")
	token, path, err := prepareLocal(dir)
	if err != nil || len(token) < 32 {
		t.Fatal(err)
	}
	if filepath.Dir(path) != dir {
		t.Fatal(path)
	}
	db := filepath.Join(dir, "state.db")
	if err = os.WriteFile(db, []byte("existing durable state"), 0600); err != nil {
		t.Fatal(err)
	}
	again, _, err := prepareLocal(dir)
	if err != nil || again != token {
		t.Fatal("credential changed", err)
	}
	b, _ := os.ReadFile(db)
	if string(b) != "existing durable state" {
		t.Fatal("data overwritten")
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0600 {
			t.Fatal(info.Mode())
		}
	}
}
func TestLocalSetupRejectsUnsafeExistingCredential(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	if err := os.WriteFile(path, []byte("short"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := prepareLocal(dir); err == nil {
		t.Fatal("replaced invalid existing credential")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "short" {
		t.Fatal("overwrote credential")
	}
}
