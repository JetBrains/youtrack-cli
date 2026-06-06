package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueDraftCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "draft", Short: "Manage issue drafts"}
	cmd.AddCommand(newIssueDraftCreateCmd())
	return cmd
}

func newIssueDraftCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an issue draft",
		RunE: func(cmd *cobra.Command, args []string) error {
			project, _ := cmd.Flags().GetString("project")
			summary, _ := cmd.Flags().GetString("summary")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			draft, err := client.CreateIssueDraft(project, summary)
			if err != nil {
				return err
			}
			return output.Print(cmd, draft)
		},
	}
	cmd.Flags().String("project", "", "Project short name (required)")
	cmd.Flags().String("summary", "", "Draft summary (required)")
	internal.MustMarkRequired(cmd, "project", "summary")
	return cmd
}
