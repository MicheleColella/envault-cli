package main

import "github.com/spf13/cobra"

func newPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Pull the latest encrypted vault from the Git remote",
		RunE:  stubRun,
	}
}
