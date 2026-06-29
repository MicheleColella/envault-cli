package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/MicheleColella/envault-cli/internal/hook"
	"github.com/MicheleColella/envault-cli/internal/ui"
)

// --- hook install --claude ---

func TestRunHookInstallClaude_InstallsHook(t *testing.T) {
	dir := t.TempDir()

	var out bytes.Buffer
	ui.Out = &out
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runHookInstallClaude(dir); err != nil {
		t.Fatalf("runHookInstallClaude: %v", err)
	}

	if !hook.IsClaudeHookInstalled(dir) {
		t.Error("hook not installed after runHookInstallClaude")
	}
	if !strings.Contains(out.String(), "installed") {
		t.Errorf("expected 'installed' in output, got %q", out.String())
	}
}

func TestRunHookInstallClaude_AlreadyInstalled(t *testing.T) {
	dir := t.TempDir()

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runHookInstallClaude(dir); err != nil {
		t.Fatalf("first install: %v", err)
	}

	var out bytes.Buffer
	ui.Out = &out

	if err := runHookInstallClaude(dir); err != nil {
		t.Fatalf("second install: %v", err)
	}
	if !strings.Contains(out.String(), "already installed") {
		t.Errorf("expected 'already installed' on second call, got %q", out.String())
	}
}

func TestRunHookUninstallClaude_RemovesHook(t *testing.T) {
	dir := t.TempDir()

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runHookInstallClaude(dir); err != nil {
		t.Fatalf("install: %v", err)
	}
	if err := runHookUninstallClaude(dir); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	if hook.IsClaudeHookInstalled(dir) {
		t.Error("hook still reported as installed after uninstall")
	}
}

func TestHookInstallCmd_ClaudeUninstallFlagWorks(t *testing.T) {
	dir := t.TempDir()

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runHookInstallClaude(dir); err != nil {
		t.Fatalf("install: %v", err)
	}

	root := newRootCmd("dev")
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})

	origWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	root.SetArgs([]string{"hook", "install", "--claude", "--uninstall"})
	if err := root.Execute(); err != nil {
		t.Fatalf("hook install --claude --uninstall: %v", err)
	}

	if hook.IsClaudeHookInstalled(dir) {
		t.Error("hook still installed after --uninstall")
	}
}
