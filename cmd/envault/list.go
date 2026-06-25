package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MicheleColella/envault-cli/internal/ui"
	"github.com/MicheleColella/envault-cli/internal/vault"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List names and algorithms of all secrets in the vault",
		RunE: func(cmd *cobra.Command, _ []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			return runList(wd)
		},
	}
}

func runList(repoRoot string) error {
	if !vault.IsInitialized(repoRoot) {
		return fmt.Errorf("vault not initialized — run `envault init` first")
	}

	store, err := vault.LoadStore(repoRoot)
	if err != nil {
		return err
	}

	if len(store.Entries) == 0 {
		ui.Info("vault is empty — use `envault add` or `envault import` to add secrets")
		return nil
	}

	ui.Header(fmt.Sprintf("%d entry(ies) in vault", len(store.Entries)))
	for _, e := range store.Entries {
		ui.Info(fmt.Sprintf("  %-40s  %-8s  %s", e.Name, e.Kind, strings.ToUpper(string(e.Algorithm))))
	}
	return nil
}
