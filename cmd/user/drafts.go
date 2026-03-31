package user

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/JetBrains/youtrack-cli/internal"
)

func newUserDraftsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "drafts", Short: "Manage user issue drafts"}
	list := &cobra.Command{
		Use: "list <userId>", Short: "List issue drafts for a user (use 'me' for current user)",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			drafts, err := client.GetUserIssueDrafts(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, drafts)
		},
	}
	del := &cobra.Command{
		Use: "delete <userId>", Short: "Delete all issue drafts for a user",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.DeleteAllUserDrafts(args[0])
		},
	}
	articles := &cobra.Command{
		Use: "articles <userId>", Short: "List article drafts for a user (use 'me' for current user)",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			drafts, err := client.GetUserArticleDrafts(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, drafts)
		},
	}
	cmd.AddCommand(list, del, articles)
	return cmd
}
