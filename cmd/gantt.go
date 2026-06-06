package cmd

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newGanttCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "gantt", Short: "Manage Gantt charts"}
	cmd.AddCommand(newGanttListCmd())
	cmd.AddCommand(newGanttDeleteCmd())
	return cmd
}

func newGanttListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all Gantt charts",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			charts, err := client.ListGanttCharts()
			if err != nil {
				return err
			}
			return output.Print(cmd, charts)
		},
	}
}

func newGanttDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a Gantt chart by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.DeleteGanttChart(args[0])
		},
	}
}
