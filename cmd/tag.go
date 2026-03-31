package cmd

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal/output"
	"github.com/JetBrains/youtrack-cli/internal"
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
			ownerLogin, _ := cmd.Flags().GetString("owner")

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			req := api.AddTagRequest{Name: name}
			if ownerLogin != "" {
				u, err := client.GetUser(ownerLogin)
				if err != nil {
					return err
				}
				req.Owner = &api.User{ID: u.ID}
			}

			tag, err := client.AddTag(req)
			if err != nil {
				return err
			}
			return output.Print(cmd, tag)
		},
	}
	cmd.Flags().String("name", "", "Tag name (required)")
	cmd.Flags().String("owner", "", "Owner login")
	cmd.MarkFlagRequired("name")
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
			return client.DeleteTag(args[0])
		},
	}
}
