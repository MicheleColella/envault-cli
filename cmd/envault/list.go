package main

import "github.com/spf13/cobra"

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List secret names stored in the vault",
		RunE:  stubRun,
	}
}
