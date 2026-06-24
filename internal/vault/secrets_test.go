package vault

import (
	"testing"
	"time"

	envcrypto "github.com/MicheleColella/envault-cli/internal/crypto"
)

func sealTestEntry(t *testing.T, name string, kind EntryKind, payload string) Entry {
	t.Helper()
	_, pub, err := envcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	env, err := envcrypto.Seal([]byte(payload), []envcrypto.PublicKey{pub}, envcrypto.AES256GCM)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	now := time.Now().UTC()
	return Entry{
		Name:       name,
		Kind:       kind,
		Algorithm:  envcrypto.AES256GCM,
		Recipients: []string{"alice@example.com"},
		CreatedAt:  now,
		UpdatedAt:  now,
		Envelope:   env,
	}
}

func TestLoadStore_MissingReturnsEmpty(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, "", false); err != nil {
		t.Fatalf("Init: %v", err)
	}

	s, err := LoadStore(root)
	if err != nil {
		t.Fatalf("LoadStore: %v", err)
	}
	if len(s.Entries) != 0 {
		t.Errorf("expected empty store, got %d entries", len(s.Entries))
	}
}

func TestSaveAndLoadStore_RoundTrip(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, "", false); err != nil {
		t.Fatalf("Init: %v", err)
	}

	s, _ := LoadStore(root)
	s = s.Upsert(sealTestEntry(t, "API_KEY", KindEnv, "shh"))
	if err := SaveStore(root, s); err != nil {
		t.Fatalf("SaveStore: %v", err)
	}

	loaded, err := LoadStore(root)
	if err != nil {
		t.Fatalf("LoadStore: %v", err)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(loaded.Entries))
	}
	if loaded.Entries[0].Name != "API_KEY" || loaded.Entries[0].Kind != KindEnv {
		t.Errorf("round-tripped entry = %+v", loaded.Entries[0])
	}
	if loaded.Entries[0].Algorithm != envcrypto.AES256GCM {
		t.Errorf("algorithm = %q, want aes-256-gcm", loaded.Entries[0].Algorithm)
	}
}

func TestUpsert_DoesNotMutateReceiver(t *testing.T) {
	original := &Store{Version: storeVersion}
	original.Upsert(sealTestEntry(t, "X", KindEnv, "v"))
	if len(original.Entries) != 0 {
		t.Errorf("Upsert mutated the receiver: %d entries", len(original.Entries))
	}
}

func TestUpsert_ReplacesSameNameAndKind(t *testing.T) {
	s := &Store{Version: storeVersion}
	s = s.Upsert(sealTestEntry(t, "TOKEN", KindEnv, "old"))
	s = s.Upsert(sealTestEntry(t, "TOKEN", KindEnv, "new"))

	if len(s.Entries) != 1 {
		t.Fatalf("expected replacement, got %d entries", len(s.Entries))
	}
}

func TestUpsert_SameNameDifferentKindCoexist(t *testing.T) {
	s := &Store{Version: storeVersion}
	s = s.Upsert(sealTestEntry(t, "config", KindEnv, "a"))
	s = s.Upsert(sealTestEntry(t, "config", KindFile, "b"))

	if len(s.Entries) != 2 {
		t.Errorf("expected env and file entries to coexist, got %d", len(s.Entries))
	}
}

func TestUpsert_PreservesCreatedAtOnReplace(t *testing.T) {
	s := &Store{Version: storeVersion}
	first := sealTestEntry(t, "TOKEN", KindEnv, "old")
	first.CreatedAt = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	s = s.Upsert(first)

	second := sealTestEntry(t, "TOKEN", KindEnv, "new")
	second.CreatedAt = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	second.UpdatedAt = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	s = s.Upsert(second)

	got := s.Entries[0]
	if !got.CreatedAt.Equal(first.CreatedAt) {
		t.Errorf("CreatedAt = %v, want preserved %v", got.CreatedAt, first.CreatedAt)
	}
	if !got.UpdatedAt.Equal(second.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, second.UpdatedAt)
	}
}
