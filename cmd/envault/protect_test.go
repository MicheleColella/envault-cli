package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/MicheleColella/envault-cli/internal/protect"
	"github.com/MicheleColella/envault-cli/internal/ui"
	"github.com/MicheleColella/envault-cli/internal/vault"
)

func makeVaultedDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := vault.Init(dir, "", false); err != nil {
		t.Fatalf("vault.Init: %v", err)
	}
	return dir
}

func TestRunProtectAdd_AddsPattern(t *testing.T) {
	dir := makeVaultedDir(t)

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runProtectAdd(dir, "config/secrets.json"); err != nil {
		t.Fatalf("runProtectAdd: %v", err)
	}
	patterns, _ := protect.LoadPatterns(dir)
	if len(patterns) != 1 || patterns[0] != "config/secrets.json" {
		t.Errorf("unexpected patterns: %v", patterns)
	}
}

func TestRunProtectAdd_FailsWithoutVault(t *testing.T) {
	dir := t.TempDir()

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runProtectAdd(dir, "secrets.json"); err == nil {
		t.Error("expected error without vault")
	}
}

func TestRunProtectList_Empty(t *testing.T) {
	dir := makeVaultedDir(t)

	var out bytes.Buffer
	ui.Out = &out
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runProtectList(dir); err != nil {
		t.Fatalf("runProtectList: %v", err)
	}
	if out.Len() == 0 {
		t.Error("expected some output for empty list")
	}
}

func TestRunProtectList_ShowsPatterns(t *testing.T) {
	dir := makeVaultedDir(t)

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	_ = protect.AddPattern(dir, "data/*.csv")
	_ = protect.AddPattern(dir, "config/")

	var out bytes.Buffer
	ui.Out = &out
	if err := runProtectList(dir); err != nil {
		t.Fatalf("runProtectList: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "data/*.csv") || !strings.Contains(output, "config/") {
		t.Errorf("patterns not shown in output: %q", output)
	}
}

func TestRunProtectRemove_RemovesPattern(t *testing.T) {
	dir := makeVaultedDir(t)

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	_ = protect.AddPattern(dir, "config/secrets.json")
	if err := runProtectRemove(dir, "config/secrets.json"); err != nil {
		t.Fatalf("runProtectRemove: %v", err)
	}
	patterns, _ := protect.LoadPatterns(dir)
	if len(patterns) != 0 {
		t.Errorf("pattern still present after remove: %v", patterns)
	}
}

func TestRunProtectRemove_ErrNotFound(t *testing.T) {
	dir := makeVaultedDir(t)

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runProtectRemove(dir, "nonexistent"); err == nil {
		t.Error("expected error removing nonexistent pattern")
	}
}

