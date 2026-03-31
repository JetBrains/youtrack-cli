package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/JetBrains/youtrack-cli/internal"
)

func newArticleCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "article", Short: "Manage knowledge base articles"}
	cmd.AddCommand(
		newArticleListCmd(),
		newArticleGetCmd(),
		newArticleAddCmd(),
		newArticleUpdateCmd(),
		newArticleDeleteCmd(),
		newArticleCommentCmd(),
		newArticleAttachmentCmd(),
		newArticleTagCmd(),
		newArticleVisibilityCmd(),
		newArticleSetParentCmd(),
	)
	return cmd
}

func newArticleListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "list", Short: "List articles",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			articles, err := client.ListArticles(query)
			if err != nil {
				return err
			}
			return output.Print(cmd, articles)
		},
	}
	cmd.Flags().String("query", "", "Filter query")
	return cmd
}

func newArticleGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <id>", Short: "Get an article by ID",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			article, err := client.GetArticle(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, article)
		},
	}
}

func newArticleAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "add", Short: "Create a new article",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectKey, _ := cmd.Flags().GetString("project")
			summary, _ := cmd.Flags().GetString("summary")
			content, _ := cmd.Flags().GetString("content")
			asUser, _ := cmd.Flags().GetString("asUser")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			proj, err := client.GetProject(projectKey)
			if err != nil {
				return err
			}
			reporterID := ""
			if asUser != "" {
				u, err := client.GetUser(asUser)
				if err != nil {
					return err
				}
				reporterID = u.ID
			}
			article, err := client.AddArticle(proj.ID, summary, content, reporterID)
			if err != nil {
				return err
			}
			return output.Print(cmd, article)
		},
	}
	cmd.Flags().String("project", "", "Project short name (required)")
	cmd.Flags().String("summary", "", "Article title (required)")
	cmd.Flags().String("content", "", "Article body")
	cmd.Flags().String("asUser", "", "Create as this user login")
	cmd.MarkFlagRequired("project")
	cmd.MarkFlagRequired("summary")
	return cmd
}

func newArticleUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "update <id>", Short: "Update an article",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, _ := cmd.Flags().GetString("summary")
			content, _ := cmd.Flags().GetString("content")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			article, err := client.UpdateArticle(args[0], summary, content)
			if err != nil {
				return err
			}
			return output.Print(cmd, article)
		},
	}
	cmd.Flags().String("summary", "", "New title")
	cmd.Flags().String("content", "", "New content")
	return cmd
}

func newArticleDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete an article",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteArticle(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "deleted article "+args[0])
			return nil
		},
	}
}

func newArticleCommentCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "comment", Short: "Manage article comments"}
	list := &cobra.Command{
		Use: "list <articleId>", Short: "List comments on an article",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			comments, err := client.ListArticleComments(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, comments)
		},
	}
	add := &cobra.Command{
		Use: "add <articleId>", Short: "Add a comment to an article",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text, _ := cmd.Flags().GetString("text")
			asUser, _ := cmd.Flags().GetString("asUser")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			authorID := ""
			if asUser != "" {
				u, err := client.GetUser(asUser)
				if err != nil {
					return err
				}
				authorID = u.ID
			}
			comment, err := client.AddArticleComment(args[0], text, authorID)
			if err != nil {
				return err
			}
			return output.Print(cmd, comment)
		},
	}
	add.Flags().String("text", "", "Comment text (required)")
	add.Flags().String("asUser", "", "Post as this user login")
	add.MarkFlagRequired("text")
	del := &cobra.Command{
		Use: "delete <articleId> <commentId>", Short: "Delete a comment from an article",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.DeleteArticleComment(args[0], args[1])
		},
	}
	update := &cobra.Command{
		Use: "update <articleId> <commentId>", Short: "Update a comment on an article",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			text, _ := cmd.Flags().GetString("text")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			comment, err := client.UpdateArticleComment(args[0], args[1], text)
			if err != nil {
				return err
			}
			return output.Print(cmd, comment)
		},
	}
	update.Flags().String("text", "", "New comment text (required)")
	update.MarkFlagRequired("text")
	cmd.AddCommand(list, add, del, update)
	return cmd
}

func newArticleTagCmd() *cobra.Command {
	return &cobra.Command{
		Use: "tag <articleId> <tagId>", Short: "Add a tag to an article",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.TagArticle(args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "tagged article "+args[0])
			return nil
		},
	}
}

func newArticleVisibilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "visibility <articleId>", Short: "Set visibility on an article",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			groups, _ := cmd.Flags().GetStringSlice("groups")
			users, _ := cmd.Flags().GetStringSlice("users")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.UpdateArticleVisibility(args[0], groups, users); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "updated visibility for article "+args[0])
			return nil
		},
	}
	cmd.Flags().StringSlice("groups", nil, "Permitted group IDs (comma-separated)")
	cmd.Flags().StringSlice("users", nil, "Permitted user IDs (comma-separated)")
	return cmd
}

func newArticleSetParentCmd() *cobra.Command {
	return &cobra.Command{
		Use: "set-parent <articleId> <parentId>", Short: "Set the parent article",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.SetParentArticle(args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "set parent of "+args[0]+" to "+args[1])
			return nil
		},
	}
}

func newArticleAttachmentCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "attachment", Short: "Manage article attachments"}
	list := &cobra.Command{
		Use: "list <articleId>", Short: "List attachments on an article",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			attachments, err := client.ListArticleAttachments(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, attachments)
		},
	}
	add := &cobra.Command{
		Use: "add <articleId>", Short: "Attach a file to an article",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			attachment, err := client.AddArticleAttachment(args[0], filePath)
			if err != nil {
				return err
			}
			return output.Print(cmd, attachment)
		},
	}
	add.Flags().String("file", "", "Path to file (required)")
	add.MarkFlagRequired("file")
	cmd.AddCommand(list, add)
	return cmd
}

