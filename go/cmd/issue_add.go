package cmd

import (
	"youtrack-cli/api"
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newIssueAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectShortName, _ := cmd.Flags().GetString("project")
			summary, _ := cmd.Flags().GetString("summary")
			description, _ := cmd.Flags().GetString("description")
			asUser, _ := cmd.Flags().GetString("asUser")

			client, err := getClient(cmd)
			if err != nil {
				return err
			}

			project, err := client.GetProject(projectShortName)
			if err != nil {
				return err
			}

			req := api.AddIssueRequest{
				Summary:     summary,
				Description: description,
				Project:     &api.Project{ID: project.ID},
			}

			if asUser != "" {
				user, err := client.GetUser(asUser)
				if err != nil {
					return err
				}
				req.Reporter = &api.User{ID: user.ID}
			}

			issue, err := client.AddIssue(req)
			if err != nil {
				return err
			}
			return output.Print(cmd, issue)
		},
	}
	cmd.Flags().String("project", "", "Project short name (required)")
	cmd.Flags().String("summary", "", "Issue summary (required)")
	cmd.Flags().String("description", "", "Issue description")
	cmd.Flags().String("asUser", "", "Create as this user (login)")
	cmd.MarkFlagRequired("project")
	cmd.MarkFlagRequired("summary")
	return cmd
}
