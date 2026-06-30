package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/MicheleColella/envault-cli/internal/hook"
	"github.com/MicheleColella/envault-cli/internal/ui"
	"github.com/MicheleColella/envault-cli/internal/vault"
)

// initVault creates a minimal Envault vault in dir so vault.IsInitialized returns true.
func initVault(t *testing.T, dir string) {
	t.Helper()
	if _, err := vault.Init(dir, "", false); err != nil {
		t.Fatalf("vault.Init: %v", err)
	}
}

// --- hook install --claude ---

func TestRunHookInstallClaude_InstallsHook(t *testing.T) {
	dir := t.TempDir()
	initVault(t, dir)

	var out bytes.Buffer
	ui.Out = &out
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runHookInstallClaude(dir, false); err != nil {
		t.Fatalf("runHookInstallClaude: %v", err)
	}

	if !hook.IsClaudeHookInstalled(dir, false) {
		t.Error("hook not installed after runHookInstallClaude")
	}
	if !strings.Contains(out.String(), "installed") {
		t.Errorf("expected 'installed' in output, got %q", out.String())
	}
}

func TestRunHookInstallClaude_AlreadyInstalled(t *testing.T) {
	dir := t.TempDir()
	initVault(t, dir)

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runHookInstallClaude(dir, false); err != nil {
		t.Fatalf("first install: %v", err)
	}

	var out bytes.Buffer
	ui.Out = &out

	if err := runHookInstallClaude(dir, false); err != nil {
		t.Fatalf("second install: %v", err)
	}
	if !strings.Contains(out.String(), "already installed") {
		t.Errorf("expected 'already installed' on second call, got %q", out.String())
	}
}

func TestRunHookInstallClaude_FailsWithoutVault(t *testing.T) {
	dir := t.TempDir()

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	err := runHookInstallClaude(dir, false)
	if err == nil {
		t.Fatal("expected error when vault not initialized, got nil")
	}
	if !strings.Contains(err.Error(), "envault init") {
		t.Errorf("expected hint about 'envault init' in error, got %q", err.Error())
	}
}

func TestRunHookInstallClaude_GlobalSkipsVaultCheck(t *testing.T) {
	// Override HOME so the global settings.json is written to a temp dir.
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := t.TempDir() // no vault

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	// Should succeed even without a vault because --global was passed.
	if err := runHookInstallClaude(dir, true); err != nil {
		t.Fatalf("runHookInstallClaude global: %v", err)
	}

	if !hook.IsClaudeHookInstalled(dir, true) {
		t.Error("global hook not installed")
	}
}

func TestRunHookInstallClaude_InjectsClaudeMD(t *testing.T) {
	dir := t.TempDir()
	initVault(t, dir)

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runHookInstallClaude(dir, false); err != nil {
		t.Fatalf("runHookInstallClaude: %v", err)
	}

	if !hook.IsClaudeMDInjected(dir) {
		t.Error("CLAUDE.md section not injected after hook install")
	}
}

func TestRunHookUninstallClaude_RemovesHook(t *testing.T) {
	dir := t.TempDir()
	initVault(t, dir)

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runHookInstallClaude(dir, false); err != nil {
		t.Fatalf("install: %v", err)
	}
	if err := runHookUninstallClaude(dir, false); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	if hook.IsClaudeHookInstalled(dir, false) {
		t.Error("hook still reported as installed after uninstall")
	}
}

func TestRunHookUninstallClaude_RemovesClaudeMDSection(t *testing.T) {
	dir := t.TempDir()
	initVault(t, dir)

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runHookInstallClaude(dir, false); err != nil {
		t.Fatalf("install: %v", err)
	}
	if err := runHookUninstallClaude(dir, false); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	if hook.IsClaudeMDInjected(dir) {
		t.Error("CLAUDE.md section still present after hook uninstall")
	}
}

func TestHookInstallCmd_ClaudeUninstallFlagWorks(t *testing.T) {
	dir := t.TempDir()
	initVault(t, dir)

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runHookInstallClaude(dir, false); err != nil {
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

	if hook.IsClaudeHookInstalled(dir, false) {
		t.Error("hook still installed after --uninstall")
	}
}
