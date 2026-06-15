package cmd

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Manage saved searches",
	}
	cmd.AddCommand(newSearchListCmd())
	cmd.AddCommand(newSearchGetCmd())
	cmd.AddCommand(newSearchAddCmd())
	cmd.AddCommand(newSearchDeleteCmd())
	cmd.AddCommand(newSearchShareCmd())
	return cmd
}

func newSearchListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved searches",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			results, err := client.ListSavedSearches()
			if err != nil {
				return err
			}
			return output.Print(cmd, results)
		},
	}
}

func newSearchGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a saved search by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			result, err := client.GetSavedSearch(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, result)
		},
	}
}

func newSearchDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a saved search by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteSavedSearch(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted saved search "+args[0])
			return nil
		},
	}
}

func newSearchAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new saved search",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			query, _ := cmd.Flags().GetString("query")

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			result, err := client.AddSavedSearch(api.AddSavedSearchRequest{
				Name:  name,
				Query: query,
			})
			if err != nil {
				return err
			}
			return output.Print(cmd, result)
		},
	}
	cmd.Flags().String("name", "", "Saved search name (required)")
	cmd.Flags().String("query", "", "YouTrack query string (required)")
	internal.MustMarkRequired(cmd, "name", "query")
	return cmd
}

func newSearchShareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "share <id>",
		Short: "Mark a saved search as shareable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.ShareSavedSearch(args[0])
		},
	}
}
