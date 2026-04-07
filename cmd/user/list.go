package user

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newUserListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			users, err := client.ListUsers(query)
			if err != nil {
				return err
			}
			return output.Print(cmd, users)
		},
	}
	cmd.Flags().String("query", "", "Filter users by query")
	return cmd
}
