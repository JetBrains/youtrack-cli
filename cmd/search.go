package cmd

import (
	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Manage saved searches",
	}
	cmd.AddCommand(newSearchAddCmd())
	cmd.AddCommand(newSearchShareCmd())
	return cmd
}

func newSearchAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new saved search",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			query, _ := cmd.Flags().GetString("query")

			client, err := getClient(cmd)
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
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("query")
	return cmd
}

func newSearchShareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "share <id>",
		Short: "Mark a saved search as shareable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return client.ShareSavedSearch(args[0])
		},
	}
}
