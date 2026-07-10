package user

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func newUserTimezoneCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "timezone", Short: "Manage user timezone"}
	internal.RequireSubcommand(cmd)
	view := &cobra.Command{
		Use: "view <userId>", Aliases: []string{"get"}, Short: "View user timezone",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			tz, err := client.GetUserTimezone(args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), tz)
			return err
		},
	}
	set := &cobra.Command{
		Use: "set <userId> <timezoneId>", Short: "Set user timezone",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.SetUserTimezone(args[0], args[1])
		},
	}
	cmd.AddCommand(view, set)
	return cmd
}
