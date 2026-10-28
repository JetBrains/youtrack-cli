package issue

import (
	"fmt"
	"slices"

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
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(newCommentListCmd())
	cmd.AddCommand(newCommentViewCmd())
	cmd.AddCommand(newCommentCreateCmd())
	cmd.AddCommand(newCommentDeleteCmd())
	cmd.AddCommand(newCommentUpdateCmd())
	cmd.AddCommand(newCommentVisibilityCmd())
	cmd.AddCommand(newCommentPinCmd())
	return cmd
}

func newCommentListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <issueId>",
		Short: "List comments on an issue",
		Long:  "List comments on an issue with their pinned state and visibility. Deleted comments are left out unless --include-deleted is passed.",
		Example: "  youtrack issue comment list FOO-42\n" +
			"  youtrack issue comment list FOO-42 --include-deleted --fields id,deleted,pinned,text",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			includeDeleted, _ := cmd.Flags().GetBool("include-deleted")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			comments, err := client.ListComments(args[0])
			if err != nil {
				return err
			}
			if !includeDeleted {
				comments = slices.DeleteFunc(comments, func(c api.Comment) bool { return c.Deleted })
			}
			return output.Print(cmd, comments)
		},
	}
	cmd.Flags().Bool("include-deleted", false, "Include deleted comments, marked as deleted")
	return cmd
}

func newCommentCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create <issueId>",
		Aliases: []string{"add"},
		Short:   "Create a comment on an issue",
		Example: "  youtrack issue comment create FOO-1 --text \"Repro steps attached\"\n" +
			"  youtrack issue comment create FOO-1 --text \"Internal note\" --visibility \"Core Team\" --visibility alice",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text, _, err := textFromFlags(cmd, "text", "text-file")
			if err != nil {
				return err
			}
			asUser, _ := cmd.Flags().GetString("asUser")

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			visibility, err := visibilityFromFlags(cmd, client)
			if err != nil {
				return err
			}
			req := api.AddCommentRequest{Text: text, Visibility: visibility}

			if asUser != "" {
				userID, err := resolveUserID(client, asUser)
				if err != nil {
					return err
				}
				req.Author = &api.User{ID: userID}
			}

			comment, err := client.AddComment(args[0], req)
			if err != nil {
				return err
			}
			if err := checkCreatedVisibility(visibility, comment.Visibility, "", "comment "+comment.ID); err != nil {
				return err
			}
			return output.Print(cmd, comment)
		},
	}
	cmd.Flags().String("text", "", "Comment text")
	addTextFileFlag(cmd, "text", "text-file", "the comment text")
	cmd.MarkFlagsOneRequired("text", "text-file")
	cmd.Flags().String("asUser", "", "Post as this user (login)")
	addVisibilityFlags(cmd, "comment")
	return cmd
}

func newCommentDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <issueId> <commentId>",
		Short: "Delete a comment",
		Long:  "Mark a comment as deleted, matching the recoverable deletion in the YouTrack web UI.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete comment %s?", args[1])); abort {
				return err
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
			text, _, err := textFromFlags(cmd, "text", "text-file")
			if err != nil {
				return err
			}
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
	cmd.Flags().String("text", "", "New comment text")
	addTextFileFlag(cmd, "text", "text-file", "the new comment text")
	cmd.MarkFlagsOneRequired("text", "text-file")
	return cmd
}

func newCommentVisibilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "visibility <issueId> <commentId>",
		Short: "Set who can see a comment",
		Long:  visibilityLong("comment"),
		Example: "  youtrack issue comment visibility FOO-1 4-17 --groups \"Core Team\"\n" +
			"  youtrack issue comment visibility FOO-1 4-17 --public",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetVisibility(cmd, "comment "+args[1], func(c *api.Client, v api.VisibilityChange) (*api.Visibility, string, error) {
				got, err := c.SetCommentVisibility(args[0], args[1], v)
				return got, "", err
			})
		},
	}
	addSetVisibilityFlags(cmd)
	return cmd
}

func newCommentViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <issueId> <commentId>",
		Aliases: []string{"get"},
		Short:   "View a comment by ID",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			comment, err := client.GetComment(args[0], args[1])
			if err != nil {
				return err
			}
			return output.Print(cmd, comment)
		},
	}
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
