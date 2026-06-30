package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/MicheleColella/envault-cli/internal/hook"
	"github.com/MicheleColella/envault-cli/internal/ui"
)

func TestRunUninstall_CleanHostIsNoop(t *testing.T) {
	root := t.TempDir()

	var out bytes.Buffer
	ui.Out = &out
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runUninstall(root, false, false); err != nil {
		t.Fatalf("runUninstall: %v", err)
	}
	if !strings.Contains(out.String(), "already clean") {
		t.Errorf("expected 'already clean', got %q", out.String())
	}
}

func TestRunUninstall_RemovesGitHookAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	mustGit(t, dir, "init", dir)

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := hook.InstallGitHook(dir); err != nil {
		t.Fatalf("InstallGitHook: %v", err)
	}

	if err := runUninstall(dir, false, false); err != nil {
		t.Fatalf("runUninstall: %v", err)
	}
	if hook.IsGitHookInstalled(dir) {
		t.Error("git hook still installed after uninstall")
	}

	// Idempotent: second run is a clean no-op.
	var out bytes.Buffer
	ui.Out = &out
	if err := runUninstall(dir, false, false); err != nil {
		t.Fatalf("second runUninstall: %v", err)
	}
	if !strings.Contains(out.String(), "already clean") {
		t.Errorf("expected 'already clean' on second run, got %q", out.String())
	}
}
