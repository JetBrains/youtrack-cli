package cmd

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newGanttCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "gantt", Short: "Manage Gantt charts"}
	cmd.AddCommand(
		newGanttListCmd(),
		newGanttUpdateCmd(),
		newGanttDeleteCmd(),
		newGanttMemberCmd(),
		newGanttSettingsCmd(),
	)
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

func newGanttUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a Gantt chart",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			startDate, _ := cmd.Flags().GetInt64("start-date")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			chart, err := client.UpdateGanttChart(args[0], name, startDate)
			if err != nil {
				return err
			}
			return output.Print(cmd, chart)
		},
	}
	cmd.Flags().String("name", "", "New name for the chart")
	cmd.Flags().Int64("start-date", 0, "New start date (Unix ms timestamp)")
	return cmd
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
			if err := client.DeleteGanttChart(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted gantt chart "+args[0])
			return nil
		},
	}
}

func newGanttMemberCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "member", Short: "Manage Gantt chart members"}
	list := &cobra.Command{
		Use:   "list <ganttId>",
		Short: "List members of a Gantt chart",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			members, err := client.ListGanttResources(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, members)
		},
	}
	add := &cobra.Command{
		Use:   "add <ganttId>",
		Short: "Add a user as a member of a Gantt chart",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, _ := cmd.Flags().GetString("user")
			startDate, _ := cmd.Flags().GetInt64("start-date")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			member, err := client.AddGanttResource(args[0], userID, startDate)
			if err != nil {
				return err
			}
			return output.Print(cmd, member)
		},
	}
	add.Flags().String("user", "", "User ID (required)")
	add.Flags().Int64("start-date", 0, "Start date (Unix ms timestamp)")
	internal.MustMarkRequired(add, "user")
	setStart := &cobra.Command{
		Use:   "set-start <ganttId> <resourceId>",
		Short: "Set the start date for a Gantt chart member",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			startDate, _ := cmd.Flags().GetInt64("start-date")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			member, err := client.SetGanttResourceStartDate(args[0], args[1], startDate)
			if err != nil {
				return err
			}
			return output.Print(cmd, member)
		},
	}
	setStart.Flags().Int64("start-date", 0, "New start date (Unix ms timestamp, required)")
	internal.MustMarkRequired(setStart, "start-date")
	cmd.AddCommand(list, add, setStart)
	return cmd
}

func newGanttSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "settings", Short: "Manage Gantt chart settings"}
	update := &cobra.Command{
		Use:   "update <ganttId>",
		Short: "Update settings of a Gantt chart",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			showWeekends, _ := cmd.Flags().GetBool("show-weekends")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			settings := map[string]any{"showWeekends": showWeekends}
			result, err := client.UpdateGanttSettings(args[0], settings)
			if err != nil {
				return err
			}
			return output.Print(cmd, result)
		},
	}
	update.Flags().Bool("show-weekends", false, "Show weekends on the chart")
	cmd.AddCommand(update)
	return cmd
}
