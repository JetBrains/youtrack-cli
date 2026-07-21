package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newArticleCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "article", Short: "Manage knowledge base articles"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(
		newArticleListCmd(),
		newArticleViewCmd(),
		newArticleCreateCmd(),
		newArticleUpdateCmd(),
		newArticleDeleteCmd(),
		newArticleCommentCmd(),
		newArticleAttachmentCmd(),
		newArticleTagCmd(),
		newArticleVisibilityCmd(),
		newArticleSetParentCmd(),
		newArticleChildCmd(),
		newArticleParentCmd(),
	)
	return cmd
}

func newArticleListCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use: "list", Short: "List articles",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			articles, err := client.ListArticles(query, limit)
			if err != nil {
				return err
			}
			if len(articles) == 0 {
				output.PrintTip(cmd, "No articles found — try a different --query or check your access")
			}
			return output.Print(cmd, articles)
		},
	}
	cmd.Flags().String("query", "", "Filter query")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of articles to return")
	return cmd
}

func newArticleViewCmd() *cobra.Command {
	return &cobra.Command{
		Use: "view <id>", Aliases: []string{"get"}, Short: "View an article by ID",
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

func newArticleCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "create", Aliases: []string{"add"}, Short: "Create a new article",
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
	internal.MustMarkRequired(cmd, "project", "summary")
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
	cmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete an article",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			confirmed, err := internal.Confirm(cmd, fmt.Sprintf("Delete article %s?", args[0]))
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
			if err := client.DeleteArticle(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted article "+args[0])
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

func newArticleCommentCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "comment", Short: "Manage article comments"}
	internal.RequireSubcommand(cmd)
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
		Use: "create <articleId>", Aliases: []string{"add"}, Short: "Create a comment on an article",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text, _ := cmd.Flags().GetString("text")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			comment, err := client.AddArticleComment(args[0], text)
			if err != nil {
				return err
			}
			return output.Print(cmd, comment)
		},
	}
	add.Flags().String("text", "", "Comment text (required)")
	internal.MustMarkRequired(add, "text")
	del := &cobra.Command{
		Use: "delete <articleId> <commentId>", Short: "Delete a comment from an article",
		Args: cobra.ExactArgs(2),
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
			if err := client.DeleteArticleComment(args[0], args[1]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted comment "+args[1])
			return nil
		},
	}
	del.Flags().Bool("force", false, "Skip the confirmation prompt")
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
	internal.MustMarkRequired(update, "text")
	reaction := newArticleCommentReactionCmd()
	cmd.AddCommand(list, add, del, update, reaction)
	return cmd
}

func newArticleCommentReactionCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "reaction", Short: "Manage reactions on article comments"}
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use: "list <articleId> <commentId>", Short: "List reactions on an article comment",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			reactions, err := client.ListArticleCommentReactions(args[0], args[1])
			if err != nil {
				return err
			}
			return output.Print(cmd, reactions)
		},
	}
	add := &cobra.Command{
		Use: "add <articleId> <commentId>", Short: "Add a reaction to an article comment",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			reaction, _ := cmd.Flags().GetString("reaction")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.AddArticleCommentReaction(args[0], args[1], reaction); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "added reaction to comment "+args[1])
			return nil
		},
	}
	add.Flags().String("reaction", "", "Reaction emoji or name (required)")
	internal.MustMarkRequired(add, "reaction")
	cmd.AddCommand(list, add)
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
			output.PrintSuccess(cmd, "tagged article "+args[0])
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
			output.PrintSuccess(cmd, "updated visibility for article "+args[0])
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
			output.PrintSuccess(cmd, "set parent of "+args[0]+" to "+args[1])
			return nil
		},
	}
}

func newArticleAttachmentCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "attachment", Short: "Manage article attachments"}
	internal.RequireSubcommand(cmd)
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
	internal.MustMarkRequired(add, "file")
	cmd.AddCommand(list, add)
	return cmd
}

func newArticleChildCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "child", Short: "Manage child articles"}
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use: "list <articleId>", Short: "List child articles of an article",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			children, err := client.ListArticleChildren(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, children)
		},
	}
	add := &cobra.Command{
		Use: "add <parentId> <childId>", Short: "Set an article as a child of another",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.AddArticleChild(args[0], args[1]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "set "+args[1]+" as child of "+args[0])
			return nil
		},
	}
	cmd.AddCommand(list, add)
	return cmd
}

func newArticleParentCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "parent", Short: "Manage article parent"}
	internal.RequireSubcommand(cmd)
	get := &cobra.Command{
		Use: "view <articleId>", Aliases: []string{"get"}, Short: "View the parent article",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			parent, err := client.GetArticleParent(args[0])
			if err != nil {
				return err
			}
			if parent == nil {
				output.PrintTip(cmd, "article has no parent")
				return nil
			}
			return output.Print(cmd, parent)
		},
	}
	cmd.AddCommand(get)
	return cmd
}
