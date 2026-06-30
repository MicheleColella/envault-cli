package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MicheleColella/envault-cli/internal/hook"
	"github.com/MicheleColella/envault-cli/internal/protect"
	"github.com/MicheleColella/envault-cli/internal/ui"
	"github.com/MicheleColella/envault-cli/internal/vault"
)

type statusResult struct {
	Initialized          bool `json:"initialized"`
	Recipients           int  `json:"recipients"`
	Secrets              int  `json:"secrets"`
	GitHook              bool `json:"git_hook"`
	ClaudeHook           bool `json:"claude_hook"`
	PrivacyShieldPatterns int  `json:"privacy_shield_patterns"`
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show vault health and integration status",
		RunE: func(_ *cobra.Command, _ []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			return runStatus(wd)
		},
	}
}

func runStatus(repoRoot string) error {
	res := statusResult{
		Initialized: vault.IsInitialized(repoRoot),
		GitHook:     hook.IsGitHookInstalled(repoRoot),
		ClaudeHook:  hook.IsClaudeHookInstalled(repoRoot, false) || hook.IsClaudeHookInstalled(repoRoot, true),
	}

	if res.Initialized {
		if recipients, err := vault.ListRecipients(repoRoot); err == nil {
			res.Recipients = len(recipients)
		}
		if store, err := vault.LoadStore(repoRoot); err == nil {
			res.Secrets = len(store.Entries)
		}
		if patterns, err := protect.LoadPatterns(repoRoot); err == nil {
			res.PrivacyShieldPatterns = len(patterns)
		}
	}

	if ui.AgentMode {
		ui.JSONResult(res)
		return nil
	}

	check := func(ok bool) string {
		if ok {
			return "✓"
		}
		return "✗"
	}

	ui.Header("Envault Status")
	ui.Info(fmt.Sprintf("  Vault initialized        %s", check(res.Initialized)))
	ui.Info(fmt.Sprintf("  Recipients               %d", res.Recipients))
	ui.Info(fmt.Sprintf("  Secrets                  %d", res.Secrets))
	ui.Info(fmt.Sprintf("  Git hook                 %s", check(res.GitHook)))
	ui.Info(fmt.Sprintf("  Claude Code hook         %s", check(res.ClaudeHook)))
	ui.Info(fmt.Sprintf("  Privacy Shield patterns  %d", res.PrivacyShieldPatterns))
	return nil
}
