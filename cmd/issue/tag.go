package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueTagCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "tag", Short: "Manage tags on an issue"}
	internal.RequireSubcommand(cmd)
	add := &cobra.Command{
		Use:   "add <issueId> <tag>",
		Short: "Add an existing tag to an issue, by tag ID or exact name",
		Long: `Add an existing tag to an issue. <tag> is a tag ID such as 6-12 or the exact,
case-sensitive tag name. A name must match exactly one tag visible to you; the command
never creates a tag.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			tagID, err := client.ResolveTagID(args[1])
			if err != nil {
				return err
			}
			if err := client.TagIssue(args[0], tagID); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "tagged issue "+args[0]+" with "+tagLabel(args[1], tagID))
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
	view := &cobra.Command{
		Use:     "view <issueId> <tagId>",
		Aliases: []string{"get"},
		Short:   "View a specific tag on an issue",
		Args:    cobra.ExactArgs(2),
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
	remove := &cobra.Command{
		Use:   "remove <issueId> <tag>",
		Short: "Remove a tag from an issue, by tag ID or exact name",
		Long: `Remove a tag from an issue. <tag> is a tag ID such as 6-12 or the exact,
case-sensitive name of a tag on the issue.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			tagID, err := client.ResolveIssueTagID(args[0], args[1])
			if err != nil {
				return err
			}
			if err := client.UntagIssue(args[0], tagID); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "removed tag "+tagLabel(args[1], tagID)+" from issue "+args[0])
			return nil
		},
	}
	cmd.AddCommand(add, list, view, remove)
	return cmd
}

func tagLabel(ref, id string) string {
	if ref == id {
		return id
	}
	return fmt.Sprintf("%q (%s)", ref, id)
}
