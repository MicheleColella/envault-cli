package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MicheleColella/envault-cli/internal/hook"
	"github.com/MicheleColella/envault-cli/internal/ui"
	"github.com/MicheleColella/envault-cli/internal/vault"
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
	var global bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install integration hooks",
		Long: "Install hooks that integrate Envault into your development workflow.\n\n" +
			"  envault hook install --git                        install the Git pre-commit hook\n" +
			"  envault hook install --git --uninstall            remove the Git pre-commit hook\n" +
			"  envault hook install --claude                     install the Claude Code hook (project-local)\n" +
			"  envault hook install --claude --global            install the Claude Code hook globally\n" +
			"  envault hook install --claude --uninstall         remove the project-local Claude Code hook\n" +
			"  envault hook install --claude --global --uninstall  remove the global Claude Code hook",
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
				return runHookUninstallClaude(wd, global)
			}
			return runHookInstallClaude(wd, global)
		},
	}

	cmd.Flags().BoolVar(&gitHook, "git", false, "target the Git pre-commit hook")
	cmd.Flags().BoolVar(&claudeHook, "claude", false, "target the Claude Code PreToolUse hook")
	cmd.Flags().BoolVar(&uninstall, "uninstall", false, "remove the hook instead of installing it")
	cmd.Flags().BoolVar(&global, "global", false, "write to ~/.claude/settings.json instead of the project-local .claude/settings.json")
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

func runHookInstallClaude(repoRoot string, global bool) error {
	if !global && !vault.IsInitialized(repoRoot) {
		return fmt.Errorf("no Envault vault found — run 'envault init' first, or use --global to install for all Claude Code sessions")
	}

	alreadyInstalled := hook.IsClaudeHookInstalled(repoRoot, global)

	if err := hook.InstallClaudeHook(repoRoot, global); err != nil {
		return err
	}

	if !global {
		// Best-effort: inject the Envault section into the project CLAUDE.md.
		_ = hook.InjectClaudeMD(repoRoot)
	}

	if alreadyInstalled {
		ui.Info("Claude Code hook already installed")
		return nil
	}

	if global {
		ui.OK("Claude Code hook installed globally (~/.claude/settings.json)")
	} else {
		ui.OK("Claude Code hook installed (.claude/settings.json)")
		ui.Info("Envault section added to CLAUDE.md")
	}
	ui.Info("Intercepts Bash tool calls: blocks envault cat/export to prevent plaintext in model context")
	ui.Info("Use --force with cat/export to explicitly override; use `envault run` for in-memory injection")
	if global {
		ui.Info("Remove with: envault hook install --claude --global --uninstall")
	} else {
		ui.Info("Remove with: envault hook install --claude --uninstall")
	}
	ui.Info("Tip: also set ENVAULT_PASSPHRASE in your Claude Code session env for non-interactive unlock")
	return nil
}

func runHookUninstallClaude(repoRoot string, global bool) error {
	if err := hook.UninstallClaudeHook(repoRoot, global); err != nil {
		return err
	}
	if !global {
		_ = hook.RemoveClaudeMDSection(repoRoot)
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
