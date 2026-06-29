package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MicheleColella/envault-cli/internal/hook"
	"github.com/MicheleColella/envault-cli/internal/ui"
)

func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Install or remove Git and Claude Code hooks",
	}
	cmd.AddCommand(newHookInstallCmd())
	cmd.AddCommand(newHookPreuseCmd())
	return cmd
}

func newHookInstallCmd() *cobra.Command {
	var gitHook bool
	var claudeHook bool
	var uninstall bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install integration hooks",
		Long: "Install hooks that integrate Envault into your development workflow.\n\n" +
			"  envault hook install --git              install the Git pre-commit hook\n" +
			"  envault hook install --git --uninstall  remove the Git pre-commit hook\n" +
			"  envault hook install --claude           install the Claude Code PreToolUse hook\n" +
			"  envault hook install --claude --uninstall  remove the Claude Code hook",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !gitHook && !claudeHook {
				return fmt.Errorf("specify --git or --claude to select the hook type")
			}
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			if gitHook {
				if uninstall {
					return runHookUninstallGit(wd)
				}
				return runHookInstallGit(wd)
			}
			// --claude
			if uninstall {
				return runHookUninstallClaude(wd)
			}
			return runHookInstallClaude(wd)
		},
	}

	cmd.Flags().BoolVar(&gitHook, "git", false, "target the Git pre-commit hook")
	cmd.Flags().BoolVar(&claudeHook, "claude", false, "target the Claude Code PreToolUse hook")
	cmd.Flags().BoolVar(&uninstall, "uninstall", false, "remove the hook instead of installing it")
	return cmd
}

func runHookInstallGit(repoRoot string) error {
	alreadyInstalled := hook.IsGitHookInstalled(repoRoot)

	if err := hook.InstallGitHook(repoRoot); err != nil {
		return err
	}

	if alreadyInstalled {
		ui.Info("Git pre-commit hook already installed")
		return nil
	}

	ui.OK("Git pre-commit hook installed (.git/hooks/pre-commit)")
	ui.Info("Scans staged diff via envault scan (12+ patterns, entropy detection, .envaultignore)")
	ui.Info("Remove with: envault hook install --git --uninstall")
	return nil
}

func runHookUninstallGit(repoRoot string) error {
	if err := hook.UninstallGitHook(repoRoot); err != nil {
		return err
	}
	ui.OK("Git pre-commit hook removed")
	return nil
}

func runHookInstallClaude(repoRoot string) error {
	alreadyInstalled := hook.IsClaudeHookInstalled(repoRoot)

	if err := hook.InstallClaudeHook(repoRoot); err != nil {
		return err
	}

	if alreadyInstalled {
		ui.Info("Claude Code hook already installed")
		return nil
	}

	ui.OK("Claude Code hook installed (.claude/settings.json)")
	ui.Info("Intercepts Bash tool calls: blocks envault cat/export to prevent plaintext in model context")
	ui.Info("Use --force with cat/export to explicitly override; use `envault run` for in-memory injection")
	ui.Info("Remove with: envault hook install --claude --uninstall")
	ui.Info("Tip: also set ENVAULT_PASSPHRASE in your Claude Code session env for non-interactive unlock")
	return nil
}

func runHookUninstallClaude(repoRoot string) error {
	if err := hook.UninstallClaudeHook(repoRoot); err != nil {
		return err
	}
	ui.OK("Claude Code hook removed")
	return nil
}

// newHookPreuseCmd returns the hidden PreToolUse hook handler invoked by Claude Code.
// Claude Code hook protocol: exit 0 = allow, exit non-zero = block.
// Stdout written before exit is displayed to Claude as the reason for blocking.
func newHookPreuseCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "preuse",
		Short:  "Claude Code PreToolUse hook handler (internal)",
		Hidden: true,
		// Suppress Cobra's default error printing — the hook writes its own
		// human-readable message to stdout for Claude to read.
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := hook.RunHookPreuse(os.Stdin, os.Stdout)
			if err != nil {
				// Non-zero exit blocks the tool call in Claude Code.
				// The block message was already written to stdout by RunHookPreuse.
				os.Exit(2)
			}
			return nil
		},
	}
}
