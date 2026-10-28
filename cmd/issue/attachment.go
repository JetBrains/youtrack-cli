package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueAttachmentCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "attachment", Short: "Manage issue attachments"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(
		newIssueAttachmentListCmd(),
		newIssueAttachmentAddCmd(),
		newIssueAttachmentDeleteCmd(),
		newIssueAttachmentVisibilityCmd(),
	)
	return cmd
}

func newIssueAttachmentListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list <issueId>", Short: "List attachments on an issue",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			attachments, err := client.GetIssueAttachments(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, attachments)
		},
	}
}

func newIssueAttachmentAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "add <issueId>", Short: "Attach a file to an issue",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			attachment, err := client.AddIssueAttachment(args[0], filePath)
			if err != nil {
				return err
			}
			return output.Print(cmd, attachment)
		},
	}
	cmd.Flags().String("file", "", "Path to file (required)")
	internal.MustMarkRequired(cmd, "file")
	return cmd
}

func newIssueAttachmentDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "delete <issueId> <attachmentId>", Short: "Delete an attachment from an issue",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"Delete attachment %s from issue %s?", args[1], args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteIssueAttachment(args[0], args[1]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted attachment "+args[1])
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

func newIssueAttachmentVisibilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "visibility <issueId> <attachmentId>",
		Short: "Set who can see an attachment",
		Long:  visibilityLong("attachment"),
		Example: "  youtrack issue attachment visibility FOO-1 12-8 --users alice\n" +
			"  youtrack issue attachment visibility FOO-1 12-8 --public",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetVisibility(cmd, "attachment "+args[1], func(c *api.Client, v api.VisibilityChange) (*api.Visibility, string, error) {
				got, err := c.SetAttachmentVisibility(args[0], args[1], v)
				return got, "", err
			})
		},
	}
	addSetVisibilityFlags(cmd)
	return cmd
}
