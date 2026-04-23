package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage issue comments",
	}
	cmd.AddCommand(newCommentListCmd())
	cmd.AddCommand(newCommentAddCmd())
	cmd.AddCommand(newCommentDeleteCmd())
	cmd.AddCommand(newCommentUpdateCmd())
	cmd.AddCommand(newCommentVisibilityCmd())
	cmd.AddCommand(newCommentPinCmd())
	return cmd
}

func newCommentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <issueId>",
		Short: "List comments on an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			comments, err := client.ListComments(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, comments)
		},
	}
}

func newCommentAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <issueId>",
		Short: "Add a comment to an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text, _ := cmd.Flags().GetString("text")
			asUser, _ := cmd.Flags().GetString("asUser")

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			req := api.AddCommentRequest{Text: text}

			if asUser != "" {
				user, err := client.GetUser(asUser)
				if err != nil {
					return err
				}
				req.Author = &api.User{ID: user.ID}
			}

			comment, err := client.AddComment(args[0], req)
			if err != nil {
				return err
			}
			return output.Print(cmd, comment)
		},
	}
	cmd.Flags().String("text", "", "Comment text (required)")
	cmd.Flags().String("asUser", "", "Post as this user (login)")
	internal.MustMarkRequired(cmd, "text")
	return cmd
}

func newCommentDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <issueId> <commentId>",
		Short: "Delete a comment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			confirmed, err := internal.Confirm(cmd, fmt.Sprintf("Delete comment %s?", args[1]))
			if err != nil {
				return err
			}
			if !confirmed {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
				return nil
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteComment(args[0], args[1]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted comment "+args[1])
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

func newCommentUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <issueId> <commentId>",
		Short: "Update a comment's text",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			text, _ := cmd.Flags().GetString("text")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			comment, err := client.UpdateComment(args[0], args[1], text)
			if err != nil {
				return err
			}
			return output.Print(cmd, comment)
		},
	}
	cmd.Flags().String("text", "", "New comment text (required)")
	internal.MustMarkRequired(cmd, "text")
	return cmd
}

func newCommentVisibilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "visibility <issueId> <commentId>",
		Short: "Set visibility on a comment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			groups, _ := cmd.Flags().GetStringSlice("groups")
			users, _ := cmd.Flags().GetStringSlice("users")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.UpdateCommentVisibility(args[0], args[1], groups, users); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "updated visibility for comment "+args[1])
			return nil
		},
	}
	cmd.Flags().StringSlice("groups", nil, "Permitted group IDs (comma-separated)")
	cmd.Flags().StringSlice("users", nil, "Permitted user IDs (comma-separated)")
	return cmd
}

func newCommentPinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pin <issueId> <commentId>",
		Short: "Pin or unpin a comment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			unpin, _ := cmd.Flags().GetBool("unpin")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.PinComment(args[0], args[1], !unpin); err != nil {
				return err
			}
			action := "pinned"
			if unpin {
				action = "unpinned"
			}
			output.PrintSuccess(cmd, action+" comment "+args[1])
			return nil
		},
	}
	cmd.Flags().Bool("unpin", false, "Unpin the comment instead")
	return cmd
}
