package issue

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"add"},
		Short:   "Create a new issue",
		Example: "  youtrack issue create --project FOO --summary \"Login button broken\"\n  youtrack issue create --project FOO --summary \"Fix\" --custom-fields '{\"State\":\"Fixed\"}'",
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

			// Parse before creating so bad input never leaves a stray, partially-fielded issue.
			var customFieldsCmd string
			if customFields != "" {
				q, err := jsonToCommand(customFields)
				if err != nil {
					return fmt.Errorf("parse custom-fields: %w", err)
				}
				customFieldsCmd = q
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

			// Custom fields and parent apply as follow-up commands; roll back the issue if either fails so create stays atomic.
			var applyErr error
			if customFieldsCmd != "" {
				if err := client.ApplyCommand(customFieldsCmd, []string{issue.IDReadable}); err != nil {
					applyErr = fmt.Errorf("apply custom fields: %w", err)
				}
			}
			if applyErr == nil && parent != "" {
				if err := client.ApplyCommand("parent for "+parent, []string{issue.IDReadable}); err != nil {
					applyErr = fmt.Errorf("set parent: %w", err)
				}
			}
			if applyErr != nil {
				if delErr := client.DeleteIssue(issue.IDReadable); delErr != nil {
					return fmt.Errorf("%w; issue %s was created but could not be rolled back: %w", applyErr, issue.IDReadable, delErr)
				}
				return fmt.Errorf("%w; created issue %s was rolled back", applyErr, issue.IDReadable)
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

// jsonToCommand converts {"State":"Fixed"} into a sorted, deterministic YouTrack command; user logins are braced downstream by api.WrapUserValues.
func jsonToCommand(jsonStr string) (string, error) {
	var fields map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &fields); err != nil {
		return "", fmt.Errorf(`%w (custom-fields must be a flat JSON object of string values, e.g. {"Assignee":"alice.qa"})`, err)
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(fields)*2)
	for _, k := range keys {
		parts = append(parts, k, fields[k])
	}
	return strings.Join(parts, " "), nil
}
