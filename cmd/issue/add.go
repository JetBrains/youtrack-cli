package issue

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new issue (alias: create)",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectShortName, _ := cmd.Flags().GetString("project")
			summary, _ := cmd.Flags().GetString("summary")
			description, _ := cmd.Flags().GetString("description")
			descriptionFile, _ := cmd.Flags().GetString("description-file")
			customFields, _ := cmd.Flags().GetString("custom-fields")
			customFieldsFile, _ := cmd.Flags().GetString("custom-fields-file")
			parent, _ := cmd.Flags().GetString("parent")
			asUser, _ := cmd.Flags().GetString("asUser")

			if descriptionFile != "" && description != "" {
				return fmt.Errorf("--description and --description-file are mutually exclusive")
			}
			if descriptionFile != "" {
				b, err := os.ReadFile(descriptionFile)
				if err != nil {
					return fmt.Errorf("read description file: %w", err)
				}
				description = string(b)
			}

			if customFieldsFile != "" && customFields != "" {
				return fmt.Errorf("--custom-fields and --custom-fields-file are mutually exclusive")
			}
			if customFieldsFile != "" {
				b, err := os.ReadFile(customFieldsFile)
				if err != nil {
					return fmt.Errorf("read custom-fields file: %w", err)
				}
				customFields = string(b)
			}

			client, err := internal.GetClient(cmd)
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

			if customFields != "" {
				cmdQuery, err := jsonToCommand(customFields)
				if err != nil {
					return fmt.Errorf("parse custom-fields: %w", err)
				}
				if err := client.ApplyCommand(cmdQuery, []string{issue.IDReadable}); err != nil {
					return fmt.Errorf("apply custom fields: %w", err)
				}
			}

			if parent != "" {
				if err := client.ApplyCommand("parent for "+parent, []string{issue.IDReadable}); err != nil {
					return fmt.Errorf("set parent: %w", err)
				}
			}

			return output.Print(cmd, issue)
		},
	}
	cmd.Flags().String("project", "", "Project short name (required)")
	cmd.Flags().String("summary", "", "Issue summary (required)")
	cmd.Flags().String("description", "", "Issue description")
	cmd.Flags().String("description-file", "", "Read description from file")
	cmd.Flags().String("custom-fields", "", `JSON object of custom field values, e.g. '{"State":"Fixed"}'`)
	cmd.Flags().String("custom-fields-file", "", "Read custom fields JSON from file")
	cmd.Flags().String("parent", "", "Parent issue ID (e.g. FOO-1)")
	cmd.Flags().String("asUser", "", "Create as this user (login)")
	internal.MustMarkRequired(cmd, "project", "summary")
	return cmd
}

// jsonToCommand converts {"State":"Fixed","Priority":"High"} → "State Fixed Priority High".
func jsonToCommand(jsonStr string) (string, error) {
	var fields map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &fields); err != nil {
		return "", err
	}
	parts := make([]string, 0, len(fields)*2)
	for k, v := range fields {
		parts = append(parts, k, v)
	}
	return strings.Join(parts, " "), nil
}
