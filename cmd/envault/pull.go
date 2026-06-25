package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MicheleColella/envault-cli/internal/git"
	"github.com/MicheleColella/envault-cli/internal/ui"
	"github.com/MicheleColella/envault-cli/internal/vault"
)

func newPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Pull the latest encrypted vault from the Git remote",
		RunE: func(cmd *cobra.Command, _ []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			return runPull(wd)
		},
	}
}

func runPull(repoRoot string) error {
	if !vault.IsInitialized(repoRoot) {
		return fmt.Errorf("vault not initialized — run `envault init` first")
	}

	before, err := vault.LoadStore(repoRoot)
	if err != nil {
		return err
	}

	if err := git.FetchOrigin(repoRoot); err != nil {
		return err
	}

	// Remove any untracked .envault/ files (e.g., from a local runInit that was
	// never pushed) so the fast-forward merge is not blocked. Committed files are
	// left untouched; the remote state takes precedence for un-pushed init output.
	if err := git.CleanVault(repoRoot); err != nil {
		return err
	}

	if err := git.MergeOrigin(repoRoot); err != nil {
		return err
	}

	after, err := vault.LoadStore(repoRoot)
	if err != nil {
		return err
	}

	changes := vault.DiffStores(before, after)
	if changes.IsEmpty() {
		ui.OK("Vault is up to date")
		return nil
	}

	ui.OK(fmt.Sprintf("Vault pulled  (%d change(s))", changes.Total()))
	if len(changes.Added) > 0 {
		ui.Info(fmt.Sprintf("added    %s", strings.Join(changes.Added, ", ")))
	}
	if len(changes.Removed) > 0 {
		ui.Info(fmt.Sprintf("removed  %s", strings.Join(changes.Removed, ", ")))
	}
	if len(changes.Rotated) > 0 {
		ui.Info(fmt.Sprintf("rotated  %s", strings.Join(changes.Rotated, ", ")))
	}
	return nil
}
