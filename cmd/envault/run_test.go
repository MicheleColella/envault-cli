package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	envcrypto "github.com/MicheleColella/envault-cli/internal/crypto"
	"github.com/MicheleColella/envault-cli/internal/ui"
	"github.com/MicheleColella/envault-cli/internal/vault"
)

func TestRunRun_NotInitialized(t *testing.T) {
	err := runRun(t.TempDir(), []string{"echo", "hi"}, newMemStore())
	if err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("expected not-initialized error, got %v", err)
	}
}

func TestRunRun_NoPrivateKey(t *testing.T) {
	root := initVaultRoot(t)
	addTestRecipient(t, root, "alice@example.com")

	ui.Out = &bytes.Buffer{}
	ui.Err = &bytes.Buffer{}
	t.Cleanup(func() {
		ui.Out = os.Stdout
		ui.Err = os.Stderr
	})

	if err := runAdd(root, "KEY", []byte("val")); err != nil {
		t.Fatalf("runAdd: %v", err)
	}

	// Empty keychain — no private key.
	err := runRun(root, []string{"true"}, newMemStore())
	if err == nil || !strings.Contains(err.Error(), "no private key") {
		t.Fatalf("expected 'no private key' error, got %v", err)
	}
}

func TestRunRun_InjectsSecrets(t *testing.T) {
	root := initVaultRoot(t)
	priv := addTestRecipient(t, root, "alice@example.com")

	kc := newMemStore()
	if err := kc.Seal("alice@example.com", priv[:]); err != nil {
		t.Fatalf("kc.Seal: %v", err)
	}

	ui.Out = &bytes.Buffer{}
	ui.Err = &bytes.Buffer{}
	t.Cleanup(func() {
		ui.Out = os.Stdout
		ui.Err = os.Stderr
	})

	if err := runAdd(root, "ENVAULT_TEST_VAR", []byte("hello-from-vault")); err != nil {
		t.Fatalf("runAdd: %v", err)
	}

	// Run printenv and capture output from the child process.
	var childOut bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRun(root, []string{"printenv", "ENVAULT_TEST_VAR"}, kc)

	_ = w.Close()
	_, _ = childOut.ReadFrom(r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runRun: %v", err)
	}
	if !strings.Contains(childOut.String(), "hello-from-vault") {
		t.Errorf("expected secret value in child output, got: %q", childOut.String())
	}
}

func TestRunRun_ExitCodePropagated(t *testing.T) {
	root := initVaultRoot(t)
	priv := addTestRecipient(t, root, "alice@example.com")

	kc := newMemStore()
	if err := kc.Seal("alice@example.com", priv[:]); err != nil {
		t.Fatalf("kc.Seal: %v", err)
	}

	ui.Out = &bytes.Buffer{}
	ui.Err = &bytes.Buffer{}
	t.Cleanup(func() {
		ui.Out = os.Stdout
		ui.Err = os.Stderr
	})

	// `false` exits with code 1 on all POSIX systems.
	err := runRun(root, []string{"false"}, kc)
	if err == nil {
		t.Fatal("expected exit-code error from 'false', got nil")
	}
	var ece exitCodeError
	if !errors.As(err, &ece) {
		t.Fatalf("expected exitCodeError, got %T: %v", err, err)
	}
	if ece.code != 1 {
		t.Errorf("exit code = %d, want 1", ece.code)
	}
}

func TestRunRun_EmptyVault(t *testing.T) {
	// Vault with a recipient but no secrets — command should still run.
	root := initVaultRoot(t)
	priv := addTestRecipient(t, root, "alice@example.com")

	kc := newMemStore()
	if err := kc.Seal("alice@example.com", priv[:]); err != nil {
		t.Fatalf("kc.Seal: %v", err)
	}

	ui.Out = &bytes.Buffer{}
	ui.Err = &bytes.Buffer{}
	t.Cleanup(func() {
		ui.Out = os.Stdout
		ui.Err = os.Stderr
	})

	if err := runRun(root, []string{"true"}, kc); err != nil {
		t.Fatalf("runRun with empty vault: %v", err)
	}
}

func TestRunRun_FileEntriesNotInjected(t *testing.T) {
	// KindFile entries must NOT be injected as env vars.
	root := initVaultRoot(t)
	priv, pub, _ := envcrypto.GenerateKeyPair()
	if err := vault.AddRecipient(root, vault.Recipient{ID: "alice@example.com", PublicKey: pub}); err != nil {
		t.Fatalf("AddRecipient: %v", err)
	}

	kc := newMemStore()
	if err := kc.Seal("alice@example.com", priv[:]); err != nil {
		t.Fatalf("kc.Seal: %v", err)
	}

	ui.Out = &bytes.Buffer{}
	ui.Err = &bytes.Buffer{}
	t.Cleanup(func() {
		ui.Out = os.Stdout
		ui.Err = os.Stderr
	})

	// Manually add a KindFile entry to the store.
	keys := []envcrypto.PublicKey{pub}
	ids := []string{"alice@example.com"}
	entry, err := sealEntry("secret.pem", vault.KindFile, []byte("pem-content"), keys, ids)
	if err != nil {
		t.Fatalf("sealEntry file: %v", err)
	}
	store, _ := vault.LoadStore(root)
	store = store.Upsert(entry)
	if err := vault.SaveStore(root, store); err != nil {
		t.Fatalf("SaveStore: %v", err)
	}

	// Output of env should not include secret.pem as a var name.
	var childOut bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runErr := runRun(root, []string{"env"}, kc)

	_ = w.Close()
	_, _ = childOut.ReadFrom(r)
	os.Stdout = oldStdout

	if runErr != nil {
		t.Fatalf("runRun: %v", runErr)
	}
	if strings.Contains(childOut.String(), "secret.pem") {
		t.Error("KindFile entry was injected as env var — it must not be")
	}
}

func TestRunRun_InfoOutput(t *testing.T) {
	root := initVaultRoot(t)
	priv := addTestRecipient(t, root, "alice@example.com")

	kc := newMemStore()
	if err := kc.Seal("alice@example.com", priv[:]); err != nil {
		t.Fatalf("kc.Seal: %v", err)
	}

	var outBuf bytes.Buffer
	ui.Out = &outBuf
	ui.Err = &bytes.Buffer{}
	t.Cleanup(func() {
		ui.Out = os.Stdout
		ui.Err = os.Stderr
	})

	if err := runAdd(root, "MY_SECRET", []byte("value")); err != nil {
		t.Fatalf("runAdd: %v", err)
	}

	outBuf.Reset()
	if err := runRun(root, []string{"true"}, kc); err != nil {
		t.Fatalf("runRun: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "decrypting") {
		t.Errorf("missing 'decrypting' in output: %q", out)
	}
	if !strings.Contains(out, "0 bytes written to disk") {
		t.Errorf("missing '0 bytes written to disk' in output: %q", out)
	}
}
