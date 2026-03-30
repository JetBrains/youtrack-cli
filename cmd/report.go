package cmd

import (
	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/spf13/cobra"
)

func newReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Manage reports and removed documents",
	}
	cmd.AddCommand(newReportDeletedListCmd())
	cmd.AddCommand(newReportRestoreCmd())
	cmd.AddCommand(newReportDeleteCmd())
	cmd.AddCommand(newReportGanttListCmd())
	cmd.AddCommand(newReportGanttDeleteCmd())
	return cmd
}

func newReportDeletedListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deleted-list",
		Short: "List removed documents",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			docs, err := client.GetRemovedDocuments()
			if err != nil {
				return err
			}
			return output.Print(cmd, docs)
		},
	}
}

func newReportRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <id>",
		Short: "Restore a removed document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return client.RestoreDocument(args[0])
		},
	}
}

func newReportDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a report by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return client.DeleteReport(args[0])
		},
	}
}

func newReportGanttListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gantt-list",
		Short: "List all Gantt charts",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
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

func newReportGanttDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gantt-delete <id>",
		Short: "Delete a Gantt chart by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return client.DeleteGanttChart(args[0])
		},
	}
}
