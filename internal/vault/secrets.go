package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	envcrypto "github.com/MicheleColella/envault-cli/internal/crypto"
)

const secretsFile = "secrets.enc"

// storeVersion is the only supported on-disk revision of the secrets store.
const storeVersion = 1

// EntryKind distinguishes a secret sourced from an env var from an arbitrary file.
type EntryKind string

const (
	// KindEnv is a single environment variable imported from a dotenv file.
	KindEnv EntryKind = "env"
	// KindFile is an arbitrary file (text, JSON, CSV, PEM, binary) stored in the vault.
	KindFile EntryKind = "file"
)

// Entry is one sealed secret or file together with its self-describing metadata:
// the ciphertext carries its own timestamp, algorithm and recipient set.
type Entry struct {
	Name       string                `json:"name"`
	Kind       EntryKind             `json:"kind"`
	Algorithm  envcrypto.CipherSuite `json:"algorithm"`
	Recipients []string              `json:"recipients"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
	Envelope   *envcrypto.Envelope   `json:"envelope"`
}

// Store is the on-disk collection of sealed entries (.envault/secrets.enc).
type Store struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

// LoadStore reads the secrets store from repoRoot. It returns an empty store
// (not an error) when the file does not yet exist.
func LoadStore(repoRoot string) (*Store, error) {
	path := filepath.Join(repoRoot, DirName, secretsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Store{Version: storeVersion}, nil
		}
		return nil, fmt.Errorf("read secrets store: %w", err)
	}

	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse secrets store: %w", err)
	}
	if s.Version != storeVersion {
		return nil, fmt.Errorf("unsupported secrets store version %d", s.Version)
	}
	return &s, nil
}

// SaveStore atomically replaces the secrets store inside repoRoot.
func SaveStore(repoRoot string, s *Store) error {
	path := filepath.Join(repoRoot, DirName, secretsFile)
	tmpPath := path + ".tmp"

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode secrets store: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write secrets temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace secrets store: %w", err)
	}
	return nil
}

// Upsert returns a new Store with e added, or replacing an existing entry that
// has the same name and kind. The original CreatedAt is preserved on replace so
// it records first-seal time, while UpdatedAt reflects the latest seal.
// The receiver is never mutated.
func (s *Store) Upsert(e Entry) *Store {
	entries := make([]Entry, 0, len(s.Entries)+1)
	replaced := false
	for _, existing := range s.Entries {
		if existing.Name == e.Name && existing.Kind == e.Kind {
			e.CreatedAt = existing.CreatedAt
			entries = append(entries, e)
			replaced = true
		} else {
			entries = append(entries, existing)
		}
	}
	if !replaced {
		entries = append(entries, e)
	}
	return &Store{Version: storeVersion, Entries: entries}
}
