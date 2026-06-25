package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MicheleColella/envault-cli/internal/git"
	"github.com/MicheleColella/envault-cli/internal/ui"
	"github.com/MicheleColella/envault-cli/internal/vault"
)

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Push the encrypted vault to the Git remote",
		RunE: func(cmd *cobra.Command, _ []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			return runPush(wd)
		},
	}
}

func runPush(repoRoot string) error {
	if !vault.IsInitialized(repoRoot) {
		return fmt.Errorf("vault not initialized — run `envault init` first")
	}

	recipients, err := vault.ListRecipients(repoRoot)
	if err != nil {
		return err
	}

	store, err := vault.LoadStore(repoRoot)
	if err != nil {
		return err
	}

	hash, commitErr := git.CommitVault(repoRoot)
	if commitErr != nil && !errors.Is(commitErr, git.ErrNothingToCommit) {
		return commitErr
	}

	if errors.Is(commitErr, git.ErrNothingToCommit) {
		// No new commit — resolve HEAD so we can still report a hash.
		hash, err = git.HeadHash(repoRoot)
		if err != nil {
			hash = ""
		}
	}

	if err := git.PushOrigin(repoRoot); err != nil {
		return err
	}

	ui.OK("Vault pushed")
	ui.Info(fmt.Sprintf("recipients  %d", len(recipients)))
	ui.Info(fmt.Sprintf("secrets     %d", len(store.Entries)))
	if hash != "" {
		ui.Info(fmt.Sprintf("commit      %s", hash))
	}
	ui.Info("GitHub stores ciphertext only — private keys never leave this machine.")
	return nil
}
