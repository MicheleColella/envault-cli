package main

import "github.com/spf13/cobra"

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Envault vault in the current repository",
		RunE:  stubRun,
	}
}
