package project

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func newProjectTimetrackCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "timetrack", Short: "Manage project time tracking settings"}
	set := &cobra.Command{
		Use: "set <shortName>", Short: "Enable or disable time tracking for a project",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, _ := cmd.Flags().GetBool("enabled")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.SetProjectTimeTracking(args[0], enabled); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "updated time tracking for project "+args[0])
			return nil
		},
	}
	set.Flags().Bool("enabled", true, "Enable time tracking")
	cmd.AddCommand(set)
	return cmd
}
