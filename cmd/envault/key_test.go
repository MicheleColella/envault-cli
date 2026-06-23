package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/MicheleColella/envault-cli/internal/keychain"
	"github.com/MicheleColella/envault-cli/internal/ui"
)

// memStore is an in-memory keychain.Store used in tests.
type memStore struct {
	keys map[string][]byte
}

func newMemStore() *memStore { return &memStore{keys: make(map[string][]byte)} }

func (m *memStore) Seal(id string, key []byte) error {
	if _, ok := m.keys[id]; ok {
		return fmt.Errorf("%w: %s", keychain.ErrAlreadyExists, id)
	}
	cp := make([]byte, len(key))
	copy(cp, key)
	m.keys[id] = cp
	return nil
}

func (m *memStore) Unseal(id string) ([]byte, error) {
	k, ok := m.keys[id]
	if !ok {
		return nil, keychain.ErrNotFound
	}
	return k, nil
}

func (m *memStore) Delete(id string) error {
	if _, ok := m.keys[id]; !ok {
		return keychain.ErrNotFound
	}
	delete(m.keys, id)
	return nil
}

func TestRunKeyNew_SealsPrivateKey(t *testing.T) {
	kc := newMemStore()

	var out bytes.Buffer
	ui.Out = &out
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runKeyNew("alice@example.com", kc); err != nil {
		t.Fatalf("runKeyNew: %v", err)
	}

	sealed, err := kc.Unseal("alice@example.com")
	if err != nil {
		t.Fatalf("Unseal: %v", err)
	}
	if len(sealed) != 32 {
		t.Errorf("sealed key length = %d, want 32", len(sealed))
	}
}

func TestRunKeyNew_PrintsExpectedOutput(t *testing.T) {
	kc := newMemStore()

	var out bytes.Buffer
	ui.Out = &out
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runKeyNew("alice@example.com", kc); err != nil {
		t.Fatalf("runKeyNew: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"alice@example.com",
		"sha256:",
		"X25519",
		"AES-256-GCM",
		"never written to disk",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRunKeyNew_FingerprintIsHex(t *testing.T) {
	kc := newMemStore()

	var out bytes.Buffer
	ui.Out = &out
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runKeyNew("bob@example.com", kc); err != nil {
		t.Fatalf("runKeyNew: %v", err)
	}

	got := out.String()
	// Extract the sha256: line and verify it contains 64 hex chars.
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "sha256:") {
			parts := strings.SplitN(line, "sha256:", 2)
			if len(parts) != 2 {
				t.Fatalf("unexpected fingerprint line: %q", line)
			}
			fp := strings.TrimSpace(parts[1])
			if len(fp) != 64 {
				t.Errorf("fingerprint hex length = %d, want 64: %q", len(fp), fp)
			}
			return
		}
	}
	t.Error("no fingerprint line found in output")
}

func TestRunKeyNew_DifferentKeysEachCall(t *testing.T) {
	kc1, kc2 := newMemStore(), newMemStore()

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runKeyNew("alice@example.com", kc1); err != nil {
		t.Fatalf("first runKeyNew: %v", err)
	}
	if err := runKeyNew("alice@example.com", kc2); err != nil {
		t.Fatalf("second runKeyNew: %v", err)
	}

	k1, _ := kc1.Unseal("alice@example.com")
	k2, _ := kc2.Unseal("alice@example.com")
	if bytes.Equal(k1, k2) {
		t.Error("two key generations produced identical private keys")
	}
}

func TestRunKeyNew_AlreadyExists(t *testing.T) {
	kc := newMemStore()

	ui.Out = &bytes.Buffer{}
	t.Cleanup(func() { ui.Out = os.Stdout })

	if err := runKeyNew("alice@example.com", kc); err != nil {
		t.Fatalf("first runKeyNew: %v", err)
	}
	err := runKeyNew("alice@example.com", kc)
	if err == nil {
		t.Fatal("expected error when key already exists")
	}
	if !errors.Is(err, keychain.ErrAlreadyExists) {
		t.Errorf("error = %v, want ErrAlreadyExists", err)
	}
}
