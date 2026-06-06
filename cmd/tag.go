package cmd

import (
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
	cmd.AddCommand(newTagListCmd())
	cmd.AddCommand(newTagAddCmd())
	cmd.AddCommand(newTagDeleteCmd())
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

func newTagAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new tag",
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
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a tag by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
}
