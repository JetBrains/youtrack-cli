package user

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newUserViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <login>",
		Aliases: []string{"get"},
		Short:   "View a user by login",
		Example: "  youtrack user view john.doe",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			user, err := client.GetUser(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, user)
		},
	}
}
