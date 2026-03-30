package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newProjectArchiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "archive <shortName>",
		Short: "Archive a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := client.ArchiveProject(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "archived project "+args[0])
			return nil
		},
	}
}
