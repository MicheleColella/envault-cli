package main

import "github.com/spf13/cobra"

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [flags] -- <command> [args...]",
		Short: "Inject decrypted secrets and execute a command",
		RunE:  stubRun,
	}
}
