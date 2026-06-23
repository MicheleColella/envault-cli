package main

import "github.com/spf13/cobra"

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name> <value>",
		Short: "Add or update a single secret in the vault",
		RunE:  stubRun,
	}
}
