package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/MicheleColella/envault-cli/internal/ui"
)

func TestRunPush_NotInitialized(t *testing.T) {
	err := runPush(t.TempDir())
	if err == nil {
		t.Fatal("expected error for uninitialized vault")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunPush_EmptyVault(t *testing.T) {
	repo, _ := initTestRepo(t, "alice@test.com", "Alice")
	if err := runInit(repo, false); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	var outBuf bytes.Buffer
	ui.Out = &outBuf
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runPush(repo); err != nil {
		t.Fatalf("runPush: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Vault pushed") {
		t.Errorf("output missing 'Vault pushed': %q", out)
	}
	if !strings.Contains(out, "recipients") {
		t.Errorf("output missing 'recipients': %q", out)
	}
	if !strings.Contains(out, "secrets") {
		t.Errorf("output missing 'secrets': %q", out)
	}
	if !strings.Contains(out, "ciphertext only") {
		t.Errorf("output missing ciphertext notice: %q", out)
	}
}

func TestRunPush_ShowsCommitHash(t *testing.T) {
	repo, _ := initTestRepo(t, "alice@test.com", "Alice")
	if err := runInit(repo, false); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	var outBuf bytes.Buffer
	ui.Out = &outBuf
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runPush(repo); err != nil {
		t.Fatalf("runPush: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "commit") {
		t.Errorf("output missing commit hash line: %q", out)
	}
}

func TestRunPush_AlreadyCommitted(t *testing.T) {
	// Pushing again when nothing changed in .envault should still succeed
	// (uses HEAD hash of existing commit).
	repo, _ := initTestRepo(t, "alice@test.com", "Alice")
	if err := runInit(repo, false); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	// First push commits + pushes the vault init files.
	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })
	if err := runPush(repo); err != nil {
		t.Fatalf("first runPush: %v", err)
	}

	// Second push: nothing new to commit, but should push and succeed.
	var outBuf bytes.Buffer
	ui.Out = &outBuf
	if err := runPush(repo); err != nil {
		t.Fatalf("second runPush: %v", err)
	}

	if !strings.Contains(outBuf.String(), "Vault pushed") {
		t.Errorf("second push missing 'Vault pushed': %q", outBuf.String())
	}
}
