package main

import "github.com/spf13/cobra"

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Push the encrypted vault to the Git remote",
		RunE:  stubRun,
	}
}
