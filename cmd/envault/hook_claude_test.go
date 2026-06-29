package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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

// --- hook preuse ---

func TestRunHookPreuse_AllowsNonSensitiveCommand(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".envault"), 0o700)

	origWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	input := map[string]interface{}{
		"tool_name": "Bash",
		"tool_input": map[string]interface{}{
			"command": "npm install",
		},
	}
	b, _ := json.Marshal(input)
	r := bytes.NewReader(b)
	var w bytes.Buffer

	if err := runHookPreuse(r, &w); err != nil {
		t.Fatalf("expected no error for non-sensitive command, got: %v", err)
	}
	if w.Len() != 0 {
		t.Errorf("expected no output for allowed command, got: %s", w.String())
	}
}

func TestRunHookPreuse_BlocksEnvaultCat(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".envault"), 0o700)

	origWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	for _, cmd := range []string{
		"envault cat DB_URL",
		"./envault cat API_KEY",
		"envault export",
		"/usr/local/bin/envault cat SECRET",
	} {
		input := map[string]interface{}{
			"tool_name":  "Bash",
			"tool_input": map[string]interface{}{"command": cmd},
		}
		b, _ := json.Marshal(input)
		r := bytes.NewReader(b)
		var w bytes.Buffer

		err := runHookPreuse(r, &w)
		if err == nil {
			t.Errorf("cmd %q: expected errBlockToolCall, got nil", cmd)
			continue
		}
		if w.Len() == 0 {
			t.Errorf("cmd %q: expected block reason written to stdout, got nothing", cmd)
		}
		if !strings.Contains(w.String(), "envault run") {
			t.Errorf("cmd %q: block message should mention 'envault run', got: %s", cmd, w.String())
		}
	}
}

func TestRunHookPreuse_AllowsCatWithForce(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".envault"), 0o700)

	origWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	input := map[string]interface{}{
		"tool_name": "Bash",
		"tool_input": map[string]interface{}{
			"command": "envault cat DB_URL --force",
		},
	}
	b, _ := json.Marshal(input)
	r := bytes.NewReader(b)
	var w bytes.Buffer

	if err := runHookPreuse(r, &w); err != nil {
		t.Fatalf("expected no error for cat --force, got: %v", err)
	}
	if w.Len() != 0 {
		t.Errorf("expected no output for cat --force, got: %s", w.String())
	}
}

func TestRunHookPreuse_NoopOutsideEnvaultRepo(t *testing.T) {
	dir := t.TempDir() // no .envault/

	origWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	// Even a sensitive command is not blocked outside an envault repo.
	input := map[string]interface{}{
		"tool_name": "Bash",
		"tool_input": map[string]interface{}{
			"command": "envault cat DB_URL",
		},
	}
	b, _ := json.Marshal(input)
	r := bytes.NewReader(b)
	var w bytes.Buffer

	if err := runHookPreuse(r, &w); err != nil {
		t.Fatalf("expected no error outside envault repo, got: %v", err)
	}
	if w.Len() != 0 {
		t.Errorf("expected no output outside envault repo, got: %s", w.String())
	}
}

func TestRunHookPreuse_NoopForNonBashTool(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".envault"), 0o700)

	origWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	input := map[string]interface{}{
		"tool_name": "Read",
		"tool_input": map[string]interface{}{
			"file_path": "/etc/hosts",
		},
	}
	b, _ := json.Marshal(input)
	r := bytes.NewReader(b)
	var w bytes.Buffer

	if err := runHookPreuse(r, &w); err != nil {
		t.Fatalf("expected no error for non-Bash tool, got: %v", err)
	}
	if w.Len() != 0 {
		t.Errorf("expected no output for non-Bash tool, got: %s", w.String())
	}
}

func TestRunHookPreuse_InvalidJSONIsNoop(t *testing.T) {
	r := strings.NewReader("not json at all")
	var w bytes.Buffer
	if err := runHookPreuse(r, &w); err != nil {
		t.Fatalf("unexpected error on invalid JSON: %v", err)
	}
	if w.Len() != 0 {
		t.Errorf("expected no output for invalid JSON input, got: %s", w.String())
	}
}

func TestIsSensitiveEnvaultCmd(t *testing.T) {
	sensitive := []string{
		"envault cat DB_URL",
		"./envault cat KEY",
		"/usr/local/bin/envault cat KEY",
		"envault export",
		"./envault export",
	}
	notSensitive := []string{
		"envault cat DB_URL --force",
		"envault list",
		"envault run -- npm start",
		"npm install",
		"echo envault cat",   // envault is not a command here
		"envault add DB_URL", // not cat/export
	}

	for _, cmd := range sensitive {
		if !isSensitiveEnvaultCmd(cmd) {
			t.Errorf("expected %q to be sensitive, got false", cmd)
		}
	}
	for _, cmd := range notSensitive {
		if isSensitiveEnvaultCmd(cmd) {
			t.Errorf("expected %q to NOT be sensitive, got true", cmd)
		}
	}
}
