package cmd

import (
	"fmt"
	"youtrack-cli/api"
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newIssueCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage issue comments",
	}
	cmd.AddCommand(newCommentListCmd())
	cmd.AddCommand(newCommentAddCmd())
	cmd.AddCommand(newCommentDeleteCmd())
	return cmd
}

func newCommentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <issueId>",
		Short: "List comments on an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
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

			client, err := getClient(cmd)
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
	cmd.MarkFlagRequired("text")
	return cmd
}

func newCommentDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <issueId> <commentId>",
		Short: "Delete a comment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteComment(args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "deleted comment "+args[1])
			return nil
		},
	}
}
