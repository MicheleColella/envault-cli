package main

import (
	"fmt"
	"time"

	envcrypto "github.com/MicheleColella/envault-cli/internal/crypto"
	"github.com/MicheleColella/envault-cli/internal/vault"
)

// loadRecipientKeys reads the vault recipients and returns their public keys
// alongside their IDs (positionally aligned). It errors when the vault has no
// recipients, since there would be no one able to decrypt the sealed data.
func loadRecipientKeys(repoRoot string) ([]envcrypto.PublicKey, []string, error) {
	recipients, err := vault.ListRecipients(repoRoot)
	if err != nil {
		return nil, nil, err
	}
	if len(recipients) == 0 {
		return nil, nil, fmt.Errorf(
			"no recipients in %s/recipients — add one with `envault key new` or `envault key import`",
			vault.DirName,
		)
	}

	keys := make([]envcrypto.PublicKey, len(recipients))
	ids := make([]string, len(recipients))
	for i, r := range recipients {
		keys[i] = envcrypto.PublicKey(r.PublicKey)
		ids[i] = r.ID
	}
	return keys, ids, nil
}

// sealEntry encrypts payload for the given recipients with AES-256-GCM and
// builds a metadata-rich vault.Entry. CreatedAt and UpdatedAt are set to now;
// Store.Upsert preserves the original CreatedAt when replacing an entry.
func sealEntry(name string, kind vault.EntryKind, payload []byte, keys []envcrypto.PublicKey, ids []string) (vault.Entry, error) {
	env, err := envcrypto.Seal(payload, keys, envcrypto.AES256GCM)
	if err != nil {
		return vault.Entry{}, err
	}

	now := time.Now().UTC()
	return vault.Entry{
		Name:       name,
		Kind:       kind,
		Algorithm:  envcrypto.AES256GCM,
		Recipients: ids,
		CreatedAt:  now,
		UpdatedAt:  now,
		Envelope:   env,
	}, nil
}
