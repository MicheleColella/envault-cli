package main

import "github.com/spf13/cobra"

func newHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hook",
		Short: "Install or remove Git and editor hooks",
		RunE:  stubRun,
	}
}
