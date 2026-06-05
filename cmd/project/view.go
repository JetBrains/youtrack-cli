package project

import "github.com/spf13/cobra"

func newProjectViewCmd() *cobra.Command {
	cmd := newProjectGetCmd()
	cmd.Use = "view <shortName>"
	cmd.Short = "View a project by short name"
	return cmd
}
