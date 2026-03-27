package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newProjectDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <shortName>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteProject(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "deleted project "+args[0])
			return nil
		},
	}
}
