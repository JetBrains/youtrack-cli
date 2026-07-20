package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newBoardCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "board", Short: "Manage agile boards and sprints"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(
		newBoardListCmd(),
		newBoardViewCmd(),
		newBoardCreateCmd(),
		newBoardDeleteCmd(),
		newBoardUpdateCmd(),
		newBoardSprintCmd(),
	)
	return cmd
}

func newBoardListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List all boards",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			boards, err := client.ListBoards()
			if err != nil {
				return err
			}
			return output.Print(cmd, boards)
		},
	}
}

func newBoardViewCmd() *cobra.Command {
	return &cobra.Command{
		Use: "view <id>", Aliases: []string{"get"}, Short: "View a board by ID",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			board, err := client.GetBoard(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, board)
		},
	}
}

func newBoardCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "create", Aliases: []string{"add"}, Short: "Create a new board",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			project, _ := cmd.Flags().GetString("project")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			board, err := client.AddBoard(name, project)
			if err != nil {
				return err
			}
			return output.Print(cmd, board)
		},
	}
	cmd.Flags().String("name", "", "Board name (required)")
	cmd.Flags().String("project", "", "Project short name to attach the board to (required)")
	internal.MustMarkRequired(cmd, "name")
	internal.MustMarkRequired(cmd, "project")
	return cmd
}

func newBoardDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a board",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete board %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteBoard(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted board "+args[0])
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

func newBoardUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "update <board-id>", Short: "Update a board",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			board, err := client.UpdateBoard(args[0], name)
			if err != nil {
				return err
			}
			return output.Print(cmd, board)
		},
	}
	cmd.Flags().String("name", "", "New board name")
	return cmd
}

func newBoardSprintCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "sprint", Short: "Manage sprints"}
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use: "list", Short: "List all sprints for a board",
		RunE: func(cmd *cobra.Command, args []string) error {
			boardID, _ := cmd.Flags().GetString("board")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			sprints, err := client.ListSprints(boardID)
			if err != nil {
				return err
			}
			return output.Print(cmd, sprints)
		},
	}
	list.Flags().String("board", "", "Board ID (required)")
	internal.MustMarkRequired(list, "board")
	cmd.AddCommand(list)
	view := &cobra.Command{
		Use: "view <sprint-id>", Aliases: []string{"get"}, Short: "View a sprint by ID",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			boardID, _ := cmd.Flags().GetString("board")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			sprint, err := client.GetSprint(boardID, args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, sprint)
		},
	}
	view.Flags().String("board", "", "Board ID (required)")
	internal.MustMarkRequired(view, "board")
	cmd.AddCommand(view)
	create := &cobra.Command{
		Use: "create", Aliases: []string{"add"}, Short: "Create a sprint on a board",
		RunE: func(cmd *cobra.Command, args []string) error {
			boardID, _ := cmd.Flags().GetString("board")
			name, _ := cmd.Flags().GetString("name")
			start, _ := cmd.Flags().GetInt64("start")
			end, _ := cmd.Flags().GetInt64("end")
			goal, _ := cmd.Flags().GetString("goal")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			sprint, err := client.AddSprint(boardID, name, start, end, goal)
			if err != nil {
				return err
			}
			return output.Print(cmd, sprint)
		},
	}
	create.Flags().String("board", "", "Board ID (required)")
	create.Flags().String("name", "", "Sprint name (required)")
	create.Flags().Int64("start", 0, "Start date (epoch ms)")
	create.Flags().Int64("end", 0, "End date (epoch ms)")
	create.Flags().String("goal", "", "Sprint goal")
	internal.MustMarkRequired(create, "board", "name")
	cmd.AddCommand(create)
	return cmd
}
