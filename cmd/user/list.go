package user

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newUserListCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			users, err := client.ListUsers(query, limit)
			if err != nil {
				return err
			}
			if len(users) == 0 {
				output.PrintTip(cmd, "No users found — try a broader --query")
			}
			return output.Print(cmd, users)
		},
	}
	cmd.Flags().String("query", "", "Filter users by query")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of users to return")
	return cmd
}
