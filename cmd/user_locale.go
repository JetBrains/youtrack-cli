package cmd

import (
	"github.com/spf13/cobra"
)

func newUserLocaleCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "locale", Short: "Manage user locale"}
	set := &cobra.Command{
		Use: "set <userId> <locale>", Short: "Set user locale (e.g. en_US)",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return client.SetUserLocale(args[0], args[1])
		},
	}
	cmd.AddCommand(set)
	return cmd
}
