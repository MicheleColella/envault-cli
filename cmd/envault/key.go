package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/spf13/cobra"

	envcrypto "github.com/MicheleColella/envault-cli/internal/crypto"
	"github.com/MicheleColella/envault-cli/internal/keychain"
	"github.com/MicheleColella/envault-cli/internal/ui"
)

func newKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage your X25519 identity key",
	}
	cmd.AddCommand(newKeyNewCmd())
	return cmd
}

func newKeyNewCmd() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Generate a new X25519 keypair and seal the private key in the OS keychain",
		RunE: func(cmd *cobra.Command, _ []string) error {
			kc, err := keychain.New()
			if err != nil {
				return err
			}
			return runKeyNew(id, kc)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "identity for this keypair, e.g. alice@example.com")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

// runKeyNew is the testable core of the key new command.
func runKeyNew(id string, kc keychain.Store) error {
	priv, pub, err := envcrypto.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate keypair: %w", err)
	}
	defer clear(priv[:])

	if err := kc.Seal(id, priv[:]); err != nil {
		return err
	}

	ui.OK(fmt.Sprintf("Key generated for %s", id))
	ui.Info(fmt.Sprintf("fingerprint  %s", pubKeyFingerprint(pub)))
	ui.Info("cipher       X25519 → AES-256-GCM")
	ui.Info("private key  sealed in OS keychain (never written to disk)")
	return nil
}

func pubKeyFingerprint(pub envcrypto.PublicKey) string {
	h := sha256.Sum256(pub[:])
	return "sha256:" + hex.EncodeToString(h[:])
}
