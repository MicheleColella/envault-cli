package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MicheleColella/envault-cli/internal/hook"
	"github.com/MicheleColella/envault-cli/internal/keychain"
	"github.com/MicheleColella/envault-cli/internal/ui"
	"github.com/MicheleColella/envault-cli/internal/vault"
)

func newUninstallCmd() *cobra.Command {
	var removeKeys bool
	var global bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove all Envault integrations (hooks, CLAUDE.md); --keys also clears keychain",
		Long: "Reverses every system integration Envault installed, leaving the host as before.\n\n" +
			"Removes: Git pre-commit hook, project Claude Code hook, the Envault CLAUDE.md section.\n" +
			"  --global  also remove the global Claude Code hook (~/.claude/settings.json)\n" +
			"  --keys    also delete this machine's vault keys from the OS keychain (IRREVERSIBLE)\n\n" +
			"Idempotent and safe to run repeatedly. Does not delete the envault binary —\n" +
			"run scripts/install.sh --uninstall (or rm the binary) for that.",
		RunE: func(_ *cobra.Command, _ []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			return runUninstall(wd, removeKeys, global)
		},
	}
	cmd.Flags().BoolVar(&removeKeys, "keys", false, "also delete this machine's vault keys from the keychain (irreversible)")
	cmd.Flags().BoolVar(&global, "global", false, "also remove the global Claude Code hook")
	return cmd
}

// runUninstall removes every Envault integration. Each step is guarded so the
// command is idempotent: re-running it on an already-clean host is a no-op.
func runUninstall(repoRoot string, removeKeys, global bool) error {
	var removed []string

	if hook.IsGitHookInstalled(repoRoot) {
		if err := hook.UninstallGitHook(repoRoot); err != nil {
			return fmt.Errorf("remove git hook: %w", err)
		}
		removed = append(removed, "Git pre-commit hook")
	}

	if hook.IsClaudeHookInstalled(repoRoot, false) {
		if err := hook.UninstallClaudeHook(repoRoot, false); err != nil {
			return fmt.Errorf("remove project Claude hook: %w", err)
		}
		removed = append(removed, "Claude Code hook (project)")
	}

	if hook.IsClaudeMDInjected(repoRoot) {
		if err := hook.RemoveClaudeMDSection(repoRoot); err != nil {
			return fmt.Errorf("remove CLAUDE.md section: %w", err)
		}
		removed = append(removed, "CLAUDE.md section")
	}

	if global && hook.IsClaudeHookInstalled(repoRoot, true) {
		if err := hook.UninstallClaudeHook(repoRoot, true); err != nil {
			return fmt.Errorf("remove global Claude hook: %w", err)
		}
		removed = append(removed, "Claude Code hook (global)")
	}

	if removeKeys {
		n, err := removeVaultKeys(repoRoot)
		if err != nil {
			ui.Warn("could not clear keychain: " + err.Error())
		} else if n > 0 {
			removed = append(removed, fmt.Sprintf("%d keychain key(s)", n))
		}
	}

	if len(removed) == 0 {
		ui.OK("Nothing to remove — host is already clean")
		return nil
	}
	for _, r := range removed {
		ui.Info("removed " + r)
	}
	ui.OK("Envault integrations removed")
	ui.Info("Binary not deleted — run scripts/install.sh --uninstall or rm it manually")
	return nil
}

// removeVaultKeys deletes from the OS keychain any vault recipient key that is
// present on this machine. Uses the raw keychain (no passphrase needed to
// delete) and ignores ErrNotFound for recipients whose keys live elsewhere.
func removeVaultKeys(repoRoot string) (int, error) {
	if !vault.IsInitialized(repoRoot) {
		return 0, nil
	}
	recipients, err := vault.ListRecipients(repoRoot)
	if err != nil {
		return 0, err
	}
	kc, err := keychain.New()
	if err != nil {
		return 0, err
	}
	var n int
	for _, r := range recipients {
		if err := kc.Delete(r.ID); err == nil {
			n++
		} else if !errors.Is(err, keychain.ErrNotFound) {
			return n, err
		}
	}
	return n, nil
}
