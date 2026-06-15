package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueActivityCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "activity", Short: "View issue activity"}
	list := &cobra.Command{
		Use:   "list <issueId>",
		Short: "List activity items for an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			items, err := client.ListIssueActivities(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, items)
		},
	}
	cmd.AddCommand(list)
	return cmd
}

func newIssueSprintCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "sprint", Short: "View sprints an issue belongs to"}
	list := &cobra.Command{
		Use:   "list <issueId>",
		Short: "List sprints that contain this issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			sprints, err := client.ListIssueSprints(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, sprints)
		},
	}
	cmd.AddCommand(list)
	return cmd
}

func newIssueVCSCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "vcs", Short: "View VCS changes linked to an issue"}
	list := &cobra.Command{
		Use:   "list <issueId>",
		Short: "List VCS changes linked to an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			changes, err := client.ListIssueVCSChanges(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, changes)
		},
	}
	cmd.AddCommand(list)
	return cmd
}
