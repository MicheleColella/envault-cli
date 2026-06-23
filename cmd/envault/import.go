package main

import "github.com/spf13/cobra"

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Import secrets from a .env file into the vault",
		RunE:  stubRun,
	}
}
