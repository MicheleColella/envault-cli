package main

import "github.com/spf13/cobra"

func newKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "key",
		Short: "Manage your X25519 identity key",
		RunE:  stubRun,
	}
}
