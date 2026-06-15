package cmd

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newBoardCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "board", Short: "Manage agile boards and sprints"}
	cmd.AddCommand(
		newBoardListCmd(),
		newBoardGetCmd(),
		newBoardViewCmd(),
		newBoardAddCmd(),
		newBoardCreateCmd(),
		newBoardDeleteCmd(),
		newBoardUpdateCmd(),
		newBoardSprintCmd(),
	)
	return cmd
}

func newBoardViewCmd() *cobra.Command {
	c := newBoardGetCmd()
	c.Use = "view <id>"
	c.Short = "View a board by ID"
	return c
}

func newBoardCreateCmd() *cobra.Command {
	c := newBoardAddCmd()
	c.Use = "create"
	c.Short = "Create a new board"
	return c
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

func newBoardGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <id>", Short: "Get a board by ID (alias: view)",
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

func newBoardAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "add", Short: "Create a new board (alias: create)",
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
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a board",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
	get := &cobra.Command{
		Use: "get <sprint-id>", Short: "Get a sprint by ID",
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
	get.Flags().String("board", "", "Board ID (required)")
	internal.MustMarkRequired(get, "board")
	cmd.AddCommand(get)
	add := &cobra.Command{
		Use: "add", Short: "Add a sprint to a board",
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
	add.Flags().String("board", "", "Board ID (required)")
	add.Flags().String("name", "", "Sprint name (required)")
	add.Flags().Int64("start", 0, "Start date (epoch ms)")
	add.Flags().Int64("end", 0, "End date (epoch ms)")
	add.Flags().String("goal", "", "Sprint goal")
	internal.MustMarkRequired(add, "board", "name")
	cmd.AddCommand(add)
	return cmd
}
