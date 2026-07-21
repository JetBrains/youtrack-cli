package user

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

// draftsTarget defaults to the current user; drafts are private, so only "me" or the
// caller's own login/id are accepted (enforced in the API layer).
func draftsTarget(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "me"
}

func newUserDraftsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "drafts", Short: "Manage your issue and article drafts"}
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use: "issues [userId]", Short: "List your issue drafts (defaults to 'me'; drafts are private to their owner)",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			drafts, err := client.GetUserIssueDrafts(draftsTarget(args))
			if err != nil {
				return err
			}
			return output.Print(cmd, drafts)
		},
	}
	del := &cobra.Command{
		Use: "delete <userId>", Short: "Delete all your issue drafts (pass 'me' or your own login; drafts are private to their owner)",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"Delete ALL issue drafts for user %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.DeleteAllUserDrafts(args[0])
		},
	}
	del.Flags().Bool("force", false, "Skip the confirmation prompt")
	articles := &cobra.Command{
		Use: "articles [userId]", Short: "List your article drafts (defaults to 'me'; drafts are private to their owner)",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			drafts, err := client.GetUserArticleDrafts(draftsTarget(args))
			if err != nil {
				return err
			}
			return output.Print(cmd, drafts)
		},
	}
	cmd.AddCommand(list, del, articles)
	return cmd
}
