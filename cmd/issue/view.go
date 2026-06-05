package issue

import "github.com/spf13/cobra"

func newIssueViewCmd() *cobra.Command {
	cmd := newIssueGetCmd()
	cmd.Use = "view <issueId>"
	cmd.Short = "View issue details"
	return cmd
}
