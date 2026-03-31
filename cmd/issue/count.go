package issue

import (
	"fmt"

	"github.com/JetBrains/youtrack-cli/internal"

	"github.com/spf13/cobra"
)

func newIssueCountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "count",
		Short: "Count issues matching a query",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			count, err := client.GetIssueCount(query)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), count)
			return nil
		},
	}
	cmd.Flags().String("query", "", "Filter query")
	return cmd
}
