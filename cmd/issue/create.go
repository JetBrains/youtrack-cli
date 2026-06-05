package issue

import "github.com/spf13/cobra"

func newIssueCreateCmd() *cobra.Command {
	cmd := newIssueAddCmd()
	cmd.Use = "create"
	cmd.Short = "Create a new issue"
	return cmd
}
