package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal/output"
	"github.com/JetBrains/youtrack-cli/internal"
)

func newBoardCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "board", Short: "Manage agile boards and sprints"}
	cmd.AddCommand(
		newBoardListCmd(),
		newBoardGetCmd(),
		newBoardAddCmd(),
		newBoardDeleteCmd(),
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

func newBoardGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <id>", Short: "Get a board by ID",
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
		Use: "add", Short: "Create a new board",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			board, err := client.AddBoard(name)
			if err != nil {
				return err
			}
			return output.Print(cmd, board)
		},
	}
	cmd.Flags().String("name", "", "Board name (required)")
	cmd.MarkFlagRequired("name")
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
			fmt.Fprintln(cmd.OutOrStdout(), "deleted board "+args[0])
			return nil
		},
	}
}

func newBoardSprintCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "sprint", Short: "Manage sprints"}
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
	add.MarkFlagRequired("board")
	add.MarkFlagRequired("name")
	cmd.AddCommand(add)
	return cmd
}
