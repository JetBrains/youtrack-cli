package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage tags",
	}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(newTagListCmd())
	cmd.AddCommand(newTagCreateCmd())
	cmd.AddCommand(newTagDeleteCmd())
	cmd.AddCommand(newTagIssueCmd())
	return cmd
}

func newTagIssueCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "issue", Short: "View issues for a tag"}
	var limit int
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use:   "list <tagId>",
		Short: "List issues tagged with a specific tag",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit < 0 {
				return fmt.Errorf("--limit must not be negative, got %d", limit)
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			issues, err := client.ListTagIssues(args[0], limit)
			if err != nil {
				return err
			}
			return output.Print(cmd, issues)
		},
	}
	list.Flags().IntVar(&limit, "limit", 100, "Maximum number of issues to return (non-negative; 0 returns none)")
	cmd.AddCommand(list)
	return cmd
}

func newTagListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			tags, err := client.ListTags()
			if err != nil {
				return err
			}
			return output.Print(cmd, tags)
		},
	}
}

func newTagCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"add"},
		Short:   "Create a new tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			tag, err := client.AddTag(api.AddTagRequest{Name: name})
			if err != nil {
				return err
			}
			return output.Print(cmd, tag)
		},
	}
	cmd.Flags().String("name", "", "Tag name (required)")
	internal.MustMarkRequired(cmd, "name")
	return cmd
}

func newTagDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a tag by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete tag %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteTag(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted tag "+args[0])
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}
