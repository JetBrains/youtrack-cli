package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueTagCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "tag", Short: "Manage tags on an issue"}
	add := &cobra.Command{
		Use:   "add <issueId> <tagId>",
		Short: "Add a tag to an issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.TagIssue(args[0], args[1]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "tagged issue "+args[0]+" with "+args[1])
			return nil
		},
	}
	list := &cobra.Command{
		Use:   "list <issueId>",
		Short: "List tags on an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			tags, err := client.ListIssueTags(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, tags)
		},
	}
	get := &cobra.Command{
		Use:   "get <issueId> <tagId>",
		Short: "Get a specific tag on an issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			tag, err := client.GetIssueTag(args[0], args[1])
			if err != nil {
				return err
			}
			return output.Print(cmd, tag)
		},
	}
	cmd.AddCommand(add, list, get)
	return cmd
}
