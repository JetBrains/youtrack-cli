package project

import "github.com/spf13/cobra"

func newProjectCreateCmd() *cobra.Command {
	cmd := newProjectAddCmd()
	cmd.Use = "create"
	cmd.Short = "Create a new project"
	return cmd
}
