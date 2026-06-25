package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/MicheleColella/envault-cli/internal/ui"
	"github.com/MicheleColella/envault-cli/internal/vault"
)

func TestRunList_RequiresInitializedVault(t *testing.T) {
	root := t.TempDir()
	err := runList(root)
	if err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("expected 'not initialized' error, got %v", err)
	}
}

func TestRunList_EmptyVault(t *testing.T) {
	root := initVaultRoot(t)

	var out bytes.Buffer
	ui.Out = &out
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runList(root); err != nil {
		t.Fatalf("runList: %v", err)
	}

	if !strings.Contains(out.String(), "empty") {
		t.Errorf("expected 'empty' message, got: %s", out.String())
	}
}

func TestRunList_ShowsNamesAndAlgorithm(t *testing.T) {
	root := initVaultRoot(t)
	addTestRecipient(t, root, "alice@example.com")

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runAdd(root, "API_KEY", []byte("s3cr3t")); err != nil {
		t.Fatalf("runAdd: %v", err)
	}

	var out bytes.Buffer
	ui.Out = &out

	if err := runList(root); err != nil {
		t.Fatalf("runList: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "API_KEY") {
		t.Errorf("expected API_KEY in output, got: %s", got)
	}
	if !strings.Contains(got, "AES-256-GCM") {
		t.Errorf("expected algorithm in output, got: %s", got)
	}
}

func TestRunList_DoesNotShowValues(t *testing.T) {
	root := initVaultRoot(t)
	addTestRecipient(t, root, "alice@example.com")

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runAdd(root, "SECRET_KEY", []byte("super-secret-value")); err != nil {
		t.Fatalf("runAdd: %v", err)
	}

	var out bytes.Buffer
	ui.Out = &out

	if err := runList(root); err != nil {
		t.Fatalf("runList: %v", err)
	}

	if strings.Contains(out.String(), "super-secret-value") {
		t.Error("list must not reveal secret values")
	}
}

func TestRunList_MultipleEntries(t *testing.T) {
	root := initVaultRoot(t)
	addTestRecipient(t, root, "alice@example.com")

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	for _, name := range []string{"KEY_A", "KEY_B", "KEY_C"} {
		if err := runAdd(root, name, []byte("val")); err != nil {
			t.Fatalf("runAdd %s: %v", name, err)
		}
	}

	store, _ := vault.LoadStore(root)
	if len(store.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(store.Entries))
	}

	var out bytes.Buffer
	ui.Out = &out

	if err := runList(root); err != nil {
		t.Fatalf("runList: %v", err)
	}

	got := out.String()
	for _, name := range []string{"KEY_A", "KEY_B", "KEY_C"} {
		if !strings.Contains(got, name) {
			t.Errorf("expected %s in output, got: %s", name, got)
		}
	}
}
