package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <issueId>",
		Short: "Update issue summary or description",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, _ := cmd.Flags().GetString("summary")
			description, descriptionSet, err := textFromFlags(cmd, "description", "description-file")
			if err != nil {
				return err
			}

			if !cmd.Flags().Changed("summary") && !descriptionSet {
				return fmt.Errorf("at least one of --summary, --description or --description-file must be provided")
			}
			req := api.UpdateIssueRequest{}
			if cmd.Flags().Changed("summary") {
				req.Summary = &summary
			}
			if descriptionSet {
				req.Description = &description
			}

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			issue, err := client.UpdateIssue(args[0], req)
			if err != nil {
				return err
			}
			return output.Print(cmd, issue)
		},
	}
	cmd.Flags().String("summary", "", "New summary")
	cmd.Flags().String("description", "", "New description (an empty string clears it)")
	addTextFileFlag(cmd, "description", "description-file", "the new description")
	return cmd
}
