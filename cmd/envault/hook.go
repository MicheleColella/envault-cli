package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

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
	ui.Info("Intercepts Bash tool calls: sets CLAUDE_CODE=1 so envault operates in agent mode")
	ui.Info("In agent mode: plaintext output is suppressed; all status is structured JSON")
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

// errBlockToolCall is returned by runHookPreuse when the Bash command must be
// blocked. The caller exits non-zero, which Claude Code interprets as "deny this
// tool call" — any text written to stdout before the exit is shown to Claude.
var errBlockToolCall = fmt.Errorf("tool call blocked by envault hook")

// newHookPreuseCmd returns the hidden PreToolUse hook handler.
// It is invoked by Claude Code's PreToolUse hook (via settings.json).
// Claude Code hook protocol: exit 0 = allow, exit non-zero = block.
// Stdout written before exit is displayed to Claude as the reason.
func newHookPreuseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "preuse",
		Short:  "Claude Code PreToolUse hook handler (internal)",
		Hidden: true,
		// Suppress Cobra's default error printing — the hook writes its own
		// human-readable message to stdout for Claude to read.
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := runHookPreuse(os.Stdin, os.Stdout)
			if err != nil {
				// Non-zero exit blocks the tool call in Claude Code.
				// The message was already written to stdout by runHookPreuse.
				os.Exit(2)
			}
			return nil
		},
	}
	return cmd
}

// preuseInput is the subset of the Claude Code PreToolUse hook JSON we care about.
type preuseInput struct {
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

// runHookPreuse reads Claude Code's PreToolUse JSON from r.
// When a Bash command in an envault repo calls `envault cat` or `envault export`
// without --force, it writes a human-readable block reason to w and returns
// errBlockToolCall. The caller must then exit non-zero so Claude Code denies the
// tool use and shows the message to Claude instead.
// For all other commands it returns nil (tool call is allowed unchanged).
func runHookPreuse(r io.Reader, w io.Writer) error {
	var input preuseInput
	if err := json.NewDecoder(r).Decode(&input); err != nil {
		return nil // non-fatal: allow the tool call unchanged
	}

	if input.ToolName != "Bash" {
		return nil
	}

	cmd, _ := input.ToolInput["command"].(string)
	if cmd == "" {
		return nil
	}

	// Only intercept when inside an envault repo.
	wd, err := os.Getwd()
	if err != nil || !isEnvaultDir(wd) {
		return nil
	}

	if isSensitiveEnvaultCmd(cmd) {
		_, _ = fmt.Fprintln(w,
			"envault: plaintext output blocked — secrets must not appear in the model context.\n"+
				"Use `envault run -- <cmd>` to inject secrets in-memory into a child process.\n"+
				"If you really need the plaintext value, pass --force to override.",
		)
		return errBlockToolCall
	}

	return nil
}

// isSensitiveEnvaultCmd reports whether cmd invokes `envault cat` or
// `envault export` as the primary command (not as an argument to another tool)
// without the --force override flag.
func isSensitiveEnvaultCmd(cmd string) bool {
	fields := strings.Fields(cmd)

	// Skip leading VAR=value environment assignments (e.g. CLAUDE_CODE=1 envault …)
	start := 0
	for start < len(fields) && strings.ContainsRune(fields[start], '=') {
		start++
	}

	if start >= len(fields) {
		return false
	}

	// Only match when envault is the first executable token.
	first := fields[start]
	if first != "envault" && !strings.HasSuffix(first, "/envault") {
		return false
	}

	if start+1 >= len(fields) {
		return false
	}
	sub := fields[start+1]
	if sub != "cat" && sub != "export" {
		return false
	}

	// Allow explicit --force override anywhere after the subcommand.
	for _, flag := range fields[start:] {
		if flag == "--force" {
			return false
		}
	}
	return true
}

// isEnvaultDir returns true when .envault/ exists under root.
func isEnvaultDir(root string) bool {
	_, err := os.Stat(root + "/.envault")
	return err == nil
}
